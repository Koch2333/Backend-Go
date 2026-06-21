import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { MODULES, prefixRoutes } from './modules'
import AdminLayout from './AdminLayout.vue'
import Dashboard from './Dashboard.vue'
import NotFound from './NotFound.vue'

// 不包裹 layout 的路由（登录页等）
const blankRoutes: RouteRecordRaw[] = MODULES.flatMap((m) =>
  prefixRoutes(m.name, m.blankRoutes),
)

// 包在 AdminLayout 里、需要登录才能访问的路由
const adminRoutes: RouteRecordRaw[] = MODULES.flatMap((m) => {
  return prefixRoutes(m.name, m.adminRoutes).map((r) => ({
    ...r,
    meta: { ...(r.meta || {}), requiresAuth: true, moduleName: m.name },
  }))
})

const routes: RouteRecordRaw[] = [
  ...blankRoutes,
  {
    path: '/',
    component: AdminLayout,
    children: [
      { path: '', component: Dashboard },
      ...adminRoutes,
    ],
  },
  { path: '/:pathMatch(.*)*', component: NotFound },
]

export const router = createRouter({
  // base 跟着 vite 的 `base` 走（默认 '/admin/'，单独部署到根域名时由 VITE_BASE=/ 覆盖）。
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior: () => ({ top: 0 }),
})

router.beforeEach((to) => {
  const requiresAuth = to.matched.some((r) => r.meta?.requiresAuth)
  if (!requiresAuth) return true
  const modName = to.matched.find((r) => r.meta?.moduleName)?.meta?.moduleName as
    | string
    | undefined
  if (!modName) return true
  const mod = MODULES.find((m) => m.name === modName)
  if (!mod) return true
  // 动态拿该模块的 auth store。所有模块的 store 都存在于 module.ts 加载阶段。
  // 这里由于模块本身不会在这被导入，还是走 localStorage 最简单。
  try {
    const raw = localStorage.getItem(`roast.admin.${modName}.auth`)
    if (raw) {
      const obj = JSON.parse(raw) as { token?: string; expiresAt?: string }
      if (obj.token && (!obj.expiresAt || new Date(obj.expiresAt).getTime() > Date.now())) {
        return true
      }
    }
  } catch {
    /* fallthrough */
  }
  return { path: `/m/${modName}/login`, query: { from: to.fullPath } }
})
