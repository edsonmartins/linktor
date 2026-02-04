import type { Metadata } from 'next'
import { Providers } from '@/components/providers'
import './globals.css'

export const metadata: Metadata = {
  title: 'Linktor Admin',
  description: 'Linktor - Multichannel Messaging Platform Admin Panel',
  icons: {
    icon: '/favicon.ico',
  },
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="pt-BR" className="dark">
      <body className="min-h-screen bg-background font-mono antialiased">
        <Providers>{children}</Providers>
      </body>
    </html>
  )
}
