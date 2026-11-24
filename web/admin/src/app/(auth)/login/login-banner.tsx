import { useTranslations } from 'next-intl'
import {
  BarChart3,
  Bot,
  Globe,
  Inbox,
  Instagram,
  Mail,
  MessageCircle,
  Phone,
  Send,
  Smartphone,
  Workflow,
  type LucideIcon,
} from 'lucide-react'
import styles from './login-banner.module.css'

/**
 * Login banner: an animated picture of what Linktor does — messages come in
 * from every channel, pass through one hub and land in the inbox, the bots,
 * the flows and the reports.
 *
 * It is a single SVG so it scales with the panel. All motion is CSS (see the
 * module) and is switched off under prefers-reduced-motion.
 */

const VIEW = { width: 960, height: 600 }
const HUB = { x: 480, y: 300, radius: 58 }
const NODE = { height: 44, radius: 12 }
const CHANNEL = { x: 36, width: 168 }
const TARGET = { x: 744, width: 180 }

interface DiagramNode {
  icon: LucideIcon
  /** Translation key under auth.banner, or a literal brand name. */
  label: string
  translate?: boolean
  /** Tailwind text colour for the icon and its travelling message. */
  tone: string
}

const channels: DiagramNode[] = [
  { icon: MessageCircle, label: 'WhatsApp', tone: 'text-green-400' },
  { icon: Send, label: 'Telegram', tone: 'text-sky-400' },
  { icon: Instagram, label: 'Instagram', tone: 'text-pink-400' },
  { icon: Mail, label: 'email', translate: true, tone: 'text-amber-400' },
  { icon: Smartphone, label: 'SMS', tone: 'text-violet-400' },
  { icon: Globe, label: 'WebChat', tone: 'text-teal-400' },
  { icon: Phone, label: 'voice', translate: true, tone: 'text-cyan-400' },
]

const targets: DiagramNode[] = [
  { icon: Inbox, label: 'inbox', translate: true, tone: 'text-primary' },
  { icon: Bot, label: 'bots', translate: true, tone: 'text-primary' },
  { icon: Workflow, label: 'flows', translate: true, tone: 'text-primary' },
  { icon: BarChart3, label: 'analytics', translate: true, tone: 'text-primary' },
]

/** Vertical centres, spread evenly around the hub. */
function spread(count: number, gap: number) {
  const first = HUB.y - ((count - 1) * gap) / 2
  return Array.from({ length: count }, (_, index) => first + index * gap)
}

const channelYs = spread(channels.length, 74)
const targetYs = spread(targets.length, 118)

function inboundPath(y: number) {
  const from = CHANNEL.x + CHANNEL.width
  const to = HUB.x - HUB.radius
  return `M ${from} ${y} C ${from + 120} ${y}, ${to - 110} ${HUB.y}, ${to} ${HUB.y}`
}

function outboundPath(y: number) {
  const from = HUB.x + HUB.radius
  const to = TARGET.x
  return `M ${from} ${HUB.y} C ${from + 110} ${HUB.y}, ${to - 120} ${y}, ${to} ${y}`
}

function delay(seconds: number) {
  return { '--delay': `${seconds}s` } as React.CSSProperties
}

function NodeBox({
  node,
  x,
  y,
  width,
  label,
  appearAfter,
}: {
  node: DiagramNode
  x: number
  y: number
  width: number
  label: string
  appearAfter: number
}) {
  const Icon = node.icon
  return (
    <g className={styles.node} style={delay(appearAfter)}>
      <rect
        x={x}
        y={y - NODE.height / 2}
        width={width}
        height={NODE.height}
        rx={NODE.radius}
        className="fill-white/[0.07] stroke-white/15"
      />
      <Icon x={x + 14} y={y - 10} width={20} height={20} className={node.tone} />
      <text x={x + 46} y={y + 5} className="fill-foreground text-[15px] font-medium">
        {label}
      </text>
    </g>
  )
}

