'use client'

import { AuthGuard } from '@/components/auth-guard'
import { Sidebar } from '@/components/layout/sidebar'
import { WebSocketProvider } from '@/hooks/use-websocket'

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <AuthGuard>
      <WebSocketProvider>
        <div className="flex h-screen overflow-hidden">
          <Sidebar />
          <main className="flex-1 overflow-auto bg-background">
            {children}
          </main>
        </div>
      </WebSocketProvider>
    </AuthGuard>
  )
}
