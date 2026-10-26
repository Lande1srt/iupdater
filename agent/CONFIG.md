# Agent 配置说明

辅助更新助手（agent）的运行时配置文件，默认路径为 agent 可执行文件同目录下的 `config.json`。

## 完整示例

```json
{
  "server_url": "http://192.168.1.100:8080",
  "access_key": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
  "poll_interval_seconds": 60,
  "app_id": "com.test.myapp",
  "channel": "stable",
  "platform": "linux",
  "architecture": "amd64",
  "target_dir": "./app",
  "backup_dir": "./backup",
  "download_dir": "./download",
  "stop_command": "",
  "start_command": "",
  "api_port": 18080
}
```

## 字段说明

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|---|---|---|---|---|
| `server_url` | string | ✅ | — | 主控 iupdater 后端地址，如 `http://192.168.1.100:8080` |
| `access_key` | string | ✅ | — | 在主控后台「助手管理 → 注册助手」时生成的 Access Key |
| `poll_interval_seconds` | int | ❌ | `60` | 轮询间隔（秒） |
| `app_id` | string | ✅ | — | 绑定的应用 AppID |
| `channel` | string | ❌ | `stable` | 发布频道：`stable` / `beta` |
| `platform` | string | ❌ | 自动检测 | 目标平台：`windows` / `linux` / `macos` / `all` |
| `architecture` | string | ❌ | 自动检测 | 目标架构：`amd64` / `arm64` / `all` |
| `target_dir` | string | ❌ | `./app` | 目标程序安装目录（agent 相对路径），下载的 zip 解压覆盖到此目录 |
| `backup_dir` | string | ❌ | `./backup` | 更新前自动备份目录（agent 相对路径），保留最近 5 个备份 |
| `download_dir` | string | ❌ | `./download` | zip 下载临时目录（agent 相对路径） |
| `stop_command` | string | ❌ | 空（跳过） | 停止目标程序的完整 shell 命令 |
| `start_command` | string | ❌ | 空（跳过） | 启动目标程序的完整 shell 命令 |
| `api_port` | int | ❌ | `18080` | agent 本地状态面板端口，访问 `http://127.0.0.1:18080` 查看实时状态和日志 |

## stop_command / start_command 详解

这两个字段是**完整的终端命令字符串**，agent 内部统一通过 shell 执行：

- **Linux**：`/bin/sh -c "<命令>"`
- **Windows**：`cmd /C "<命令>"`

因此它支持 shell 所有语法：空格、引号、分号、管道、重定向、后台运行符 `&` 等。

### 常见形式

#### 1Panel 面板管理脚本（Linux）

```json
{
  "stop_command":  "/in1panel/go/linux-amd64/run.sh stop",
  "start_command": "/in1panel/go/linux-amd64/run.sh start"
}
```

#### systemd 服务

```json
{
  "stop_command":  "systemctl stop myapp",
  "start_command": "systemctl start myapp"
}
```

#### init.d 脚本

```json
{
  "stop_command":  "/etc/init.d/myapp stop",
  "start_command": "/etc/init.d/myapp start"
}
```

#### Docker 容器

```json
{
  "stop_command":  "docker stop mycontainer",
  "start_command": "docker start mycontainer"
}
```

#### Windows .bat 脚本

```json
{
  "stop_command":  "./stop.bat",
  "start_command": "./start.bat"
}
```

#### kill + 后台启动（无 service 管理器的自建进程）

```json
{
  "stop_command":  "pkill -f myapp; sleep 2",
  "start_command": "nohup ./myapp > /dev/null 2>&1 &"
}
```

#### 多命令用分号串联

```json
{
  "stop_command":  "pkill -f myapp; sleep 3; echo stopped",
  "start_command": "cd /opt/myapp && ./myapp --daemon"
}
```

### 路径解析规则

命令里的**第一个词**如果包含 `/` 或 `\`（可执行文件 / 脚本路径），且是**相对路径**，agent 会自动解析为相对于自身目录的绝对路径；后面的参数原样保留。

| 配置值 | 解析后（agent 在 `/opt/agent/`） |
|---|---|
| `"./run.sh stop"` | `"/opt/agent/run.sh stop"` |
| `"run.sh stop"` | `run.sh stop`（首词无路径分隔符 → 视作 PATH 里的系统命令） |
| `"/etc/init.d/myapp restart"` | 不变（已是绝对路径） |
| `"systemctl restart myapp"` | 不变 |

### 超时与工作目录

- 每个命令超时 **30 秒**（系统级 exec 超时）
- shell 工作目录固定为 **agent 自身所在目录**（不是脚本所在目录），确保 `./xxx` 相对路径稳定

### 命令失败时的行为

- **stop_command 失败** → 中止更新（返回错误，不做下载/覆盖），向上层主控上报 `fail`
- **start_command 失败** → 文件已成功替换，仍然上报 `success`，但 `message` 字段里会附带启动失败错误，后台可看到覆盖已生效但需要手动启动

### 向后兼容

旧字段 `stop_script` / `start_script` 仍可用，未设置 `stop_command` 时会自动兜底使用它们。新项目**统一用 `stop_command` / `start_command`**。

## 完整更新流程

```
┌──────────────────────────────────────────────────────┐
│ 1. 轮询 checkin → 主控返回 UpdateTask                │
├──────────────────────────────────────────────────────┤
│ 2. 下载 zip（跟随 302，进度百分比回传到本地面板）       │
│ 3. SHA256 校验（主控提供了 file_hash 时）             │
│ 4. stop_command（30s 超时，失败 → 中止更新）          │
│ 5. 全目录备份 target_dir → backup_dir/{时间戳}         │
│ 6. 解压到临时目录 → os.Rename 原子覆盖 target_dir      │
│    （覆盖中任何步骤失败 → 自动从备份恢复 target_dir）   │
│ 7. start_command（30s 超时，失败不回滚但上报）        │
│ 8. 清理旧备份（保留最近 5 个）                         │
│ 9. 上报 success / fail 到主控                          │
└──────────────────────────────────────────────────────┘
```

## 本地状态面板

agent 内置一个轻量 HTTP 状态面板，无需额外运行。启动后浏览器访问：

```
http://127.0.0.1:18080
```

功能：
- 实时状态（空闲 / 检查中 / 下载中 / 更新中 / 成功 / 失败）
- 下载进度条（下载时实时百分比）
- 当前版本号 / 下次检查倒计时
- 手动「立即检查」按钮
- 最近 200 行实时日志

## 运行方式

```powershell
# 直接运行（默认读同目录 config.json）
./agent

# 指定配置文件
./agent --config /etc/myapp/agent.json

# 后台运行（推荐 Linux systemd / Windows 服务包装）
# Linux 示例：
nohup ./agent > /var/log/agent.log 2>&1 &

# Windows 示例：注册为计划任务或用 NSSM 包装成服务
```

## 注册 Access Key

Access Key 由主控后台管理员生成：

1. 登录主控后端管理 UI
2. 侧边栏「助手管理」→ 右上角「注册助手」
3. 填写名称、选择要监控的 App、Channel、Platform、Architecture
4. 提交后**立即复制生成的 Access Key**（后端只显示这一次）
5. 粘贴到 agent 的 `config.json` 的 `access_key` 字段

access_key 格式为 32 字符 hex，例如 `99bde104a3c3ed3f6bfd1d93b700a86d`。
