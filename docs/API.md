# iUpdater API 文档

## 概述

iUpdater 是一个通用的程序版本更新服务，支持多应用、多渠道、多平台的版本管理和组件包分发。

**基础路径**: `/api`

---

## 公共接口

### 1. 检查更新

检查指定应用是否有可用更新。

**接口**: `POST /api/check-update`

**请求体**:
```json
{
  "app_id": "com.example.myapp",
  "version_code": 202607050001,
  "channel": "stable",
  "platform": "windows",
  "architecture": "x86_64",
  "access_key": "a1b2c3d4e5f6g7h8"
}
```

**参数说明**:
| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| app_id | string | 是 | 应用唯一识别码（包名） |
| version_code | int64 | 是 | 当前版本号（YYYYMMDDXXXX 格式） |
| channel | string | 否 | 渠道：`stable`(正式版) / `beta`(测试版)，默认 stable |
| platform | string | 否 | 平台：windows/macos/linux/android/ios/all，默认 all |
| architecture | string | 否 | 架构：x86_64/arm64/x86/all，默认 all |
| access_key | string | 条件 | 访问密钥，应用启用访问校验时必填 |

**响应**:
```json
{
  "need_update": true,
  "is_force_update": false,
  "latest_version": {
    "app_id": "com.example.myapp",
    "version_code": 202607050100,
    "version_name": "1.1.0",
    "channel": "stable",
    "platform": "windows",
    "architecture": "x86_64",
    "package_type": "full",
    "file_size": 52428800,
    "file_hash": "abc123def456...",
    "download_url": "https://cdn.example.com/app/xxx.zip",
    "release_notes": "更新说明..."
  },
  "components": [
    {
      "component_key": "core-module",
      "name": "核心模块",
      "version_code": 202607050100,
      "version_name": "1.0.0",
      "file_size": 10485760,
      "file_hash": "def789ghi012...",
      "download_url": "https://cdn.example.com/components/core.zip"
    }
  ]
}
```

**响应字段说明**:
| 字段 | 类型 | 说明 |
|------|------|------|
| need_update | bool | 是否需要更新 |
| is_force_update | bool | 是否强制更新 |
| latest_version | object | 最新完整包版本信息 |
| components | array | 可用组件包列表 |

---

### 2. 下载代理

通过系统代理转发文件下载，用于 S3 代理模式和 WebDAV 存储。

**接口**: `GET /api/download/{uuid}/{filename}`

**路径参数**:
- `uuid`: 文件唯一标识
- `filename`: 文件名

**说明**:
- 自动识别文件所属应用的存储配置
- 流式转发，不占用服务器存储空间
- 支持 S3 和 WebDAV 存储的透明代理
- 无需客户端处理存储服务认证

**响应**: 文件二进制流

---

## 管理后台接口

所有管理接口需要在请求头中携带 JWT Token：
```
Authorization: Bearer {token}
```

### 2. 登录

**接口**: `POST /api/login`

**请求体**:
```json
{
  "username": "admin",
  "password": "admin123"
}
```

