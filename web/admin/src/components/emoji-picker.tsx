'use client'

import { useEffect, useRef } from 'react'

interface EmojiPickerProps {
  /** Called with the selected emoji's native character (e.g. "😀"). */
  onSelect: (emoji: string) => void
}

/**
 * Thin wrapper around emoji-mart's framework-agnostic Picker. The official React
 * wrapper (@emoji-mart/react) only declares peer support up to React 18, so we
 * mount the vanilla custom element ourselves — which also keeps the (heavy)
 * picker + data out of the main bundle via dynamic import.
 */
export function EmojiPicker({ onSelect }: EmojiPickerProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  // Keep the latest callback without re-instantiating the picker on each render.
  const onSelectRef = useRef(onSelect)
  onSelectRef.current = onSelect

  useEffect(() => {
    let cancelled = false
    const container = containerRef.current

    ;(async () => {
      const [{ Picker }, data] = await Promise.all([
        import('emoji-mart'),
        import('@emoji-mart/data'),
      ])
      if (cancelled || !container) return

      const picker = new Picker({
        data: (data as { default: unknown }).default ?? data,
        onEmojiSelect: (emoji: { native?: string }) => {
          if (emoji?.native) onSelectRef.current(emoji.native)
        },
        theme: 'dark',
        previewPosition: 'none',
        skinTonePosition: 'search',
        navPosition: 'bottom',
      })
      container.appendChild(picker as unknown as Node)
    })()

    return () => {
      cancelled = true
      if (container) container.innerHTML = ''
    }
  }, [])

  return <div ref={containerRef} />
}
