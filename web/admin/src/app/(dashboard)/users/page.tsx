'use client'

import { useState, useMemo, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import {
  Plus,
  MoreVertical,
  Search,
  UserPlus,
  Shield,
  ShieldCheck,
  ShieldAlert,
  Mail,
  Trash2,
  Edit,
  UserX,
  UserCheck,
  RefreshCw,
  Eye,
  EyeOff,
  Check,
  X,
} from 'lucide-react'
import { Header } from '@/components/layout/header'
import { toastError, toastSuccess } from '@/hooks/use-toast'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Avatar } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib/utils'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { User, PaginatedResponse } from '@/types'

type UserRole = 'admin' | 'agent' | 'supervisor'
type UserStatus = 'active' | 'inactive'

interface CreateUserInput {
  name: string
  email: string
  password: string
  role: UserRole
}

interface UpdateUserInput {
  name?: string
  email?: string
  role?: UserRole
  status?: UserStatus
}

/**
 * Password strength checker
 */
function checkPasswordStrength(password: string): {
  score: number
  level: 'weak' | 'medium' | 'strong' | 'veryStrong'
  checks: {
    minLength: boolean
    uppercase: boolean
    lowercase: boolean
    number: boolean
    special: boolean
  }
} {
  const checks = {
    minLength: password.length >= 8,
    uppercase: /[A-Z]/.test(password),
    lowercase: /[a-z]/.test(password),
    number: /[0-9]/.test(password),
    special: /[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(password),
  }

  const score = Object.values(checks).filter(Boolean).length

  let level: 'weak' | 'medium' | 'strong' | 'veryStrong' = 'weak'
  if (score >= 5) level = 'veryStrong'
  else if (score >= 4) level = 'strong'
  else if (score >= 3) level = 'medium'

  return { score, level, checks }
}

/**
 * Generate a strong random password
 */
function generateStrongPassword(): string {
  const uppercase = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'
  const lowercase = 'abcdefghijklmnopqrstuvwxyz'
  const numbers = '0123456789'
  const special = '!@#$%^&*'

  const allChars = uppercase + lowercase + numbers + special

  // Ensure at least one of each required type
  let password = ''
  password += uppercase[Math.floor(Math.random() * uppercase.length)]
  password += lowercase[Math.floor(Math.random() * lowercase.length)]
  password += numbers[Math.floor(Math.random() * numbers.length)]
  password += special[Math.floor(Math.random() * special.length)]

  // Fill remaining with random chars
  for (let i = 0; i < 8; i++) {
    password += allChars[Math.floor(Math.random() * allChars.length)]
  }

  // Shuffle the password
  return password.split('').sort(() => Math.random() - 0.5).join('')
}

/**
 * Password Strength Indicator
 */
function PasswordStrengthIndicator({ password, t }: { password: string; t: (key: string) => string }) {
  const { level, checks } = useMemo(() => checkPasswordStrength(password), [password])

  if (!password) return null

  const colors = {
    weak: 'bg-red-500',
    medium: 'bg-yellow-500',
    strong: 'bg-green-500',
    veryStrong: 'bg-emerald-500',
  }

  const widths = {
    weak: 'w-1/4',
    medium: 'w-2/4',
    strong: 'w-3/4',
    veryStrong: 'w-full',
  }

  return (
    <div className="space-y-2 mt-2">
      <div className="flex items-center gap-2">
        <div className="flex-1 h-1.5 bg-secondary rounded-full overflow-hidden">
          <div className={cn('h-full transition-all', colors[level], widths[level])} />
        </div>
        <span className={cn('text-xs font-medium', {
          'text-red-500': level === 'weak',
          'text-yellow-500': level === 'medium',
          'text-green-500': level === 'strong',
          'text-emerald-500': level === 'veryStrong',
        })}>
          {t(`passwordStrength.${level}`)}
        </span>
      </div>

      <div className="text-xs space-y-1">
        <p className="text-muted-foreground font-medium">{t('passwordRequirements.title')}</p>
        <div className="grid grid-cols-2 gap-1">
          {Object.entries(checks).map(([key, passed]) => (
            <div key={key} className={cn('flex items-center gap-1', passed ? 'text-green-500' : 'text-muted-foreground')}>
              {passed ? <Check className="h-3 w-3" /> : <X className="h-3 w-3" />}
              <span>{t(`passwordRequirements.${key}`)}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

/**
 * Role Badge
 */
function RoleBadge({ role, t }: { role: UserRole; t: (key: string) => string }) {
  const configs: Record<UserRole, { icon: React.ReactNode; color: string }> = {
    admin: {
      icon: <ShieldCheck className="h-4 w-4" />,
      color: 'text-red-500',
    },
    supervisor: {
      icon: <ShieldAlert className="h-4 w-4" />,
      color: 'text-amber-500',
    },
    agent: {
      icon: <Shield className="h-4 w-4" />,
      color: 'text-blue-500',
    },
  }
  const config = configs[role]
  return (
    <Badge variant="outline" className={cn('gap-1', config.color)}>
      {config.icon}
      {t(`roles.${role}`)}
    </Badge>
  )
}

/**
 * Status Badge
 */
function StatusBadge({ status, tCommon }: { status: UserStatus; tCommon: (key: string) => string }) {
  return (
    <Badge variant={status === 'active' ? 'success' : 'secondary'}>
      {status === 'active' ? tCommon('active') : tCommon('inactive')}
    </Badge>
  )
}

/**
 * User Card Component
 */
function UserCard({
  user,
  t,
  tCommon,
  onEdit,
  onToggleStatus,
  onDelete,
}: {
  user: User
  t: (key: string) => string
  tCommon: (key: string) => string
  onEdit: () => void
  onToggleStatus: () => void
  onDelete: () => void
}) {
  return (
    <Card className="hover:border-primary/30 transition-colors">
      <CardContent className="pt-6">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-4">
            <Avatar
              src={user.avatar_url}
              fallback={user.name}
              size="lg"
              status={user.status === 'active' ? 'online' : 'offline'}
            />
            <div>
              <h3 className="font-medium">{user.name}</h3>
              <p className="text-sm text-muted-foreground flex items-center gap-1">
                <Mail className="h-3 w-3" />
                {user.email}
              </p>
              <div className="flex items-center gap-2 mt-2">
                <RoleBadge role={user.role} t={t} />
                <StatusBadge status={user.status} tCommon={tCommon} />
              </div>
            </div>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={onEdit}>
                <Edit className="h-4 w-4 mr-2" />
                {tCommon('edit')}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onToggleStatus}>
                {user.status === 'active' ? (
                  <>
                    <UserX className="h-4 w-4 mr-2" />
                    {t('deactivate')}
                  </>
                ) : (
                  <>
                    <UserCheck className="h-4 w-4 mr-2" />
                    {t('activate')}
                  </>
                )}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="text-destructive" onClick={onDelete}>
                <Trash2 className="h-4 w-4 mr-2" />
                {tCommon('delete')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardContent>
    </Card>
  )
}

/**
 * Create/Edit User Dialog
 */
function UserFormDialog({
  open,
  onOpenChange,
  user,
  onSuccess,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  user?: User
  onSuccess: () => void
}) {
  const t = useTranslations('users')
  const tCommon = useTranslations('common')
  const tValidation = useTranslations('validation')
  const tErrors = useTranslations('errors')
  const isEditing = !!user

  const getInitialFormData = () => user
    ? { name: user.name, email: user.email, password: '', confirmPassword: '', role: user.role }
    : { name: '', email: '', password: '', confirmPassword: '', role: 'agent' as UserRole }

  const [formData, setFormData] = useState<CreateUserInput & { confirmPassword: string }>(getInitialFormData())
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)

  // Reset form when dialog opens/closes or user changes
  useEffect(() => {
    if (open) {
      setFormData(getInitialFormData())
      setShowPassword(false)
      setShowConfirmPassword(false)
    }
  }, [open, user])

  const queryClient = useQueryClient()

  const passwordStrength = useMemo(() => checkPasswordStrength(formData.password), [formData.password])
  const passwordsMatch = formData.password === formData.confirmPassword
  const isPasswordValid = isEditing || (passwordStrength.score >= 4 && passwordsMatch && formData.password.length > 0)

  // Translate known error messages
  const getErrorMessage = (error: Error & { code?: string }) => {
    if (error.message?.toLowerCase().includes('email already in use')) {
      return tErrors('emailInUse')
    }
    if (error.code === 'CONFLICT') {
      return tErrors('conflict')
    }
    return error.message || tErrors('generic')
  }

  const createMutation = useMutation({
    mutationFn: (data: CreateUserInput) => api.post<User>('/users', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all })
      toastSuccess(tCommon('success'), t('addUser'))
      onOpenChange(false)
      onSuccess()
    },
    onError: (error: Error & { code?: string }) => {
      toastError(tCommon('error'), getErrorMessage(error))
    },
  })

  const updateMutation = useMutation({
    mutationFn: (data: UpdateUserInput) => api.put<User>(`/users/${user?.id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all })
      toastSuccess(tCommon('success'), t('updateUser'))
      onOpenChange(false)
      onSuccess()
    },
    onError: (error: Error & { code?: string }) => {
      toastError(tCommon('error'), getErrorMessage(error))
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!isPasswordValid) return

    if (isEditing) {
      updateMutation.mutate({ name: formData.name, email: formData.email, role: formData.role })
    } else {
      createMutation.mutate({ name: formData.name, email: formData.email, password: formData.password, role: formData.role })
    }
  }

  const handleGeneratePassword = () => {
    const newPassword = generateStrongPassword()
    setFormData({ ...formData, password: newPassword, confirmPassword: newPassword })
    setShowPassword(true)
    setShowConfirmPassword(true)
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  const roleConfigs: Record<UserRole, { labelKey: string; descKey: string; icon: React.ReactNode; color: string }> = {
    admin: {
      labelKey: 'roles.admin',
      descKey: 'roles.adminDesc',
      icon: <ShieldCheck className="h-4 w-4" />,
      color: 'text-red-500',
    },
    supervisor: {
      labelKey: 'roles.supervisor',
      descKey: 'roles.supervisorDesc',
      icon: <ShieldAlert className="h-4 w-4" />,
      color: 'text-amber-500',
    },
    agent: {
      labelKey: 'roles.agent',
      descKey: 'roles.agentDesc',
      icon: <Shield className="h-4 w-4" />,
      color: 'text-blue-500',
    },
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{isEditing ? t('editUser') : t('addMember')}</DialogTitle>
            <DialogDescription>
              {isEditing ? t('updateUserInfo') : t('inviteNewMember')}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="name">{t('fullName')}</Label>
              <Input
                id="name"
                placeholder={t('fullNamePlaceholder')}
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                required
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="email">{t('email')}</Label>
              <Input
                id="email"
                type="email"
                placeholder={t('emailPlaceholder')}
                value={formData.email}
                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                required
              />
            </div>

            {!isEditing && (
              <>
                <div className="grid gap-2">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="password">{t('password')}</Label>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="h-auto py-1 px-2 text-xs"
                      onClick={handleGeneratePassword}
                    >
                      <RefreshCw className="h-3 w-3 mr-1" />
                      {t('generatePassword')}
                    </Button>
                  </div>
                  <div className="relative">
                    <Input
                      id="password"
                      type={showPassword ? 'text' : 'password'}
                      placeholder={t('passwordPlaceholder')}
                      value={formData.password}
                      onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                      required
                      className="pr-10"
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                      onClick={() => setShowPassword(!showPassword)}
                    >
                      {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </Button>
                  </div>
                  <PasswordStrengthIndicator password={formData.password} t={t} />
                </div>

                <div className="grid gap-2">
                  <Label htmlFor="confirmPassword">{t('confirmPassword')}</Label>
                  <div className="relative">
                    <Input
                      id="confirmPassword"
                      type={showConfirmPassword ? 'text' : 'password'}
                      placeholder={t('passwordPlaceholder')}
                      value={formData.confirmPassword}
                      onChange={(e) => setFormData({ ...formData, confirmPassword: e.target.value })}
                      required
                      className={cn('pr-10', formData.confirmPassword && !passwordsMatch && 'border-red-500')}
                    />
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="absolute right-0 top-0 h-full px-3 hover:bg-transparent"
                      onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                    >
                      {showConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                    </Button>
                  </div>
                  {formData.confirmPassword && !passwordsMatch && (
                    <p className="text-xs text-red-500">{tValidation('passwordMatch')}</p>
                  )}
                </div>
              </>
            )}

            <div className="grid gap-2">
              <Label htmlFor="role">{t('role')}</Label>
              <Select
                value={formData.role}
                onValueChange={(value: UserRole) => setFormData({ ...formData, role: value })}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(roleConfigs).map(([key, config]) => (
                    <SelectItem key={key} value={key}>
                      <div className="flex items-center gap-2">
                        <span className={config.color}>{config.icon}</span>
                        <div>
                          <span className="font-medium">{t(config.labelKey)}</span>
                          <span className="text-xs text-muted-foreground ml-2">
                            {t(config.descKey)}
                          </span>
                        </div>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              {tCommon('cancel')}
            </Button>
            <Button type="submit" disabled={isPending || (!isEditing && !isPasswordValid)}>
              {isPending ? t('saving') : isEditing ? t('updateUser') : t('addUser')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

/**
 * Users Page
 */
export default function UsersPage() {
  const t = useTranslations('users')
  const tCommon = useTranslations('common')
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState<string>('all')
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [editUser, setEditUser] = useState<User | undefined>()
  const [userToDelete, setUserToDelete] = useState<User | undefined>()

  // Fetch users
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.users.list({ search, role: roleFilter }),
    queryFn: () =>
      api.get<PaginatedResponse<User>>('/users', {
        ...(search && { search }),
        ...(roleFilter !== 'all' && { role: roleFilter }),
      }),
  })

  const users = data?.data ?? []

  // Update user mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateUserInput }) =>
      api.put(`/users/${id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all })
    },
  })

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/users/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all })
      toastSuccess(tCommon('success'), t('deleteUser'))
      setUserToDelete(undefined)
    },
    onError: (error: Error) => {
      toastError(tCommon('error'), error.message)
    },
  })

  const filteredUsers = users.filter(
    (user) =>
      user.name.toLowerCase().includes(search.toLowerCase()) ||
      user.email.toLowerCase().includes(search.toLowerCase())
  )

  // Group users by role for stats
  const stats = {
    total: users.length,
    admins: users.filter((u) => u.role === 'admin').length,
    supervisors: users.filter((u) => u.role === 'supervisor').length,
    agents: users.filter((u) => u.role === 'agent').length,
    active: users.filter((u) => u.status === 'active').length,
  }

  return (
    <div className="flex flex-col h-full">
      <Header title={t('title')} />

      <div className="p-6 space-y-6 overflow-auto">
        {/* Stats */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold">{stats.total}</div>
              <p className="text-xs text-muted-foreground">{t('totalMembers')}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold text-green-500">{stats.active}</div>
              <p className="text-xs text-muted-foreground">{tCommon('active')}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold text-red-500">{stats.admins}</div>
              <p className="text-xs text-muted-foreground">{t('administrators')}</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold text-blue-500">{stats.agents}</div>
              <p className="text-xs text-muted-foreground">{t('agents')}</p>
            </CardContent>
          </Card>
        </div>

        {/* Header with search and create */}
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-4 flex-1">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder={t('searchUsers')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>
            <Select value={roleFilter} onValueChange={setRoleFilter}>
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder={t('filterByRole')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t('allRoles')}</SelectItem>
                <SelectItem value="admin">{t('administrators')}</SelectItem>
                <SelectItem value="supervisor">{t('supervisors')}</SelectItem>
                <SelectItem value="agent">{t('agents')}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button onClick={() => setCreateDialogOpen(true)}>
            <UserPlus className="h-4 w-4 mr-2" />
            {t('addMember')}
          </Button>
        </div>

        {/* Users Grid */}
        {isLoading ? (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <Card key={i}>
                <CardContent className="pt-6">
                  <div className="flex items-center gap-4">
                    <Skeleton className="h-12 w-12 rounded-full" />
                    <div className="space-y-2">
                      <Skeleton className="h-4 w-24" />
                      <Skeleton className="h-3 w-32" />
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        ) : filteredUsers.length > 0 ? (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {filteredUsers.map((user) => (
              <UserCard
                key={user.id}
                user={user}
                t={t}
                tCommon={tCommon}
                onEdit={() => setEditUser(user)}
                onToggleStatus={() =>
                  updateMutation.mutate({
                    id: user.id,
                    data: { status: user.status === 'active' ? 'inactive' : 'active' },
                  })
                }
                onDelete={() => setUserToDelete(user)}
              />
            ))}
          </div>
        ) : (
          <Card className="border-dashed">
            <CardContent className="py-12 text-center">
              <UserPlus className="mx-auto h-12 w-12 text-muted-foreground opacity-50" />
              <p className="mt-4 text-lg font-medium">{t('noTeamMembers')}</p>
              <p className="text-sm text-muted-foreground">
                {t('addFirstMember')}
              </p>
              <Button className="mt-4" onClick={() => setCreateDialogOpen(true)}>
                <UserPlus className="h-4 w-4 mr-2" />
                {t('addMember')}
              </Button>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Create Dialog */}
      <UserFormDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onSuccess={() => {}}
      />

      {/* Edit Dialog */}
      {editUser && (
        <UserFormDialog
          open={!!editUser}
          onOpenChange={(open) => !open && setEditUser(undefined)}
          user={editUser}
          onSuccess={() => setEditUser(undefined)}
        />
      )}

      {/* Delete Confirmation Dialog */}
      <AlertDialog open={!!userToDelete} onOpenChange={(open) => !open && setUserToDelete(undefined)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('confirmDeleteTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('confirmDeleteDescription', { name: userToDelete?.name })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tCommon('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => userToDelete && deleteMutation.mutate(userToDelete.id)}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? tCommon('loading') : tCommon('delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
