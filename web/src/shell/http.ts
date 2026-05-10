import axios, { type AxiosInstance } from 'axios'
import type { ApiResult } from './types'

export interface CreateHttpOptions {
  baseURL: string
  /** 请求发起时调用，返回当前模块的 JWT（或 null）。 */
  getToken?: () => string | null | undefined
  /** 401 时回调，常用于清 token + 跳登录页。 */
  onUnauthorized?: () => void
}

export function createHttp(opts: CreateHttpOptions): AxiosInstance {
  const http = axios.create({ baseURL: opts.baseURL, timeout: 15_000 })
  http.interceptors.request.use((config) => {
    const t = opts.getToken?.()
    if (t) {
      config.headers = config.headers ?? {}
      ;(config.headers as Record<string, string>).Authorization = `Bearer ${t}`
    }
    return config
  })
  http.interceptors.response.use(
    (resp) => resp,
    (error) => {
      if (error?.response?.status === 401) opts.onUnauthorized?.()
      return Promise.reject(error)
    },
  )
  return http
}

/** 后端统一返回 { code, message, data }。非 0 的 code 看作业务错误。 */
export function unwrap<T>(res: { data: ApiResult<T> }): T {
  if (res.data?.code !== 0) {
    throw new Error(res.data?.message || 'request failed')
  }
  return res.data.data
}

export function extractMessage(err: unknown, fallback = '请求失败'): string {
  const e = err as {
    response?: { data?: { message?: string } }
    message?: string
  }
  return e?.response?.data?.message || e?.message || fallback
}
