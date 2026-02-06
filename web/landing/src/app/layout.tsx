import type { Metadata } from 'next'
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
    <html lang="en" className="dark">
      <body className="min-h-screen bg-background antialiased">
        {children}
      </body>
    </html>
  )
}
