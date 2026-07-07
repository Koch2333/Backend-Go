# RoundNFC Tencent Cloud COS Setup

RoundNFC uses Tencent Cloud COS for direct image uploads.

The backend generates a short-lived COS `PUT` authorization header. Android uploads the image directly to COS, then sends the returned `objectKey` back to the backend in JSON.

## Backend Flow

For coser photo binding:

```text
Android -> Backend:
POST /api/roundnfc/app/badges/{id}/coser-photo/presign

Backend -> Android:
uploadUrl + objectKey + headers

Android -> COS:
PUT <uploadUrl>
Authorization: <headers.Authorization>
Content-Type: image/jpeg

Android -> Backend:
POST /api/roundnfc/app/badges/{id}/coser-binding
{
  "cn": "CoserName",
  "photoObjectKey": "<objectKey>"
}
```

For normal NFC write photos:

```http
POST /api/roundnfc/app/uploads/presign
```

Both flows use the same COS bucket.

## Create Bucket

Create one COS bucket for RoundNFC images.

Recommended settings:

- Bucket permission: `私有读写`.
- Do not use public write.
- Region: choose the region closest to backend/users, for example `ap-shanghai` or `ap-guangzhou`.
- Bucket name in backend config must be the full bucket name, including APPID.

Example full bucket name:

```text
roundnfc-1250000000
```

Tencent Cloud docs:

- Create bucket: https://cloud.tencent.com/document/product/436/14106
- CORS: https://cloud.tencent.com/document/product/436/13318

## CORS

Native Android is not blocked by browser CORS, but configure CORS now so WebView/debug tools/admin direct uploads can work later.

In COS console:

```text
Bucket detail -> Security Management -> CORS -> Add Rule
```

Recommended rule:

```text
Allowed Origins:
*
```

Or use stricter origins:

```text
https://your-admin-domain.example
http://localhost:5174
http://localhost:8081
```

Methods:

```text
PUT
POST
GET
HEAD
OPTIONS
```

Allow-Headers:

```text
*
```

Expose-Headers:

```text
ETag
x-cos-request-id
```

Max-Age:

```text
600
```

## Backend Environment

Set these in the RoundNFC config env file:

```env
ROUNDNFC_COS_BUCKET=roundnfc-1250000000
ROUNDNFC_COS_REGION=ap-shanghai
ROUNDNFC_COS_SECRET_ID=your-secret-id
ROUNDNFC_COS_SECRET_KEY=your-secret-key
ROUNDNFC_COS_SCHEME=https
```

Notes:

- On local `go run`, this is usually `config/roundnfc/.env` under the repo.
- If `CONFIG_DIR` is set, the backend reads `$CONFIG_DIR/config/roundnfc/.env`.
- If running a compiled binary and `CONFIG_DIR` is not set, the backend reads `config/roundnfc/.env` next to the binary.
- `config/roundnfc/.env` is loaded into the backend process on startup.
- `config/roundnfc/local.env` overrides `config/roundnfc/.env` when present.
- `ROUNDNFC_COS_BUCKET` must be the full bucket name, including APPID.
- `ROUNDNFC_COS_REGION` must match the bucket region.
- Restart the RoundNFC backend after changing these values.
- Startup logs include `[paths] config base = ...` and `[roundnfc/config] ... cos_secret_id_prefix=...`; use those lines to confirm the backend is reading the expected file and key.

## CAM Key Permissions

For a quick test, a main-account key can work, but production should use a CAM sub-user/key limited to this bucket.

Minimum required permissions:

- `PutObject`
- `GetObject`
- `HeadObject`
- `OptionsObject` if needed by tooling/CORS checks

Restrict the resource to the RoundNFC bucket, and preferably to the object prefix:

```text
roundnfc/*
```

Practical Tencent Cloud console setup:

1. Go to `CAM -> Users -> target sub-user -> Permissions`.
2. Attach a COS policy that allows write access to the target bucket.
3. If using a custom policy, include at least object write permission for:

```text
bucket: roundnfc-1300177615
region: ap-shanghai
prefix: roundnfc/*
actions: PutObject, GetObject, HeadObject
```

For local testing, the fastest way to confirm whether the issue is only CAM permission is to temporarily attach Tencent Cloud's built-in COS full-access policy to this CAM user, retry one upload, then replace it with a narrower bucket/prefix policy after it works.

Do not place `SecretKey` in the Android app. The Android app only receives a short-lived COS upload URL and authorization header from the backend.

## Quick Test

After backend env is configured, pair Android app or call with app token:

```http
POST /api/roundnfc/app/badges/badge-001/coser-photo/presign
Content-Type: application/json
X-RoundNFC-App-Token: <token>

{
  "fileName": "coser.jpg",
  "contentType": "image/jpeg"
}
```

Expected backend response data:

```json
{
  "uploadUrl": "https://roundnfc-1250000000.cos.ap-shanghai.myqcloud.com/...",
  "objectKey": "roundnfc/coser-photos/badge-001/uuid.jpg",
  "method": "PUT",
  "headers": {
    "Authorization": "q-sign-algorithm=sha1&..."
  },
  "expiresIn": 300
}
```

Android upload requirements:

- Use a separate/plain HTTP client for the COS `PUT`.
- Copy response `headers` into the COS request.
- Do not send `X-RoundNFC-App-Token` to COS.
- Do not replace `headers.Authorization` with the backend app token.

If COS is not configured, backend returns:

```text
503 cos not configured
```

## Troubleshooting 403

### `SignatureDoesNotMatch`

The request was signed incorrectly or Android changed the signed request. Check:

- Android uses `method = PUT`.
- Android uploads to the exact `uploadUrl` from backend.
- Android copies every response `headers` key/value into the COS request.
- Android does not send backend `X-RoundNFC-App-Token` or backend `Authorization` to COS.
- Backend `ROUNDNFC_COS_BUCKET` and `ROUNDNFC_COS_REGION` match the bucket.

### `AccessDenied`

The signature was accepted, but Tencent Cloud rejected the operation by permission policy. Check:

- The `SecretId` in backend log is the CAM user/key you expect.
- That CAM user has `PutObject` permission on the same bucket shown in backend log.
- The allowed resource/prefix includes the uploaded object key, for example `roundnfc/coser-photos/XG29D9L/...`.
- The bucket is under the same Tencent Cloud account as the key.
- No bucket policy explicitly denies write operations.

Backend logs one line for every presign:

```text
roundnfc cos presign badge_id=... purpose=... object_key=... bucket=... region=... secret_id_prefix=... expires_in=300s
```

Compare this log with the COS error `Resource` field. The bucket, region, and object prefix must all match the policy.
