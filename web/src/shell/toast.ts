// Minimal M3-styled snackbar replacement for Vant's showSuccessToast / showFailToast.
type Kind = 'success' | 'fail'

function ensureStyle() {
  if (document.getElementById('m3-toast-style')) return
  const s = document.createElement('style')
  s.id = 'm3-toast-style'
  s.textContent = `
.m3-toast-wrap{position:fixed;left:50%;bottom:32px;transform:translateX(-50%);z-index:9999;display:flex;flex-direction:column;gap:8px;pointer-events:none}
.m3-toast{pointer-events:auto;min-width:160px;max-width:80vw;padding:10px 18px;border-radius:9999px;font-size:14px;line-height:1.4;color:#fff;background:#323232;box-shadow:0 4px 12px rgba(0,0,0,0.18);opacity:0;transform:translateY(10px);transition:opacity .18s ease,transform .18s ease;text-align:center}
.m3-toast.show{opacity:1;transform:translateY(0)}
.m3-toast.success{background:#146c43}
.m3-toast.fail{background:#b3261e}
`
  document.head.appendChild(s)
}

function ensureWrap(): HTMLElement {
  let w = document.getElementById('m3-toast-wrap')
  if (!w) {
    w = document.createElement('div')
    w.id = 'm3-toast-wrap'
    w.className = 'm3-toast-wrap'
    document.body.appendChild(w)
  }
  return w
}

function show(msg: string, kind: Kind, duration = 2000) {
  ensureStyle()
  const wrap = ensureWrap()
  const el = document.createElement('div')
  el.className = `m3-toast ${kind}`
  el.textContent = msg
  wrap.appendChild(el)
  requestAnimationFrame(() => el.classList.add('show'))
  setTimeout(() => {
    el.classList.remove('show')
    setTimeout(() => el.remove(), 220)
  }, duration)
}

export function showSuccessToast(msg: string) {
  show(msg ?? '', 'success')
}

export function showFailToast(msg: string) {
  show(msg ?? '', 'fail')
}
