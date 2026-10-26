# iUpdater

通用程序版本更新管理服务 + 跨主机辅助更新助手。

一个主控后端（Go + Vue3），多个 agent 助手（Go，零依赖，单二进制），适合管理分布式部署的服务器程序的灰度版本更新。


## 功能特性

### 主控后端

- **多应用管理**：每个应用独立 AppID、多 Channel（stable / beta）、多 Platform（win / linux / mac / all）、多 Architecture（amd64 / arm64 / all）
- **日期序号版本号**：`YYYYMMDDXXXX`，每天最多 9999 个版本，可读性强
- **强制更新标记**：客户端可判断是否必须升级
- **完整包 + 组件包**：两种分发模式
- **哈希校验 + 文件大小**：自动计算 SHA256，下载前校验
- **多存储策略（通过复用存储档案）**：
  - **S3 兼容对象存储**：阿里云 OSS、腾讯云 COS、AWS S3、MinIO、Cloudflare R2；支持 Virtual-Hosted / Path-Style 路径风格切换；可配置外部 CDN 加速域名（Redirect + CDN 直连）
  - **WebDAV**：系统代理转发，不暴露凭证
  - **元数据模式**：文件链接外部托管
  - 多个 App 可以复用同一个 StorageProfile，各自覆盖 base_path
- **访问校验**：应用级 AccessKey 鉴权，未授权返回 401
- **TOTP 两步验证（可选）**：配置 `TOTP_SECRET` 后强制两步登录，密码正确才进入 TOTP 环节；未配置则单步登录
- **辅助更新助手（Agent）管理**：
  - 注册 agent，生成唯一 Access Key（32 字符 hex）
  - 查看各 agent 实时状态（state / hostname / 当前版本 / 上次 checkin）
  - 远程控制：立即检查更新 / 重启 agent / 下发远程配置覆盖
  - agent 下一次 checkin 时自动领取控制指令，执行后主控自动清空
- **公共查询 API**：`/u/:appid` 返回 JSON，`/u/getlink/:appid` 302 重定向
- **完整下载代理**：`/api/download/*filekey`，跟随 302，支持 S3 直连 / 主控代理

### Agent 辅助更新助手

- **零依赖单二进制**：Go 静态编译，不依赖任何系统库
- **主动轮询**：可配置间隔（默认 60s）checkin 主控
- **完整更新流程**：stop → SHA256 校验 → 备份 → 解压 → 原子覆盖 → start
- **安全回滚**：覆盖任何步骤失败自动从备份还原，保留最近 5 个
- **shell 命令支持**：`stop_command` / `start_command` 完整终端命令（支持 `/in1panel/go/linux-amd64/run.sh stop`、`systemctl restart myapp`、`pkill -f xx; sleep 2` 等）
- **远程配置 merge**：主控下发 `config_patch`，agent 自动合并写入本地 `config.json` 并生效
- **本地状态面板**：`http://127.0.0.1:18080`，实时状态 / 进度 / 最近 200 行日志 / 手动检查
- **配置验证器**：独立 `agent-check` 小工具，检查 JSON 语法、主控连通性、真实 checkin、本地脚本可执行性
- **Linux 启动脚本**：`run.sh` start/stop/restart/status/check + systemd 服务模板注释

## 技术栈

| 模块 | 技术 |
|---|---|
| 主控后端 | Go 1.24 / Gin / GORM + SQLite / JWT / TOTP (pquerna/otp) / AWS SDK v2 / Go embed |
| 前端 UI | Vue 3 + Composition API / Vite / Tailwind CSS / Pinia / Axios |
| Agent 助手 | Go 1.24，**零外部依赖**（仅标准库 + embed） |
| 验证器 | Go 1.24，独立子模块 |

## 快速开始

### 环境要求

- Go 1.24+
- Node.js 18+ （仅开发前端）

### 一键编译（Windows）

```powershell
./build.ps1
```

产出物：`backend/iupdater.exe`（单二进制，内嵌前端 dist）。

### 开发模式

```powershell
# 后端
cd backend
cp .env.example .env     # 改 PORT / ADMIN_USER / ADMIN_PASS / TOTP_SECRET
go mod tidy
go run .

# 前端（另一个终端，已配 vite proxy 到后端）
cd frontend
npm install
npm run dev
```

### 默认账号

| 变量 | 默认值 |
|---|---|
| ADMIN_USER | `admin` |
| ADMIN_PASS | `admin123` |
| JWT_SECRET | `iupdater-secret-key-change-in-production` |
| PORT | `8080` |

## 配置

### 主控后端环境变量

