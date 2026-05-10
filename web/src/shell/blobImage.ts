import { onBeforeUnmount, ref, watch, type Ref } from 'vue'

/**
 * 后端的「一次性 URL」加载器：fetch 后转 blob 贴到 <img>。
 * 源 URL 变化或组件卸载时会 revokeObjectURL。
 *
 * 如果请求需要携带 JWT，传 fetcher：最简单可以 (url) => http(url)。
 */
export function useBlobImage(
  source: () => string | undefined | null,
  fetcher?: (url: string, signal: AbortSignal) => Promise<Blob>,
) {
  const src = ref<string | null>(null)
  const loading = ref(false)
  const error = ref<unknown>(null)

  let currentObjectUrl: string | null = null
  let abort: AbortController | null = null

  function release() {
    if (currentObjectUrl) {
      URL.revokeObjectURL(currentObjectUrl)
      currentObjectUrl = null
    }
  }

  async function load(url: string) {
    abort?.abort()
    abort = new AbortController()
    loading.value = true
    error.value = null
    try {
      const blob = fetcher
        ? await fetcher(url, abort.signal)
        : await defaultFetch(url, abort.signal)
      release()
      currentObjectUrl = URL.createObjectURL(blob)
      src.value = currentObjectUrl
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

async function defaultFetch(url: string, signal: AbortSignal): Promise<Blob> {
  const resp = await fetch(url, { signal, credentials: 'omit' })
  if (!resp.ok) throw new Error(`fetch ${resp.status}`)
  return resp.blob()
}
