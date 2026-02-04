/**
 * Zustand Stores - Plugin Pattern
 *
 * Each store is a "plugin" that manages a specific domain of state:
 * - auth-store: Authentication and user state
 * - ui-store: UI preferences and transient state
 *
 * Stores are designed to be:
 * - Self-contained: Each manages its own domain
 * - Composable: Can be used together or separately
 * - Persistent: Key state persisted to localStorage
 */

export {
  useAuthStore,
  useUser,
  useIsAuthenticated,
  useAuthLoading,
  useAuthError,
  type User,
} from './auth-store'

export {
  useUIStore,
  useSidebarCollapsed,
  useSidebarMobileOpen,
  useActiveConversation,
  useCommandPaletteOpen,
  useUnreadCount,
} from './ui-store'
