import { ref, type Ref } from 'vue'

const KEY_PREFIX = 'roast.'
const KEY_SUFFIX = '.apiBase'

const refs = new Map<string, Ref<string>>()
const listeners = new Set<(name: string, value: string) => void>()

// 后台 server 在 admin 端口的 index.html 注入：
//   <script>window.__ROAST_RUNTIME={apiBase:"http://host:8080"}</script>
// 这样 SPA 一加载就知道 API 在哪，不用手点 BackendSwitcher。
declare global {
  interface Window {
    __ROAST_RUNTIME?: { apiBase?: string }
  }
}

export function getRuntimeApiBase(): string {
  if (typeof window === 'undefined') return ''
  const v = window.__ROAST_RUNTIME?.apiBase
  if (typeof v !== 'string') return ''
  let s = v.trim()
  while (s.endsWith('/')) s = s.slice(0, -1)
  return s
}

function storageKey(moduleName: string): string {
  return `${KEY_PREFIX}${moduleName}${KEY_SUFFIX}`
}

function readRaw(moduleName: string): string {
  if (typeof localStorage === 'undefined') return ''
  try {
    return localStorage.getItem(storageKey(moduleName)) || ''
  } catch {
    return ''
  }
}

function writeRaw(moduleName: string, value: string) {
  if (typeof localStorage === 'undefined') return
  try {
    if (value) localStorage.setItem(storageKey(moduleName), value)
    else localStorage.removeItem(storageKey(moduleName))
  } catch {
    /* ignore */
  }
}

function normalize(input: string): string {
  let v = (input || '').trim()
  if (!v) return ''
  // strip trailing slashes
  while (v.endsWith('/')) v = v.slice(0, -1)
  return v
}

function isValidBase(v: string): boolean {
  if (!v) return true
  try {
    const u = new URL(v)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}

/**
 * 模块当前生效的 API base URL：
 *   1) localStorage 里用户显式设置的（BackendSwitcher / ?api=）
 *   2) 否则 server 注入的运行时默认（不同端口部署时通常是 http://host:8080）
 *   3) 否则空字符串（=> 同源相对路径）
 */
export function getApiBase(moduleName: string): string {
  return readRaw(moduleName) || getRuntimeApiBase()
}

/** 是否用了用户自己设置的 override（区别于运行时默认）。给 UI 显示用。 */
export function hasUserOverride(moduleName: string): boolean {
  return readRaw(moduleName) !== ''
}

export function setApiBase(moduleName: string, value: string): void {
  const v = normalize(value)
  if (!isValidBase(v)) {
    throw new Error('请输入合法的 http(s):// URL')
  }
  writeRaw(moduleName, v)
  const r = refs.get(moduleName)
  if (r) r.value = v
  listeners.forEach((fn) => {
    try {
      fn(moduleName, v)
    } catch {
      /* ignore */
    }
  })
}

/** Reactive ref that tracks the apiBase for the given module. */
export function useApiBaseRef(moduleName: string): Ref<string> {
  let r = refs.get(moduleName)
  if (!r) {
    r = ref(readRaw(moduleName))
    refs.set(moduleName, r)
  }
  return r
}

/** Subscribe to apiBase changes for any module. Returns an unsubscribe fn. */
export function onApiBaseChange(fn: (name: string, value: string) => void): () => void {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

/**
 * Read ?api=<url> from current location and persist it to whichever module
 * is implied by the path (/m/<name>/...). Then strip the param from the URL.
 * Safe to call multiple times; only acts when the param is present.
 */
export function consumeApiQueryParam(): void {
  if (typeof window === 'undefined') return
  try {
    const url = new URL(window.location.href)
    const apiParam = url.searchParams.get('api')
    if (apiParam === null) return
    // Determine module from path: <base>/m/<name>/...
    const parts = url.pathname.split('/').filter(Boolean)
    const mIdx = parts.indexOf('m')
    const moduleName = mIdx >= 0 && parts.length > mIdx + 1 ? parts[mIdx + 1] : ''
    if (moduleName) {
      try {
        setApiBase(moduleName, apiParam)
      } catch {
        /* ignore invalid */
      }
    }
    url.searchParams.delete('api')
    const newSearch = url.searchParams.toString()
    const newUrl =
      url.pathname + (newSearch ? `?${newSearch}` : '') + url.hash
    window.history.replaceState({}, '', newUrl)
  } catch {
    /* ignore */
  }
}
