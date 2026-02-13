'use client'

import { useLocale } from 'next-intl'
import { useRouter } from 'next/navigation'
import { useTransition } from 'react'
import { Globe } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { locales, type Locale } from '@/i18n/config'
import { cn } from '@/lib/utils'

const localeNames: Record<Locale, string> = {
  'pt-BR': 'Português',
  'es': 'Español',
  'en': 'English',
}

const localeFlags: Record<Locale, string> = {
  'pt-BR': '🇧🇷',
  'es': '🇪🇸',
  'en': '🇺🇸',
}

interface LocaleSwitcherProps {
  collapsed?: boolean
}

export function LocaleSwitcher({ collapsed = false }: LocaleSwitcherProps) {
  const locale = useLocale() as Locale
  const router = useRouter()
  const [isPending, startTransition] = useTransition()

  const handleLocaleChange = (newLocale: Locale) => {
    // Set cookie to persist locale preference
    document.cookie = `locale=${newLocale};path=/;max-age=31536000`
    startTransition(() => {
      router.refresh()
    })
  }

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
          disabled={isPending}
        >
          <Globe className="h-4 w-4" />
          {!collapsed && (
            <span className="text-xs">
              {localeFlags[locale]} {locale.toUpperCase()}
            </span>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-40">
        {locales.map((loc) => (
          <DropdownMenuItem
            key={loc}
            onClick={() => handleLocaleChange(loc)}
            className={cn(
              'gap-2 cursor-pointer',
              locale === loc && 'bg-primary/10 text-primary'
            )}
          >
            <span>{localeFlags[loc]}</span>
            <span>{localeNames[loc]}</span>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
