'use client'

import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
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
} from 'lucide-react'
import { Header } from '@/components/layout/header'
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
import type { User } from '@/types'

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
 * Role configurations
 */
const roleConfigs: Record<UserRole, { label: string; description: string; icon: React.ReactNode; color: string }> = {
  admin: {
    label: 'Administrator',
    description: 'Full access to all features',
    icon: <ShieldCheck className="h-4 w-4" />,
    color: 'text-red-500',
  },
  supervisor: {
    label: 'Supervisor',
    description: 'Manage team and view reports',
    icon: <ShieldAlert className="h-4 w-4" />,
    color: 'text-amber-500',
  },
  agent: {
    label: 'Agent',
    description: 'Handle conversations',
    icon: <Shield className="h-4 w-4" />,
    color: 'text-blue-500',
  },
}

/**
 * Role Badge
 */
function RoleBadge({ role }: { role: UserRole }) {
  const config = roleConfigs[role]
  return (
    <Badge variant="outline" className={cn('gap-1', config.color)}>
      {config.icon}
      {config.label}
    </Badge>
  )
}

/**
 * Status Badge
 */
function StatusBadge({ status }: { status: UserStatus }) {
  return (
    <Badge variant={status === 'active' ? 'success' : 'secondary'}>
      {status === 'active' ? 'Active' : 'Inactive'}
    </Badge>
  )
}

/**
 * User Card Component
 */
function UserCard({
  user,
  onEdit,
  onToggleStatus,
  onDelete,
}: {
  user: User
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
                <RoleBadge role={user.role} />
                <StatusBadge status={user.status} />
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
                Edit
              </DropdownMenuItem>
              <DropdownMenuItem onClick={onToggleStatus}>
                {user.status === 'active' ? (
                  <>
                    <UserX className="h-4 w-4 mr-2" />
                    Deactivate
                  </>
                ) : (
                  <>
                    <UserCheck className="h-4 w-4 mr-2" />
                    Activate
                  </>
                )}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="text-destructive" onClick={onDelete}>
                <Trash2 className="h-4 w-4 mr-2" />
                Delete
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
  const isEditing = !!user
  const [formData, setFormData] = useState<CreateUserInput | UpdateUserInput>(
    user
      ? { name: user.name, email: user.email, role: user.role }
      : { name: '', email: '', password: '', role: 'agent' as UserRole }
  )

  const queryClient = useQueryClient()

  const createMutation = useMutation({
    mutationFn: (data: CreateUserInput) => api.post<User>('/users', data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all })
      onOpenChange(false)
      onSuccess()
    },
  })

  const updateMutation = useMutation({
    mutationFn: (data: UpdateUserInput) => api.put<User>(`/users/${user?.id}`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.users.all })
      onOpenChange(false)
      onSuccess()
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (isEditing) {
      updateMutation.mutate(formData as UpdateUserInput)
    } else {
      createMutation.mutate(formData as CreateUserInput)
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[450px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{isEditing ? 'Edit User' : 'Add Team Member'}</DialogTitle>
            <DialogDescription>
              {isEditing ? 'Update user information' : 'Invite a new member to your team'}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="name">Full Name</Label>
              <Input
                id="name"
                placeholder="John Doe"
                value={(formData as CreateUserInput).name || ''}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                required
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                placeholder="john@example.com"
                value={(formData as CreateUserInput).email || ''}
                onChange={(e) => setFormData({ ...formData, email: e.target.value })}
                required
              />
            </div>

            {!isEditing && (
              <div className="grid gap-2">
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="Minimum 8 characters"
                  value={(formData as CreateUserInput).password || ''}
                  onChange={(e) =>
                    setFormData({ ...(formData as CreateUserInput), password: e.target.value })
                  }
                  required
                  minLength={8}
                />
              </div>
            )}

            <div className="grid gap-2">
              <Label htmlFor="role">Role</Label>
              <Select
                value={(formData as CreateUserInput).role}
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
                          <span className="font-medium">{config.label}</span>
                          <span className="text-xs text-muted-foreground ml-2">
                            {config.description}
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
              Cancel
            </Button>
            <Button type="submit" disabled={isPending}>
              {isPending ? 'Saving...' : isEditing ? 'Update User' : 'Add User'}
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
  const queryClient = useQueryClient()
  const [search, setSearch] = useState('')
  const [roleFilter, setRoleFilter] = useState<string>('all')
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [editUser, setEditUser] = useState<User | undefined>()

  // Fetch users
  const { data, isLoading } = useQuery({
    queryKey: queryKeys.users.list({ search, role: roleFilter }),
    queryFn: () =>
      api.get<{ data: User[] }>('/users', {
        ...(search && { search }),
        ...(roleFilter !== 'all' && { role: roleFilter }),
      }),
  })

  const users = data?.data || []

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
      <Header title="Team" />

      <div className="p-6 space-y-6 overflow-auto">
        {/* Stats */}
        <div className="grid gap-4 md:grid-cols-4">
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold">{stats.total}</div>
              <p className="text-xs text-muted-foreground">Total Members</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold text-green-500">{stats.active}</div>
              <p className="text-xs text-muted-foreground">Active</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold text-red-500">{stats.admins}</div>
              <p className="text-xs text-muted-foreground">Administrators</p>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="text-2xl font-bold text-blue-500">{stats.agents}</div>
              <p className="text-xs text-muted-foreground">Agents</p>
            </CardContent>
          </Card>
        </div>

        {/* Header with search and create */}
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-4 flex-1">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search users..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-9"
              />
            </div>
            <Select value={roleFilter} onValueChange={setRoleFilter}>
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="Filter by role" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Roles</SelectItem>
                <SelectItem value="admin">Administrators</SelectItem>
                <SelectItem value="supervisor">Supervisors</SelectItem>
                <SelectItem value="agent">Agents</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <Button onClick={() => setCreateDialogOpen(true)}>
            <UserPlus className="h-4 w-4 mr-2" />
            Add Member
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
                onEdit={() => setEditUser(user)}
                onToggleStatus={() =>
                  updateMutation.mutate({
                    id: user.id,
                    data: { status: user.status === 'active' ? 'inactive' : 'active' },
                  })
                }
                onDelete={() => {
                  if (confirm('Are you sure you want to delete this user?')) {
                    deleteMutation.mutate(user.id)
                  }
                }}
              />
            ))}
          </div>
        ) : (
          <Card className="border-dashed">
            <CardContent className="py-12 text-center">
              <UserPlus className="mx-auto h-12 w-12 text-muted-foreground opacity-50" />
              <p className="mt-4 text-lg font-medium">No team members</p>
              <p className="text-sm text-muted-foreground">
                Add your first team member to get started
              </p>
              <Button className="mt-4" onClick={() => setCreateDialogOpen(true)}>
                <UserPlus className="h-4 w-4 mr-2" />
                Add Member
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
    </div>
  )
}
