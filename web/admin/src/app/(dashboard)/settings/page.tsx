'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useLocale } from 'next-intl'
import { useRouter } from 'next/navigation'
import { User, Bell, Shield, Palette, Key, Building, Copy, Globe } from 'lucide-react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'
import { Avatar } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useUser } from '@/stores/auth-store'
import { cn } from '@/lib/utils'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { locales, type Locale } from '@/i18n/config'

type APIKeyRecord = {
  id: string
  name: string
  key_prefix: string
  scopes: string[]
  last_used_at?: string | null
  expires_at?: string | null
  created_at: string
}

type CreatedAPIKey = APIKeyRecord & {
  key: string
}

type Tenant = {
  id: string
  name: string
  slug: string
  plan: string
  settings: Record<string, string>
}

const localeNames: Record<Locale, string> = {
  'pt-BR': '🇧🇷 Português (Brasil)',
  'es': '🇪🇸 Español',
  'en': '🇺🇸 English',
}

/**
 * Profile Settings Section
 */
function ProfileSettings({ t }: { t: ReturnType<typeof useTranslations<'settings'>> }) {
  const user = useUser()
  const { toast } = useToast()
  const locale = useLocale() as Locale
  const router = useRouter()
  const [name, setName] = useState(user?.name || '')

  const updateProfileMutation = useMutation({
    mutationFn: (data: { name?: string; avatar_url?: string }) =>
      api.put('/me', data),
    onSuccess: () => {
      toast({ title: t('profileUpdated'), description: t('profileUpdatedDesc') })
    },
    onError: () => {
      toast({ title: t('profileUpdateFailed'), variant: 'error' })
    },
  })

  const handleSaveProfile = () => {
    updateProfileMutation.mutate({ name })
  }

  const handleLocaleChange = (newLocale: Locale) => {
    document.cookie = `locale=${newLocale};path=/;max-age=31536000`
    router.refresh()
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">{t('profile')}</h3>
        <p className="text-sm text-muted-foreground">{t('managePersonalInfo')}</p>
      </div>

      <Separator />

      <div className="flex items-center gap-4">
        <Avatar src={user?.avatar_url} fallback={user?.name || 'U'} size="xl" />
        <div>
          <Button variant="outline" size="sm">{t('changeAvatar')}</Button>
          <p className="mt-1 text-xs text-muted-foreground">{t('avatarHint')}</p>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="name">{t('profile')}</Label>
          <Input
            id="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <Input id="email" type="email" defaultValue={user?.email} disabled />
        </div>
      </div>

      <div className="space-y-2">
        <Label>{t('role')}</Label>
        <div>
          <Badge variant="outline" className="capitalize">{user?.role}</Badge>
        </div>
      </div>

      {/* Language selector */}
      <div className="space-y-2">
        <Label className="flex items-center gap-2">
          <Globe className="h-4 w-4" />
          {t('language')}
        </Label>
        <Select value={locale} onValueChange={(v) => handleLocaleChange(v as Locale)}>
          <SelectTrigger className="w-full md:w-[280px]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {locales.map((loc) => (
              <SelectItem key={loc} value={loc}>{localeNames[loc]}</SelectItem>
            ))}
          </SelectContent>
        </Select>
        <p className="text-xs text-muted-foreground">{t('languageDesc')}</p>
      </div>

      <Button
        onClick={handleSaveProfile}
        disabled={updateProfileMutation.isPending}
      >
        {updateProfileMutation.isPending ? t('saving') : t('saveChanges')}
      </Button>
    </div>
  )
}

/**
 * Notifications Settings Section
 */
