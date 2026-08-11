import type { Metadata } from 'next'
import { NextIntlClientProvider } from 'next-intl'
import { getLocale, getMessages } from 'next-intl/server'
import { Providers } from '@/components/providers'
import { RUNTIME_CONFIG_ATTRIBUTE, resolveRuntimeConfig } from '@/lib/runtime-config'
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

  return (
    // The deployment's URLs ride on the document element so one published image
    // can serve any host. An attribute — not an inline script — because Next
    // emits its bundles as async scripts in the head, which may run before the
    // parser reaches the body. See lib/runtime-config.ts.
    <html
      lang={locale}
      className="dark"
      {...{ [RUNTIME_CONFIG_ATTRIBUTE]: JSON.stringify(runtimeConfig) }}
    >
      <body className="min-h-screen bg-background font-mono antialiased">
        <NextIntlClientProvider locale={locale} messages={messages}>
          <Providers>{children}</Providers>
        </NextIntlClientProvider>
      </body>
    </html>
  )
}
