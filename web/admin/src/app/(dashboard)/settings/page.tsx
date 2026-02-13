'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { User, Bell, Shield, Palette, Key, Building } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { Avatar } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { useUser } from '@/stores/auth-store'
import { cn } from '@/lib/utils'

/**
 * Profile Settings Section
 */
function ProfileSettings({ t }: { t: (key: string) => string }) {
  const user = useUser()

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">{t('profile')}</h3>
        <p className="text-sm text-muted-foreground">
          {t('managePersonalInfo')}
        </p>
      </div>

      <Separator />

      <div className="flex items-center gap-4">
        <Avatar src={user?.avatar_url} fallback={user?.name || 'U'} size="xl" />
        <div>
          <Button variant="outline" size="sm">
            {t('changeAvatar')}
          </Button>
          <p className="mt-1 text-xs text-muted-foreground">
            {t('avatarHint')}
          </p>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="name">{t('profile')}</Label>
          <Input id="name" defaultValue={user?.name} />
        </div>
        <div className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <Input id="email" type="email" defaultValue={user?.email} disabled />
        </div>
      </div>

      <div className="space-y-2">
        <Label>{t('role')}</Label>
        <div>
          <Badge variant="outline" className="capitalize">
            {user?.role}
          </Badge>
        </div>
      </div>

      <Button>{t('saveChanges')}</Button>
    </div>
  )
}

/**
 * Notifications Settings Section
 */
