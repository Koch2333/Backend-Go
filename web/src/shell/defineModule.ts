import type { AxiosInstance } from 'axios'
import { defineModuleAuthStore } from './auth'
import { createHttp, unwrap } from './http'
import { getApiBase, onApiBaseChange } from './backend'

/**
 * 为一个后台模块生成「一套运行时」：
 *  - useAuth：该模块专属的 Pinia auth store
 *  - http：含 JWT 注入、01 自动跳登录的 axios 实例
 *  - unwrap：ApiResult 解包
 */
export function defineModule(opts: { name: string; apiPrefix: string }) {
  const useAuth = defineModuleAuthStore(opts.name)
  let httpInstance: AxiosInstance | null = null

  // 当本模块的 apiBase 变了，下次 http() 重新构建
  onApiBaseChange((name) => {
    if (name === opts.name) httpInstance = null
  })

  const http = (): AxiosInstance => {
    if (!httpInstance) {
      const base = getApiBase(opts.name)
      httpInstance = createHttp({
        baseURL: (base || '') + opts.apiPrefix,
        getToken: () => useAuth().token,
        onUnauthorized: () => {
          useAuth().clear()
          if (typeof window !== 'undefined') {
            // BASE_URL 已带尾斜杠（默认 '/admin/'），与 SPA 路由 base 保持一致；
            // 否则整页跳转会丢掉前缀，落到后端 404。
            const base = import.meta.env.BASE_URL.replace(/\/$/, '')
            const loginPath = `${base}/m/${opts.name}/login`
            if (window.location.pathname !== loginPath) {
              window.location.assign(loginPath)
            }
          }
        },
      })
    }
    return httpInstance
  }

  return {
    name: opts.name,
    apiPrefix: opts.apiPrefix,
    useAuth,
    http,
    unwrap,
    /** 常用：POST /admin/login 后保存 token 并跳转。 */
    async signIn(token: string, username: string, expiresAt: string) {
      useAuth().set(token, username, expiresAt)
    },
  }
}

export type DefinedModule = ReturnType<typeof defineModule>
