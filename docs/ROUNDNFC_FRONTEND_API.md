# RoundNFC 前端接口文档

本文档面向 Web 前端管理后台和公开徽章页。Android 写卡 App 的 `/app/*` 接口见 `docs/ROUNDNFC_ANDROID_APP.md`。

## 基础信息

- 默认前缀：`/api/roundnfc`
- 管理后台接口前缀：`/api/roundnfc/admin`
- 请求体默认使用 `Content-Type: application/json`
- 上传接口使用 `multipart/form-data`
- 时间字段使用 RFC3339 字符串，例如 `2026-07-09T12:00:00Z`

统一 JSON 响应：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

错误响应：

```json
{
  "code": 400,
  "message": "invalid body",
  "data": null
}
```

管理后台除登录、通行密钥登录开始/完成外，均需要：

```http
Authorization: Bearer <jwt>
```

部分管理接口也兼容服务端配置的 `X-App-Token` 或已创建的 `X-RoundNFC-App-Token`，但 Web 管理后台建议只使用 JWT。

分页参数：

- `limit`：默认 `50`，最大 `200`
- `offset`：默认 `0`

列表响应统一形态：

```json
{
  "items": [],
  "total": 0
}
```

## 数据结构

### Badge

```ts
interface Badge {
  id: string
  title: string
  series?: string
  type?: string
  styleKey?: string
  imageUrl?: string
  styleImageUrl?: string
  styleImageOriginalUrl?: string
  description?: string
  serialNo?: string
  releasedAt?: string
  coserBinding?: BadgeCoserBinding
  createdAt?: string
  updatedAt?: string
}
```

### BadgeStyleTemplate

```ts
interface BadgeStyleTemplate {
  key: string
  label: string
  description?: string
  imageUrl?: string
  imageOriginalUrl?: string
  imagePreviewUrl?: string
  payload?: unknown
  enabled: boolean
  createdAt?: string
  updatedAt?: string
}
```

### RequestStatus

```ts
type RequestStatus = 'new' | 'handled' | 'rejected'
```

## 公开徽章页接口

### 获取公开徽章信息

```http
GET /api/roundnfc/badges/{id}
```

响应 `data`：`Badge`

说明：

- `imageUrl`、`styleImageOriginalUrl` 可能是 `/api/roundnfc/objects/{token}` 一次性对象链接。
- 一次性对象链接会过期，前端应按需重新获取徽章信息。

### 提交返图申请

```http
POST /api/roundnfc/badges/{id}/photo-requests
```

请求：

```json
{
  "name": "昵称",
  "contact": "联系方式",
  "message": "备注",
  "attachmentKeys": ["uploads/xx.jpg"],
  "turnstileToken": "<cf-turnstile-token>"
}
```

响应 `data`：

```json
{
  "requestId": "ph_xxx"
}
```

必填：`name`、`contact`。接口有 Turnstile 校验和按 IP + 徽章限流。

### 提交 To 签申请

```http
POST /api/roundnfc/badges/{id}/autograph-requests
```

请求：

```json
{
  "name": "昵称",
  "contact": "联系方式",
  "target": "To 谁",
  "content": "签名内容",
  "attachmentKeys": ["uploads/xx.jpg"],
  "turnstileToken": "<cf-turnstile-token>"
}
```

响应 `data`：

```json
{
  "requestId": "au_xxx"
}
```

必填：`name`、`contact`、`content`。接口有 Turnstile 校验和按 IP + 徽章限流。

### 上传公开附件

```http
POST /api/roundnfc/uploads
Content-Type: multipart/form-data
```

表单字段：

- `file`：图片文件
- `turnstileToken`：Cloudflare Turnstile token，也可通过 `CF-Turnstile-Response` 请求头传入

响应 `data`：

```json
{
  "key": "uploads/ab/xxx.jpg",
  "mime": "image/jpeg",
  "size": 12345
}
```

限制：

- 只允许 `image/jpeg`、`image/png`、`image/webp`、`image/gif`
- 默认最大上传大小由 `ROUNDNFC_MAX_UPLOAD_MB` 控制，默认 `8MB`

### 获取一次性对象

```http
GET /api/roundnfc/objects/{token}
```

响应：图片二进制。

说明：

- 成功后返回 blob，前端可用 `URL.createObjectURL()` 显示。
- 链接是一次性消费，过期或已消费返回 `410`。
- 无效 token 返回 `403`，对象不存在返回 `404`。

