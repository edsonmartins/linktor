import { NextRequest, NextResponse } from 'next/server'

const PUBLIC_PATHS = ['/login', '/forgot-password', '/reset-password']

/**
 * Server-side route protection.
 *
 * Auth tokens live in localStorage (not readable here), so the client mirrors
 * their presence into the non-sensitive `linktor_authed` marker cookie (see
 * src/lib/api.ts). Unauthenticated visitors are redirected to /login before
 * any protected markup is rendered. The real security boundary remains the
 * API's Authorization check — this is a UX/defense-in-depth layer only.
 * Locale detection is handled in i18n/request.ts.
 */
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl

  if (pathname === '/' || PUBLIC_PATHS.some((path) => pathname.startsWith(path))) {
    return NextResponse.next()
  }

  const authed = request.cookies.get('linktor_authed')?.value === '1'
  if (!authed) {
    return NextResponse.redirect(new URL('/login', request.url))
  }

  return NextResponse.next()
}

export const config = {
  matcher: [
    // Match all pathnames except for static files and API routes
    '/((?!api|_next|_vercel|images|favicon.ico|.*\\..*).*)',
  ],
}
