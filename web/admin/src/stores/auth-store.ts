import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'
import { api, tokenStorage } from '@/lib/api'

/**
 * User type
 */
export interface User {
  id: string
  email: string
  name: string
  role: string
  avatar_url?: string
  tenant_id: string
}

/**
 * Auth Store - Zustand with persistence
 * Manages authentication state and user data
 */
interface AuthState {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  error: string | null
}

interface AuthActions {
  login: (email: string, password: string) => Promise<void>
  logout: () => void
  refreshToken: () => Promise<void>
  setUser: (user: User | null) => void
  clearError: () => void
}

type AuthStore = AuthState & AuthActions

interface LoginResponse {
  access_token: string
  refresh_token: string
  user: User
}

export const useAuthStore = create<AuthStore>()(
  persist(
    (set, get) => ({
      // State
      user: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,

      // Actions
      login: async (email: string, password: string) => {
        set({ isLoading: true, error: null })

        try {
          const response = await api.post<LoginResponse>('/auth/login', {
            email,
            password,
          })

          tokenStorage.setTokens(response.access_token, response.refresh_token)

          set({
            user: response.user,
            isAuthenticated: true,
            isLoading: false,
          })
        } catch (error) {
          const message =
            error instanceof Error ? error.message : 'Login failed'
          set({
            error: message,
            isLoading: false,
            isAuthenticated: false,
            user: null,
          })
          throw error
        }
      },

      logout: () => {
        tokenStorage.clearTokens()
        set({
          user: null,
          isAuthenticated: false,
          error: null,
        })
      },

      refreshToken: async () => {
        const refreshToken = tokenStorage.getRefreshToken()
        if (!refreshToken) {
          get().logout()
          return
        }

        try {
          const response = await api.post<LoginResponse>('/auth/refresh', {
            refresh_token: refreshToken,
          })

          tokenStorage.setTokens(response.access_token, response.refresh_token)
          set({ user: response.user, isAuthenticated: true })
        } catch {
          get().logout()
        }
      },

      setUser: (user) => {
        set({ user, isAuthenticated: !!user })
      },

      clearError: () => {
        set({ error: null })
      },
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
)

/**
 * Selector hooks for better performance
 */
export const useUser = () => useAuthStore((state) => state.user)
export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated)
export const useAuthLoading = () => useAuthStore((state) => state.isLoading)
export const useAuthError = () => useAuthStore((state) => state.error)
