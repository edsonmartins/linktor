'use client'

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from 'react'
import {
  SYSTEM_THEME_COOKIE,
  THEME_COOKIE,
  resolveTheme,
  type ResolvedTheme,
  type Theme,
} from '@/lib/theme'

interface ThemeContextValue {
  /** What the user picked: light, dark or system. */
  theme: Theme
  /** What is actually on screen. */
  resolvedTheme: ResolvedTheme
  setTheme: (theme: Theme) => void
}

const ThemeContext = createContext<ThemeContextValue | null>(null)

const DARK_QUERY = '(prefers-color-scheme: dark)'
const ONE_YEAR = 60 * 60 * 24 * 365

function writeCookie(name: string, value: string) {
  document.cookie = `${name}=${value};path=/;max-age=${ONE_YEAR};samesite=lax`
}

function subscribeToSystemTheme(onChange: () => void) {
  const query = window.matchMedia(DARK_QUERY)
  query.addEventListener('change', onChange)
  return () => query.removeEventListener('change', onChange)
}

function readSystemTheme(): ResolvedTheme {
  return window.matchMedia(DARK_QUERY).matches ? 'dark' : 'light'
}

interface ThemeProviderProps {
  children: ReactNode
  /** The preference the server rendered <html> with. */
  initialTheme: Theme
  /** The server's last known OS preference (from the mirror cookie). */
  initialSystemTheme: ResolvedTheme
}

export function ThemeProvider({
  children,
  initialTheme,
  initialSystemTheme,
}: ThemeProviderProps) {
  const [theme, setThemeState] = useState<Theme>(initialTheme)
  const systemTheme = useSyncExternalStore(
    subscribeToSystemTheme,
    readSystemTheme,
    () => initialSystemTheme
  )
  const resolvedTheme = resolveTheme(theme, systemTheme)

  // The server already rendered the class from the cookies; this keeps the
  // document in step when the user switches or the OS preference changes.
  useEffect(() => {
    document.documentElement.classList.toggle('dark', resolvedTheme === 'dark')
  }, [resolvedTheme])

  // Mirror the OS preference so the server can resolve "system" next time.
  useEffect(() => {
    writeCookie(SYSTEM_THEME_COOKIE, systemTheme)
  }, [systemTheme])

  const setTheme = useCallback((next: Theme) => {
    writeCookie(THEME_COOKIE, next)
    setThemeState(next)
  }, [])

  const value = useMemo(
    () => ({ theme, resolvedTheme, setTheme }),
    [theme, resolvedTheme, setTheme]
  )

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useTheme must be used within a ThemeProvider')
  }
  return context
}