## 管理后台鉴权接口

### 密码登录

```http
POST /api/roundnfc/admin/login
```

请求：

```json
{
  "username": "admin",
  "password": "password",
  "totpCode": "123456"
}
```

响应 `data`：

```json
{
  "token": "<jwt>",
  "expiresAt": "2026-07-09T12:00:00Z",
  "username": "admin"
}
```

如果账号已开启 TOTP 且未传 `totpCode`：

```json
{
  "needsTOTP": true
}
```

### 当前登录用户

```http
GET /api/roundnfc/admin/me
Authorization: Bearer <jwt>
```

响应 `data`：

```json
{
  "username": "admin"
}
```

### TOTP 状态

```http
GET /api/roundnfc/admin/totp/status
```

响应 `data`：

```json
{
  "enabled": true
}
```

### 创建 TOTP 配置

```http
POST /api/roundnfc/admin/totp/setup
```

响应 `data`：

```json
{
  "uri": "otpauth://totp/...",
  "secret": "BASE32SECRET"
}
```

### 启用 TOTP

```http
POST /api/roundnfc/admin/totp/enable
```

请求：

```json
{
  "code": "123456"
}
```

响应 `data`：

```json
{
  "ok": true
}
```

### 关闭 TOTP

```http
DELETE /api/roundnfc/admin/totp
```

响应 `data`：

```json
{
  "ok": true
}
```

### 通行密钥登录

开始登录：

```http
POST /api/roundnfc/admin/webauthn/login/begin
```

请求：

```json
{
  "username": "admin"
}
```

响应 `data`：

```ts
interface LoginBeginOptions {
  challenge: string
  rpId: string
  timeout: number
  allowCredentials: Array<{ type: string; id: string }>
  sessionId: string
}
```

完成登录：

```http
POST /api/roundnfc/admin/webauthn/login/finish
```

请求：

```json
{
  "sessionId": "xxx",
  "credential": {}
}
```

响应同密码登录成功响应。

### 管理通行密钥

开始注册：

```http
POST /api/roundnfc/admin/webauthn/register/begin
```

响应 `data`：

```ts
interface RegBeginOptions {
  challenge: string
  rpId: string
  rpName: string
  userId: string
  userName: string
  userDisplayName: string
  timeout: number
  excludeCredentialIds: string[]
  sessionId: string
}
```

完成注册：

```http
POST /api/roundnfc/admin/webauthn/register/finish
```

请求：

```json
{
  "sessionId": "xxx",
  "name": "MacBook Touch ID",
  "credential": {}
}
```

响应 `data`：

```json
{
  "ok": true,
  "id": "credential-id"
}
```

列出通行密钥：

```http
GET /api/roundnfc/admin/webauthn/credentials
```

响应 `data`：

```json
{
  "items": [
    {
      "id": "credential-id",
      "name": "MacBook Touch ID",
      "createdAt": "2026-07-09T12:00:00Z"
    }
  ]
}
```

删除通行密钥：

```http
DELETE /api/roundnfc/admin/webauthn/credentials/{id}
```

响应 `data`：

```json
{
  "ok": true
}
```

## 徽章管理接口

### 列出徽章

```http
GET /api/roundnfc/admin/badges?q=&limit=50&offset=0
```

响应 `data`：`{ items: Badge[], total: number }`

### 获取徽章

```http
GET /api/roundnfc/admin/badges/{id}
```

响应 `data`：`Badge`

### 创建徽章

```http
POST /api/roundnfc/admin/badges
```

请求：

```json
{
  "id": "badge-001",
  "title": "徽章标题",
  "series": "系列",
  "type": "类型",
  "styleKey": "sakura",
  "imageUrl": "badges/xx.jpg",
  "description": "描述",
  "serialNo": "001",
  "releasedAt": "2026-07-09"
}
```

响应 `data`：`Badge`

说明：

- `id` 必填。
- `styleKey` 必须为空字符串或存在于样式模板列表中。

### 更新徽章

```http
PUT /api/roundnfc/admin/badges/{id}
```

请求字段同创建徽章。路径中的 `{id}` 优先于请求体 `id`。

响应 `data`：`Badge`

### 删除徽章

```http
DELETE /api/roundnfc/admin/badges/{id}
```

