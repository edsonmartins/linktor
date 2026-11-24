'use client'

import { useState } from 'react'
import { usePathname } from 'next/navigation'
import Link from 'next/link'
import Image from 'next/image'
import { useTranslations } from 'next-intl'
import { ChevronLeft, ChevronRight, Info, LogOut } from 'lucide-react'
import { cn, getInitials } from '@/lib/utils'
import { SimpleTooltip } from '@/components/ui/tooltip'
import { useAuthStore, useUser } from '@/stores/auth-store'
import { useUIStore, useSidebarCollapsed, useUnreadCount } from '@/stores/ui-store'
import { VersionsDialog } from './versions-dialog'
import { isAdmin } from '@/lib/rbac'
import { navGroups, settingsItem, type NavItem } from './nav-config'

/**
 * Sidebar: the brand's own panel (deep green in both themes), so navigation
 * does not blend into the content. The look lives in globals.css
 * (`.lk-sidebar*`, `.lk-nav-*`, `--sb-*` tokens); the map in nav-config.ts.
 */
export function Sidebar() {
  const [versoesAbertas, setVersoesAbertas] = useState(false)
  const t = useTranslations('nav')
  const tVersoes = useTranslations('versions')
  const pathname = usePathname()
  const user = useUser()
  const { logout } = useAuthStore()
  const collapsed = useSidebarCollapsed()
  const unreadCount = useUnreadCount()
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)

  // Hide admin-only entries from non-admin roles (the API enforces this too).
  const userIsAdmin = isAdmin(user?.role)
  const visibleGroups = navGroups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => !item.adminOnly || userIsAdmin),
    }))
    .filter((group) => group.items.length > 0)

  const renderItem = (item: NavItem) => {
    const isActive = pathname.startsWith(item.href)
    const Icon = item.icon
    const label = t(item.labelKey)
    const showBadge = item.badge && unreadCount > 0

    const link = (
      <Link
        href={item.href}
        className="lk-nav-item"
        data-active={isActive || undefined}
        aria-current={isActive ? 'page' : undefined}
        aria-label={collapsed ? label : undefined}
      >
        <Icon className="h-[18px] w-[18px] shrink-0" strokeWidth={1.8} />
        {!collapsed && <span className="lk-nav-label">{label}</span>}
        {showBadge &&
          (collapsed ? (
            <span className="absolute right-2 top-1.5 h-2 w-2 rounded-full bg-red-400" />
          ) : (
            <span className="lk-nav-badge">{unreadCount > 99 ? '99+' : unreadCount}</span>
          ))}
      </Link>
    )

    return collapsed ? (
      <SimpleTooltip key={item.href} content={label} side="right">
        {link}
      </SimpleTooltip>
    ) : (
      <div key={item.href}>{link}</div>
    )
  }

  const footerButton = (
    label: string,
    icon: React.ReactNode,
    onClick: () => void,
    className?: string
  ) => (
    <SimpleTooltip content={label} side={collapsed ? 'right' : 'top'}>
      <button
        type="button"
        onClick={onClick}
        aria-label={label}
        className={cn('lk-sidebar-action', className)}
      >
        {icon}
      </button>
    </SimpleTooltip>
  )

  return (
    <aside
      className={cn(
        'lk-sidebar relative flex h-screen shrink-0 flex-col transition-[width] duration-300',
        collapsed ? 'w-[68px]' : 'w-60'
      )}
      data-collapsed={collapsed || undefined}
    >
      {/* Brand */}
      <Link href="/dashboard" className="lk-sidebar-brand" aria-label="Linktor">
        {/* The mark has wide transparent margins, hence the oversized box. */}
        <Image
          src="/images/logo_linktor_single.png"
          alt=""
          width={1024}
          height={1024}
          className="-mx-5 h-[76px] w-[76px] max-w-none shrink-0"
          priority
        />
        {!collapsed && <span className="lk-sidebar-wordmark">Linktor</span>}
      </Link>

      {/* Navigation */}
      <nav className="lk-sidebar-scroll min-h-0 flex-1 overflow-y-auto overflow-x-hidden px-2.5 pb-2">
        {visibleGroups.map((group, index) => (
          <div key={group.key}>
            {collapsed ? (
              index > 0 && <div className="lk-sidebar-sep" aria-hidden />
            ) : (
              <div className="lk-sidebar-group">{t(`groups.${group.key}`)}</div>
            )}
            {group.items.map(renderItem)}
          </div>
        ))}
      </nav>

      <div className="border-t border-[var(--sb-divider)] px-2.5 pt-2">{renderItem(settingsItem)}</div>

      {/* Who is signed in, versions and logout */}
      <div className="lk-sidebar-user" data-collapsed={collapsed || undefined}>
        {!collapsed && (
          <>
            <span className="lk-sidebar-avatar" aria-hidden>
              {getInitials(user?.name || 'U')}
            </span>
            <span className="flex min-w-0 flex-1 flex-col">
              <span className="truncate text-xs font-semibold text-white">{user?.name}</span>
              <span className="truncate text-[11px] text-[var(--sb-text-dim)]">{user?.email}</span>
            </span>
          </>
        )}
        {/* Versões no ar — responde "subiu ou não" sem entrar na VPS */}
        {footerButton(tVersoes('title'), <Info className="h-[17px] w-[17px]" />, () =>
          setVersoesAbertas(true)
        )}
        {footerButton(
          t('logout'),
          <LogOut className="h-[17px] w-[17px]" />,
          logout,
          'hover:!bg-red-400/15 hover:!text-red-300'
        )}
      </div>

      <VersionsDialog open={versoesAbertas} onOpenChange={setVersoesAbertas} />

      {/* Collapse toggle */}
      <button
        type="button"
        onClick={toggleSidebar}
        className="absolute -right-3 top-[52px] z-20 flex h-6 w-6 items-center justify-center rounded-full border border-border bg-card text-muted-foreground shadow-sm hover:text-foreground"
      >
        {collapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}
      </button>
    </aside>
  )
}
