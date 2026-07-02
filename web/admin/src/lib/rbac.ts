/**
 * Frontend RBAC helpers — mirror the backend's role gating so the UI does not
 * surface features the API will reject with 403.
 *
 * Source of truth is the backend: routes guarded by RequireRole("admin","owner")
 * are admin-only (users/team, roles, audit logs, observability, api keys,
 * tenant + operational settings updates). Everything else is available to all
 * authenticated roles. Keep this list in sync with cmd/server/main.go.
 */

export type Role = 'owner' | 'admin' | 'supervisor' | 'agent' | string

const ADMIN_ROLES = new Set(['owner', 'admin'])

/** Roles that may manage users, roles, audit logs, observability, etc. */
export function isAdmin(role: Role | undefined | null): boolean {
  return !!role && ADMIN_ROLES.has(role)
}

/**
 * Dashboard routes that require an admin/owner role. A non-admin reaching one
 * of these (by typing the URL) should see an access-denied screen rather than
 * a wall of 403s.
 */
export const ADMIN_ONLY_ROUTES = [
  '/users',
  '/roles',
  '/audit-logs',
  '/observability',
] as const

export function isAdminOnlyRoute(pathname: string): boolean {
  return ADMIN_ONLY_ROUTES.some(
    (route) => pathname === route || pathname.startsWith(`${route}/`)
  )
}

export function canAccessRoute(pathname: string, role: Role | undefined | null): boolean {
  if (isAdminOnlyRoute(pathname)) return isAdmin(role)
  return true
}
