'use client'

import Link from 'next/link'
import { ArrowRight, Github, Book, MessageCircle } from 'lucide-react'

export function CTA() {
  return (
    <section className="py-24 relative overflow-hidden">
      {/* Background Effects */}
      <div className="absolute inset-0 grid-pattern opacity-20" />
      <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] bg-primary/5 rounded-full blur-3xl" />

      <div className="relative z-10 mx-auto max-w-4xl px-4 sm:px-6 lg:px-8">
        <div className="terminal-card rounded-2xl p-8 sm:p-12 text-center border-gradient">
          <div className="bg-card rounded-xl p-8 sm:p-12">
            <span className="inline-block px-3 py-1 text-sm font-mono text-primary border border-primary/30 rounded-full bg-primary/10 mb-6">
              Get Started
            </span>

            <h2 className="text-3xl sm:text-4xl font-bold mb-4">
              Start Building
              <span className="text-gradient"> Today</span>
            </h2>

            <p className="text-lg text-muted-foreground max-w-xl mx-auto mb-8">
              Join thousands of developers building amazing conversational experiences.
              Free to get started, no credit card required.
            </p>

            <div className="flex flex-col sm:flex-row gap-4 justify-center mb-12">
              <Link
                href="https://app.linktor.io/register"
                className="inline-flex items-center justify-center gap-2 px-8 py-4 bg-primary text-primary-foreground rounded-lg font-medium text-lg hover:bg-primary/90 transition-all hover-glow group animate-glow-pulse"
              >
                Get Started Free
                <ArrowRight className="h-5 w-5 group-hover:translate-x-1 transition-transform" />
              </Link>
              <Link
                href="https://docs.linktor.io"
                className="inline-flex items-center justify-center gap-2 px-8 py-4 border border-border bg-card text-foreground rounded-lg font-medium text-lg hover:bg-secondary transition-colors"
              >
                <Book className="h-5 w-5" />
                Read the Docs
              </Link>
            </div>

            {/* Links */}
            <div className="flex flex-wrap justify-center gap-8 text-muted-foreground">
              <Link
                href="https://github.com/linktor/linktor"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-2 hover:text-foreground transition-colors"
              >
                <Github className="h-5 w-5" />
                <span>GitHub</span>
              </Link>
              <Link
                href="https://docs.linktor.io"
                className="flex items-center gap-2 hover:text-foreground transition-colors"
              >
                <Book className="h-5 w-5" />
                <span>Documentation</span>
              </Link>
              <Link
                href="https://discord.gg/linktor"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-2 hover:text-foreground transition-colors"
              >
                <MessageCircle className="h-5 w-5" />
                <span>Discord</span>
              </Link>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
