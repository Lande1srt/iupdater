# iUpdater Agent — 辅助更新助手

iUpdater 的跨主机自动更新客户端。**Go 单二进制，零外部依赖**，可直接丢到任何 Linux / Windows 服务器上跑。

主控后端发布新版本 → agent 主动 checkin → 下载 → 停服 → 覆盖 → 启动，全程自动。

---

## 它能做什么

- ✅ **主动轮询主控**：可配置间隔（默认 60s）checkin 后端，发现新版本自动拉取
- ✅ **完整 shell 命令**：`stop_command` / `start_command` 支持任意终端命令（1Panel 管理脚本、systemctl、docker、pkill、后台 nohup 等）
- ✅ **SHA256 校验**：主控下发的文件 hash 与本地计算不匹配直接中止
- ✅ **自动备份 + 回滚**：覆盖前备份 target_dir，任何步骤失败自动还原；保留最近 5 份
- ✅ **本地状态面板**：`http://127.0.0.1:18080` 实时显示状态、进度、最近 200 行日志
- ✅ **主控远程控制**：主控后台可下发 `check_now`（立即检查）、`restart`、`heartbeat`（探活），agent 下次 poll 自动执行
- ✅ **远程配置 merge**：主控下发 `config_patch`，agent 自动合并写入本地 config.json 并生效（无需登录服务器改文件）
- ✅ **跨平台**：Linux amd64 / arm64、Windows，全静态编译（CGO_ENABLED=0）

## 不能做什么

- ❌ 不走 WebSocket / 长连接 — 保持极简，所有指令在 checkin 里双向传递
- ❌ 不能真正自重启（Windows 进程不能 self-exec，Linux 建议用 systemd / NSSM 包装）
- ❌ 不做真正的"覆盖中还能用" — 停服 → 覆盖 → 启动，这是最可靠的模型

## 编译产物

仓库提供交叉编译好的二进制，直接拿：

| 文件 | 平台 | 说明 |
|---|---|---|
| `agent-linux-amd64` | Linux x86_64 | 推荐大多数服务器 |
| `agent-linux-arm64` | Linux ARM64 | 树莓派 / 国产 ARM 服务器 |
| `agent.exe` | Windows | WSL 兼容 |

重新编译：

```bash
# Linux amd64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o agent-linux-amd64 .
# Linux arm64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o agent-linux-arm64 .
# Windows
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o agent.exe .
```

验证器（独立子模块）：

```bash
cd check
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o agent-check-linux-amd64 .
```

## 快速开始（Linux）

```bash
# 1. 拷贝到目标服务器
scp agent-linux-amd64 run.sh config.example.json server:/opt/myapp/

# 2. 重命名 + 权限
cd /opt/myapp
mv agent-linux-amd64 agent
cp config.example.json config.json
chmod +x agent run.sh

# 3. 编辑 config.json（access_key 从主控后台「助手管理 → 注册助手」获取）
vim config.json

# 4. 先验一下配置 + 连通性
./agent-check-linux-amd64 ./config.json
# 或
./run.sh check

# 5. 启动
./run.sh start
./run.sh status          # 查看状态
# 本地面板：浏览器打开 http://127.0.0.1:18080
```

`run.sh` 支持的子命令：

| 命令 | 说明 |
|---|---|
| `./run.sh start` | nohup 后台启动，PID 文件 `.agent.pid`，日志 `logs/agent.log` |
| `./run.sh stop` | 10s 优雅退出 → 超时 `kill -9` |
| `./run.sh restart` | stop + start |
| `./run.sh status` | 查看 PID / 运行状态 / 日志大小 / 面板地址 |
| `./run.sh check` | **推荐首次必跑** — 检查 config.json 语法 + 必填项 + 主控连通性 + 真实 checkin + 本地脚本可执行性 |

## 完整更新流程

