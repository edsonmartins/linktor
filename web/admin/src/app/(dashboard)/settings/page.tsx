'use client'

import { useState } from 'react'
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
 * Settings Navigation Items
 */
const settingsNav = [
  { id: 'profile', label: 'Profile', icon: User },
  { id: 'notifications', label: 'Notifications', icon: Bell },
  { id: 'security', label: 'Security', icon: Shield },
  { id: 'appearance', label: 'Appearance', icon: Palette },
  { id: 'api-keys', label: 'API Keys', icon: Key },
  { id: 'organization', label: 'Organization', icon: Building },
]

/**
 * Profile Settings Section
 */
function ProfileSettings() {
  const user = useUser()

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">Profile</h3>
        <p className="text-sm text-muted-foreground">
          Manage your personal information
        </p>
      </div>

      <Separator />

      <div className="flex items-center gap-4">
        <Avatar src={user?.avatar_url} fallback={user?.name || 'U'} size="xl" />
        <div>
          <Button variant="outline" size="sm">
            Change avatar
          </Button>
          <p className="mt-1 text-xs text-muted-foreground">
            JPG, PNG or GIF. Max 2MB.
          </p>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="name">Name</Label>
          <Input id="name" defaultValue={user?.name} />
        </div>
        <div className="space-y-2">
          <Label htmlFor="email">Email</Label>
          <Input id="email" type="email" defaultValue={user?.email} disabled />
        </div>
      </div>

      <div className="space-y-2">
        <Label>Role</Label>
        <div>
          <Badge variant="outline" className="capitalize">
            {user?.role}
          </Badge>
        </div>
      </div>

      <Button>Save changes</Button>
    </div>
  )
}

/**
 * Notifications Settings Section
 */
function NotificationsSettings() {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">Notifications</h3>
        <p className="text-sm text-muted-foreground">
          Configure how you receive notifications
        </p>
      </div>

      <Separator />

      <div className="space-y-4">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base">Email Notifications</CardTitle>
            <CardDescription>
              Receive email notifications for important events
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <label className="flex items-center justify-between">
                <span className="text-sm">New conversation assigned</span>
                <input type="checkbox" defaultChecked className="toggle" />
              </label>
              <label className="flex items-center justify-between">
                <span className="text-sm">Conversation resolved</span>
                <input type="checkbox" defaultChecked className="toggle" />
              </label>
              <label className="flex items-center justify-between">
                <span className="text-sm">Daily summary</span>
                <input type="checkbox" className="toggle" />
              </label>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base">Push Notifications</CardTitle>
            <CardDescription>
              Receive push notifications in your browser
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              <label className="flex items-center justify-between">
                <span className="text-sm">New messages</span>
                <input type="checkbox" defaultChecked className="toggle" />
              </label>
              <label className="flex items-center justify-between">
                <span className="text-sm">Mentions</span>
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
function SecuritySettings() {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">Security</h3>
        <p className="text-sm text-muted-foreground">
          Manage your account security settings
        </p>
      </div>

      <Separator />

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Change Password</CardTitle>
          <CardDescription>
            Update your password regularly to keep your account secure
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="current-password">Current password</Label>
            <Input id="current-password" type="password" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="new-password">New password</Label>
            <Input id="new-password" type="password" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="confirm-password">Confirm new password</Label>
            <Input id="confirm-password" type="password" />
          </div>
          <Button>Update password</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Two-Factor Authentication</CardTitle>
          <CardDescription>
            Add an extra layer of security to your account
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Button variant="outline">Enable 2FA</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Active Sessions</CardTitle>
          <CardDescription>
            Manage your active login sessions
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <div className="flex items-center justify-between p-3 bg-secondary/50 rounded-lg">
              <div>
                <p className="text-sm font-medium">Current session</p>
                <p className="text-xs text-muted-foreground">
                  macOS - Chrome - Last active: Now
                </p>
              </div>
              <Badge variant="success" dot>
                Active
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
function AppearanceSettings() {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">Appearance</h3>
        <p className="text-sm text-muted-foreground">
          Customize how Linktor looks for you
        </p>
      </div>

      <Separator />

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Theme</CardTitle>
          <CardDescription>
            Select your preferred color theme
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-3 grid-cols-3">
            <button className="p-4 rounded-lg border-2 border-primary bg-[hsl(180,3%,10%)] text-left">
              <div className="h-2 w-2 rounded-full bg-primary mb-2" />
              <p className="text-sm font-medium text-white">Terminal</p>
              <p className="text-xs text-gray-400">Default dark theme</p>
            </button>
            <button className="p-4 rounded-lg border border-border bg-gray-900 text-left opacity-50 cursor-not-allowed">
              <div className="h-2 w-2 rounded-full bg-blue-500 mb-2" />
              <p className="text-sm font-medium text-white">Ocean</p>
              <p className="text-xs text-gray-400">Coming soon</p>
            </button>
            <button className="p-4 rounded-lg border border-border bg-white text-left opacity-50 cursor-not-allowed">
              <div className="h-2 w-2 rounded-full bg-gray-900 mb-2" />
              <p className="text-sm font-medium text-gray-900">Light</p>
              <p className="text-xs text-gray-500">Coming soon</p>
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
function ApiKeysSettings() {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">API Keys</h3>
        <p className="text-sm text-muted-foreground">
          Manage API keys for external integrations
        </p>
      </div>

      <Separator />

      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">
          API keys allow external applications to access your data
        </p>
        <Button>
          Generate new key
        </Button>
      </div>

      <Card className="border-dashed">
        <CardContent className="py-8 text-center">
          <Key className="mx-auto h-12 w-12 text-muted-foreground opacity-50" />
          <p className="mt-4 text-sm font-medium">No API keys</p>
          <p className="text-xs text-muted-foreground">
            Generate a key to start integrating with external services
          </p>
        </CardContent>
      </Card>
    </div>
  )
}

/**
 * Organization Settings Section
 */
function OrganizationSettings() {
  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-lg font-medium">Organization</h3>
        <p className="text-sm text-muted-foreground">
          Manage your organization settings
        </p>
      </div>

      <Separator />

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Organization Details</CardTitle>
          <CardDescription>
            Basic information about your organization
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="org-name">Organization name</Label>
            <Input id="org-name" defaultValue="Demo Company" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="org-slug">Slug</Label>
            <Input id="org-slug" defaultValue="demo" disabled />
          </div>
          <Button>Save changes</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Subscription</CardTitle>
          <CardDescription>
            Your current plan and billing information
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between p-4 bg-primary/10 rounded-lg border border-primary/30">
            <div>
              <p className="font-medium">Professional Plan</p>
              <p className="text-sm text-muted-foreground">
                $99/month - Unlimited agents
              </p>
            </div>
            <Button variant="outline">Manage plan</Button>
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
  const [activeSection, setActiveSection] = useState('profile')

  const renderSection = () => {
    switch (activeSection) {
      case 'profile':
        return <ProfileSettings />
      case 'notifications':
        return <NotificationsSettings />
      case 'security':
        return <SecuritySettings />
      case 'appearance':
        return <AppearanceSettings />
      case 'api-keys':
        return <ApiKeysSettings />
      case 'organization':
        return <OrganizationSettings />
      default:
        return <ProfileSettings />
    }
  }

  return (
    <div className="flex flex-col h-full">
      <Header title="Settings" />

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
