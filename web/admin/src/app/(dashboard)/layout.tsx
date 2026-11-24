'use client'

import { AuthGuard } from '@/components/auth-guard'
import { RoleGuard } from '@/components/role-guard'
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
          <main className="lk-main flex-1 overflow-auto">
            <RoleGuard>{children}</RoleGuard>
          </main>
        </div>
      </WebSocketProvider>
    </AuthGuard>
  )
}
