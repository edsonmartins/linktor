import type { Metadata } from 'next'
import { themeInitScript } from '@/lib/theme'
import './globals.css'

export const metadata: Metadata = {
  title: 'Linktor - Omnichannel Conversations Powered by AI',
  description: 'Connect WhatsApp, Telegram, Email and more. Build AI-powered flows. Engage customers everywhere.',
  keywords: ['omnichannel', 'chatbot', 'ai', 'whatsapp', 'telegram', 'customer engagement'],
  openGraph: {
    title: 'Linktor - Omnichannel Conversations Powered by AI',
    description: 'Connect WhatsApp, Telegram, Email and more. Build AI-powered flows. Engage customers everywhere.',
    type: 'website',
  },
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    // `dark` is the default; the script below corrects the class before first
    // paint from the stored preference, hence suppressHydrationWarning.
    <html lang="en" className="dark" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeInitScript }} />
      </head>
      <body className="min-h-screen bg-background antialiased">
        {children}
      </body>
    </html>
  )
}
