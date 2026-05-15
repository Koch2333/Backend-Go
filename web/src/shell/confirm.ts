// Promise-based confirm dialog using <md-dialog>, mirroring Vant's showConfirmDialog API.
// Resolves on confirm; rejects on cancel (so callers can `try { await ... } catch { return }`).
import '@material/web/dialog/dialog.js'
import '@material/web/button/text-button.js'
import '@material/web/button/filled-button.js'

export interface ConfirmOptions {
  title?: string
  message?: string
  confirmText?: string
  cancelText?: string
}

export function showConfirmDialog(opts: ConfirmOptions): Promise<void> {
  return new Promise((resolve, reject) => {
    const dialog = document.createElement('md-dialog') as HTMLElement & {
      show: () => void
      close: (reason?: string) => void
    }

    const headline = document.createElement('div')
    headline.setAttribute('slot', 'headline')
    headline.textContent = opts.title ?? '提示'
    dialog.appendChild(headline)

    const content = document.createElement('div')
    content.setAttribute('slot', 'content')
    content.style.fontSize = '14px'
    content.style.lineHeight = '1.5'
    content.textContent = opts.message ?? ''
    dialog.appendChild(content)

    const actions = document.createElement('div')
    actions.setAttribute('slot', 'actions')

    const cancelBtn = document.createElement('md-text-button')
    cancelBtn.textContent = opts.cancelText ?? '取消'
    cancelBtn.addEventListener('click', () => dialog.close('cancel'))
    actions.appendChild(cancelBtn)

    const okBtn = document.createElement('md-filled-button')
    okBtn.textContent = opts.confirmText ?? '确定'
    okBtn.addEventListener('click', () => dialog.close('confirm'))
    actions.appendChild(okBtn)

    dialog.appendChild(actions)

    dialog.addEventListener('closed', () => {
      const reason = (dialog as any).returnValue
      dialog.remove()
      if (reason === 'confirm') resolve()
      else reject(new Error('cancel'))
    })

    document.body.appendChild(dialog)
    // Wait a tick for the custom element to upgrade before calling show().
    requestAnimationFrame(() => {
      try {
        dialog.show()
      } catch {
        setTimeout(() => dialog.show(), 0)
      }
    })
  })
}
