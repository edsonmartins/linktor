import {
  Activity,
  BarChart3,
  BookOpen,
  Bot,
  FileText,
  FlaskConical,
  GitBranch,
  LayoutDashboard,
  Megaphone,
  MessageSquare,
  MessageSquareText,
  Radio,
  ScrollText,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Sparkles,
  Users,
  UsersRound,
  type LucideIcon,
} from 'lucide-react'

/**
 * Navigation map, shared by the sidebar (which lists it) and the page header
 * (which borrows the icon and colour of the section you are in).
 */

/** Section colours; the look of each lives in globals.css ([data-accent]). */
export type Accent = 'lime' | 'teal' | 'violet' | 'amber' | 'rose' | 'blue' | 'slate'

export interface NavItem {
  /** Key under the `nav` translations. */
  labelKey: string
  href: string
  icon: LucideIcon
  /** Shows the unread count. */
  badge?: boolean
  adminOnly?: boolean
}

export interface NavGroup {
  /** Key under `nav.groups`. */
  key: string
  accent: Accent
  items: NavItem[]
}

export const navGroups: NavGroup[] = [
  {
    key: 'service',
    accent: 'lime',
    items: [
      { labelKey: 'dashboard', href: '/dashboard', icon: LayoutDashboard },
      { labelKey: 'conversations', href: '/conversations', icon: MessageSquare, badge: true },
      { labelKey: 'contacts', href: '/contacts', icon: Users },
    ],
  },
  {
    key: 'automation',
    accent: 'teal',
    items: [
      { labelKey: 'channels', href: '/channels', icon: Radio },
      { labelKey: 'bots', href: '/bots', icon: Bot },
      { labelKey: 'knowledgeBase', href: '/knowledge-base', icon: BookOpen },
      { labelKey: 'templates', href: '/templates', icon: FileText },
      { labelKey: 'campaigns', href: '/campaigns', icon: Megaphone },
      { labelKey: 'cannedResponses', href: '/canned-responses', icon: MessageSquareText },
      { labelKey: 'vre', href: '/vre', icon: Sparkles },
      { labelKey: 'flows', href: '/flows', icon: GitBranch },
    ],
  },
  {
    key: 'insights',
    accent: 'violet',
    items: [
      { labelKey: 'analytics', href: '/analytics', icon: BarChart3 },
      { labelKey: 'observability', href: '/observability', icon: Activity, adminOnly: true },
    ],
  },
  {
    key: 'admin',
    accent: 'amber',
    items: [
      { labelKey: 'team', href: '/users', icon: UsersRound, adminOnly: true },
      { labelKey: 'roles', href: '/roles', icon: ShieldCheck, adminOnly: true },
      { labelKey: 'operations', href: '/operations', icon: SlidersHorizontal },
      { labelKey: 'audit', href: '/audit-logs', icon: ScrollText, adminOnly: true },
      { labelKey: 'sandboxAllowlist', href: '/sandbox-allowlist', icon: FlaskConical, adminOnly: true },
    ],
  },
]

export const settingsItem: NavItem = { labelKey: 'settings', href: '/settings', icon: Settings }

/** The section a pathname belongs to, for the header's icon. */
export function findNavSection(pathname: string): { item: NavItem; accent: Accent } | null {
  for (const group of navGroups) {
    const item = group.items.find((candidate) => pathname.startsWith(candidate.href))
    if (item) return { item, accent: group.accent }
  }
  if (pathname.startsWith(settingsItem.href)) return { item: settingsItem, accent: 'slate' }
  return null
}
