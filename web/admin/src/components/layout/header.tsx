'use client'

import { Bell, Search, Command } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { SimpleTooltip } from '@/components/ui/tooltip'
import { useUIStore, useUnreadCount } from '@/stores/ui-store'

interface HeaderProps {
  title?: string
  children?: React.ReactNode
}

/**
 * Header Component
 * Top bar with search and notifications
 */
export function Header({ title, children }: HeaderProps) {
  const unreadCount = useUnreadCount()
  const setCommandPaletteOpen = useUIStore((s) => s.setCommandPaletteOpen)

  return (
    <header className="flex h-16 items-center justify-between border-b border-border bg-card px-6">
      {/* Title */}
      <div className="flex items-center gap-4">
        {title && (
          <h1 className="text-xl font-semibold text-foreground">{title}</h1>
        )}
        {children}
      </div>

      {/* Actions */}
      <div className="flex items-center gap-2">
        {/* Search / Command Palette */}
        <Button
          variant="outline"
          className="hidden w-64 justify-start text-muted-foreground lg:flex"
          onClick={() => setCommandPaletteOpen(true)}
        >
          <Search className="mr-2 h-4 w-4" />
          <span>Search...</span>
          <kbd className="ml-auto flex h-5 items-center gap-1 rounded border border-border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground">
            <Command className="h-3 w-3" />K
          </kbd>
        </Button>

        {/* Mobile search */}
        <SimpleTooltip content="Search (Cmd+K)">
          <Button
            variant="ghost"
            size="icon"
            className="lg:hidden"
            onClick={() => setCommandPaletteOpen(true)}
          >
            <Search className="h-5 w-5" />
          </Button>
        </SimpleTooltip>

        {/* Notifications */}
        <SimpleTooltip content="Notifications">
          <Button variant="ghost" size="icon" className="relative">
            <Bell className="h-5 w-5" />
            {unreadCount > 0 && (
              <span className="absolute right-1 top-1 flex h-4 w-4 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-white">
                {unreadCount > 9 ? '9+' : unreadCount}
              </span>
            )}
          </Button>
        </SimpleTooltip>
      </div>
    </header>
  )
}