响应 `data`：

```json
{
  "ok": true
}
```

### 上传徽章图片

```http
POST /api/roundnfc/admin/badges/{id}/image
Content-Type: multipart/form-data
```

表单字段：

- `file`：图片文件

响应 `data`：

```json
{
  "key": "badges/badge-001/ab/xxx.jpg"
}
```

上传成功后后端会把该 key 写入徽章的 `imageUrl`。

## 样式模板接口

### 列出样式模板

```http
GET /api/roundnfc/admin/style-templates
```

响应 `data`：

```json
{
  "items": []
}
```

每个 item 为 `BadgeStyleTemplate`。如果模板图片是本地对象，响应会补充 `imageOriginalUrl` 和 `imagePreviewUrl` 一次性对象链接。

兼容接口：

```http
GET /api/roundnfc/admin/styles
```

### 创建样式模板

```http
POST /api/roundnfc/admin/style-templates
```

请求：

```json
{
  "key": "sakura",
  "label": "樱花",
  "description": "粉色主题",
  "imageUrl": "style-templates/sakura/xx.png",
  "payload": {},
  "enabled": true
}
```

响应 `data`：`BadgeStyleTemplate`

必填：`key`、`label`。

### 更新样式模板

```http
PUT /api/roundnfc/admin/style-templates/{key}
```

请求字段同创建。路径中的 `{key}` 优先于请求体 `key`。

响应 `data`：`BadgeStyleTemplate`

### 上传样式模板图片

```http
POST /api/roundnfc/admin/style-templates/{key}/image
Content-Type: multipart/form-data
```

表单字段：

- `file`：图片文件

响应 `data`：

```json
{
  "key": "style-templates/sakura/ab/xxx.png",
  "item": {}
}
```

上传成功后后端会把该 key 写入样式模板的 `imageUrl`。

### 删除样式模板

```http
DELETE /api/roundnfc/admin/style-templates/{key}
```

响应 `data`：

```json
{
  "ok": true
}
```

## 申请管理接口

### 列出返图申请

```http
GET /api/roundnfc/admin/photo-requests?badgeId=&status=&limit=50&offset=0
```

响应 `data`：

```ts
interface PhotoRequest {
  id: string
  badgeId: string
  name: string
  contact: string
  message?: string
  status: 'new' | 'handled' | 'rejected'
  attachmentKeys?: string[]
  createdAt: string
  updatedAt: string
}
```

### 更新返图申请状态

```http
PATCH /api/roundnfc/admin/photo-requests/{id}
```

请求：

```json
{
  "status": "handled"
}
```

响应 `data`：

```json
{
  "ok": true
}
```

### 列出 To 签申请

```http
GET /api/roundnfc/admin/autograph-requests?badgeId=&status=&limit=50&offset=0
```

响应 `data`：

```ts
interface AutographRequest {
  id: string
  badgeId: string
  name: string
  contact: string
  target: string
  content: string
  status: 'new' | 'handled' | 'rejected'
  attachmentKeys?: string[]
  createdAt: string
  updatedAt: string
}
```

### 更新 To 签申请状态

```http
PATCH /api/roundnfc/admin/autograph-requests/{id}
```

请求：

```json
{
  "status": "rejected"
}
```

响应 `data`：

```json
{
  "ok": true
}
```

## COS 直传和写卡记录

### 获取 COS 直传签名

```http
POST /api/roundnfc/admin/uploads/presign
```

请求：

```json
{
  "badgeId": "badge-001",
  "fileName": "photo.jpg",
  "contentType": "image/jpeg",
  "purpose": "nfc-write"
}
```

响应 `data`：

```json
{
  "uploadUrl": "https://bucket.cos.ap-shanghai.myqcloud.com/roundnfc/nfc-writes/badge-001/uuid.jpg",
  "objectKey": "roundnfc/nfc-writes/badge-001/uuid.jpg",
  "method": "PUT",
  "headers": {
    "Authorization": "q-sign-algorithm=sha1&..."
  },
  "expiresIn": 300
}
```

前端上传文件到 COS 时：

- 使用响应里的 `method`，当前固定为 `PUT`
- 请求 URL 使用 `uploadUrl`
- 附加响应里的 `headers`
- 不要把后端 JWT、`X-App-Token` 或 `X-RoundNFC-App-Token` 发给 COS

