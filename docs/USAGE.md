# 使用说明

> 关于：构建、首次运行、配置文件如何释放、各模块的后台入口和前端开发流程。

## 1. 目录与组件

```
cmd/server     # 完整服务：挂载 internal/<module> 下所有模块
cmd/roundnfc   # 单模块构建：只跑 RoundNFC（移交给只关心徽章的人）
cmd/genpw      # 生成 bcrypt 哈希（用于 *_ADMIN_PASSWORD_HASH）
cmd/genmod     # 重新生成 internal/bootstrap/mod/autogen_imports.go
internal/      # 模块实现
web/           # 后台 SPA（Vue 3 + Vite + @material/web）
internal/adminui/dist/  # web 构建产物，server 启动时通过 embed.FS 内嵌
```

## 2. 首次运行：配置文件释放在哪里？

每个模块都实现了 `InitEnv()`。启动时，`internal/bootstrap/mod` 会遍历所有已启用的模块，按下面的规则把 `config/<module>/.env` 放到“运行目录”下：

| 场景             | 实际目录                       |
| ---------------- | ------------------------------ |
| `go run ./cmd/server`            | 当前工作目录（`os.Getwd()`） |
| 直接执行已构建二进制              | 二进制所在目录                 |
| 设置了环境变量 `CONFIG_DIR=/path` | `/path`                        |

判定逻辑见 `pkg/paths/ExecDir()`：
1. 优先用 `CONFIG_DIR`；
2. 否则取 `os.Executable()` 的目录；
3. 但如果该目录在 `/tmp/.../go-build…/` 里（即 `go run` 临时编译产物），认为是开发环境，退回 `os.Getwd()`；
4. 最终兑底返回 `.`。

涉及的模块：`avatar / email / redirect / rhythmgames / roundnfc / integrations/aicweb`。

每个模块只在 `.env` 不存在时写默认值（不会覆盖你已经改过的配置）。`local.env` 则会强覆盖（建议用于本地开发，已 gitignore）。

举例（首次运行 `go build -o bin/server ./cmd/server && cd /opt/roast && /opt/roast/server`）将得到：

```
/opt/roast/
├── server
├── config/
│   ├── avatar/.env
│   ├── email/{.env.example, local.env.example}
│   ├── redirect/.env       # 含随机生成的 JWT_SECRET
│   ├── rhythmgames/.env
│   ├── roundnfc/.env       # 含随机 JWT_SECRET + OBJECT_HMAC_KEY
│   └── aicweb/.env
└── databases/              # 由各模块运行时再建
```

## 3. 构建

### 3.1 完整服务

```bash
# 1) 构建前端 SPA → internal/adminui/dist
cd web
pnpm install --no-frozen-lockfile
pnpm build
cd ..

# 2) 重新生成模块导入表（如果加了新模块）
go generate ./internal/bootstrap/mod

# 3) 构建二进制（dist 已 embed 进去）
go build -o bin/server ./cmd/server
```

CI（`.github/workflows/build.yml`）已经替你完成上面所有步骤，并把 `web-dist` 和成品二进制都作为构建产物保留 30 天。

### 3.2 单独的 RoundNFC

```bash
go build -o bin/roundnfc ./cmd/roundnfc
```

### 3.3 生成密码哈希

```bash
go run ./cmd/genpw "your-password"
# 把输出粘到 config/<mod>/.env 里的 *_ADMIN_PASSWORD_HASH
```

## 4. 后台 UI（Material 3）

`web/` 是一个 Vue 3 + Vite SPA，统一用 `@material/web`（Google 官方 Material 3 Web Components）。Tailwind 用来做布局，Vant 已经完全移除。

### 4.1 已注册的 M3 组件

见 `web/src/shell/m3.ts`：filled-button / outlined-button / text-button / icon-button / outlined-text-field / filled-text-field / switch / dialog / divider / icon / list / list-item / circular-progress / chip-set / assist-chip。

