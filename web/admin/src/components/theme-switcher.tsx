'use client'

import { useTranslations } from 'next-intl'
import { Monitor, Moon, Sun, type LucideIcon } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { useTheme } from '@/components/theme-provider'
import { themes, type Theme } from '@/lib/theme'
import { cn } from '@/lib/utils'

export const themeIcons: Record<Theme, LucideIcon> = {
  light: Sun,
  dark: Moon,
  system: Monitor,
}

interface ThemeSwitcherProps {
  collapsed?: boolean
}

export function ThemeSwitcher({ collapsed = false }: ThemeSwitcherProps) {
  const t = useTranslations('settings')
  const { theme, setTheme } = useTheme()
  const Icon = themeIcons[theme]

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size={collapsed ? 'icon' : 'sm'}
          className={cn(
            'text-muted-foreground hover:text-foreground',
            collapsed ? 'w-10 h-10' : 'gap-2'
          )}
          aria-label={t('theme')}
        >
          <Icon className="h-4 w-4" />
          {!collapsed && <span className="text-xs">{t(theme)}</span>}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        {themes.map((option) => {
          const OptionIcon = themeIcons[option]
          return (
            <DropdownMenuItem
              key={option}
              onClick={() => setTheme(option)}
              className={cn(
                'gap-2 cursor-pointer',
                theme === option && 'bg-primary/10 text-primary'
              )}
            >
              <OptionIcon className="h-4 w-4" />
              <span>{t(option)}</span>
            </DropdownMenuItem>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
