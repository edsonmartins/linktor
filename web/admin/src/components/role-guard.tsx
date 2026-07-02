'use client'

import { usePathname } from 'next/navigation'
import { useTranslations } from 'next-intl'
import { ShieldAlert } from 'lucide-react'
import { useUser } from '@/stores/auth-store'
import { canAccessRoute } from '@/lib/rbac'

/**
 * RoleGuard - blocks non-admin roles from admin-only dashboard routes reached
 * by typing the URL directly (the sidebar already hides these entries). The
 * API enforces the same rule with 403; this is the matching UX layer.
 */
export function RoleGuard({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const user = useUser()
  const t = useTranslations('common')

  if (!canAccessRoute(pathname, user?.role)) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 p-8 text-center">
        <ShieldAlert className="h-12 w-12 text-muted-foreground" />
        <h2 className="text-lg font-semibold">{t('accessDeniedTitle')}</h2>
        <p className="max-w-md text-sm text-muted-foreground">
          {t('accessDeniedDescription')}
        </p>
      </div>
    )
  }

  return <>{children}</>
}
