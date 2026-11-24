'use client'

import { usePathname } from 'next/navigation'
import { Bell, Search, Command } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { SimpleTooltip } from '@/components/ui/tooltip'
import { LocaleSwitcher } from '@/components/locale-switcher'
import { ThemeSwitcher } from '@/components/theme-switcher'
import { useUIStore, useUnreadCount } from '@/stores/ui-store'
import { findNavSection } from './nav-config'

interface HeaderProps {
  title?: string
  children?: React.ReactNode
}

/**
 * Header Component
 * Top bar: the page's identity on the left (the section's icon in a gradient
 * square, then the title), search, language, theme and notifications on the
 * right.
 */
export function Header({ title, children }: HeaderProps) {
  const pathname = usePathname()
  const unreadCount = useUnreadCount()
  const setCommandPaletteOpen = useUIStore((s) => s.setCommandPaletteOpen)
  const section = findNavSection(pathname)
  const SectionIcon = section?.item.icon

  return (
    <header className="lk-header flex h-16 shrink-0 items-center justify-between gap-4 px-6">
      {/* Title */}
      <div className="flex min-w-0 items-center gap-3">
        {SectionIcon && (
          <span className="lk-page-icon" data-accent={section.accent} aria-hidden>
            <SectionIcon className="h-5 w-5" strokeWidth={1.8} />
          </span>
        )}
        {title && (
          <h1 className="truncate text-xl font-bold tracking-tight text-foreground">{title}</h1>
        )}
        {children}
      </div>

      {/* Actions */}
      <div className="flex shrink-0 items-center gap-1">
        {/* Search / Command Palette */}
        <Button
          variant="outline"
          className="mr-1 hidden w-56 justify-start bg-background/60 text-muted-foreground xl:flex"
          onClick={() => setCommandPaletteOpen(true)}
        >
          <Search className="mr-2 h-4 w-4" />
          <span>Search...</span>
          <kbd className="ml-auto flex h-5 items-center gap-1 rounded border border-border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground">
            <Command className="h-3 w-3" />K
          </kbd>
        </Button>

        {/* Compact search */}
        <SimpleTooltip content="Search (Cmd+K)">
          <Button
            variant="ghost"
            size="icon"
            className="xl:hidden"
            onClick={() => setCommandPaletteOpen(true)}
          >
            <Search className="h-5 w-5" />
          </Button>
        </SimpleTooltip>

        <LocaleSwitcher collapsed />
        <ThemeSwitcher collapsed />

        {/* Notifications */}
        <SimpleTooltip content="Notifications">
          <Button variant="ghost" size="icon" className="relative">
            <Bell className="h-5 w-5" />
            {unreadCount > 0 && (
              <span className="absolute right-1 top-1 flex h-4 w-4 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-destructive-foreground">
                {unreadCount > 9 ? '9+' : unreadCount}
              </span>
            )}
          </Button>
        </SimpleTooltip>
      </div>
    </header>
  )
}
