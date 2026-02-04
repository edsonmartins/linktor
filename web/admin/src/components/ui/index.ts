/**
 * UI Components - Plugin Design Pattern
 *
 * All components follow these principles:
 * 1. Variant Adapters - Different visual implementations via variants
 * 2. Compound Components - Composable parts (Header, Content, Footer)
 * 3. Slot Pattern - asChild prop for composition
 * 4. Provider Pattern - Global providers for shared state
 */

// Primitives
export { Button, buttonVariants } from './button'
export { Input, inputVariants } from './input'
export { Label } from './label'
export { Separator } from './separator'

// Containers
export {
  Card,
  CardHeader,
  CardFooter,
  CardTitle,
  CardDescription,
  CardContent,
  cardVariants,
} from './card'

// Feedback
export { Badge, badgeVariants } from './badge'
export { Avatar, avatarVariants } from './avatar'
export { Spinner, FullPageSpinner, InlineSpinner, spinnerVariants } from './spinner'
export { Skeleton, SkeletonCard, SkeletonMessage, SkeletonList, SkeletonTable } from './skeleton'

// Overlays
export {
  Dialog,
  DialogPortal,
  DialogOverlay,
  DialogClose,
  DialogTrigger,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
} from './dialog'

export {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuCheckboxItem,
  DropdownMenuRadioItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuGroup,
  DropdownMenuPortal,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuRadioGroup,
} from './dropdown-menu'

export {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
  TooltipProvider,
  SimpleTooltip,
} from './tooltip'

// Toast (needs separate import for hook)
export {
  Toast,
  ToastAction,
  ToastClose,
  ToastDescription,
  ToastProvider,
  ToastTitle,
  ToastViewport,
} from './toast'
export { Toaster } from './toaster'

// Layout
export { ScrollArea, ScrollBar } from './scroll-area'
