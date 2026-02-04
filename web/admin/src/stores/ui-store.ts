import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'

/**
 * UI Store - Zustand store for UI state
 * Manages sidebar, theme, and other UI preferences
 */

interface UIState {
  // Sidebar
  sidebarCollapsed: boolean
  sidebarMobileOpen: boolean

  // Active conversation (for highlighting in list)
  activeConversationId: string | null

  // Command palette
  commandPaletteOpen: boolean

  // Notifications
  unreadCount: number
}

interface UIActions {
  // Sidebar
  toggleSidebar: () => void
  setSidebarCollapsed: (collapsed: boolean) => void
  setSidebarMobileOpen: (open: boolean) => void

  // Active conversation
  setActiveConversation: (id: string | null) => void

  // Command palette
  toggleCommandPalette: () => void
  setCommandPaletteOpen: (open: boolean) => void

  // Notifications
  setUnreadCount: (count: number) => void
  incrementUnread: () => void
  clearUnread: () => void
}

type UIStore = UIState & UIActions

export const useUIStore = create<UIStore>()(
  persist(
    (set) => ({
      // Initial state
      sidebarCollapsed: false,
      sidebarMobileOpen: false,
      activeConversationId: null,
      commandPaletteOpen: false,
      unreadCount: 0,

      // Actions
      toggleSidebar: () =>
        set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),

      setSidebarCollapsed: (collapsed) =>
        set({ sidebarCollapsed: collapsed }),

      setSidebarMobileOpen: (open) =>
        set({ sidebarMobileOpen: open }),

      setActiveConversation: (id) =>
        set({ activeConversationId: id }),

      toggleCommandPalette: () =>
        set((state) => ({ commandPaletteOpen: !state.commandPaletteOpen })),

      setCommandPaletteOpen: (open) =>
        set({ commandPaletteOpen: open }),

      setUnreadCount: (count) =>
        set({ unreadCount: count }),

      incrementUnread: () =>
        set((state) => ({ unreadCount: state.unreadCount + 1 })),

      clearUnread: () =>
        set({ unreadCount: 0 }),
    }),
    {
      name: 'ui-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
      }),
    }
  )
)

/**
 * Selector hooks
 */
export const useSidebarCollapsed = () => useUIStore((state) => state.sidebarCollapsed)
export const useSidebarMobileOpen = () => useUIStore((state) => state.sidebarMobileOpen)
export const useActiveConversation = () => useUIStore((state) => state.activeConversationId)
export const useCommandPaletteOpen = () => useUIStore((state) => state.commandPaletteOpen)
export const useUnreadCount = () => useUIStore((state) => state.unreadCount)
