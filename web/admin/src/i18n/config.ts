export const locales = ['pt-BR', 'es', 'en'] as const
export const defaultLocale = 'pt-BR' as const

export type Locale = (typeof locales)[number]

export function isValidLocale(locale: string): locale is Locale {
  return locales.includes(locale as Locale)
}
