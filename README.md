# iUpdater - 通用程序版本更新器

基于 Go + Vue3 开发的通用程序版本更新服务，支持多应用、多渠道、多平台的版本管理和组件包分发。

## 功能特性

- **多应用管理**: 支持管理多个应用，每个应用独立配置
- **双渠道发布**: 正式版(stable) / 测试版(beta) 双渠道
- **多平台架构**: 支持 Windows / macOS / Linux / Android / iOS，支持 x86_64 / arm64 等架构
- **日期序号版本号**: 可读的纯数字版本号（YYYYMMDDXXXX）
- **强制更新**: 支持标记强制更新版本
- **完整包与组件包**: 支持完整安装包和分散组件包两种分发模式
- **哈希校验**: 自动计算文件 SHA256 哈希值，确保文件完整性
- **多存储策略**: 每个应用可独立选择存储策略（S3 / WebDAV / 元数据）
- **系统代理转发**: WebDAV 统一通过系统代理下载，S3 可选代理或直连
- **访问校验**: 应用级 AccessKey 鉴权，未授权返回 401
- **文件探针**: 元数据模式下支持探测远程文件大小（HEAD 请求）
- **现代化 UI**: Vue3 + Tailwind CSS 构建的扁平化管理后台
- **一体化编译**: Go embed 嵌入前端资源，单文件部署
- **Rust Demo 客户端**: egui 构建的跨平台测试客户端

## 技术栈

**后端**:
- Go 1.21+
- Gin Web 框架
- GORM + SQLite
- JWT 认证
- Go embed（前端资源嵌入）

**前端**:
- Vue 3 + Composition API
- Vite
- Tailwind CSS
- Pinia 状态管理
- Vue Router
- Axios

**Demo 测试客户端**:
- Rust
- egui (eframe)
- reqwest
- tokio

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 18+
- npm 或 yarn

### 一体化编译（推荐）

一键编译前端和后端，生成独立的可执行文件：

**Windows (PowerShell)**:
```powershell
.\build.ps1
```

**Windows (CMD)**:
```bat
build.bat
```

编译完成后，可执行文件位于 `backend\iupdater.exe`，直接运行即可。

### 开发模式

#### 后端启动

```bash
cd backend

# 复制配置文件
copy .env.example .env

# 安装依赖
go mod tidy

# 构建
go build -o iupdater.exe .

# 运行
./iupdater.exe
```

后端默认监听 `:8080` 端口。

### 前端启动

```bash
cd frontend

# 安装依赖
npm install

# 开发模式
npm run dev

# 生产构建
npm run build
```

前端开发服务器默认运行在 `http://localhost:5173`，已配置代理转发 API 请求到后端。

### 默认账号

- 用户名: `admin`
- 密码: `admin123`

可通过环境变量 `ADMIN_USER` 和 `ADMIN_PASS` 修改。

## 配置说明

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| PORT | 8080 | 服务监听端口 |
| ADMIN_USER | admin | 管理员用户名 |
| ADMIN_PASS | admin123 | 管理员密码 |
| JWT_SECRET | iupdater-secret-key-change-in-production | JWT 密钥，生产环境请修改 |
| DB_PATH | data/iupdater.db | SQLite 数据库文件路径 |

### 存储策略配置

每个应用可独立配置存储策略，在管理后台图形化配置，无需手动编辑 JSON。

#### 1. 元数据模式

仅存储版本元数据，文件下载链接手动填写外部地址，适合文件托管在第三方平台的场景。

- 发布版本时填写外部下载链接
- 可选填写文件大小和 SHA256 哈希
- 支持一键探测远程文件大小（HTTP HEAD 请求）
- 下载时直接使用外部链接，不经过系统转发

#### 2. S3 兼容对象存储

支持阿里云 OSS、腾讯云 COS、MinIO 等 S3 兼容存储。

配置项：
- **Endpoint**: S3 服务地址
- **Access Key**: 访问密钥
- **Secret Key**: 密钥
- **Bucket**: 存储桶
- **Region**: 区域
- **下载方式**: 外部 CDN 直连 / 系统代理转发
  - 外部 CDN 直连：配置 CDN 基础 URL，客户端直接访问 CDN
  - 系统代理转发：下载请求通过本系统代理，隐藏存储服务地址

#### 3. WebDAV 存储

配置项：
- **URL**: WebDAV 服务地址
- **用户名**: 账号
- **密码**: 密码
- **基础路径**: 远程目录路径

下载方式：统一通过系统代理转发，不暴露 WebDAV 地址和凭证。

### 访问校验

每个应用可独立启用访问校验：
- 启用后自动生成 16 位随机 AccessKey
- 客户端请求 `/api/check-update` 必须携带 `access_key`
- 未携带或错误的 AccessKey 返回 401 Unauthorized

## 版本号规则

采用可读的日期序号格式：`YYYYMMDDXXXX`

- **前 8 位**: 年月日，如 `20260705`
- **后 4 位**: 当天序号，从 `0001` 开始递增

示例：
- 2026年7月5日第1个版本: `202607050001`
- 2026年7月5日第100个版本: `202607050100`
- 2026年7月6日第1个版本: `202607060001`

## API 文档

详细的 API 文档请参考 [API 文档](docs/API.md)

### 核心接口

**公共接口**:
- `POST /api/check-update` - 检查更新
- `GET /api/download/*` - 下载代理（S3 代理/WebDAV 代理）

**管理接口**（需认证）:
- `POST /api/login` - 登录
- `GET/POST/PUT/DELETE /api/admin/apps` - 应用管理
- `GET/POST/PUT/DELETE /api/admin/versions` - 版本管理
- `POST /api/admin/versions/upload` - 上传版本文件
- `POST /api/admin/versions/probe` - 探针获取文件大小
- `PUT /api/admin/versions/:id/status` - 更新版本状态
- `GET/POST/PUT/DELETE /api/admin/components` - 组件管理
- `POST /api/admin/components/upload` - 上传组件文件
- `PUT /api/admin/components/:id/status` - 更新组件状态

## Demo 测试客户端

Rust + egui 构建的跨平台测试客户端，用于测试更新接口。

### 功能

- 配置服务器地址、App ID、版本号、AccessKey
- 渠道切换（正式版/测试版）
- 完整包信息展示
- 组件列表与本地版本对比
- 选项卡切换查看完整包和组件
- 下载链接一键复制

### 编译运行

```bash
cd demo
cargo build --release
./target/release/iupdater-demo.exe
```

## 许可证

MIT License