**响应**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "user": {
    "username": "admin"
  }
}
```

---

### 3. 应用管理

#### 3.1 获取应用列表

**接口**: `GET /api/admin/apps`

**查询参数**:
- `page`: 页码，默认 1
- `page_size`: 每页条数，默认 20
- `keyword`: 搜索关键词（按名称或App ID搜索）

#### 3.2 获取应用详情

**接口**: `GET /api/admin/apps/:id`

#### 3.3 创建应用

**接口**: `POST /api/admin/apps`

**请求体**:
```json
{
  "app_id": "com.example.myapp",
  "name": "我的应用",
  "description": "应用描述",
  "icon": "https://example.com/icon.png",
  "storage_type": "metadata",
  "storage_config": "{\"base_url\": \"https://cdn.example.com\"}",
  "enable_auth": false
}
```

**访问校验说明**:
- `enable_auth: true` - 启用访问校验，自动生成 16 位随机 AccessKey
- `enable_auth: false` - 禁用访问校验，客户端无需提供 access_key
- 启用后，客户端请求 /api/check-update 必须携带正确的 access_key，否则返回 401

**存储类型说明**:
- `metadata`: 元数据模式
  - 仅存储版本元数据，不存储文件
  - 发布版本时手动填写外部下载链接
  - 可选填写文件大小和 SHA256 哈希值
  - 支持探针功能：一键探测远程文件大小（HTTP HEAD 请求）
  - 配置: `{ "base_url": "https://your-domain.com" }` (可选)
- `s3`: S3 兼容对象存储
  - 支持阿里云 OSS、腾讯云 COS、MinIO 等
  - 下载方式：外部 CDN 直连 / 系统代理转发
  - 配置: `{ "endpoint": "", "access_key": "", "secret_key": "", "bucket": "", "region": "", "use_proxy": false, "cdn_url": "" }`
  - `use_proxy: false` - 外部 CDN 直连，需配置 cdn_url
  - `use_proxy: true` - 系统代理转发，下载路径为 /api/download/{uuid}/{filename}
- `webdav`: WebDAV 存储
  - 统一通过系统代理转发，不暴露 WebDAV 地址和凭证
  - 下载路径：/api/download/{uuid}/{filename}
  - 配置: `{ "url": "", "username": "", "password": "", "base_path": "" }`

#### 3.4 更新应用

**接口**: `PUT /api/admin/apps/:id`

#### 3.5 删除应用

**接口**: `DELETE /api/admin/apps/:id`

---

### 4. 版本管理

#### 4.1 获取版本列表

**接口**: `GET /api/admin/versions`

**查询参数**:
- `app_id`: 应用ID
- `channel`: 渠道筛选
- `page`: 页码
- `page_size`: 每页条数

#### 4.2 创建版本（手动指定下载地址）

**接口**: `POST /api/admin/versions`

**请求体**:
```json
{
  "app_id": "com.example.myapp",
  "version_name": "1.0.0",
  "channel": "stable",
  "platform": "windows",
  "architecture": "x86_64",
  "package_type": "full",
  "is_force_update": false,
  "download_url": "https://example.com/app.zip",
  "file_hash": "abc123...",
  "file_size": 52428800,
  "release_notes": "更新说明"
}
```

#### 4.3 上传版本文件

**接口**: `POST /api/admin/versions/upload` (multipart/form-data)

**表单字段**:
- `app_id`: 应用ID
- `version_name`: 版本名称
- `channel`: 渠道
- `platform`: 平台
- `architecture`: 架构
- `package_type`: 包类型
- `is_force_update`: 是否强制更新
- `release_notes`: 更新说明
- `file`: 安装包文件（S3/WebDAV 模式）
- `download_url`: 外部下载链接（元数据模式）
- `file_size`: 文件大小（元数据模式，可选）
- `file_hash`: SHA256 哈希（元数据模式，可选）

**说明**:
- S3/WebDAV 模式：上传 `file` 文件，系统自动计算哈希
- 元数据模式：填写 `download_url`，可选填 `file_size` 和 `file_hash`

#### 4.4 探针获取文件大小

**接口**: `POST /api/admin/versions/probe`

**请求体**:
```json
{
  "url": "https://example.com/app.zip"
}
```

**响应**:
```json
{
  "size": 52428800
}
```

**说明**:
- 使用 HTTP HEAD 请求获取远程文件大小
- 不下载文件内容，仅读取 Content-Length 头
- 用于元数据模式快速获取文件信息

#### 4.5 更新版本状态

**接口**: `PUT /api/admin/versions/:id/status`

**请求体**:
```json
{
  "status": 1
}
```
- `status: 1` 已发布
- `status: 0` 已下架

#### 4.6 删除版本

**接口**: `DELETE /api/admin/versions/:id`

---

### 5. 组件管理

#### 5.1 获取组件列表

**接口**: `GET /api/admin/components`

**查询参数**:
- `app_id`: 应用ID
- `channel`: 渠道筛选
- `page`: 页码
- `page_size`: 每页条数

#### 5.2 创建组件（手动指定下载地址）

**接口**: `POST /api/admin/components`

**请求体**:
```json
{
  "app_id": "com.example.myapp",
  "component_key": "core-module",
  "name": "核心模块",
  "description": "组件描述",
  "version_name": "1.0.0",
  "channel": "stable",
  "platform": "all",
  "architecture": "all",
  "download_url": "https://example.com/core.zip",
  "file_hash": "abc123...",
  "file_size": 10485760
}
```

#### 5.3 上传组件文件

**接口**: `POST /api/admin/components/upload` (multipart/form-data)

**表单字段**:
- `app_id`: 应用ID
- `component_key`: 组件Key
- `name`: 组件名称
- `description`: 组件描述
- `version_name`: 版本名称
- `channel`: 渠道
- `platform`: 平台
- `architecture`: 架构
- `file`: 组件文件（S3/WebDAV 模式）
- `download_url`: 外部下载链接（元数据模式）
- `file_size`: 文件大小（元数据模式，可选）
- `file_hash`: SHA256 哈希（元数据模式，可选）

**说明**:
- S3/WebDAV 模式：上传 `file` 文件，系统自动计算哈希
- 元数据模式：填写 `download_url`，可选填 `file_size` 和 `file_hash`

#### 5.4 更新组件状态

**接口**: `PUT /api/admin/components/:id/status`

#### 5.5 删除组件

**接口**: `DELETE /api/admin/components/:id`

---

## 版本号规则

- **version_code**: 日期序号格式 YYYYMMDDXXXX（如 202607050001）
  - 前 8 位：年月日（YYYYMMDD）
  - 后 4 位：当天序号（0001-9999）
  - 每天从 0001 开始自动递增
  - 每天最多 9999 个版本，超出时返回溢出错误
- **version_name**: 自定义版本名称（如 1.0.0、v2.1.0-beta），仅用于展示
- 比较逻辑：服务端返回的 version_code 大于客户端传入的 version_code 时判定为有更新

**版本号示例**:
- 2026年7月5日第1个版本：`202607050001`
- 2026年7月5日第100个版本：`202607050100`
- 2026年7月6日第1个版本：`202607060001`

## 渠道说明

- `stable`: 正式版渠道，面向所有用户
- `beta`: 测试版渠道，面向测试用户

## 平台与架构

支持的平台标识：`windows`, `macos`, `linux`, `android`, `ios`, `all`

支持的架构标识：`x86_64`, `arm64`, `x86`, `all`

匹配规则：优先匹配精确值，其次匹配 `all`