function NotificationsSettings({ t }: { t: ReturnType<typeof useTranslations<'settings'>> }) {
  const { toast } = useToast()

  const { data: tenant } = useQuery({
    queryKey: queryKeys.tenant.current(),
    queryFn: () => api.get<Tenant>('/tenant'),
  })

  const settings = tenant?.settings || {}

  const [emailNewConvo, setEmailNewConvo] = useState(settings['notif.email.new_conversation'] !== 'false')
  const [emailResolved, setEmailResolved] = useState(settings['notif.email.resolved'] !== 'false')
  const [emailDaily, setEmailDaily] = useState(settings['notif.email.daily_summary'] === 'true')
  const [pushMessages, setPushMessages] = useState(settings['notif.push.messages'] !== 'false')
  const [pushMentions, setPushMentions] = useState(settings['notif.push.mentions'] !== 'false')

  const saveMutation = useMutation({
    mutationFn: (newSettings: Record<string, string>) =>
      api.put('/tenant', { settings: newSettings }),
    onSuccess: () => {
      toast({ title: t('notificationsSaved'), description: t('notificationsSavedDesc') })
    },
    onError: () => {
      toast({ title: t('notificationsSaveFailed'), variant: 'error' })
    },
  })

  const handleSave = () => {
    saveMutation.mutate({
      'notif.email.new_conversation': String(emailNewConvo),
      'notif.email.resolved': String(emailResolved),
      'notif.email.daily_summary': String(emailDaily),
      'notif.push.messages': String(pushMessages),
      'notif.push.mentions': String(pushMentions),
    })
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">{t('notifications')}</h3>
        <p className="text-sm text-muted-foreground">{t('configureNotifications')}</p>
      </div>

      <Separator />

      <div className="space-y-4">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base">{t('emailNotifications')}</CardTitle>
            <CardDescription>{t('emailNotificationsDesc')}</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <Label htmlFor="email-new-convo" className="font-normal">{t('newConversationAssigned')}</Label>
                <Switch id="email-new-convo" checked={emailNewConvo} onCheckedChange={setEmailNewConvo} />
              </div>
              <div className="flex items-center justify-between">
                <Label htmlFor="email-resolved" className="font-normal">{t('conversationResolved')}</Label>
                <Switch id="email-resolved" checked={emailResolved} onCheckedChange={setEmailResolved} />
              </div>
              <div className="flex items-center justify-between">
                <Label htmlFor="email-daily" className="font-normal">{t('dailySummary')}</Label>
                <Switch id="email-daily" checked={emailDaily} onCheckedChange={setEmailDaily} />
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base">{t('pushNotifications')}</CardTitle>
            <CardDescription>{t('pushNotificationsDesc')}</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <Label htmlFor="push-messages" className="font-normal">{t('newMessages')}</Label>
                <Switch id="push-messages" checked={pushMessages} onCheckedChange={setPushMessages} />
              </div>
              <div className="flex items-center justify-between">
                <Label htmlFor="push-mentions" className="font-normal">{t('mentions')}</Label>
                <Switch id="push-mentions" checked={pushMentions} onCheckedChange={setPushMentions} />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Button onClick={handleSave} disabled={saveMutation.isPending}>
        {saveMutation.isPending ? t('saving') : t('saveChanges')}
      </Button>
    </div>
  )
}

/**
 * Security Settings Section
 */