```
┌──────────────────────────────────────────────────────────────────┐
│ 轮询 checkin → 主控返回 {task_id, download_url, file_hash, ...}  │
│                      同时可能包含                                │
│                       remote_command  (check_now / restart / ...)│
│                       config_patch    (远程配置覆盖)              │
├──────────────────────────────────────────────────────────────────┤
│ 1. 收到 config_patch → 原子 merge 到本地 config.json.tmp → rename │
│                                                                  │
│ 2. 收到 remote_command  → check_now: 2s 后再 poll               │
│                       restart: 延迟执行（外部 supervisor 接管）    │
│                       heartbeat: 无操作                          │
│                                                                  │
│ 3. 下载 zip（主控可能返回 S3 预签名 / CDN 直连 / 主控代理 URL）    │
│                                                                  │
│ 4. SHA256 校验（主控提供了 file_hash 时）                          │
│                                                                  │
│ 5. stop_command                                                  │
│    - 超时 30s                                                    │
│    - 失败 → 中止更新，不上报 success，继续运行                     │
│                                                                  │
│ 6. 备份 target_dir → backup_dir/{YYYYMMDD-HHMMSS}/              │
│                                                                  │
│ 7. 解压到临时目录 → os.Rename 原子覆盖 target_dir                 │
│    - 任何步骤失败 → 自动从备份还原 target_dir                     │
│                                                                  │
│ 8. start_command                                                 │
│    - 超时 30s                                                     │
│    - 失败不回滚但上报 fail（文件已替换）                           │
│                                                                  │
│ 9. 清理旧备份（保留最近 5 个）                                    │
│                                                                  │
│ 10. 上报 success / fail / message 到主控                          │
└──────────────────────────────────────────────────────────────────┘
```

## stop_command / start_command 完整终端命令示例

agent 通过 `/bin/sh -c "<cmd>"` 执行命令，支持任意 shell 语法。第一个词如果是相对 / 绝对路径会自动解析为 agent 自身目录下的绝对路径。

```jsonc
{
  // 1Panel 面板管理脚本
  "stop_command":  "/in1panel/go/linux-amd64/run.sh stop",
  "start_command": "/in1panel/go/linux-amd64/run.sh start",

  // systemd 服务
  "stop_command":  "systemctl stop myapp",
  "start_command": "systemctl start myapp",

  // init.d 脚本
  "stop_command":  "/etc/init.d/myapp stop",

  // Docker 容器
  "stop_command":  "docker stop mycontainer",
  "start_command": "docker start mycontainer",

  // Windows .bat
  "stop_command":  "./stop.bat",

  // 自建进程：pkill + 后台启动
  "stop_command":  "pkill -f myapp; sleep 2",
  "start_command": "nohup ./myapp > /dev/null 2>&1 &",

  // 多命令串联
  "stop_command":  "pkill -f myapp; sleep 3; echo stopped",
  "start_command": "cd /opt/myapp && ./myapp --daemon"
}
```

**超时**：每个命令 30 秒 exec 超时。工作目录固定为 agent 自身目录。

## 主控远程控制（无需登录服务器）

主控后台「助手管理 → 某行的 ⚙」弹窗可以下发两种东西：

### 控制指令（command）

| 值 | agent 行为 |
|---|---|
| `check_now` | 跳过等待间隔，2 秒内再 poll 一次 |
| `restart` | 3 秒后尝试重 exec 自身（Linux，Windows 不生效） |
| `heartbeat` | 无操作，主控只是探活 |

### 远程配置覆盖（config_patch）

```json
{
  "poll_interval_seconds": 30,
  "stop_command":  "./run.sh stop",
  "start_command": "./run.sh start"
}
```

agent 下一次 checkin 时领取，自动 merge 到本地 `config.json` 并原子写入（`.tmp` → `rename`，不会写坏）。只有 patch 里非空的字段会覆盖；本地已有的其他字段保留。

## checkin 请求体（agent → 主控）

```json
POST /api/agent/checkin
Authorization: Bearer <agent_access_key>
Content-Type: application/json

{
  "app_id":          "com.test.myapp",
  "channel":         "stable",
  "platform":        "linux",
  "architecture":    "amd64",
  "current_version": "1.2.3",
  "current_code":    202610130001,
  "status": {
    "state":          "idle",
    "current_version": "1.2.3",
    "progress":       0,
    "hostname":       "prod-server-01",
    "cfg_poll_sec":   60
  }
}
```

## checkin 响应体（主控 → agent）

可能同时包含任务和远程控制指令：

```json
{
  "task_id":       "a1b2c3d4",
  "version_name":  "1.2.4",
  "version_code":  202610130002,
  "download_url":  "https://cdn.example.com/app-1.2.4.zip",
  "file_hash":     "sha256:abc123...",
  "file_size":     12345678,

  "remote_command": "check_now",
  "command_at":     "2026-10-13T10:20:30Z",

  "config_patch": {
    "poll_interval_seconds": 30
  }
}
```

没有更新任务时 task_id 为空或整个响应是空对象 `{}`。

## 本地状态面板

启动后浏览器访问 `http://127.0.0.1:18080`：

- 实时状态徽章（空闲 / 检查中 / 下载中 / 更新中 / ✓ 成功 / ✗ 失败）
- 下载进度条（正在下载时）
- 当前版本号、上次更新时间、下次检查倒计时
- 手动「立即检查」按钮
- 最近 200 行实时日志（滚动刷新）

面板地址在 config.json 的 `api_port` 字段，默认 `18080`。