function NotificationsSettings({ t }: { t: (key: string) => string }) {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">{t('notifications')}</h3>
        <p className="text-sm text-muted-foreground">
          {t('configureNotifications')}
        </p>
      </div>

      <Separator />

      <div className="space-y-4">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base">{t('emailNotifications')}</CardTitle>
            <CardDescription>
              {t('emailNotificationsDesc')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <label className="flex items-center justify-between">
                <span className="text-sm">{t('newConversationAssigned')}</span>
                <input type="checkbox" defaultChecked className="toggle" />
              </label>
              <label className="flex items-center justify-between">
                <span className="text-sm">{t('conversationResolved')}</span>
                <input type="checkbox" defaultChecked className="toggle" />
              </label>
              <label className="flex items-center justify-between">
                <span className="text-sm">{t('dailySummary')}</span>
                <input type="checkbox" className="toggle" />
              </label>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base">{t('pushNotifications')}</CardTitle>
            <CardDescription>
              {t('pushNotificationsDesc')}
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <label className="flex items-center justify-between">
                <span className="text-sm">{t('newMessages')}</span>
                <input type="checkbox" defaultChecked className="toggle" />
              </label>
              <label className="flex items-center justify-between">
                <span className="text-sm">{t('mentions')}</span>
                <input type="checkbox" defaultChecked className="toggle" />
              </label>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

/**
 * Security Settings Section
 */
function SecuritySettings({ t, tCommon }: { t: (key: string) => string; tCommon: (key: string) => string }) {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">{t('security')}</h3>
        <p className="text-sm text-muted-foreground">
          {t('manageAccountSecurity')}
        </p>
      </div>

      <Separator />

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{t('changePassword')}</CardTitle>
          <CardDescription>
            {t('changePasswordDesc')}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="current-password">{t('currentPassword')}</Label>
            <Input id="current-password" type="password" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="new-password">{t('newPassword')}</Label>
            <Input id="new-password" type="password" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="confirm-password">{t('confirmNewPassword')}</Label>
            <Input id="confirm-password" type="password" />
          </div>
          <Button>{t('updatePassword')}</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{t('twoFactorAuth')}</CardTitle>
          <CardDescription>
            {t('twoFactorAuthDesc')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="outline">{t('enable2FA')}</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{t('activeSessions')}</CardTitle>
          <CardDescription>
            {t('activeSessionsDesc')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <div className="flex items-center justify-between p-3 bg-secondary/50 rounded-lg">
              <div>
                <p className="text-sm font-medium">{t('currentSession')}</p>
                <p className="text-xs text-muted-foreground">
                  macOS - Chrome - {t('lastActive')}: Now
                </p>
              </div>
              <Badge variant="success" dot>
                {tCommon('active')}
              </Badge>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

/**
 * Appearance Settings Section
 */
function AppearanceSettings({ t }: { t: (key: string) => string }) {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">{t('appearance')}</h3>
        <p className="text-sm text-muted-foreground">
          {t('customizeAppearance')}
        </p>
      </div>

      <Separator />

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{t('theme')}</CardTitle>
          <CardDescription>
            {t('selectTheme')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-3 grid-cols-3">
            <button className="p-4 rounded-lg border-2 border-primary bg-[hsl(180,3%,10%)] text-left">
              <div className="h-2 w-2 rounded-full bg-primary mb-2" />
              <p className="text-sm font-medium text-white">{t('terminal')}</p>
              <p className="text-xs text-gray-400">{t('defaultDarkTheme')}</p>
            </button>
            <button className="p-4 rounded-lg border border-border bg-gray-900 text-left opacity-50 cursor-not-allowed">
              <div className="h-2 w-2 rounded-full bg-blue-500 mb-2" />
              <p className="text-sm font-medium text-white">{t('ocean')}</p>
              <p className="text-xs text-gray-400">{t('comingSoon')}</p>
            </button>
            <button className="p-4 rounded-lg border border-border bg-white text-left opacity-50 cursor-not-allowed">
              <div className="h-2 w-2 rounded-full bg-gray-900 mb-2" />
              <p className="text-sm font-medium text-gray-900">{t('light')}</p>
              <p className="text-xs text-gray-500">{t('comingSoon')}</p>
            </button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

/**
 * API Keys Settings Section
 */
function ApiKeysSettings({ t }: { t: (key: string) => string }) {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">{t('apiKeys')}</h3>
        <p className="text-sm text-muted-foreground">
          {t('manageApiKeys')}
        </p>
      </div>

      <Separator />

      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          {t('apiKeysAllowAccess')}
        </p>
        <Button>
          {t('generateNewKey')}
        </Button>
      </div>

      <Card className="border-dashed">
        <CardContent className="py-8 text-center">
          <Key className="mx-auto h-12 w-12 text-muted-foreground opacity-50" />
          <p className="mt-4 text-sm font-medium">{t('noApiKeys')}</p>
          <p className="text-xs text-muted-foreground">
            {t('generateKeyHint')}
          </p>
        </CardContent>
      </Card>
    </div>
  )
}

/**
 * Organization Settings Section
 */
function OrganizationSettings({ t }: { t: (key: string) => string }) {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">{t('organization')}</h3>
        <p className="text-sm text-muted-foreground">
          {t('manageOrganization')}
        </p>
      </div>

      <Separator />

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{t('organizationDetails')}</CardTitle>
          <CardDescription>
            {t('organizationDetailsDesc')}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="org-name">{t('organizationName')}</Label>
            <Input id="org-name" defaultValue="Demo Company" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="org-slug">{t('slug')}</Label>
            <Input id="org-slug" defaultValue="demo" disabled />
          </div>
          <Button>{t('saveChanges')}</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{t('subscription')}</CardTitle>
          <CardDescription>
            {t('subscriptionDesc')}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between p-4 bg-primary/10 rounded-lg border border-primary/30">
            <div>
              <p className="font-medium">{t('professionalPlan')}</p>
              <p className="text-sm text-muted-foreground">
                $99/month - {t('unlimitedAgents')}
              </p>
            </div>
            <Button variant="outline">{t('managePlan')}</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

/**
 * Settings Page
 */
export default function SettingsPage() {
  const t = useTranslations('settings')
  const tCommon = useTranslations('common')
  const [activeSection, setActiveSection] = useState('profile')

  // Settings Navigation Items with translations
  const settingsNav = [
    { id: 'profile', label: t('profile'), icon: User },
    { id: 'notifications', label: t('notifications'), icon: Bell },
    { id: 'security', label: t('security'), icon: Shield },
    { id: 'appearance', label: t('appearance'), icon: Palette },
    { id: 'api-keys', label: t('apiKeys'), icon: Key },
    { id: 'organization', label: t('organization'), icon: Building },
  ]

  const renderSection = () => {
    switch (activeSection) {
      case 'profile':
        return <ProfileSettings t={t} />
      case 'notifications':
        return <NotificationsSettings t={t} />
      case 'security':
        return <SecuritySettings t={t} tCommon={tCommon} />
      case 'appearance':
        return <AppearanceSettings t={t} />
      case 'api-keys':
        return <ApiKeysSettings t={t} />
      case 'organization':
        return <OrganizationSettings t={t} />
      default:
        return <ProfileSettings t={t} />
    }
  }

  return (
    <div className="flex flex-col h-full">
      <Header title={t('title')} />

      <div className="flex flex-1 overflow-hidden">
        {/* Settings Navigation */}
        <nav className="w-64 border-r border-border p-4 overflow-auto">
          <div className="space-y-1">
            {settingsNav.map((item) => {
              const Icon = item.icon
              return (
                <button
                  key={item.id}
                  onClick={() => setActiveSection(item.id)}
                  className={cn(
                    'flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    activeSection === item.id
                      ? 'bg-primary/10 text-primary'
                      : 'text-muted-foreground hover:bg-secondary hover:text-foreground'
                  )}
                >
                  <Icon className="h-4 w-4" />
                  {item.label}
                </button>
              )
            })}
          </div>
        </nav>

        {/* Settings Content */}
        <main className="flex-1 overflow-auto p-6">
          <div className="max-w-2xl">{renderSection()}</div>
        </main>
      </div>
    </div>
  )
}