| 变量 | 默认 | 说明 |
|---|---|---|
| `PORT` | `8080` | 监听端口 |
| `DB_PATH` | `data/iupdater.db` | SQLite 路径 |
| `ADMIN_USER` / `ADMIN_PASS` | `admin` / `admin123` | 管理员登录凭证 |
| `JWT_SECRET` | iupdater-secret-key-change-in-production | JWT 签名密钥，**生产必须改** |
| `TOTP_SECRET` | 空（不启用） | TOTP 两步验证秘钥（Base32）；设置后登录强制两步 |

### Agent 配置

Agent 配置为 JSON 文件（默认与 agent 二进制同目录 `config.json`），完整字段与示例见 [agent/CONFIG.md](agent/CONFIG.md)。

核心字段：

```json
{
  "server_url": "http://主控服务器:8080",
  "access_key": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",  // 主控注册 agent 后生成
  "app_id": "com.test.myapp",
  "channel": "stable",
  "poll_interval_seconds": 60,
  "target_dir": "./app",
  "stop_command":  "/in1panel/go/linux-amd64/run.sh stop",
  "start_command": "/in1panel/go/linux-amd64/run.sh start",
  "api_port": 18080
}
```

### S3 路径风格

阿里云 OSS / 腾讯云 COS / AWS S3 / Cloudflare R2 / 七牛 → **Virtual-Hosted（默认）**
MinIO / Ceph / 自建 S3 → **Path-Style**

前端在「存储档案」表单里选择，后端自动切换。

### 存储档案复用

一个 StorageProfile（S3 或 WebDAV）可被多个 App 复用，App 自己通过 `storage_path` 覆盖档案的 `base_path`。

## Agent 远程控制（主控下发）

主控管理端「助手管理 → ⚙」弹窗支持：

- **立即检查更新**（`check_now`）：agent 下次 poll 跳过间隔立即执行 checkin
- **重启 agent**（`restart`）：让 agent 自身重启动（完整重启建议外部 systemd/NSSM）
- **远程配置覆盖**（`config_patch`）：JSON patch，agent merge 写入本地 config.json 后自动生效，**无需登录目标服务器**

控制指令在 agent 下一次 checkin 时领取，领取后主控立即清空。

## API 路由速查

### 公共（无需认证）

| 方法 | 路由 | 说明 |
|---|---|---|
| GET | `/u/:appid` | 返回最新版本 JSON |
| GET | `/u/getlink/:appid` | 302 重定向到最终下载链接 |
| GET | `/api/login/config` | 返回 `{totp_enabled: bool}` |
| POST | `/api/login` | 用户名密码 + 可选 TOTP code |
| POST | `/api/check-update` | 传统方式检查更新（携带 access_key） |
| GET | `/api/download/*filekey` | S3/WebDAV 代理下载 |

### Agent 侧（Bearer agent_access_key）

| 方法 | 路由 | 说明 |
|---|---|---|
| POST | `/api/agent/checkin` | 上报状态 + 拉取更新任务 + 领取远程 command / config_patch |
| POST | `/api/agent/report` | 上报更新结果（success / fail） |

### 管理端（JWT）

| 方法 | 路由 | 说明 |
|---|---|---|
| POST | `/api/admin/agents` | 注册新 agent（返回唯一 access_key） |
| GET | `/api/admin/agents` | 列表（含实时 status_json / pending_command） |
| GET | `/api/admin/agents/:id` | 详情 |
| PUT | `/api/admin/agents/:id` | 编辑 / 重置 access_key |
| DELETE | `/api/admin/agents/:id` | 删除（**物理删除，清干净关联表**） |
| POST | `/api/admin/agents/:id/control` | 下发 command 或 config_patch |
| /api/admin/apps | 应用 CRUD |  |
| /api/admin/storage-profiles | 存储档案 CRUD + 连通性检查 |  |
| /api/admin/versions | 版本 CRUD + 上传 + 探针 |  |
| /api/admin/components | 组件 CRUD + 上传 |  |

## Linux 服务器部署（Agent）

```bash
scp agent-linux-amd64 run.sh config.json server:/opt/myapp/
cd /opt/myapp
mv agent-linux-amd64 agent
chmod +x agent run.sh

# 先验一下
./run.sh check

# 启动
./run.sh start

# (可选) systemd 开机自启
# 按 run.sh 顶部注释复制 /etc/systemd/system/iupdater-agent.service
sudo systemctl daemon-reload
sudo systemctl enable --now iupdater-agent
```

完整说明见 [agent/CONFIG.md](agent/CONFIG.md)。

## 许可证

MIT License
