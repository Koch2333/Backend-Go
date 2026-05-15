import { defineStore } from 'pinia'

export interface AuthState {
  token: string | null
  username: string | null
  expiresAt: string | null
}

function storageKey(name: string) {
  return `roast.admin.${name}.auth`
}

function load(name: string): AuthState {
  if (typeof localStorage === 'undefined') {
    return { token: null, username: null, expiresAt: null }
  }
  try {
    const raw = localStorage.getItem(storageKey(name))
    if (!raw) return { token: null, username: null, expiresAt: null }
    const obj = JSON.parse(raw) as AuthState
    if (obj.expiresAt && new Date(obj.expiresAt).getTime() < Date.now()) {
      return { token: null, username: null, expiresAt: null }
    }
    return obj
  } catch {
    return { token: null, username: null, expiresAt: null }
  }
}

/**
 * 为某个模块定义 auth store。不同模块的 token 分别存仓，互不干扰。
 */
export function defineModuleAuthStore(moduleName: string) {
  return defineStore(`auth.${moduleName}`, {
    state: (): AuthState => load(moduleName),
    getters: {
      isLoggedIn: (s) => !!s.token,
    },
    actions: {
      set(token: string, username: string, expiresAt: string) {
        this.token = token
        this.username = username
        this.expiresAt = expiresAt
        try {
          localStorage.setItem(
            storageKey(moduleName),
            JSON.stringify({ token, username, expiresAt }),
          )
        } catch {
          /* storage 不可用时静默 */
        }
      },
      clear() {
        this.token = null
        this.username = null
        this.expiresAt = null
        try {
          localStorage.removeItem(storageKey(moduleName))
        } catch {
          /* ignore */
        }
      },
    },
  })
}
