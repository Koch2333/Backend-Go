# 后台模块前端包规范

本仓库里 `web/` 是「后端所有模块共享的后台 SPA」。每个后端模块下发
一份「前端包」，按本路进走加一个文件夹即可，不需什么 Vite 工程、auth、layout
都重新启一遇。

## 思想

- **静态发现**：Shell 启动时用 `import.meta.glob('../modules/*/module.ts')` 扫描
  所有模块。加一个文件夹 = 多一个模块。
- **模块自描述**：每个模块的 `module.ts` default-export 一个 `ModuleManifest`，
  告诉 Shell 路由、侧边栏、API 前缀。
- **运行时隔离**：每个模块一份独立的 auth store + axios 实例，token 互不干扰。
  401 自动跳本模块的登录页。
- **路由自动加前缀**：你填 `path: 'badges'`，Shell 拼为 `/m/<name>/badges`。
  而这个 SPA 本身跑在 `/admin/`。所以完整 URL 是 `/admin/m/roundnfc/badges`。

## 加一个新模块的 5 步

1. 在 `web/src/modules/` 里复制 `_template/` 为 `<你的模块名>/`。
2. 改 `core.ts`：填 `name`（与 `internal/<name>` 同名）与 `apiPrefix`。
3. 改 `api.ts`：加你的接口函数。靠 `M.http().get(...)` 发请求，靠 `M.unwrap(resp)` 拆包。
4. 改 `module.ts`：改 `title`、`description`、`nav`、`adminRoutes`、`blankRoutes`。
5. 重跑 `pnpm dev`。模块自动出现在侧边栏；访问 `/admin/m/<name>/...` 即可。

## 后端合契

任何想接入本 Shell 的后端模块应提供：

| 路径 | 说明 |
|------|------|
| `POST {apiPrefix}/admin/login`        | 接受 `{username, password}`，返回 `{token, expiresAt, username}` |
| `GET  {apiPrefix}/admin/me`           | JWT 中间件保护，返回 `{username}`（或你需要的 profile） |
| 其余 `{apiPrefix}/admin/...` | 按业务需要设计。响应必须包 `{code, message, data}` |

JWT 中间件推荐复用 `internal/auth.Required(secret)`。每个模块可以用自己的 secret
（像 RoundNFC 用 `ROUNDNFC_JWT_SECRET`），也可以多模块共享一个「全站后台」secret，
看你是否需要「一次登录多模块生效」。

## 运行时 API、Shell 提供什么

```ts
import { defineModule } from '@/shell/defineModule'

export const M = defineModule({
  name: 'myfeat',
  apiPrefix: '/api/myfeat',
})

// M.useAuth   — Pinia store: { token, username, expiresAt, isLoggedIn, set(), clear() }
// M.http()    — AxiosInstance：自动携带 Bearer，401 跳本模块 /m/<name>/login
// M.unwrap    — ApiResult 拆包帮手
// M.signIn(token, username, expiresAt) — 登录后调一下存到 store
```

公共组件 / 函数还有：

```ts
import { useBlobImage } from '@/shell/blobImage'   // 一次性 URL → blob
import { extractMessage } from '@/shell/http'      // axios 错误 → 人话
import AdminLayout from '@/shell/AdminLayout.vue'  // 跳过默认 layout 包裹时定制用
```

## 规范上的其他约定

- `module.ts` 只负责描述，不要在里面跳路由或调 API；Shell 是运行时 import 这个文件的。
- 路由里 `path` 填**相对路径**：Shell 会拼为绝对路径。你自己需要跳别的页时，
  请用带模块名的绝对路径，比如 `router.push('/m/myfeat/badges')`。
- `nav[].to` 必须是绝对路径。最常见的写法：`/m/<name>/<page>`。
- 登录页放 `blankRoutes`，其他页放 `adminRoutes`。`adminRoutes` 的路由会被
  Shell 自动上 `requiresAuth + AdminLayout`。
- 不要在模块里手动 `import` 别的模块文件 —— 会闹源码圈子、违反模块隔离。
  纯展示只读的东西靠共享 API（`@/shell/...`）打出去。

## Build & 部署

```bash
cd web
pnpm install
pnpm build         # 输出到 ../internal/adminui/dist
# 后端二进制会通过 embed.FS 直接内嵌该目录到 /admin/* 路径
back in repo root:
go build ./cmd/server
```
