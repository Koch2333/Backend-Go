import type { RouteRecordRaw } from 'vue-router'

export interface AdminNavItem {
  /**
   * 全路径，例如 '/m/roundnfc/badges'。
   * 为了不在模块里额外计算前缀，这里明说需是全路径。
   */
  to: string
  label: string
  /** Vant 图标名，可选。 */
  icon?: string
}

/**
 * 一个后端模块的「后台前端包」描述。
 * 每个模块在 src/modules/<name>/module.ts 里 default-export 一个。
 */
export interface ModuleManifest {
  /** 与后端 internal/<name> 同名。全局唯一。 */
  name: string
  /** 人读标题，出现在侧边栏 / 顶部 / Dashboard。 */
  title: string
  /** 一句话描述，出现在 Dashboard 卡片。 */
  description?: string
  /** 后端 API 前缀，例如 '/api/roundnfc'。 */
  apiPrefix: string

  /**
   * 全屏路由（不包 AdminLayout）。登录页放这里。
   * 路径会被自动拼为 /m/<name>/<your-path>，填相对值即可。
   */
  blankRoutes?: RouteRecordRaw[]

  /**
   * 需要鉴权 + 被 AdminLayout 包裹的路由。
   * 未登录访问会被 Shell 引到 /m/<name>/login。
   */
  adminRoutes?: RouteRecordRaw[]

  /** 出现在左侧侧边栏的入口。 */
  nav?: AdminNavItem[]
}

export interface ApiResult<T = unknown> {
  code: number
  message: string
  data: T
}

export interface ListResult<T> {
  items: T[]
  total: number
}
