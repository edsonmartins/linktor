'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuthStore } from '@/stores/auth-store'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { toastError } from '@/hooks/use-toast'
import { Terminal, Lock, Mail } from 'lucide-react'

export default function LoginPage() {
  const router = useRouter()
  const { login, isLoading } = useAuthStore()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    try {
      await login(email, password)
      router.push('/dashboard')
    } catch (error) {
      toastError(
        'Login failed',
        error instanceof Error ? error.message : 'Invalid credentials'
      )
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-4">
      {/* Terminal-style background grid */}
      <div className="absolute inset-0 bg-[linear-gradient(rgba(34,197,94,0.03)_1px,transparent_1px),linear-gradient(90deg,rgba(34,197,94,0.03)_1px,transparent_1px)] bg-[size:50px_50px]" />

      <Card variant="glow" className="relative w-full max-w-md">
        <CardHeader className="text-center">
          {/* Terminal icon with glow */}
          <div className="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10 ring-2 ring-primary/30">
            <Terminal className="h-8 w-8 text-primary" />
          </div>

          <CardTitle className="text-2xl font-bold tracking-tight">
            <span className="text-gradient">LINKTOR</span>
          </CardTitle>
          <CardDescription>
            <span className="terminal-prompt">admin panel access</span>
          </CardDescription>
        </CardHeader>

        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                placeholder="admin@demo.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                variant="terminal"
                leftIcon={<Mail className="h-4 w-4" />}
                required
                autoComplete="email"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
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
              {isLoading ? 'Authenticating...' : 'Login'}
            </Button>
          </form>

          {/* Demo credentials hint */}
          <div className="mt-6 rounded-md border border-border bg-secondary/50 p-3">
            <p className="text-xs text-muted-foreground">
              <span className="font-semibold text-primary">Demo credentials:</span>
            </p>
            <p className="mt-1 font-mono text-xs text-muted-foreground">
              Email: admin@demo.com
              <br />
              Password: admin123
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
