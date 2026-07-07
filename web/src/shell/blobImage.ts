import { onBeforeUnmount, ref, watch, type Ref } from 'vue'

type BlobFetcher = (url: string, signal: AbortSignal) => Promise<Blob>

interface BlobCacheEntry {
  objectUrl: string | null
  promise: Promise<string> | null
  refs: number
}

interface BlobImageOptions {
  cacheKey?: string | null
  fetcher?: BlobFetcher
  signal?: AbortSignal
}

const blobCache = new Map<string, BlobCacheEntry>()

export async function acquireBlobImage(sourceUrl: string, options: BlobImageOptions = {}) {
  const key = normalizeKey(options.cacheKey || sourceUrl)
  let entry = blobCache.get(key)
  if (!entry) {
    entry = { objectUrl: null, promise: null, refs: 0 }
    blobCache.set(key, entry)
  }
  entry.refs += 1

  if (entry.objectUrl) return { key, src: entry.objectUrl }

  if (!entry.promise) {
    const fetcher = options.fetcher ?? defaultFetch
    const signal = options.signal ?? new AbortController().signal
    entry.promise = fetcher(sourceUrl, signal)
      .then((blob) => {
        const current = blobCache.get(key)
        if (!current) {
          return URL.createObjectURL(blob)
        }
        current.objectUrl = URL.createObjectURL(blob)
        current.promise = null
        return current.objectUrl
      })
      .catch((err) => {
        const current = blobCache.get(key)
        if (current) {
          current.promise = null
          current.refs = Math.max(0, current.refs - 1)
          if (current.refs === 0 && !current.objectUrl) blobCache.delete(key)
        }
        throw err
      })
  }

  const src = await entry.promise
  return { key, src }
}

export function releaseBlobImage(cacheKey: string | null | undefined) {
  if (!cacheKey) return
  const key = normalizeKey(cacheKey)
  const entry = blobCache.get(key)
  if (!entry) return
  entry.refs = Math.max(0, entry.refs - 1)
  if (entry.refs > 0) return
  if (entry.objectUrl) URL.revokeObjectURL(entry.objectUrl)
  blobCache.delete(key)
}

export function clearBlobImageCache() {
  for (const entry of blobCache.values()) {
    if (entry.objectUrl) URL.revokeObjectURL(entry.objectUrl)
  }
  blobCache.clear()
}

export function blobImageCacheKey(...parts: Array<string | number | null | undefined>) {
  return parts
    .filter((p) => p !== null && p !== undefined && String(p).trim() !== '')
    .map((p) => String(p).trim())
    .join(':')
}

/**
 * 后端的「一次性 URL」加载器：fetch 后转 blob 贴到 <img>。
 * 默认按 source URL 缓存；如果后端每次返回新的 one-shot URL，传稳定的 cacheKey。
 *
 * 如果请求需要携带 JWT，传 fetcher：最简单可以 (url) => http(url)。
 */
export function useBlobImage(
  source: () => string | undefined | null,
  options: BlobImageOptions | BlobFetcher = {},
) {
  const src = ref<string | null>(null)
  const loading = ref(false)
  const error = ref<unknown>(null)

  const resolvedOptions: BlobImageOptions =
    typeof options === 'function' ? { fetcher: options } : options
  let currentKey: string | null = null
  let abort: AbortController | null = null

  function release() {
    releaseBlobImage(currentKey)
    currentKey = null
  }

  async function load(url: string) {
    abort?.abort()
    abort = new AbortController()
    loading.value = true
    error.value = null
    try {
      const out = await acquireBlobImage(url, {
        ...resolvedOptions,
        signal: abort.signal,
      })
      release()
      currentKey = out.key
      src.value = out.src
    } catch (err) {
      if ((err as { name?: string }).name === 'AbortError') return
      error.value = err
    } finally {
      loading.value = false
    }
  }

  watch(
    source,
    (v) => {
      release()
      src.value = null
      if (!v) return
      void load(v)
    },
    { immediate: true },
  )

  onBeforeUnmount(() => {
    abort?.abort()
    release()
  })

  return { src: src as Ref<string | null>, loading, error }
}

function normalizeKey(v: string) {
  return v.trim()
}

async function defaultFetch(url: string, signal: AbortSignal): Promise<Blob> {
  const resp = await fetch(url, { signal, credentials: 'omit' })
  if (!resp.ok) throw new Error(`fetch ${resp.status}`)
  return resp.blob()
}
