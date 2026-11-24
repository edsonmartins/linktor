import type { Metadata } from 'next'
import { NextIntlClientProvider } from 'next-intl'
import { getLocale, getMessages } from 'next-intl/server'
import { cookies } from 'next/headers'
import { Providers } from '@/components/providers'
import { RUNTIME_CONFIG_ATTRIBUTE, resolveRuntimeConfig } from '@/lib/runtime-config'
import {
  SYSTEM_THEME_COOKIE,
  THEME_COOKIE,
  parseResolvedTheme,
  parseTheme,
  resolveTheme,
} from '@/lib/theme'
import './globals.css'

export const metadata: Metadata = {
  title: 'Linktor Admin',
  description: 'Linktor - Multichannel Messaging Platform Admin Panel',
  // Favicon/apple-icon are provided by the App Router file convention
  // (src/app/icon.png + src/app/apple-icon.png), so no manual icons entry.
}

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const locale = await getLocale()
  const messages = await getMessages()
  const runtimeConfig = resolveRuntimeConfig()

  // Theme comes from cookies so the class is right on the first byte — see
  // lib/theme.ts. The provider keeps it in step on the client afterwards.
  const cookieStore = await cookies()
  const theme = parseTheme(cookieStore.get(THEME_COOKIE)?.value)
  const systemTheme = parseResolvedTheme(cookieStore.get(SYSTEM_THEME_COOKIE)?.value)
  const isDark = resolveTheme(theme, systemTheme) === 'dark'

  return (
    // The deployment's URLs ride on the document element so one published image
    // can serve any host. An attribute — not an inline script — because Next
    // emits its bundles as async scripts in the head, which may run before the
    // parser reaches the body. See lib/runtime-config.ts.
    <html
      lang={locale}
      className={isDark ? 'dark' : undefined}
      {...{ [RUNTIME_CONFIG_ATTRIBUTE]: JSON.stringify(runtimeConfig) }}
    >
      <body className="min-h-screen bg-background font-mono antialiased">
        <NextIntlClientProvider locale={locale} messages={messages}>
          <Providers initialTheme={theme} initialSystemTheme={systemTheme}>
            {children}
          </Providers>
        </NextIntlClientProvider>
      </body>
    </html>
  )
}
