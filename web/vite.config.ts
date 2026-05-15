import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { fileURLToPath, URL } from 'node:url'

// 输出产物到 internal/adminui/dist/，供 Go 在服务启动时通过 embed.FS 直接内嵌到二进制。
// 开发期 vite dev 仍正常在 :5174，通过 proxy 跳后端；避免与 RoundNFC 公开站 :5173 冲突。
// base 默认 '/admin/'（被 Go 二进制挂载在 /admin/ 下）。SPA 单独部署到根域名时
// 用 `VITE_BASE=/ pnpm build` 改成 '/'。
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const base = env.VITE_BASE || '/admin/'
  return {
    base,
    plugins: [
      vue({
        template: {
          // <md-*> are @material/web custom elements; Vue compiler must leave them alone.
          compilerOptions: { isCustomElement: (tag) => tag.startsWith('md-') },
        },
      }),
      Components({
        dts: 'src/components.d.ts',
      }),
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      host: true,
      port: 5174,
      proxy: {
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true,
        },
      },
    },
    build: {
      outDir: '../internal/adminui/dist',
      emptyOutDir: true,
    },
  }
})
