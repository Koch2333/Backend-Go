# Roast

A Go backend for my API service, powered by Gin.

## Features

- Atom IT Club's some basic backend functions. (Thanks to [@Chemio9](https://github.com/chemio9))
- Redirect Service
- Email Sender (SMTP and Microsoft Graph)
- Avatar provider
- **RoundNFC**: badge metadata + photo / To-sign request collection, with admin API,
  one-shot signed image URLs and pluggable object storage (Local now, COS / OSS planned).

## Layout

```
cmd/
  server/        # full server, mounts every internal/<mod>
  roundnfc/      # standalone build, only the RoundNFC module
  genpw/         # bcrypt helper for ROUNDNFC_ADMIN_PASSWORD_HASH
  genmod/        # generates internal/bootstrap/mod/autogen_imports.go
internal/
  app/           # entrypoint shared by cmd/server
  auth/          # JWT + bcrypt + gin middleware (used by RoundNFC admin)
  risk/          # Cloudflare Turnstile verifier + sliding-window rate limiter
  bootstrap/     # plug registry + auto-mount
  avatar/        # avatar upload/serve
  email/         # SMTP / Graph senders
  redirect/      # /api/redirect/...
  roundnfc/      # NEW — RoundNFC public + admin API
  rhythmgames/
  integrations/
  handler/
pkg/
  paths/
  objstore/      # NEW — Storage interface + Local driver, COS/OSS stubs (build tags)
```

## Quick start

Prerequisites: **Go 1.23+**, **Node 24+**, **pnpm 10+**.

```bash
# Linux / macOS
make setup          # go mod + web SPA + codegen, one step

# Windows (cmd)
make setup
```

| Target | What it does |
|--------|-------------|
| `setup` | Full init: `deps` → `web` → `generate` |
| `deps` | `go mod download` + `go mod tidy` |
| `web` | `web-deps` + `spa` |
| `web-deps` | `pnpm install` (web dependencies only) |
| `spa` | `pnpm build` (compile admin SPA only) |
| `generate` | `go generate` — scan `internal/**/module.go`, write `autogen_imports.go` |
| `build` | `setup` + compile release binary → `dist/` |
| `dev` | `deps` + `generate` + compile (skip web, for daily dev) |
| `run` | `dev` + run the binary |
| `clean` | Remove `dist/` and `autogen_imports.go` |

> POSIX uses the `Makefile`; Windows uses `make.bat` — same targets, same names.

## Build

### Full server

```bash
go generate ./internal/bootstrap/mod   # scans internal/**/module.go
go build -o bin/server ./cmd/server
```

### Standalone RoundNFC

```bash
go build -o bin/roundnfc ./cmd/roundnfc
```

This binary contains only the RoundNFC module (no avatar / email / redirect /
rhythmgames). Suitable for handing off to someone who only wants the badge
backend.

### Object storage drivers

Local driver is the default. To compile in a cloud driver:

```bash
go build -tags=cos ./cmd/roundnfc   # Tencent Cloud COS (TODO)
go build -tags=oss ./cmd/roundnfc   # Aliyun OSS (TODO)
```

Driver implementations live in `pkg/objstore/cos.go` and `pkg/objstore/oss.go`
behind build tags; fill in the TODOs when picking a vendor.

## RoundNFC quick start

```bash
go run ./cmd/roundnfc                 # first run writes config/roundnfc/.env
go run ./cmd/genpw "choose-a-pw"      # paste hash into ROUNDNFC_ADMIN_PASSWORD_HASH
```

Endpoints (mounted at `/api/roundnfc` by default):

| Method | Path                                       | Auth     | Purpose                                |
|--------|--------------------------------------------|----------|----------------------------------------|
| GET    | `/badges/:id`                              | -        | Public badge view; `imageUrl` is one-shot |
| POST   | `/badges/:id/photo-requests`               | Turnstile| Fan submits a return-photo request     |
| POST   | `/badges/:id/autograph-requests`           | Turnstile| Fan submits a To-sign request          |
| POST   | `/uploads`                                 | Turnstile| Fan-uploaded attachment                |
| GET    | `/objects/:token`                          | -        | One-shot blob fetch                    |
| POST   | `/admin/login`                             | -        | Returns JWT                            |
| GET    | `/admin/me`                                | JWT      | Probe                                  |
| GET/POST/PUT/DELETE | `/admin/badges[...]`              | JWT      | Badge CRUD                             |
| POST   | `/admin/badges/:id/image`                  | JWT      | Replace badge image (multipart)        |
| POST   | `/admin/uploads/presign`                   | JWT / X-App-Token | Return 5-minute COS PUT URL     |
| POST   | `/admin/nfc-writes`                        | JWT / X-App-Token | Save NFC write metadata + photo key |
| GET/PATCH | `/admin/photo-requests[...]`            | JWT      | List + status update                   |
| GET/PATCH | `/admin/autograph-requests[...]`        | JWT      | List + status update                   |

For Android direct COS upload, configure `ROUNDNFC_COS_BUCKET`,
`ROUNDNFC_COS_REGION`, `ROUNDNFC_COS_SECRET_ID`, `ROUNDNFC_COS_SECRET_KEY`, and
optionally `ROUNDNFC_ADMIN_APP_TOKEN` on the backend only. The Android client
calls `/admin/uploads/presign`, uploads with HTTP `PUT` to `uploadUrl`, then
saves only `photoObjectKey` through `/admin/nfc-writes`. Keep the COS bucket
private; do not store permanent COS URLs in the database.

## TODO

- Implement COS / OSS drivers in `pkg/objstore`.
- Connect frontend admin against `/api/roundnfc/admin/*`.
- API documentation generation.
