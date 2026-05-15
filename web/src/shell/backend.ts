import { ref, type Ref } from 'vue'

const KEY_PREFIX = 'roast.'
const KEY_SUFFIX = '.apiBase'

const refs = new Map<string, Ref<string>>()
const listeners = new Set<(name: string, value: string) => void>()

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

export function getApiBase(moduleName: string): string {
  return readRaw(moduleName)
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
