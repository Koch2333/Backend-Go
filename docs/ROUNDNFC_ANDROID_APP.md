# RoundNFC Android Writer App Integration

This document is for the Android app Codex session. The backend side is already implemented in this repo.

## Goal

Android writer app should pair with the backend by scanning a QR code from the RoundNFC admin UI, then use the token in that QR code to call writer APIs.

Admin path:

1. Open RoundNFC admin.
2. Go to `安全设置`.
3. Use `Android App 配对`.
4. Create a pairing item.
5. Scan the QR code with the Android app.

## QR Code Payload

The QR code content is a JSON string. Example shape:

```json
{
  "protocol": "roundnfc-writer",
  "version": 1,
  "name": "写卡手机",
  "apiBase": "https://example.com",
  "apiPrefix": "/api/roundnfc",
  "tokenHeader": "X-RoundNFC-App-Token",
  "token": "rnfca_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
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
  "createdAt": "2026-07-05T00:00:00Z"
}
```

Android should persist at least:

- `apiBase`
- `apiPrefix`
- `tokenHeader`
- `token`
- `endpoints`

Build request URLs with:

```text
fullUrl = apiBase + endpoint
```

Do not add another `apiPrefix` if using values from `endpoints`, because endpoint values already include the prefix.

## Style Presets

The backend is the source of truth for badge style templates. Android should not hard-code style keys.

Always load style presets from backend before showing the style picker:

```http
GET /api/roundnfc/app/style-templates
```

Response data:

```json
{
  "items": [
    {
      "key": "sakura",
      "label": "樱花粉",
      "description": "",
      "imageUrl": "style-templates/sakura/xx.png",
      "imageOriginalUrl": "/api/roundnfc/objects/one-shot-token",
      "imagePreviewUrl": "/api/roundnfc/objects/one-shot-token",
      "payload": { "theme": "sakura" },
      "enabled": true
    }
  ]
}
```

Android write UI should let the user choose one of these returned `key` values, or empty string for no preset/custom image. Use `imageOriginalUrl` for the original template image bytes, keep `imageUrl` as backend metadata/object key, and treat `payload` as template-specific JSON. `imagePreviewUrl` is kept as a backward-compatible alias and currently points to the same original image. If Android sends a `styleKey` not present in the enabled backend template list, backend returns `400 invalid styleKey`.

## Authentication

Every `/app/*` request must include:

```http
X-RoundNFC-App-Token: <token from QR>
```

The legacy static header `X-App-Token` may still work for backend compatibility, but the Android app should use `tokenHeader` from the QR payload.

If the token is disabled or deleted in admin, backend returns `401`.

## Endpoints

Base prefix is usually:

```text
/api/roundnfc
```

Android-specific APIs are under:

```text
/api/roundnfc/app
```

### List Style Templates

```http
GET /api/roundnfc/app/style-templates
```

Use this to populate the Android style picker. The selected `key` is sent as `styleKey` when upserting a badge.

### List Badges

```http
GET /api/roundnfc/app/badges?q=&limit=50&offset=0
```

Response data:

```json
{
  "items": [
    {
      "id": "badge-001",
      "title": "badge-001",
      "series": "",
      "type": "badge",
      "styleKey": "sakura",
      "imageUrl": "",
      "description": "",
      "serialNo": "",
      "releasedAt": "",
      "createdAt": "2026-07-05T00:00:00Z",
      "updatedAt": "2026-07-05T00:00:00Z"
    }
  ],
  "total": 1
}
```

Backend wraps all successful responses as:

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

### Get Badge

```http
GET /api/roundnfc/app/badges/{id}
```

### Upsert Badge Style

Used by Android when creating/updating a badge by NFC/card ID.

```http
POST /api/roundnfc/app/badges
Content-Type: application/json
```

Body:

```json
{
  "id": "badge-001",
  "styleKey": "sakura"
}
```

Allowed `styleKey` values are returned by `GET /api/roundnfc/app/style-templates`. Empty string is also accepted for no preset/custom image.

## Second Write: Bind CN and Coser Photo

The second write flow binds `cn` (coser name) and a photo object to an existing badge ID.

Recommended Android flow:

1. Read/know `badgeId`.
2. Ask backend for a one-time COS upload URL.
3. Upload the photo directly to COS with `PUT`.
4. Confirm binding with JSON payload.

### Presign Coser Photo Upload

```http
POST /api/roundnfc/app/badges/{id}/coser-photo/presign
Content-Type: application/json
```

Body:

```json
{
  "fileName": "coser.jpg",
  "contentType": "image/jpeg"
}
```

Response data:

```json
{
  "uploadUrl": "https://bucket.cos.region.myqcloud.com/...",
  "objectKey": "roundnfc/coser-photos/badge-001/uuid.jpg",
  "method": "PUT",
  "headers": {
    "Authorization": "q-sign-algorithm=sha1&..."
  },
  "expiresIn": 300
}
```

