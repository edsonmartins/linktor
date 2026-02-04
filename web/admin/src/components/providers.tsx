'use client'

import { ReactNode } from 'react'
import { QueryProvider } from '@/lib/query'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Toaster } from '@/components/ui/toaster'

/**
 * Providers - Plugin Pattern
 * Composes all global providers in a single component
 * Each provider is a "plugin" that adds functionality
 */
export function Providers({ children }: { children: ReactNode }) {
  return (
    <QueryProvider>
      <TooltipProvider delayDuration={300}>
        {children}
        <Toaster />
      </TooltipProvider>
    </QueryProvider>
  )
}