export function LoginBanner() {
  const t = useTranslations('auth.banner')
  const labelOf = (node: DiagramNode) => (node.translate ? t(node.label) : node.label)

  return (
    // The brand's stage: the sidebar's deep green, with `dark` pinning the
    // tokens inside it in both themes. The form beside it follows the user's
    // theme.
    <div className="dark relative flex h-full flex-col overflow-hidden text-foreground" style={{ background: 'var(--sb-bg)' }}>
      {/* Backdrop: grid plus two soft lime glows */}
      <div className="absolute inset-0 bg-[linear-gradient(hsl(var(--primary)/0.07)_1px,transparent_1px),linear-gradient(90deg,hsl(var(--primary)/0.07)_1px,transparent_1px)] bg-[size:48px_48px] [mask-image:radial-gradient(ellipse_at_center,black_35%,transparent_85%)]" />
      <div className={`${styles.glow} absolute -left-32 top-1/4 h-[28rem] w-[28rem] rounded-full bg-lime-400/25 blur-3xl`} />
      <div className={`${styles.glow} absolute -right-24 bottom-0 h-96 w-96 rounded-full bg-teal-400/20 blur-3xl`} style={{ animationDelay: '2.5s' }} />

      <div className="relative z-10 px-12 pt-12 xl:px-16 xl:pt-16">
        <p className="terminal-prompt text-sm">{t('eyebrow')}</p>
        <h2 className="mt-3 max-w-2xl text-3xl font-bold leading-tight tracking-tight xl:text-4xl">
          {t('titleStart')}
          <span className="text-gradient block w-fit">{t('titleEnd')}</span>
        </h2>
        <p className="mt-3 max-w-xl text-sm text-muted-foreground xl:text-base">{t('subtitle')}</p>
      </div>

      <div className="relative z-10 flex min-h-0 flex-1 items-center px-6 pb-8 xl:px-10">
        <svg
          viewBox={`0 0 ${VIEW.width} ${VIEW.height}`}
          className="h-full max-h-[640px] w-full font-mono"
          role="img"
          aria-label={t('diagramLabel')}
        >
          {/* Wires */}
          <g fill="none" strokeWidth={1.5} className="stroke-primary/50">
            {channelYs.map((y) => (
              <path key={`in-${y}`} d={inboundPath(y)} className={styles.flow} />
            ))}
            {targetYs.map((y) => (
              <path key={`out-${y}`} d={outboundPath(y)} className={styles.flow} />
            ))}
          </g>

          {/* Messages travelling in: one per channel, in the channel's colour */}
          {channels.map((channel, index) => (
            <circle
              key={`msg-in-${channel.label}`}
              r={4.5}
              fill="currentColor"
              className={`${styles.packet} ${channel.tone}`}
              style={{
                offsetPath: `path('${inboundPath(channelYs[index])}')`,
                ...delay(1 + index * 0.55),
              }}
            />
          ))}

          {/* ...and out, unified */}
          {targets.map((target, index) => (
            <circle
              key={`msg-out-${target.label}`}
              r={4.5}
              fill="currentColor"
              className={`${styles.packet} text-primary`}
              style={{
                offsetPath: `path('${outboundPath(targetYs[index])}')`,
                ...delay(2.4 + index * 0.85),
              }}
            />
          ))}

          {channels.map((channel, index) => (
            <NodeBox
              key={channel.label}
              node={channel}
              x={CHANNEL.x}
              y={channelYs[index]}
              width={CHANNEL.width}
              label={labelOf(channel)}
              appearAfter={0.1 + index * 0.08}
            />
          ))}

          {targets.map((target, index) => (
            <NodeBox
              key={target.label}
              node={target}
              x={TARGET.x}
              y={targetYs[index]}
              width={TARGET.width}
              label={labelOf(target)}
              appearAfter={0.9 + index * 0.1}
            />
          ))}

          {/* Hub */}
          <g className={styles.node} style={delay(0.5)}>
            <circle cx={HUB.x} cy={HUB.y} r={HUB.radius} className={`${styles.ring} fill-none stroke-primary`} strokeWidth={1.5} />
            <circle cx={HUB.x} cy={HUB.y} r={HUB.radius} className={`${styles.ring} fill-none stroke-primary`} strokeWidth={1.5} style={delay(1.6)} />
            <circle cx={HUB.x} cy={HUB.y} r={HUB.radius} className="fill-[#0c2619] stroke-primary" strokeWidth={2} />
            <image
              href="/images/logo_linktor_single.png"
              x={HUB.x - 105}
              y={HUB.y - 105}
              width={210}
              height={210}
            />
            <text x={HUB.x} y={HUB.y + HUB.radius + 28} textAnchor="middle" className="fill-foreground text-[17px] font-semibold">
              Linktor
            </text>
          </g>
        </svg>
      </div>

      <p className="relative z-10 px-12 pb-8 text-xs text-muted-foreground xl:px-16">
        {t('footnote')}
      </p>
    </div>
  )
}
