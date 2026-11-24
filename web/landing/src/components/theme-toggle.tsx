'use client'

import { useSyncExternalStore } from 'react'
import { Monitor, Moon, Sun, type LucideIcon } from 'lucide-react'
import {
  defaultTheme,
  getTheme,
  setTheme,
  subscribeToTheme,
  themes,
  type Theme,
} from '@/lib/theme'

const themeMeta: Record<Theme, { icon: LucideIcon; label: string }> = {
  light: { icon: Sun, label: 'Light' },
  dark: { icon: Moon, label: 'Dark' },
  system: { icon: Monitor, label: 'System' },
}

/** One button that cycles light → dark → system. */
export function ThemeToggle({ showLabel = false }: { showLabel?: boolean }) {
  const theme = useSyncExternalStore(subscribeToTheme, getTheme, () => defaultTheme)
  const next = themes[(themes.indexOf(theme) + 1) % themes.length]
  const { icon: Icon, label } = themeMeta[theme]

  return (
    <button
      type="button"
      onClick={() => setTheme(next)}
      className="inline-flex items-center gap-2 rounded-lg p-2 text-sm font-medium text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
      aria-label={`Theme: ${label}. Switch to ${themeMeta[next].label}`}
      title={`Theme: ${label}`}
    >
      <Icon className="h-5 w-5" />
      {showLabel && <span>{label} theme</span>}
    </button>
  )
}
