'use client'

import { BookOpen, Play, Terminal, Sparkles, ArrowRight } from 'lucide-react'

const docSections = [
  {
    title: 'Getting Started',
    description: 'Quick start guides and installation instructions',
    href: 'https://docs.linktor.io/getting-started/installation',
  },
  {
    title: 'API Reference',
    description: 'Complete REST API documentation with examples',
    href: 'https://docs.linktor.io/api/overview',
  },
  {
    title: 'SDKs',
    description: 'Client libraries for 7 programming languages',
    href: 'https://docs.linktor.io/sdks/overview',
  },
  {
    title: 'Self-Hosting',
    description: 'Deploy with Docker or Kubernetes',
    href: 'https://docs.linktor.io/self-hosting/docker',
  },
]

export function Docs() {
  return (
    <section id="docs" className="py-24 relative overflow-hidden">
      {/* Background */}
      <div className="absolute inset-0 bg-gradient-to-b from-background via-primary/5 to-background" />
      <div className="absolute inset-0 grid-pattern opacity-10" />

      <div className="relative z-10 mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="text-center mb-16">
          <span className="inline-block px-3 py-1 text-sm font-mono text-primary border border-primary/30 rounded-full bg-primary/10 mb-4">
            Documentation
          </span>
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            Interactive docs with
            <span className="text-gradient"> MCP Playground</span>
          </h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            Explore our comprehensive documentation and test the API directly in your browser with our interactive MCP Playground.
          </p>
        </div>

        {/* MCP Playground Feature Card */}
        <div className="mb-16">
          <div className="terminal-card p-8 lg:p-12 hover-glow transition-all duration-300 relative overflow-hidden">
            {/* Gradient overlay */}
            <div className="absolute top-0 right-0 w-1/2 h-full bg-gradient-to-l from-primary/10 to-transparent pointer-events-none" />

            <div className="grid lg:grid-cols-2 gap-8 items-center">
              {/* Left: Content */}
              <div className="relative z-10">
                <div className="inline-flex items-center gap-2 px-3 py-1 text-sm font-mono text-terminal-green border border-terminal-green/30 rounded-full bg-terminal-green/10 mb-4">
                  <Sparkles className="h-4 w-4" />
                  New Feature
                </div>
                <h3 className="text-2xl sm:text-3xl font-bold mb-4">
                  MCP Playground
                </h3>
                <p className="text-muted-foreground mb-6">
                  Test Linktor&apos;s 30+ MCP tools directly in your browser. No code required.
                  The playground automatically discovers all available tools, generates dynamic
                  forms from their schemas, and executes calls in real-time.
                </p>

                <ul className="space-y-3 mb-8">
                  {[
                    'Auto-discovery of tools, resources, and prompts',
                    'Dynamic forms generated from JSON Schema',
                    'Real-time execution with JSON-RPC 2.0',
                    'Response visualization with timing metrics',
                  ].map((item, i) => (
                    <li key={i} className="flex items-center gap-3 text-sm">
                      <div className="h-1.5 w-1.5 rounded-full bg-terminal-green" />
                      {item}
                    </li>
                  ))}
                </ul>

                <div className="flex flex-wrap gap-4">
                  <a
                    href="https://docs.linktor.io/mcp/playground"
                    className="inline-flex items-center gap-2 px-6 py-3 bg-primary text-primary-foreground rounded-lg font-semibold hover:bg-primary/90 transition-colors"
                  >
                    <Play className="h-4 w-4" />
                    Try Playground
                  </a>
                  <a
                    href="https://docs.linktor.io/mcp/overview"
                    className="inline-flex items-center gap-2 px-6 py-3 border border-border rounded-lg font-semibold hover:bg-accent/10 transition-colors"
                  >
                    <BookOpen className="h-4 w-4" />
                    MCP Docs
                  </a>
                </div>
              </div>

              {/* Right: Terminal Preview */}
              <div className="relative">
                <div className="terminal-card overflow-hidden">
                  {/* Terminal Header */}
                  <div className="flex items-center gap-2 px-4 py-3 border-b border-border bg-card/50">
                    <div className="flex gap-1.5">
                      <div className="h-3 w-3 rounded-full bg-terminal-coral/60" />
                      <div className="h-3 w-3 rounded-full bg-terminal-yellow/60" />
                      <div className="h-3 w-3 rounded-full bg-terminal-green/60" />
                    </div>
                    <span className="text-xs text-muted-foreground font-mono ml-2">mcp-playground</span>
                  </div>

                  {/* Terminal Content */}
                  <div className="p-4 font-mono text-sm space-y-3">
                    <div className="flex items-center gap-2">
                      <Terminal className="h-4 w-4 text-primary" />
                      <span className="text-muted-foreground">Connected to</span>
                      <span className="text-terminal-green">linktor-mcp-server</span>
                    </div>

                    <div className="border-t border-border/50 pt-3">
                      <div className="text-muted-foreground text-xs mb-2">Available Tools (30)</div>
                      <div className="flex flex-wrap gap-2">
                        {['list_conversations', 'send_message', 'get_contact', 'search_knowledge'].map((tool) => (
                          <span key={tool} className="px-2 py-1 text-xs bg-primary/10 text-primary rounded border border-primary/20">
                            {tool}
                          </span>
                        ))}
                        <span className="px-2 py-1 text-xs text-muted-foreground">+26 more</span>
                      </div>
                    </div>

                    <div className="border-t border-border/50 pt-3">
                      <div className="text-muted-foreground text-xs mb-2">Execute: list_conversations</div>
                      <pre className="text-xs text-terminal-green bg-background/50 p-2 rounded overflow-x-auto">
{`{
  "result": {
    "conversations": [
      { "id": "cv_abc", "status": "open" },
      { "id": "cv_def", "status": "open" }
    ]
  }
}`}
                      </pre>
                      <div className="text-xs text-muted-foreground mt-1">Completed in 142ms</div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Documentation Links Grid */}
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {docSections.map((section, index) => (
            <a
              key={section.title}
              href={section.href}
              className="terminal-card p-5 hover-glow transition-all duration-300 group opacity-0 animate-fade-in"
              style={{ animationDelay: `${index * 100}ms`, animationFillMode: 'forwards' }}
            >
              <div className="flex items-start justify-between mb-3">
                <BookOpen className="h-5 w-5 text-primary" />
                <ArrowRight className="h-4 w-4 text-muted-foreground group-hover:text-primary group-hover:translate-x-1 transition-all" />
              </div>
              <h3 className="font-semibold mb-1">{section.title}</h3>
              <p className="text-sm text-muted-foreground">{section.description}</p>
            </a>
          ))}
        </div>
      </div>
    </section>
  )
}
