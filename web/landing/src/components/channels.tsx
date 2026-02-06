'use client'

import {
  MessageCircle,
  Send,
  Phone,
  Mail,
  Globe,
  Instagram,
  Facebook,
  MessageSquare,
  Smartphone,
} from 'lucide-react'

const channels = [
  {
    name: 'WhatsApp',
    description: 'Official Business API',
    icon: MessageCircle,
    color: 'text-[#25D366]',
    bgColor: 'bg-[#25D366]/10',
    borderColor: 'border-[#25D366]/30',
  },
  {
    name: 'Telegram',
    description: 'Bot API Integration',
    icon: Send,
    color: 'text-[#0088cc]',
    bgColor: 'bg-[#0088cc]/10',
    borderColor: 'border-[#0088cc]/30',
  },
  {
    name: 'SMS',
    description: 'Twilio & Vonage',
    icon: Smartphone,
    color: 'text-terminal-yellow',
    bgColor: 'bg-terminal-yellow/10',
    borderColor: 'border-terminal-yellow/30',
  },
  {
    name: 'Email',
    description: 'SMTP & IMAP',
    icon: Mail,
    color: 'text-terminal-coral',
    bgColor: 'bg-terminal-coral/10',
    borderColor: 'border-terminal-coral/30',
  },
  {
    name: 'Voice',
    description: 'Twilio & Vonage',
    icon: Phone,
    color: 'text-terminal-purple',
    bgColor: 'bg-terminal-purple/10',
    borderColor: 'border-terminal-purple/30',
  },
  {
    name: 'WebChat',
    description: 'Embeddable Widget',
    icon: Globe,
    color: 'text-primary',
    bgColor: 'bg-primary/10',
    borderColor: 'border-primary/30',
  },
  {
    name: 'Instagram',
    description: 'Messenger API',
    icon: Instagram,
    color: 'text-[#E4405F]',
    bgColor: 'bg-[#E4405F]/10',
    borderColor: 'border-[#E4405F]/30',
  },
  {
    name: 'Facebook',
    description: 'Messenger Platform',
    icon: Facebook,
    color: 'text-[#1877F2]',
    bgColor: 'bg-[#1877F2]/10',
    borderColor: 'border-[#1877F2]/30',
  },
  {
    name: 'RCS',
    description: 'Rich Business Messaging',
    icon: MessageSquare,
    color: 'text-accent',
    bgColor: 'bg-accent/10',
    borderColor: 'border-accent/30',
  },
]

export function Channels() {
  return (
    <section id="channels" className="py-24 bg-card/50 relative">
      <div className="relative z-10 mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="text-center mb-16">
          <span className="inline-block px-3 py-1 text-sm font-mono text-accent border border-accent/30 rounded-full bg-accent/10 mb-4">
            Channels
          </span>
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            Connect with customers
            <span className="text-gradient"> everywhere</span>
          </h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            One inbox for all your messaging channels. Unified conversations across 10+ platforms.
          </p>
        </div>

        {/* Channels Grid */}
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
          {channels.map((channel, index) => (
            <div
              key={channel.name}
              className={`terminal-card p-4 text-center hover-glow transition-all duration-300 group cursor-pointer opacity-0 animate-fade-in`}
              style={{ animationDelay: `${index * 50}ms`, animationFillMode: 'forwards' }}
            >
              <div className={`inline-flex p-3 rounded-lg ${channel.bgColor} border ${channel.borderColor} mb-3 group-hover:scale-110 transition-transform`}>
                <channel.icon className={`h-6 w-6 ${channel.color}`} />
              </div>
              <h3 className="font-semibold text-sm mb-1">{channel.name}</h3>
              <p className="text-xs text-muted-foreground">{channel.description}</p>
            </div>
          ))}
        </div>

        {/* Visual Connection Line */}
        <div className="mt-16 relative">
          <div className="absolute left-1/2 -translate-x-1/2 w-px h-16 bg-gradient-to-b from-primary/50 to-transparent" />
          <div className="text-center pt-20">
            <div className="terminal-card inline-block px-8 py-4 glow-primary">
              <div className="flex items-center gap-3">
                <div className="w-3 h-3 rounded-full bg-primary animate-pulse" />
                <span className="font-mono text-sm">
                  <span className="text-primary">linktor</span>
                  <span className="text-muted-foreground">.unified()</span>
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
