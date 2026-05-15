import { ref } from 'vue'

const STORAGE_KEY = 'roast.admin.theme'
type Theme = 'light' | 'dark'

function readStored(): Theme | null {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    return v === 'light' || v === 'dark' ? v : null
  } catch {
    return null
  }
}

function systemPrefersDark(): boolean {
  return typeof window !== 'undefined'
    && window.matchMedia?.('(prefers-color-scheme: dark)').matches === true
}

function apply(theme: Theme) {
  document.documentElement.setAttribute('data-theme', theme)
}

const initial: Theme = readStored() ?? (systemPrefersDark() ? 'dark' : 'light')
apply(initial)

export const theme = ref<Theme>(initial)

export function toggleTheme() {
  const next: Theme = theme.value === 'dark' ? 'light' : 'dark'
  theme.value = next
  apply(next)
  try { localStorage.setItem(STORAGE_KEY, next) } catch { /* */ }
}
