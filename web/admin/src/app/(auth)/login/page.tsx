'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import Image from 'next/image'
import { useTranslations } from 'next-intl'
import { useAuthStore } from '@/stores/auth-store'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toastError } from '@/hooks/use-toast'
import { LocaleSwitcher } from '@/components/locale-switcher'
import { ThemeSwitcher } from '@/components/theme-switcher'
import { Lock, Mail } from 'lucide-react'
import { LoginBanner } from './login-banner'

export default function LoginPage() {
  const t = useTranslations('auth')
  const router = useRouter()
  const { login, isLoading, isAuthenticated } = useAuthStore()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  // Already-authenticated visitors (e.g. bounced here by middleware before
  // the auth marker cookie was set) go straight to the dashboard.
  useEffect(() => {
    if (isAuthenticated) {
      router.replace('/dashboard')
    }
  }, [isAuthenticated, router])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    try {
      await login(email, password)
      router.push('/dashboard')
    } catch (error) {
      toastError(
        t('loginFailed'),
        error instanceof Error ? error.message : t('invalidCredentials')
      )
    }
  }

  return (
    <div className="grid min-h-screen bg-background lg:grid-cols-[minmax(0,7fr)_minmax(380px,3fr)]">
      {/* Banner: 70% of the width on desktop, dropped on small screens */}
      <aside className="hidden lg:block">
        <LoginBanner />
      </aside>

      <main className="lk-main relative flex min-h-screen flex-col">
        {/* Small screens have no banner, so they keep the terminal grid */}
        <div className="absolute inset-0 bg-[linear-gradient(hsl(var(--primary)/0.06)_1px,transparent_1px),linear-gradient(90deg,hsl(var(--primary)/0.06)_1px,transparent_1px)] bg-[size:50px_50px] lg:hidden" />

        <div className="relative z-10 flex items-center justify-end gap-1 p-4">
          <ThemeSwitcher />
          <LocaleSwitcher />
        </div>

        <div className="relative z-10 flex flex-1 items-center justify-center px-6 pb-10 sm:px-10">
          <div className="w-full max-w-sm">
            <div className="mb-8 w-fit rounded-lg dark:bg-foreground dark:px-3 dark:py-1.5">
              <Image
                src="/images/logo_linktor_texto.png"
                alt="Linktor"
                width={888}
                height={386}
                className="h-12 w-auto"
                priority
              />
            </div>

            <h1 className="text-2xl font-bold tracking-tight">{t('adminPanel')}</h1>
            <p className="mb-8 mt-1 text-sm text-muted-foreground">
              <span className="terminal-prompt">{t('signIn')}</span>
            </p>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="email">{t('email')}</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="you@company.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  variant="terminal"
                  leftIcon={<Mail className="h-4 w-4" />}
                  required
                  autoComplete="email"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="password">{t('password')}</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="********"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  variant="terminal"
                  leftIcon={<Lock className="h-4 w-4" />}
                  required
                  autoComplete="current-password"
                />
              </div>

              <Button
                type="submit"
                className="w-full"
                loading={isLoading}
              >
                {isLoading ? t('authenticating') : t('login')}
              </Button>
            </form>

            {/* Demo credentials hint — only rendered when explicitly enabled */}
            {process.env.NEXT_PUBLIC_SHOW_DEMO_CREDENTIALS === 'true' && (
              <div className="mt-6 rounded-md border border-border bg-secondary/50 p-3">
                <p className="text-xs text-muted-foreground">
                  <span className="font-semibold text-primary">{t('demoCredentials')}</span>
                </p>
                <p className="mt-1 font-mono text-xs text-muted-foreground">
                  Email: admin@demo.com
                  <br />
                  Password: admin123
                </p>
              </div>
            )}
          </div>
        </div>

        <p className="relative z-10 px-6 pb-6 text-center text-xs text-muted-foreground">
          {t('footer', { year: new Date().getFullYear() })}
        </p>
      </main>
    </div>
  )
}
