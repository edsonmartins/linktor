'use client'

import { useEffect, useState } from 'react'
import { useRouter, usePathname } from 'next/navigation'
import { useAuthStore } from '@/stores/auth-store'
import { FullPageSpinner } from '@/components/ui/spinner'

const PUBLIC_PATHS = ['/login', '/forgot-password', '/reset-password']

/**
 * AuthGuard - Protected Route Component
 * Redirects unauthenticated users to login
 */
export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter()
  const pathname = usePathname()
  const { isAuthenticated, refreshToken } = useAuthStore()
  const [isChecking, setIsChecking] = useState(true)

  const isPublicPath = PUBLIC_PATHS.some((path) => pathname.startsWith(path))

  useEffect(() => {
    const checkAuth = async () => {
      // Try to refresh token on mount
      if (!isAuthenticated) {
        try {
          await refreshToken()
        } catch {
          // Token refresh failed, will redirect below
        }
      }
      setIsChecking(false)
    }

    checkAuth()
  }, [isAuthenticated, refreshToken])

  useEffect(() => {
    if (isChecking) return

    if (!isAuthenticated && !isPublicPath) {
      // Redirect to login with return URL
      const returnUrl = encodeURIComponent(pathname)
      router.push(`/login?returnUrl=${returnUrl}`)
    } else if (isAuthenticated && isPublicPath) {
      // Redirect authenticated users away from auth pages
      router.push('/dashboard')
    }
  }, [isAuthenticated, isPublicPath, isChecking, pathname, router])

  // Show loading while checking auth
  if (isChecking) {
    return <FullPageSpinner />
  }

  // Show loading while redirecting
  if (!isAuthenticated && !isPublicPath) {
    return <FullPageSpinner />
  }

  if (isAuthenticated && isPublicPath) {
    return <FullPageSpinner />
  }

  return <>{children}</>
}
