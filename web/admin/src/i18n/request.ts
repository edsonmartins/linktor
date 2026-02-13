import { getRequestConfig } from 'next-intl/server'
import { cookies, headers } from 'next/headers'
import { defaultLocale, isValidLocale, type Locale } from './config'

export default getRequestConfig(async () => {
  const cookieStore = await cookies()
  const headersList = await headers()

  // Try to get locale from cookie first
  let locale: Locale = defaultLocale
  const cookieLocale = cookieStore.get('locale')?.value

  if (cookieLocale && isValidLocale(cookieLocale)) {
    locale = cookieLocale
  } else {
    // Try Accept-Language header
    const acceptLanguage = headersList.get('accept-language')
    if (acceptLanguage) {
      const browserLocale = acceptLanguage.split(',')[0].split('-')[0]
      if (browserLocale === 'es') {
        locale = 'es'
      } else if (browserLocale === 'en') {
        locale = 'en'
      }
    }
  }

  return {
    locale,
    messages: (await import(`./locales/${locale}.json`)).default
  }
})
