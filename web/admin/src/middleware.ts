import { NextRequest, NextResponse } from 'next/server'

export function middleware(request: NextRequest) {
  // Simply pass through - locale detection is handled in i18n/request.ts
  return NextResponse.next()
}

export const config = {
  matcher: [
    // Match all pathnames except for static files and API routes
    '/((?!api|_next|_vercel|images|favicon.ico|.*\\..*).*)',
  ],
}