`purpose` 规则：

- 空、`nfc-write`、`nfc-writes`、`download`、`user-download`：只允许 JPEG，生成 `roundnfc/nfc-writes/{badgeId}/...jpg`
- `coser-photo`、`cn-photo`、`badge-coser`：允许 JPEG、PNG、WebP，生成 `roundnfc/coser-photos/{badgeId}/...`

### 创建 NFC 写卡记录

```http
POST /api/roundnfc/admin/nfc-writes
```

请求：

```json
{
  "badgeId": "badge-001",
  "tagUid": "04AABBCCDD",
  "ndefUrl": "https://example.com/nfc/badge-001",
  "deviceId": "writer-01",
  "writeStatus": "success",
  "photoObjectKey": "roundnfc/nfc-writes/badge-001/uuid.jpg",
  "writtenAt": "2026-07-09T12:00:00Z"
}
```

响应 `data`：

```ts
interface NFCWrite {
  id: string
  badgeId: string
  tagUid: string
  ndefUrl: string
  deviceId: string
  writeStatus: string
  photoObjectKey?: string
  writtenAt: string
  createdAt: string
}
```

说明：

- `badgeId` 必填。
- `photoObjectKey` 必须是对象 key，不能是 `http://` 或 `https://` URL。
- `writtenAt` 可不传；后端会使用当前时间。

## Android 写卡 App 配对令牌管理

这些接口供 Web 管理后台生成和管理 Android 写卡 App 的 token 与二维码。

### 列出 App Token

```http
GET /api/roundnfc/admin/app-tokens
```

响应 `data`：

```json
{
  "items": [
    {
      "id": "uuid",
      "name": "手机 A",
      "tokenPrefix": "rnfca_xxx",
      "enabled": true,
      "lastUsedAt": "2026-07-09T12:00:00Z",
      "createdAt": "2026-07-09T12:00:00Z",
      "updatedAt": "2026-07-09T12:00:00Z"
    }
  ]
}
```

### 创建 App Token

```http
POST /api/roundnfc/admin/app-tokens
```

请求：

```json
{
  "name": "手机 A",
  "apiBase": "https://example.com"
}
```

响应 `data`：

```json
{
  "item": {},
  "token": "rnfca_xxxxxxxxx",
  "pairing": {
    "protocol": "roundnfc-writer",
    "version": 1,
    "name": "手机 A",
    "apiBase": "https://example.com",
    "apiPrefix": "/api/roundnfc",
    "tokenHeader": "X-RoundNFC-App-Token",
    "token": "rnfca_xxxxxxxxx",
    "endpoints": {
      "listStyleTemplates": "/api/roundnfc/app/style-templates",
      "listBadges": "/api/roundnfc/app/badges",
      "getBadge": "/api/roundnfc/app/badges/{id}",
      "upsertBadge": "/api/roundnfc/app/badges",
      "presignUpload": "/api/roundnfc/app/uploads/presign",
      "createWrite": "/api/roundnfc/app/nfc-writes",
      "presignCoserPhoto": "/api/roundnfc/app/badges/{id}/coser-photo/presign",
      "upsertCoserBinding": "/api/roundnfc/app/badges/{id}/coser-binding",
      "getCoserBinding": "/api/roundnfc/app/badges/{id}/coser-binding"
    },
    "createdAt": "2026-07-09T12:00:00Z"
  }
}
```

说明：

- `token` 明文只会在创建时返回一次，前端应立即生成二维码或展示给用户。
- `apiBase` 可不传；后端会根据请求推断。

### 启用或禁用 App Token

```http
PATCH /api/roundnfc/admin/app-tokens/{id}
```

请求：

```json
{
  "enabled": false
}
```

响应 `data`：

```json
{
  "ok": true
}
```

### 删除 App Token

```http
DELETE /api/roundnfc/admin/app-tokens/{id}
```

响应 `data`：

```json
{
  "ok": true
}
```

## 常见状态码

- `200`：成功
- `400`：请求体或参数非法
- `401`：缺少或无效 JWT
- `403`：Turnstile 校验失败，或对象 token 无效
- `404`：资源不存在
- `410`：一次性对象链接已过期或已消费
- `413`：上传文件过大
- `415`：上传媒体类型不支持
- `429`：公开提交或上传触发限流
- `503`：COS 未配置或管理后台鉴权未配置
