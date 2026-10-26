#!/usr/bin/env bash
# ==============================================================================
# iUpdater Agent 启动脚本
# 用法: ./run.sh {start|stop|restart|status|check}
#
# 目录约定: 把本脚本和 agent-linux-* 二进制、config.json 放在同一目录
#   /opt/myapp/
#   ├── run.sh                 ← 本脚本
#   ├── agent-linux-amd64      ← 重命名为 agent 即可 (或让脚本自动选择)
#   ├── config.json            ← 用户配置
#   ├── logs/agent.log         ← 运行日志 (自动创建)
#   └── .agent.pid             ← PID 文件 (自动创建)
#
# systemd 集成 (可选):
#   cp run.sh /usr/local/bin/iupdater-agent.sh
#   # /etc/systemd/system/iupdater-agent.service:
#   # [Unit]
#   # Description=iUpdater Agent
#   # After=network.target
#   # [Service]
#   # Type=forking
#   # User=root
#   # WorkingDirectory=/opt/myapp
#   # ExecStart=/usr/local/bin/iupdater-agent.sh start
#   # ExecStop=/usr/local/bin/iupdater-agent.sh stop
#   # PIDFile=/opt/myapp/.agent.pid
#   # [Install]
#   # WantedBy=multi-user.target
#   systemctl daemon-reload
#   systemctl enable iupdater-agent
#   systemctl start iupdater-agent
# ==============================================================================

set -e

# 脚本所在目录（绝对路径）
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

PID_FILE="$SCRIPT_DIR/.agent.pid"
LOG_DIR="$SCRIPT_DIR/logs"
LOG_FILE="$LOG_DIR/agent.log"

mkdir -p "$LOG_DIR"

# 彩色输出（终端才生效）
if [ -t 1 ]; then
    C_GREEN=$'\e[32m'; C_RED=$'\e[31m'; C_YELLOW=$'\e[33m'; C_BOLD=$'\e[1m'; C_RESET=$'\e[0m'
else
    C_GREEN=""; C_RED=""; C_YELLOW=""; C_BOLD=""; C_RESET=""
fi

log()  { echo    "${C_BOLD}[run.sh]${C_RESET} $*"; }
ok()   { echo    "${C_GREEN}✓${C_RESET} $*"; }
warn() { echo    "${C_YELLOW}!${C_RESET} $*"; }
fail() { echo    "${C_RED}✗${C_RESET} $*" >&2; }

# ---- 自动选择可用的 agent 二进制 ----
# 优先级：有执行权限的 > 只有读权限的（自动帮用户加 +x）
pick_bin() {
    local -a candidates=(
        "./agent"
        "./agent-linux-amd64"
        "./agent-linux-arm64"
        "./agent-linux-x86_64"
        "./agent.exe"
    )
    local c
    # 第一阶段：挑有执行权限的
    for c in "${candidates[@]}"; do
        if [ -x "$c" ]; then
            echo "$c"
            return 0
        fi
    done
    # 第二阶段：找存在的文件，自动补 +x
    for c in "${candidates[@]}"; do
        if [ -f "$c" ]; then
            warn "$c 没有执行权限，自动 chmod +x"
            chmod +x "$c" 2>/dev/null || true
            echo "$c"
            return 0
        fi
    done
    return 1
}
AGENT_BIN=$(pick_bin) || true

# 检查 config.json 是否存在
check_config() {
    if [ ! -f "./config.json" ]; then
        fail "config.json 不存在于 $(pwd)"
        fail "  复制一份 config.example.json 为 config.json 并填写参数"
        return 1
    fi
    return 0
}

# 检查二进制是否可用
check_bin() {
    if [ -z "$AGENT_BIN" ]; then
        fail "当前目录下找不到 agent 二进制"
        warn "可用文件名: agent / agent-linux-amd64 / agent-linux-arm64"
        warn "如果是 PowerShell 上传的文件，可能需要先上传到 Linux 再 chmod +x"
        ls -1 ./agent* 2>/dev/null || true
        return 1
    fi
    if [ ! -x "$AGENT_BIN" ]; then
        # pick_bin 里已经尝试加过了（例如只读系统盘），再试一次
        chmod +x "$AGENT_BIN" 2>/dev/null || true
    fi
    if [ ! -x "$AGENT_BIN" ]; then
        fail "$AGENT_BIN 没有执行权限"
        warn "手动执行: chmod +x $AGENT_BIN"
        return 1
    fi
    ok "使用二进制: $AGENT_BIN"
    return 0
}

