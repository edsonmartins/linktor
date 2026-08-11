import type { Metadata } from 'next'
import { NextIntlClientProvider } from 'next-intl'
import { getLocale, getMessages } from 'next-intl/server'
import { Providers } from '@/components/providers'
import { RUNTIME_CONFIG_GLOBAL, resolveRuntimeConfig } from '@/lib/runtime-config'
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
    <html lang={locale} className="dark">
      <body className="min-h-screen bg-background font-mono antialiased">
        {/* Publishes the deployment's API/WS URLs before any app module runs,
            so one published image can serve any host. Next's bundles load
            after this inline script. See lib/runtime-config.ts. */}
        <script
          dangerouslySetInnerHTML={{
            __html: `window.${RUNTIME_CONFIG_GLOBAL}=${JSON.stringify(
              runtimeConfig
            ).replace(/</g, '\\u003c')};`,
          }}
        />
        <NextIntlClientProvider locale={locale} messages={messages}>
          <Providers>{children}</Providers>
        </NextIntlClientProvider>
      </body>
    </html>
  )
}