function SecuritySettings({ t, tCommon }: { t: ReturnType<typeof useTranslations<'settings'>>; tCommon: ReturnType<typeof useTranslations<'common'>> }) {
  const { toast } = useToast()
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  const changePasswordMutation = useMutation({
    mutationFn: (data: { current_password: string; new_password: string }) =>
      api.put('/me/password', data),
    onSuccess: () => {
      toast({ title: t('passwordChanged'), description: t('passwordChangedDesc') })
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
    },
    onError: (error: Error) => {
      toast({ title: t('passwordChangeFailed'), description: error.message, variant: 'error' })
    },
  })

  const handleChangePassword = () => {
    if (!currentPassword) {
      toast({ title: t('currentPasswordRequired'), variant: 'error' })
      return
    }
    if (newPassword.length < 8) {
      toast({ title: t('passwordMinLength'), variant: 'error' })
      return
    }
    if (newPassword !== confirmPassword) {
      toast({ title: t('passwordsDoNotMatch'), variant: 'error' })
      return
    }
    changePasswordMutation.mutate({
      current_password: currentPassword,
      new_password: newPassword,
    })
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">{t('security')}</h3>
        <p className="text-sm text-muted-foreground">{t('manageAccountSecurity')}</p>
      </div>

      <Separator />

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{t('changePassword')}</CardTitle>
          <CardDescription>{t('changePasswordDesc')}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="current-password">{t('currentPassword')}</Label>
            <Input
              id="current-password"
              type="password"
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="new-password">{t('newPassword')}</Label>
            <Input
              id="new-password"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="confirm-password">{t('confirmNewPassword')}</Label>
            <Input
              id="confirm-password"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
            />
          </div>
          <Button
            onClick={handleChangePassword}
            disabled={changePasswordMutation.isPending}
          >
            {changePasswordMutation.isPending ? t('saving') : t('updatePassword')}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{t('twoFactorAuth')}</CardTitle>
          <CardDescription>{t('twoFactorAuthDesc')}</CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="outline">{t('enable2FA')}</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{t('activeSessions')}</CardTitle>
          <CardDescription>{t('activeSessionsDesc')}</CardDescription>
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
function AppearanceSettings({ t }: { t: ReturnType<typeof useTranslations<'settings'>> }) {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">{t('appearance')}</h3>
        <p className="text-sm text-muted-foreground">{t('customizeAppearance')}</p>
      </div>

      <Separator />

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{t('theme')}</CardTitle>
          <CardDescription>{t('selectTheme')}</CardDescription>
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
function ApiKeysSettings({ t }: { t: ReturnType<typeof useTranslations<'settings'>> }) {
  const { toast } = useToast()
  const queryClient = useQueryClient()
  const [newlyCreatedKey, setNewlyCreatedKey] = useState<CreatedAPIKey | null>(null)

  const { data: apiKeys = [], isLoading } = useQuery({
    queryKey: queryKeys.apiKeys.list(),
    queryFn: () => api.get<APIKeyRecord[]>('/api-keys'),
  })

  const createMutation = useMutation({
    mutationFn: () => api.post<CreatedAPIKey>('/api-keys', {
      name: `Admin API Key ${new Date().toLocaleString()}`,
      scopes: ['*'],
    }),
    onSuccess: (createdKey) => {
      setNewlyCreatedKey(createdKey)
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys.all })
      toast({ title: t('apiKeyGenerated'), description: t('apiKeyGeneratedDesc') })
    },
    onError: () => {
      toast({ title: t('apiKeyGenerateFailed'), variant: 'error' })
    },
  })

  const handleCopyKey = async (key: string) => {
    await navigator.clipboard.writeText(key)
    toast({ title: t('apiKeyCopied'), description: t('apiKeyCopiedDesc') })
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">{t('apiKeys')}</h3>
        <p className="text-sm text-muted-foreground">{t('manageApiKeys')}</p>
      </div>

      <Separator />

      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{t('apiKeysAllowAccess')}</p>
        <Button onClick={() => createMutation.mutate()} disabled={createMutation.isPending}>
          {createMutation.isPending ? t('generating') : t('generateNewKey')}
        </Button>
      </div>

      {newlyCreatedKey && (
        <Card>
          <CardContent className="flex items-center justify-between gap-3 py-4">
            <div className="min-w-0">
              <p className="truncate font-mono text-sm">{newlyCreatedKey.key}</p>
              <p className="text-xs text-muted-foreground">{t('storeKeyWarning')}</p>
            </div>
            <Button variant="outline" size="sm" onClick={() => handleCopyKey(newlyCreatedKey.key)}>
              <Copy className="mr-2 h-4 w-4" />
              {t('copy')}
            </Button>
          </CardContent>
        </Card>
      )}

      {isLoading ? (
        <Card className="border-dashed">
          <CardContent className="py-8 text-center text-sm text-muted-foreground">
            {t('loadingApiKeys')}
          </CardContent>
        </Card>
      ) : apiKeys.length > 0 ? (
        <div className="space-y-3">
          {apiKeys.map((apiKey) => (
            <Card key={apiKey.id}>
              <CardContent className="flex items-center justify-between gap-3 py-4">
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{apiKey.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {t('prefix')} {apiKey.key_prefix} - {t('created')} {new Date(apiKey.created_at).toLocaleString()}
                  </p>
                </div>
                <Badge variant="outline">{apiKey.scopes.join(', ') || '*'}</Badge>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <Card className="border-dashed">
          <CardContent className="py-8 text-center">
            <Key className="mx-auto h-12 w-12 text-muted-foreground opacity-50" />
            <p className="mt-4 text-sm font-medium">{t('noApiKeys')}</p>
            <p className="text-xs text-muted-foreground">{t('generateKeyHint')}</p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

/**
 * Organization Settings Section
 */
function OrganizationSettings({ t }: { t: ReturnType<typeof useTranslations<'settings'>> }) {
  const { toast } = useToast()
  const queryClient = useQueryClient()

  const { data: tenant } = useQuery({
    queryKey: queryKeys.tenant.current(),
    queryFn: () => api.get<Tenant>('/tenant'),
  })

  const [orgName, setOrgName] = useState(tenant?.name || '')

  // Sync state when tenant loads
  useState(() => {
    if (tenant?.name && !orgName) setOrgName(tenant.name)
  })

  const updateOrgMutation = useMutation({
    mutationFn: (data: { name: string }) =>
      api.put('/tenant', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tenant.all })
      toast({ title: t('orgUpdated'), description: t('orgUpdatedDesc') })
    },
    onError: () => {
      toast({ title: t('orgUpdateFailed'), variant: 'error' })
    },
  })

  const handleSaveOrg = () => {
    updateOrgMutation.mutate({ name: orgName })
  }

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">{t('organization')}</h3>
        <p className="text-sm text-muted-foreground">{t('manageOrganization')}</p>
      </div>

      <Separator />

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{t('organizationDetails')}</CardTitle>
          <CardDescription>{t('organizationDetailsDesc')}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="org-name">{t('organizationName')}</Label>
            <Input
              id="org-name"
              value={orgName || tenant?.name || ''}
              onChange={(e) => setOrgName(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="org-slug">{t('slug')}</Label>
            <Input id="org-slug" value={tenant?.slug || ''} disabled />
          </div>
          <Button
            onClick={handleSaveOrg}
            disabled={updateOrgMutation.isPending}
          >
            {updateOrgMutation.isPending ? t('saving') : t('saveChanges')}
          </Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">{t('subscription')}</CardTitle>
          <CardDescription>{t('subscriptionDesc')}</CardDescription>
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

        <main className="flex-1 overflow-auto p-6">
          <div className="max-w-2xl">{renderSection()}</div>
        </main>
      </div>
    </div>
  )
}