### 4.2 全局辅助

| 文件                         | 作用                                                        |
| ---------------------------- | ----------------------------------------------------------- |
| `web/src/shell/toast.ts`     | `showSuccessToast(msg)` / `showFailToast(msg)`，胶囊型提示 |
| `web/src/shell/confirm.ts`   | `showConfirmDialog({ title, message })`，返回 Promise        |
| `web/src/shell/m3.ts`        | M3 组件副作用注册（main.ts 已 import）                     |
| `web/src/shell/AdminLayout.vue` | 左侧导航 + 顶部条；导航图标用 Material Symbols 名         |

`@/shell/toast` 和 `@/shell/confirm` 是 Vant 同名 API 的替代品，按需引入即可。

### 4.3 写视图的样板

```vue
<script setup lang="ts">
import { ref } from 'vue'
import { showFailToast, showSuccessToast } from '@/shell/toast'

const value = ref('')
const editDialog = ref<HTMLDialogElement & { show: () => void; close: () => void } | null>(null)
</script>

<template>
  <md-outlined-text-field
    label="名称"
    :value="value"
    @input="(e: any) => (value = e.target.value)"
    class="w-full"
  />

  <md-filled-button @click="editDialog?.show()">
    <md-icon slot="icon">add</md-icon>
    新建
  </md-filled-button>

  <md-dialog ref="editDialog">
    <div slot="headline">编辑</div>
    <div slot="content">…</div>
    <div slot="actions">
      <md-text-button @click="editDialog?.close()">取消</md-text-button>
      <md-filled-button @click="onSave">保存</md-filled-button>
    </div>
  </md-dialog>
</template>
```

注意：
- `md-outlined-text-field` 用 `:value="x" @input="(e:any)=>(x=e.target.value)"`，不能直接 `v-model`。
- `md-switch` 用 `:selected="x" @change="(e:any)=>(x=e.target.selected)"`。
- `md-icon` 的内容是 Material Symbols Outlined 的图标名（例如 `add`、`delete`、`search`、`fingerprint`）。

### 4.4 开发

```bash
cd web
pnpm dev          # http://localhost:5174/admin/
```

Vite 已经设好 `/api` 代理到 `http://localhost:8080`，正常起后端即可。

## 5. 各模块后台入口

| 模块     | 后台路由前缀                  | 登录页                  |
| -------- | ----------------------------- | ----------------------- |
| redirect | `/api/redirect/admin/*`       | `/admin/m/redirect/login`  |
| roundnfc | `/api/roundnfc/admin/*`       | `/admin/m/roundnfc/login`  |

两个模块共用同一个 `authflow` 包：用户名 + 密码 + 可选 TOTP + 可选 Passkey/WebAuthn。账号配置在对应的 `config/<mod>/.env` 中（`*_ADMIN_USERNAME` / `*_ADMIN_PASSWORD_HASH`）。

## 6. 常见问题

**Q：用 `go run` 启动，配置没出现在项目根目录？**
A：旧版本会把 `.env` 放到 `/tmp/go-build…/` 下。已修复：检测到 go-build 路径会自动回退到 `os.Getwd()`。或者直接 `export CONFIG_DIR=$(pwd)` 强制指定。

**Q：怎么把账号密码改掉？**
A：`go run ./cmd/genpw "newpw"` 得到 bcrypt 哈希，覆盖 `config/<mod>/.env` 的 `*_ADMIN_PASSWORD_HASH`，重启服务。

**Q：怎么加新的 M3 组件？**
A：在 `web/src/shell/m3.ts` 里 `import '@material/web/<dir>/<name>.js'`，模板里就能写 `<md-…>`。

**Q：Vue 警告 "unknown custom element: md-…"？**
A：`web/vite.config.ts` 里 `compilerOptions.isCustomElement = (tag) => tag.startsWith('md-')` 已经处理。如果新增了别的前缀，扩展这里即可。
