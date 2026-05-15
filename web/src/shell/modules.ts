import type { RouteRecordRaw } from 'vue-router'
import type { ModuleManifest } from './types'

/**
 * 自动发现所有 src/modules/<name>/module.ts。
 *
 * Vite 推荐用 import.meta.glob 静态扫描：产物仍是 tree-shake 友好的代码拆分。
 * 这里用 eager 加载；Shell 启动时需要拿到全部 manifest 才能拼路由/侧边栏。
 */
const eager = import.meta.glob<{ default: ModuleManifest }>(
  '../modules/*/module.ts',
  { eager: true },
)

/** 已识别的模块。_template 不进入运行时。 */
export const MODULES: ModuleManifest[] = Object.values(eager)
  .map((m) => m.default)
  .filter((m): m is ModuleManifest => !!m && m.name !== '_template')
  .sort((a, b) => a.name.localeCompare(b.name))

/** 在路由表里拼接为 /m/<name>/<your-path>。 */
export function prefixRoutes(
  modName: string,
  routes: RouteRecordRaw[] | undefined,
): RouteRecordRaw[] {
  if (!routes) return []
  return routes.map((r) => ({
    ...r,
    path: joinPath(`/m/${modName}`, r.path),
  }))
}

function joinPath(base: string, sub: string): string {
  if (!sub) return base
  if (sub.startsWith('/')) return base + sub
  return base + '/' + sub
}
