'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { ArrowRight, Play } from 'lucide-react'

const terminalLines = [
  { prefix: '$', text: 'linktor init --project my-chatbot', delay: 0 },
  { prefix: '>', text: 'Connecting to WhatsApp...', delay: 800, color: 'text-muted-foreground' },
  { prefix: '>', text: 'Connecting to Telegram...', delay: 1200, color: 'text-muted-foreground' },
  { prefix: '>', text: 'Loading AI models...', delay: 1600, color: 'text-muted-foreground' },
  { prefix: '✓', text: 'All channels connected!', delay: 2200, color: 'text-terminal-green' },
  { prefix: '✓', text: 'Bot is ready to receive messages', delay: 2600, color: 'text-terminal-green' },
]

export function Hero() {
  const [visibleLines, setVisibleLines] = useState<number[]>([])

  useEffect(() => {
    terminalLines.forEach((line, index) => {
      setTimeout(() => {
        setVisibleLines((prev) => [...prev, index])
      }, line.delay)
    })
  }, [])

  return (
    <section className="relative min-h-screen flex items-center justify-center pt-16 overflow-hidden">
      {/* Background Effects */}
      <div className="absolute inset-0 grid-pattern opacity-30" />
      <div className="absolute top-1/4 left-1/4 w-96 h-96 bg-primary/10 rounded-full blur-3xl" />
      <div className="absolute bottom-1/4 right-1/4 w-96 h-96 bg-accent/10 rounded-full blur-3xl" />

      <div className="relative z-10 mx-auto max-w-7xl px-4 sm:px-6 lg:px-8 py-20">
        <div className="grid lg:grid-cols-2 gap-12 items-center">
          {/* Left: Content */}
          <div className="text-center lg:text-left">
            <div className="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-primary/30 bg-primary/10 text-primary text-sm font-mono mb-6 opacity-0 animate-fade-in">
              <span className="relative flex h-2 w-2">
                <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-primary opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-primary"></span>
              </span>
              Open Source Omnichannel Platform
            </div>

            <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold tracking-tight mb-6 opacity-0 animate-fade-in delay-100">
              <span className="text-foreground">Omnichannel</span>
              <br />
              <span className="text-foreground">Conversations,</span>
              <br />
              <span className="text-gradient">Powered by AI</span>
            </h1>

            <p className="text-lg sm:text-xl text-muted-foreground max-w-xl mx-auto lg:mx-0 mb-8 opacity-0 animate-fade-in delay-200">
              Connect WhatsApp, Telegram, Email and more. Build AI-powered flows.
              Engage customers everywhere from a single platform.
            </p>

            <div className="flex flex-col sm:flex-row gap-4 justify-center lg:justify-start opacity-0 animate-fade-in delay-300">
              <Link
                href="https://app.linktor.io/register"
                className="inline-flex items-center justify-center gap-2 px-6 py-3 bg-primary text-primary-foreground rounded-lg font-medium hover:bg-primary/90 transition-all hover-glow group"
              >
                Get Started Free
                <ArrowRight className="h-4 w-4 group-hover:translate-x-1 transition-transform" />
              </Link>
              <Link
                href="https://docs.linktor.io"
                className="inline-flex items-center justify-center gap-2 px-6 py-3 border border-border bg-card text-foreground rounded-lg font-medium hover:bg-secondary transition-colors"
              >
                <Play className="h-4 w-4" />
                View Documentation
              </Link>
            </div>

            {/* Stats */}
            <div className="mt-12 grid grid-cols-3 gap-8 opacity-0 animate-fade-in delay-400">
              <div>
                <div className="text-2xl sm:text-3xl font-bold text-primary font-mono">10+</div>
                <div className="text-sm text-muted-foreground">Channels</div>
              </div>
              <div>
                <div className="text-2xl sm:text-3xl font-bold text-accent font-mono">7</div>
                <div className="text-sm text-muted-foreground">SDKs</div>
              </div>
              <div>
                <div className="text-2xl sm:text-3xl font-bold text-terminal-yellow font-mono">100%</div>
                <div className="text-sm text-muted-foreground">Open Source</div>
              </div>
            </div>
          </div>

          {/* Right: Terminal Animation */}
          <div className="opacity-0 animate-slide-up delay-200">
            <div className="terminal-card rounded-xl overflow-hidden glow-primary">
              {/* Terminal Header */}
              <div className="flex items-center gap-2 px-4 py-3 bg-card border-b border-border">
                <div className="flex gap-2">
                  <div className="w-3 h-3 rounded-full bg-terminal-coral" />
                  <div className="w-3 h-3 rounded-full bg-terminal-yellow" />
                  <div className="w-3 h-3 rounded-full bg-terminal-green" />
                </div>
                <span className="text-xs text-muted-foreground font-mono ml-2">linktor-cli</span>
              </div>

              {/* Terminal Content */}
              <div className="p-4 font-mono text-sm space-y-2 min-h-[200px]">
                {terminalLines.map((line, index) => (
                  <div
                    key={index}
                    className={`flex gap-2 transition-opacity duration-300 ${
                      visibleLines.includes(index) ? 'opacity-100' : 'opacity-0'
                    }`}
                  >
                    <span className={line.prefix === '✓' ? 'text-terminal-green' : 'text-primary'}>
                      {line.prefix}
                    </span>
                    <span className={line.color || 'text-foreground'}>{line.text}</span>
                  </div>
                ))}
                {visibleLines.length === terminalLines.length && (
                  <div className="flex gap-2">
                    <span className="text-primary">$</span>
                    <span className="cursor-blink" />
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
