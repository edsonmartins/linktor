'use client'

import { ReactNode } from 'react'
import { QueryProvider } from '@/lib/query'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from '@/components/ui/toaster'
import { ThemeProvider } from '@/components/theme-provider'
import type { ResolvedTheme, Theme } from '@/lib/theme'

/**
 * Providers - Plugin Pattern
 * Composes all global providers in a single component
 * Each provider is a "plugin" that adds functionality
 */
interface ProvidersProps {
  children: ReactNode
  initialTheme: Theme
  initialSystemTheme: ResolvedTheme
}

export function Providers({ children, initialTheme, initialSystemTheme }: ProvidersProps) {
  return (
    <ThemeProvider initialTheme={initialTheme} initialSystemTheme={initialSystemTheme}>
      <QueryProvider>
        <TooltipProvider delayDuration={300}>
          {children}
          <Toaster />
        </TooltipProvider>
      </QueryProvider>
    </ThemeProvider>
  )
}
