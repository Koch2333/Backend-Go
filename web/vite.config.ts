import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { VantResolver } from 'unplugin-vue-components/resolvers'
import { fileURLToPath, URL } from 'node:url'

// 输出产物到 internal/adminui/dist/，供 Go 在服务启动时通过 embed.FS 直接内嵌到二进制。
// 开发期 vite dev 仍正常在 :5174，通过 proxy 跳后端；避免与 RoundNFC 公开站 :5173 冲突。
export default defineConfig({
  base: '/admin/',
  plugins: [
    vue({
      template: {
        // <md-*> are @material/web custom elements; Vue compiler must leave them alone.
        compilerOptions: { isCustomElement: (tag) => tag.startsWith('md-') },
      },
    }),
    Components({
      resolvers: [VantResolver()],
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
})
