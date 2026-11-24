'use client'

import * as React from 'react'
import * as AvatarPrimitive from '@radix-ui/react-avatar'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn, getInitials } from '@/lib/utils'

/**
 * Avatar variants - Plugin Pattern adapters for sizing
 */
const avatarVariants = cva(
  'relative flex shrink-0 overflow-hidden rounded-full',
  {
    variants: {
      size: {
        default: 'h-10 w-10',
        xs: 'h-6 w-6',
        sm: 'h-8 w-8',
        lg: 'h-12 w-12',
        xl: 'h-16 w-16',
      },
    },
    defaultVariants: {
      size: 'default',
    },
  }
)

const avatarFallbackVariants = cva(
  'flex h-full w-full items-center justify-center rounded-full font-semibold',
  {
    variants: {
      size: {
        default: 'text-sm',
        xs: 'text-[10px]',
        sm: 'text-xs',
        lg: 'text-base',
        xl: 'text-lg',
      },
    },
    defaultVariants: {
      size: 'default',
    },
  }
)

/** Soft tint + readable ink per person, stable for the same name. */
const avatarTones = [
  'bg-lime-500/20 text-lime-700 dark:text-lime-300',
  'bg-teal-500/20 text-teal-700 dark:text-teal-300',
  'bg-sky-500/20 text-sky-700 dark:text-sky-300',
  'bg-violet-500/20 text-violet-700 dark:text-violet-300',
  'bg-amber-500/20 text-amber-700 dark:text-amber-300',
  'bg-rose-500/20 text-rose-700 dark:text-rose-300',
]

function avatarTone(name?: string) {
  if (!name) return 'bg-muted text-muted-foreground'
  let hash = 0
  for (const char of name) hash = (hash * 31 + char.charCodeAt(0)) >>> 0
  return avatarTones[hash % avatarTones.length]
}

export interface AvatarProps
  extends React.ComponentPropsWithoutRef<typeof AvatarPrimitive.Root>,
    VariantProps<typeof avatarVariants> {
  src?: string
  alt?: string
  fallback?: string
  status?: 'online' | 'offline' | 'busy' | 'away'
}

/**
 * Avatar Component - Compound Component Pattern (Plugin Design)
 * - Composes Radix primitives with custom styling
 * - Auto-generates initials fallback
 * - Status indicator slot
 */
const Avatar = React.forwardRef<
  React.ElementRef<typeof AvatarPrimitive.Root>,
  AvatarProps
>(({ className, size, src, alt, fallback, status, ...props }, ref) => (
  <div className="relative inline-flex">
    <AvatarPrimitive.Root
      ref={ref}
      className={cn(avatarVariants({ size }), className)}
      {...props}
    >
      <AvatarPrimitive.Image
        src={src}
        alt={alt}
        className="aspect-square h-full w-full object-cover"
      />
      <AvatarPrimitive.Fallback
        className={cn(avatarFallbackVariants({ size }), avatarTone(fallback || alt))}
      >
        {fallback ? getInitials(fallback) : alt ? getInitials(alt) : '?'}
      </AvatarPrimitive.Fallback>
    </AvatarPrimitive.Root>
    {status && (
      <span
        className={cn(
          'absolute bottom-0 right-0 rounded-full border-2 border-background',
          size === 'xs' && 'h-1.5 w-1.5',
          size === 'sm' && 'h-2 w-2',
          (!size || size === 'default') && 'h-2.5 w-2.5',
          size === 'lg' && 'h-3 w-3',
          size === 'xl' && 'h-4 w-4',
          status === 'online' && 'bg-terminal-green',
          status === 'offline' && 'bg-muted-foreground',
          status === 'busy' && 'bg-terminal-coral',
          status === 'away' && 'bg-terminal-yellow'
        )}
      />
    )}
  </div>
))
Avatar.displayName = 'Avatar'

export { Avatar, avatarVariants }