pid_running() {
    [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null
}

# ---- 子命令 ----
do_start() {
    log "启动 iUpdater Agent..."
    check_config || return 1
    check_bin    || return 1

    if pid_running; then
        warn "agent 已经在运行 (PID $(cat "$PID_FILE"))"
        return 0
    fi

    # 后台启动，标准输出/错误追加到日志
    nohup "$AGENT_BIN" --config "./config.json" >> "$LOG_FILE" 2>&1 &
    local pid=$!
    echo "$pid" > "$PID_FILE"
    sleep 1
    if kill -0 "$pid" 2>/dev/null; then
        ok "启动成功 PID=$pid"
        ok "日志: tail -f $LOG_FILE"
        ok "面板: 浏览器打开 http://127.0.0.1:18080"
    else
        fail "启动后立即退出，查看日志:"
        tail -n 20 "$LOG_FILE" 2>/dev/null || true
        rm -f "$PID_FILE"
        return 1
    fi
}

do_stop() {
    log "停止 iUpdater Agent..."
    if ! pid_running; then
        warn "没有运行中的 agent"
        rm -f "$PID_FILE"
        return 0
    fi
    local pid
    pid=$(cat "$PID_FILE")
    kill "$pid" 2>/dev/null || true
    # 等待最多 10s 优雅退出
    for i in $(seq 1 20); do
        sleep 0.5
        if ! kill -0 "$pid" 2>/dev/null; then
            ok "已停止 (PID $pid)"
            rm -f "$PID_FILE"
            return 0
        fi
    done
    # 强杀
    warn "优雅退出超时，强制 kill -9"
    kill -9 "$pid" 2>/dev/null || true
    rm -f "$PID_FILE"
    ok "已强制停止"
}

do_restart() {
    do_stop
    sleep 1
    do_start
}

do_status() {
    printf '%-20s %s\n' "二进制:"   "$AGENT_BIN"
    printf '%-20s %s\n' "配置文件:" "$( [ -f "./config.json" ] && echo "存在" || echo "缺失!")"
    printf '%-20s %s\n' "日志文件:" "$LOG_FILE"
    if pid_running; then
        local pid=$(cat "$PID_FILE")
        local ppid
        ppid=$(ps -o ppid= -p "$pid" 2>/dev/null | tr -d ' ')
        printf '%-20s %s\n' "运行状态:" "${C_GREEN}运行中${C_RESET}"
        printf '%-20s %s\n' "PID:"     "$pid"
        printf '%-20s %s\n' "PPID:"    "$ppid"
        printf '%-20s %s\n' "面板:"    "http://127.0.0.1:18080"
        printf '%-20s %s\n' "日志大小:" "$(du -h "$LOG_FILE" 2>/dev/null | cut -f1)"
    else
        printf '%-20s %s\n' "运行状态:" "${C_RED}未运行${C_RESET}"
    fi
}

do_check() {
    log "配置 + 连通性检查..."
    check_config || return 1
    check_bin    || return 1

    echo
    log "=== 1. config.json 语法 + 必填项 ==="
    # JSON 语法校验
    if command -v jq >/dev/null 2>&1; then
        if ! jq empty "./config.json" >/dev/null 2>&1; then
            fail "config.json JSON 语法错误"
            return 1
        fi
        ok "JSON 语法合法"
        # 必填项
        local server_url app_id access_key
        server_url=$(jq -r '.server_url // ""'  ./config.json)
        app_id=$(jq      -r '.app_id     // ""'  ./config.json)
        access_key=$(jq  -r '.access_key // ""'  ./config.json)
        [ "$server_url" != "" ] && ok "server_url = $server_url"  || fail "server_url 不能为空"
        [ "$app_id"     != "" ] && ok "app_id     = $app_id"      || fail "app_id 不能为空"
        [ "$access_key" != "" ] && ok "access_key 已填写 (${#access_key} 字符)" || warn "access_key 为空 (将匿名 checkin)"
    else
        warn "未安装 jq，跳过 JSON 语法校验 (apt install jq / yum install jq)"
    fi

    echo
    log "=== 2. 主控后端连通性 ==="
    local server_url
    server_url=$(python3 -c "import json; print(json.load(open('config.json')).get('server_url',''))" 2>/dev/null || echo "")
    if [ "$server_url" = "" ]; then
        fail "读不到 server_url"
        return 1
    fi
    local code
    code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$server_url/api/login/config")
    if [ "$code" = "200" ] || [ "$code" = "404" ]; then
        ok "主控可达 (HTTP $code)"
    else
        fail "主控不可达 (HTTP $code)，检查 server_url / 防火墙"
        return 1
    fi

    echo
    log "=== 3. 本地面板端口占用 ==="
    local api_port
    api_port=$(python3 -c "import json; print(json.load(open('config.json')).get('api_port', 18080))" 2>/dev/null || echo "18080")
    if ss -tln 2>/dev/null | grep -q ":$api_port " || netstat -tln 2>/dev/null | grep -q ":$api_port "; then
        warn "端口 $api_port 已被占用 (本地面板无法启动)"
    else
        ok "端口 $api_port 空闲"
    fi

    echo
    log "=== 4. 目标目录与脚本 ==="
    local target_dir
    target_dir=$(python3 -c "import json; print(json.load(open('config.json')).get('target_dir','./app'))" 2>/dev/null || echo "./app")
    [ -d "$target_dir" ] && ok "target_dir 存在: $target_dir" || warn "target_dir 不存在 (首次安装正常)"

    # 检查 stop_command / start_command 里引用的第一个文件是否存在
    local cmd_first
    for field in stop_command start_command stop_script start_script; do
        cmd_first=$(python3 -c "
import json
c = json.load(open('config.json'))
v = c.get('$field','') or c.get('${field}_command','') or c.get('${field%_script}_command','')
# 兼容旧字段
if not v and '$field' == 'stop_script': v = c.get('stop_command','')
if not v and '$field' == 'start_script': v = c.get('start_command','')
import shlex
parts = shlex.split(v) if v else []
print(parts[0] if parts else '')
" 2>/dev/null || echo "")
        if [ -n "$cmd_first" ] && [[ "$cmd_first" == ./* || "$cmd_first" == /* ]]; then
            if [ -e "$cmd_first" ]; then
                ok "  $field:  $cmd_first ✓"
            else
                warn "  $field:  $cmd_first 不存在 (命令执行时会失败)"
            fi
        fi
    done

    echo
    ok "=== 检查完成，运行 $0 start 启动 agent ==="
}

# ---- 分发 ----
case "${1:-}" in
    start)    do_start  ;;
    stop)     do_stop   ;;
    restart)  do_restart ;;
    status)   do_status ;;
    check)    do_check  ;;
    *)
        echo "用法: $0 {start|stop|restart|status|check}"
        exit 1
        ;;
esac
