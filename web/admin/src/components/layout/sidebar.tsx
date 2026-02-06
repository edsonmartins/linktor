'use client'

import { usePathname } from 'next/navigation'
import Link from 'next/link'
import Image from 'next/image'
import {
  LayoutDashboard,
  MessageSquare,
  Users,
  UsersRound,
  Radio,
  BookOpen,
  GitBranch,
  BarChart3,
  Settings,
  LogOut,
  ChevronLeft,
  ChevronRight,
  Bell,
  Bot,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Avatar } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { SimpleTooltip } from '@/components/ui/tooltip'
import { Separator } from '@/components/ui/separator'
import { useAuthStore, useUser } from '@/stores/auth-store'
import { useUIStore, useSidebarCollapsed, useUnreadCount } from '@/stores/ui-store'

/**
 * Navigation items - Plugin Pattern
 * Each item is a "plugin" that can be enabled/disabled
 */
const navItems = [
  {
    label: 'Dashboard',
    href: '/dashboard',
    icon: LayoutDashboard,
  },
  {
    label: 'Conversations',
    href: '/conversations',
    icon: MessageSquare,
    badge: true, // Shows unread count
  },
  {
    label: 'Contacts',
    href: '/contacts',
    icon: Users,
  },
  {
    label: 'Channels',
    href: '/channels',
    icon: Radio,
  },
  {
    label: 'Bots',
    href: '/bots',
    icon: Bot,
  },
  {
    label: 'Knowledge Base',
    href: '/knowledge-base',
    icon: BookOpen,
  },
  {
    label: 'Flows',
    href: '/flows',
    icon: GitBranch,
  },
  {
    label: 'Analytics',
    href: '/analytics',
    icon: BarChart3,
  },
  {
    label: 'Team',
    href: '/users',
    icon: UsersRound,
  },
]

const bottomNavItems = [
  {
    label: 'Settings',
    href: '/settings',
    icon: Settings,
  },
]

/**
 * Sidebar Component - Plugin Pattern
 * Collapsible sidebar with navigation
 */
export function Sidebar() {
  const pathname = usePathname()
  const user = useUser()
  const { logout } = useAuthStore()
  const collapsed = useSidebarCollapsed()
  const unreadCount = useUnreadCount()
  const toggleSidebar = useUIStore((s) => s.toggleSidebar)

  return (
    <aside
      className={cn(
        'relative flex h-screen flex-col border-r border-border bg-card transition-all duration-300',
        collapsed ? 'w-16' : 'w-64'
      )}
    >
      {/* Header */}
      <div className="flex h-16 items-center justify-between px-4">
        {!collapsed && (
          <Link href="/dashboard" className="flex items-center">
            <Image
              src="/images/logo_fundo_escuro.png"
              alt="Linktor"
              width={120}
              height={32}
              className="h-8 w-auto"
              priority
            />
          </Link>
        )}
        {collapsed && (
          <Link href="/dashboard" className="mx-auto">
            <Image
              src="/images/logo_single.png"
              alt="Linktor"
              width={32}
              height={32}
              className="h-8 w-8"
              priority
            />
          </Link>
        )}
      </div>

      <Separator />

      {/* Navigation */}
      <nav className="flex-1 space-y-1 p-2">
        {navItems.map((item) => {
          const isActive = pathname.startsWith(item.href)
          const Icon = item.icon

          const linkContent = (
            <Link
              href={item.href}
              className={cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
                collapsed && 'justify-center px-2'
              )}
            >
              <Icon className="h-5 w-5 shrink-0" />
              {!collapsed && (
                <>
                  <span>{item.label}</span>
                  {item.badge && unreadCount > 0 && (
                    <Badge variant="error" className="ml-auto">
                      {unreadCount > 99 ? '99+' : unreadCount}
                    </Badge>
                  )}
                </>
              )}
              {collapsed && item.badge && unreadCount > 0 && (
                <span className="absolute right-1 top-1 h-2 w-2 rounded-full bg-destructive" />
              )}
            </Link>
          )

          if (collapsed) {
            return (
              <SimpleTooltip key={item.href} content={item.label} side="right">
                <div className="relative">{linkContent}</div>
              </SimpleTooltip>
            )
          }

          return <div key={item.href}>{linkContent}</div>
        })}
      </nav>

      {/* Bottom Navigation */}
      <div className="space-y-1 p-2">
        {bottomNavItems.map((item) => {
          const isActive = pathname.startsWith(item.href)
          const Icon = item.icon

          const linkContent = (
            <Link
              href={item.href}
              className={cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-primary/10 text-primary'
                  : 'text-muted-foreground hover:bg-secondary hover:text-foreground',
                collapsed && 'justify-center px-2'
              )}
            >
              <Icon className="h-5 w-5 shrink-0" />
              {!collapsed && <span>{item.label}</span>}
            </Link>
          )

          if (collapsed) {
            return (
              <SimpleTooltip key={item.href} content={item.label} side="right">
                {linkContent}
              </SimpleTooltip>
            )
          }

          return <div key={item.href}>{linkContent}</div>
        })}
      </div>

      <Separator />

      {/* User section */}
      <div className="p-2">
        <div
          className={cn(
            'flex items-center gap-3 rounded-md p-2',
            collapsed && 'justify-center'
          )}
        >
          <Avatar
            src={user?.avatar_url}
            fallback={user?.name || 'U'}
            size="sm"
            status="online"
          />
          {!collapsed && (
            <div className="flex-1 overflow-hidden">
              <p className="truncate text-sm font-medium">{user?.name}</p>
              <p className="truncate text-xs text-muted-foreground">
                {user?.email}
              </p>
            </div>
          )}
        </div>

        {/* Logout button */}
        {collapsed ? (
          <SimpleTooltip content="Logout" side="right">
            <Button
              variant="ghost"
              size="icon"
              className="mt-1 w-full"
              onClick={logout}
            >
              <LogOut className="h-5 w-5" />
            </Button>
          </SimpleTooltip>
        ) : (
          <Button
            variant="ghost"
            className="mt-1 w-full justify-start gap-3 text-muted-foreground"
            onClick={logout}
          >
            <LogOut className="h-5 w-5" />
            Logout
          </Button>
        )}
      </div>

      {/* Collapse toggle */}
      <button
        onClick={toggleSidebar}
        className="absolute -right-3 top-20 flex h-6 w-6 items-center justify-center rounded-full border border-border bg-card text-muted-foreground hover:bg-secondary hover:text-foreground"
      >
        {collapsed ? (
          <ChevronRight className="h-4 w-4" />
        ) : (
          <ChevronLeft className="h-4 w-4" />
        )}
      </button>
    </aside>
  )
}