## systemd 开机自启（Linux）

`run.sh` 顶部注释里有完整模板，复制即可：

```bash
sudo cp run.sh /usr/local/bin/iupdater-agent.sh

# 创建 /etc/systemd/system/iupdater-agent.service：
sudo tee /etc/systemd/system/iupdater-agent.service <<'EOF'
[Unit]
Description=iUpdater Agent
After=network.target

[Service]
Type=forking
User=root
WorkingDirectory=/opt/myapp
ExecStart=/usr/local/bin/iupdater-agent.sh start
ExecStop=/usr/local/bin/iupdater-agent.sh stop
PIDFile=/opt/myapp/.agent.pid

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now iupdater-agent
sudo systemctl status iupdater-agent
```

## 目录约定

```
/opt/myapp/                     ← agent 自身目录 (run.sh 的工作目录)
├── agent                       二进制（chmod +x）
├── run.sh                      启动脚本（可选，但推荐）
├── config.json                 配置文件（必填）
├── logs/
│   └── agent.log               运行日志（自动创建）
├── .agent.pid                  PID 文件（run.sh start 写）
├── app/                        ← target_dir：目标程序安装目录
│   ├── myapp
│   ├── run.sh
│   └── ...
├── download/                   ← 下载临时目录
└── backup/                     ← 自动备份目录（保留最近 5 个）
    ├── 20261013-100500/
    └── 20261013-110000/
```

## 配置字段速查

完整字段与示例见 [CONFIG.md](CONFIG.md)。高频字段：

| 字段 | 必填 | 默认 | 说明 |
|---|---|---|---|
| `server_url` | ✅ | — | 主控后端地址 |
| `access_key` | ✅ | — | 主控后台「助手管理 → 注册助手」生成 |
| `app_id` | ✅ | — | 绑定的应用 |
| `poll_interval_seconds` | ❌ | `60` | checkin 间隔 |
| `target_dir` | ❌ | `./app` | 目标程序安装目录 |
| `backup_dir` | ❌ | `./backup` | 备份目录 |
| `download_dir` | ❌ | `./download` | 下载临时目录 |
| `stop_command` | ❌ | 空（跳过） | 完整停止命令 |
| `start_command` | ❌ | 空（跳过） | 完整启动命令 |
| `api_port` | ❌ | `18080` | 本地状态面板端口 |

## 疑难排查

| 现象 | 原因 | 解决 |
|---|---|---|
| checkin 返回 401 | access_key 写错 / 主控重置了 | 主控后台编辑 agent → 重置 access_key → 更新 config.json |
| 主控不可达 / 超时 | server_url 错 / 防火墙 / 主控挂了 | `./run.sh check` 验证；curl 一下 server_url |
| start_command 返回 "权限不足" | 脚本缺执行权限 | `chmod +x <脚本>`，或直接在 config.json 里用 `sh ./run.sh start` 绕过 |
| 更新后 start_command 失败但文件已替换 | 文件对了，启动脚本报错 | agent 上报 fail 但文件保留。修复脚本后下次更新就能用，或手动 `./run.sh start` |
| 主控看不到实时状态 / 面板 | agent 离线（>5min 没 checkin） | `./run.sh status` 看是否运行；tail logs/agent.log；确认 poll_interval_seconds |
| 远程 config_patch 没生效 | agent 还没轮询到下一次 checkin | `./run.sh check` 手动触发一次 |
| Windows agent 不能真正自重启 | Go self-exec 在 Windows 不完整 | 用 NSSM 包装成 Windows 服务，主控下发 restart 后由服务管理器接管 |

## 文件清单

```
agent/
├── README.md           ← 本文件
├── CONFIG.md           完整配置说明
├── go.mod              独立 Go module，零外部依赖
├── main.go             入口 + HTTP 面板
├── client.go           checkin / report / download
├── updater.go          下载 → SHA256 → 备份 → 解压覆盖 → start → 回滚
├── config.go           配置加载 + 远程 config_patch merge
├── logger.go           环形日志缓冲 + 控制台
├── run.sh              Linux 启动脚本
├── config.example.json 配置示例
├── embed.go            Go embed 入口
├── web/index.html      原生 H5 状态面板
└── check/              agent-check 独立验证器
    ├── go.mod
    └── main.go
```

## 零依赖

agent 主程序仅依赖 Go 标准库：

```
encoding/json
fmt
io
io/fs
net/http
os
path/filepath
runtime
sync
time
embed
archive/zip    ← 解压覆盖用
crypto/sha256  ← 文件校验用
context        ← 命令超时
errors
```

无 Gin / GORM / sql driver / tls / otp — 交叉编译任意平台秒级完成。

## 许可证

MIT License
