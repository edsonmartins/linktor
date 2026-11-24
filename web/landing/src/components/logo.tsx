import Image from 'next/image'

/**
 * Linktor brand: the mark plus the wordmark set in type, so one asset works
 * on the light and on the dark theme.
 */
export function Logo() {
  return (
    <span className="flex items-center">
      {/* The mark has wide transparent margins, hence the oversized box. */}
      <Image
        src="/images/logo_linktor_single.png"
        alt=""
        width={1024}
        height={1024}
        className="-mx-4 h-16 w-16 max-w-none shrink-0"
        priority
      />
      <span className="text-xl font-bold tracking-tight text-foreground">Linktor</span>
    </span>
  )
}
