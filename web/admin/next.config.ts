import type { NextConfig } from 'next'
import { PHASE_PRODUCTION_BUILD } from 'next/constants'
import createNextIntlPlugin from 'next-intl/plugin'

const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts')

// NEXT_PUBLIC_* values are inlined at build time; a production build without
// them would silently ship localhost fallbacks. Fail the build instead. They
// are the default, not the last word: a deployment overrides them at runtime
// with LINKTOR_ADMIN_* (see src/lib/runtime-config.ts), which is what lets one
// published image serve both the SaaS and an on-prem install.
//
// The check is scoped to the build phase on purpose. `next start` re-reads this
// file, and the runner stage of the image carries no NEXT_PUBLIC_* — demanding
// them there would force every deployment to repeat build-time values in its
// environment just to let the container boot.
function assertBuildTimeEnv() {
  for (const required of ['NEXT_PUBLIC_API_URL', 'NEXT_PUBLIC_WS_URL']) {
    if (!process.env[required]) {
      throw new Error(
        `${required} must be set for production builds (see .env.example)`
      )
    }
  }
}

const apiOrigin =
  process.env.NEXT_PUBLIC_API_URL?.replace(/\/api\/v1\/?$/, '') ||
  process.env.NEXT_PROXY_API_ORIGIN ||
  'http://localhost:8081'

const nextConfig: NextConfig = {
  output: 'standalone',
  reactStrictMode: true,
  reactCompiler: true,
  allowedDevOrigins: ['http://localhost:3001'],
  // API proxy para o backend Go
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: `${apiOrigin}/api/:path*`,
      },
    ]
  },
}

export default function config(phase: string): NextConfig {
  if (phase === PHASE_PRODUCTION_BUILD) {
    assertBuildTimeEnv()
  }
  return withNextIntl(nextConfig)
}
