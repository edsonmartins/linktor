'use client'

import {
  Bot,
  GitBranch,
  BookOpen,
  BarChart3,
  Code2,
  Zap,
  Shield,
  Users,
} from 'lucide-react'

const features = [
  {
    icon: Bot,
    title: 'AI-Powered Bots',
    description: 'Build intelligent chatbots with OpenAI, Anthropic, or custom LLM providers. Natural conversations that understand context.',
    color: 'text-terminal-green',
    bgColor: 'bg-terminal-green/10',
    borderColor: 'border-terminal-green/30',
  },
  {
    icon: GitBranch,
    title: 'Visual Flow Builder',
    description: 'Design conversation flows with drag-and-drop simplicity. Conditions, loops, and integrations made visual.',
    color: 'text-terminal-cyan',
    bgColor: 'bg-terminal-cyan/10',
    borderColor: 'border-terminal-cyan/30',
  },
  {
    icon: BookOpen,
    title: 'Knowledge Base',
    description: 'Train your AI with documents, FAQs, and custom data. Automatic embeddings for semantic search.',
    color: 'text-terminal-yellow',
    bgColor: 'bg-terminal-yellow/10',
    borderColor: 'border-terminal-yellow/30',
  },
  {
    icon: BarChart3,
    title: 'Real-time Analytics',
    description: 'Monitor conversations, track engagement, and measure bot performance with detailed dashboards.',
    color: 'text-terminal-purple',
    bgColor: 'bg-terminal-purple/10',
    borderColor: 'border-terminal-purple/30',
  },
  {
    icon: Code2,
    title: 'Multi-language SDKs',
    description: 'Integrate with TypeScript, Python, Go, Java, Rust, .NET, or PHP. Comprehensive APIs for any stack.',
    color: 'text-primary',
    bgColor: 'bg-primary/10',
    borderColor: 'border-primary/30',
  },
  {
    icon: Zap,
    title: 'Real-time Messaging',
    description: 'WebSocket support for instant message delivery. Type indicators, read receipts, and live updates.',
    color: 'text-accent',
    bgColor: 'bg-accent/10',
    borderColor: 'border-accent/30',
  },
  {
    icon: Shield,
    title: 'Enterprise Security',
    description: 'End-to-end encryption, role-based access control, and compliance-ready infrastructure.',
    color: 'text-terminal-coral',
    bgColor: 'bg-terminal-coral/10',
    borderColor: 'border-terminal-coral/30',
  },
  {
    icon: Users,
    title: 'Human Handoff',
    description: 'Seamlessly escalate conversations to human agents when needed. Smart routing and queuing.',
    color: 'text-terminal-green',
    bgColor: 'bg-terminal-green/10',
    borderColor: 'border-terminal-green/30',
  },
]

export function Features() {
  return (
    <section id="features" className="py-24 relative">
      <div className="absolute inset-0 grid-pattern opacity-20" />

      <div className="relative z-10 mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="text-center mb-16">
          <span className="inline-block px-3 py-1 text-sm font-mono text-primary border border-primary/30 rounded-full bg-primary/10 mb-4">
            Features
          </span>
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            Everything you need for
            <span className="text-gradient"> omnichannel success</span>
          </h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            A complete platform for building, deploying, and scaling conversational experiences across all channels.
          </p>
        </div>

        {/* Features Grid */}
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
          {features.map((feature, index) => (
            <div
              key={feature.title}
              className={`terminal-card p-6 hover-glow transition-all duration-300 group opacity-0 animate-fade-in`}
              style={{ animationDelay: `${index * 100}ms`, animationFillMode: 'forwards' }}
            >
              <div className={`inline-flex p-3 rounded-lg ${feature.bgColor} border ${feature.borderColor} mb-4 group-hover:scale-110 transition-transform`}>
                <feature.icon className={`h-6 w-6 ${feature.color}`} />
              </div>
              <h3 className="text-lg font-semibold mb-2">{feature.title}</h3>
              <p className="text-sm text-muted-foreground">{feature.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
