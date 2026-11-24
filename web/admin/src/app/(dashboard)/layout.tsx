'use client'

import { AuthGuard } from '@/components/auth-guard'
import { RoleGuard } from '@/components/role-guard'
import { Sidebar } from '@/components/layout/sidebar'
import { WebSocketProvider } from '@/hooks/use-websocket'
import { usePathname } from 'next/navigation'
import { findNavSection } from '@/components/layout/nav-config'

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  // The section's colour, inherited by metric cards and empty states.
  const accent = findNavSection(usePathname())?.accent ?? 'lime'

  return (
    <AuthGuard>
      <WebSocketProvider>
        <div className="flex h-screen overflow-hidden">
          <Sidebar />
          <main className="lk-main flex-1 overflow-auto" data-accent={accent}>
            <RoleGuard>{children}</RoleGuard>
          </main>
        </div>
      </WebSocketProvider>
    </AuthGuard>
  )
}