Then upload:

```http
PUT <uploadUrl>
Authorization: <headers.Authorization>
Content-Type: image/jpeg
```

Upload notes:

- Use a clean HTTP client for COS upload, not the RoundNFC backend API client with interceptors.
- Do not send `X-RoundNFC-App-Token` or backend `Authorization` to COS.
- Copy every key/value from response `headers` into the COS `PUT` request.
- `uploadUrl` is the plain COS object URL; COS auth is carried by response `headers.Authorization`.

### Confirm CN Binding

```http
POST /api/roundnfc/app/badges/{id}/coser-binding
Content-Type: application/json
```

Body:

```json
{
  "cn": "CoserName",
  "photoObjectKey": "roundnfc/coser-photos/badge-001/uuid.jpg",
  "deviceId": "android-device-id-or-name",
  "tagUid": "04AABBCCDD",
  "writtenAt": "2026-07-06T12:34:56Z"
}
```

Notes:

- `cn` is required.
- `photoObjectKey` is required and must be an object key, not a full URL.
- This is an upsert keyed by `badgeId`; calling it again replaces the previous `cn/photoObjectKey` binding for that badge.

### Get CN Binding

```http
GET /api/roundnfc/app/badges/{id}/coser-binding
```

Returns the currently bound `cn` and `photoObjectKey`.

If badge does not exist, backend creates it with:

- `id` from request
- `title` same as `id`
- `type` as `badge`
- `styleKey` from request

### Presign Upload

Used before uploading write-photo JPEG to COS.

```http
POST /api/roundnfc/app/uploads/presign
Content-Type: application/json
```

Body:

```json
{
  "badgeId": "badge-001",
  "fileName": "write.jpg",
  "contentType": "image/jpeg",
  "purpose": "nfc-write"
}
```

Response data:

```json
{
  "uploadUrl": "https://bucket.cos.region.myqcloud.com/...",
  "objectKey": "roundnfc/nfc-writes/badge-001/uuid.jpg",
  "method": "PUT",
  "headers": {
    "Authorization": "q-sign-algorithm=sha1&..."
  },
  "expiresIn": 300
}
```

Android then uploads the JPEG with:

```http
PUT <uploadUrl>
Authorization: <headers.Authorization>
Content-Type: image/jpeg
```

Save `objectKey`; pass it to `createWrite`.

If COS is not configured, backend returns `503` with message `cos not configured`.

### Create NFC Write Record

Call this after the NFC write succeeds or fails. If a photo was uploaded, include `photoObjectKey`.

```http
POST /api/roundnfc/app/nfc-writes
Content-Type: application/json
```

Body:

```json
{
  "badgeId": "badge-001",
  "tagUid": "04AABBCCDD",
  "ndefUrl": "https://example.com/nfc/badge-001",
  "deviceId": "android-device-id-or-name",
  "writeStatus": "success",
  "photoObjectKey": "roundnfc/nfc-writes/badge-001/uuid.jpg",
  "writtenAt": "2026-07-05T12:34:56Z"
}
```

Notes:

- `badgeId` is required.
- `writtenAt` is optional; backend uses current UTC time if omitted.
- `photoObjectKey` must be an object key, not an absolute URL.
- `writeStatus` is currently free text. Recommended values: `success`, `failed`.

## Android Implementation Checklist

1. Add QR scanner flow for pairing.
2. Parse QR as JSON and verify:
   - `protocol == "roundnfc-writer"`
   - `version == 1`
   - `apiBase`, `tokenHeader`, `token` are non-empty
3. Persist pairing config securely. Store token in encrypted storage if available.
4. Build an HTTP client that injects the configured token header on `/app/*` calls.
5. Add connection test by calling `GET listBadges`.
6. Add write flow:
   - Load style templates with `GET listStyleTemplates`.
   - Let user choose a backend-returned style key.
   - Read/select badge ID.
   - Optionally call `upsertBadge`.
   - Write NFC tag.
   - Optionally request presign and upload JPEG.
   - Call `createWrite`.
7. Add second-write flow:
   - Request `presignCoserPhoto`.
   - Upload photo to COS.
   - Call `upsertCoserBinding` with `cn` and returned `objectKey`.
8. On `401`, show a clear message that pairing was revoked or expired and ask the user to pair again.

## Backend Source References

- Route definitions: `internal/roundnfc/router.go`
- Pairing response builder: `internal/roundnfc/handler_admin.go`
- Token verification: `internal/roundnfc/app_token.go`
- NFC write payload: `internal/roundnfc/handler_admin.go`
- Frontend pairing UI: `web/src/modules/roundnfc/views/Security.vue`
