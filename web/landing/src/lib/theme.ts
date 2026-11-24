/**
 * Theme preference for the landing page.
 *
 * The site is a static export, so there is no server to read a cookie: the
 * preference lives in localStorage and `themeInitScript` runs in <head>,
 * before first paint, to put the right class on <html>. Anyone who never
 * picked a theme keeps the original dark look.
 */
export const themes = ['light', 'dark', 'system'] as const
export type Theme = (typeof themes)[number]

export const THEME_STORAGE_KEY = 'theme'
export const defaultTheme: Theme = 'dark'
const DARK_QUERY = '(prefers-color-scheme: dark)'

/** Kept dependency-free and tiny: it is inlined as a blocking script. */
export const themeInitScript = `(function(){try{var t=localStorage.getItem('${THEME_STORAGE_KEY}');var d=t==='light'?false:t==='system'?matchMedia('${DARK_QUERY}').matches:true;document.documentElement.classList.toggle('dark',d)}catch(e){}})()`

function isTheme(value: string | null): value is Theme {
  return themes.includes(value as Theme)
}

const listeners = new Set<() => void>()

function applyTheme(theme: Theme) {
  const dark =
    theme === 'system' ? window.matchMedia(DARK_QUERY).matches : theme === 'dark'
  document.documentElement.classList.toggle('dark', dark)
}

export function getTheme(): Theme {
  try {
    const stored = localStorage.getItem(THEME_STORAGE_KEY)
    return isTheme(stored) ? stored : defaultTheme
  } catch {
    return defaultTheme
  }
}

export function setTheme(theme: Theme) {
  try {
    localStorage.setItem(THEME_STORAGE_KEY, theme)
  } catch {
    // Storage blocked (private mode): the choice still applies to this visit.
  }
  applyTheme(theme)
  listeners.forEach((listener) => listener())
}

/** For useSyncExternalStore: also follows the OS and other tabs. */
export function subscribeToTheme(listener: () => void) {
  const sync = () => {
    applyTheme(getTheme())
    listener()
  }
  const query = window.matchMedia(DARK_QUERY)
  listeners.add(listener)
  query.addEventListener('change', sync)
  window.addEventListener('storage', sync)
  return () => {
    listeners.delete(listener)
    query.removeEventListener('change', sync)
    window.removeEventListener('storage', sync)
  }
}
