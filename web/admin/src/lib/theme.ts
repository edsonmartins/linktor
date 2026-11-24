/**
 * Theme preference — shared by the server (root layout) and the client
 * (ThemeProvider).
 *
 * The preference lives in a cookie, like the locale, so the server can render
 * <html> with the right class on the first byte: no flash of the wrong theme
 * and no inline script. "system" cannot be resolved on the server, so the
 * client mirrors the OS preference into a second cookie that the server reads
 * on the next request.
 */
export const themes = ['light', 'dark', 'system'] as const
export type Theme = (typeof themes)[number]
export type ResolvedTheme = 'light' | 'dark'

export const THEME_COOKIE = 'theme'
export const SYSTEM_THEME_COOKIE = 'theme_system'

/** The app shipped dark-only; keep that for anyone who never picked a theme. */
export const defaultTheme: Theme = 'dark'
const fallbackResolved: ResolvedTheme = 'dark'

export function parseTheme(value: string | undefined): Theme {
  return themes.includes(value as Theme) ? (value as Theme) : defaultTheme
}

export function parseResolvedTheme(value: string | undefined): ResolvedTheme {
  return value === 'light' || value === 'dark' ? value : fallbackResolved
}

export function resolveTheme(theme: Theme, system: ResolvedTheme): ResolvedTheme {
  return theme === 'system' ? system : theme
}
