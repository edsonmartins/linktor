'use client'

import { useState } from 'react'
import { Copy, Check } from 'lucide-react'

const sdks = [
  {
    name: 'TypeScript',
    package: '@linktor/sdk',
    install: 'npm install @linktor/sdk',
    color: 'text-[#3178C6]',
    code: `import { Linktor } from '@linktor/sdk'

const client = new Linktor({
  apiKey: process.env.LINKTOR_API_KEY
})

// Send a message
await client.messages.send({
  channelId: 'whatsapp_123',
  to: '+1234567890',
  content: 'Hello from Linktor!'
})`,
  },
  {
    name: 'Python',
    package: 'linktor',
    install: 'pip install linktor',
    color: 'text-[#3776AB]',
    code: `from linktor import Linktor

client = Linktor(api_key="your_api_key")

# Send a message
client.messages.send(
    channel_id="whatsapp_123",
    to="+1234567890",
    content="Hello from Linktor!"
)`,
  },
  {
    name: 'Go',
    package: 'linktor-go',
    install: 'go get github.com/linktor/linktor-go',
    color: 'text-[#00ADD8]',
    code: `package main

import "github.com/linktor/linktor-go"

func main() {
    client := linktor.New("your_api_key")

    // Send a message
    client.Messages.Send(linktor.SendParams{
        ChannelID: "whatsapp_123",
        To:        "+1234567890",
        Content:   "Hello from Linktor!",
    })
}`,
  },
  {
    name: 'Java',
    package: 'io.linktor:linktor-sdk',
    install: 'maven: io.linktor:linktor-sdk:1.0.0',
    color: 'text-[#ED8B00]',
    code: `import io.linktor.Linktor;
import io.linktor.models.Message;

Linktor client = new Linktor("your_api_key");

// Send a message
client.messages().send(
    Message.builder()
        .channelId("whatsapp_123")
        .to("+1234567890")
        .content("Hello from Linktor!")
        .build()
);`,
  },
  {
    name: 'Rust',
    package: 'linktor',
    install: 'cargo add linktor',
    color: 'text-[#DEA584]',
    code: `use linktor::Client;

#[tokio::main]
async fn main() {
    let client = Client::new("your_api_key");

    // Send a message
    client.messages().send(
        "whatsapp_123",
        "+1234567890",
        "Hello from Linktor!"
    ).await.unwrap();
}`,
  },
  {
    name: '.NET',
    package: 'Linktor.SDK',
    install: 'dotnet add package Linktor.SDK',
    color: 'text-[#512BD4]',
    code: `using Linktor;

var client = new LinktorClient("your_api_key");

// Send a message
await client.Messages.SendAsync(new SendMessageRequest
{
    ChannelId = "whatsapp_123",
    To = "+1234567890",
    Content = "Hello from Linktor!"
});`,
  },
  {
    name: 'PHP',
    package: 'linktor/linktor-php',
    install: 'composer require linktor/linktor-php',
    color: 'text-[#777BB4]',
    code: `<?php
use Linktor\\Client;

$client = new Client('your_api_key');

// Send a message
$client->messages->send([
    'channel_id' => 'whatsapp_123',
    'to' => '+1234567890',
    'content' => 'Hello from Linktor!'
]);`,
  },
]

export function SDKs() {
  const [activeSDK, setActiveSDK] = useState(0)
  const [copied, setCopied] = useState(false)

  const copyCode = () => {
    navigator.clipboard.writeText(sdks[activeSDK].code)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <section id="sdks" className="py-24 relative">
      <div className="absolute inset-0 grid-pattern opacity-20" />

      <div className="relative z-10 mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="text-center mb-16">
          <span className="inline-block px-3 py-1 text-sm font-mono text-terminal-yellow border border-terminal-yellow/30 rounded-full bg-terminal-yellow/10 mb-4">
            SDKs
          </span>
          <h2 className="text-3xl sm:text-4xl font-bold mb-4">
            Integrate with
            <span className="text-gradient"> any stack</span>
          </h2>
          <p className="text-lg text-muted-foreground max-w-2xl mx-auto">
            Official SDKs for 7 programming languages. Well-documented, type-safe, and easy to use.
          </p>
        </div>

        {/* SDK Selector & Code */}
        <div className="terminal-card rounded-xl overflow-hidden glow-primary">
          {/* SDK Tabs */}
          <div className="flex flex-wrap border-b border-border bg-card/50">
            {sdks.map((sdk, index) => (
              <button
                key={sdk.name}
                onClick={() => setActiveSDK(index)}
                className={`px-4 py-3 text-sm font-medium transition-colors relative ${
                  activeSDK === index
                    ? 'text-primary'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                {sdk.name}
                {activeSDK === index && (
                  <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary" />
                )}
              </button>
            ))}
          </div>

          {/* Install Command */}
          <div className="flex items-center justify-between px-4 py-3 bg-card border-b border-border">
            <div className="flex items-center gap-2">
              <span className="text-primary font-mono text-sm">$</span>
              <code className="text-sm font-mono text-foreground">
                {sdks[activeSDK].install}
              </code>
            </div>
            <button
              onClick={copyCode}
              className="p-2 text-muted-foreground hover:text-foreground transition-colors"
            >
              {copied ? (
                <Check className="h-4 w-4 text-terminal-green" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </button>
          </div>

          {/* Code Block */}
          <div className="relative">
            <pre className="p-4 overflow-x-auto text-sm">
              <code className="font-mono">
                {sdks[activeSDK].code.split('\n').map((line, i) => (
                  <div key={i} className="leading-relaxed">
                    <span className="text-muted-foreground select-none mr-4">
                      {String(i + 1).padStart(2, ' ')}
                    </span>
                    {highlightCode(line)}
                  </div>
                ))}
              </code>
            </pre>
          </div>
        </div>

        {/* SDK Cards */}
        <div className="mt-12 grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-4">
          {sdks.map((sdk, index) => (
            <button
              key={sdk.name}
              onClick={() => setActiveSDK(index)}
              className={`terminal-card p-4 text-center transition-all duration-300 ${
                activeSDK === index
                  ? 'border-primary glow-primary'
                  : 'hover-glow'
              }`}
            >
              <span className={`text-lg font-bold ${sdk.color}`}>{sdk.name}</span>
            </button>
          ))}
        </div>
      </div>
    </section>
  )
}

// Simple syntax highlighting
function highlightCode(line: string) {
  const parts: React.ReactNode[] = []
  let remaining = line

  // Keywords
  const keywords = ['import', 'from', 'const', 'await', 'async', 'new', 'package', 'func', 'use', 'let', 'var', 'main', 'return']
  const keywordRegex = new RegExp(`\\b(${keywords.join('|')})\\b`, 'g')

  // Strings
  const stringRegex = /(["'`])(?:(?!\1)[^\\]|\\.)*\1/g

  // Comments
  const commentRegex = /(\/\/.*$|#.*$|\/\*[\s\S]*?\*\/)/g

  // Process line
  let lastIndex = 0
  const matches: { start: number; end: number; content: string; type: string }[] = []

  // Find strings
  let match
  while ((match = stringRegex.exec(line)) !== null) {
    matches.push({ start: match.index, end: match.index + match[0].length, content: match[0], type: 'string' })
  }

  // Find keywords (only if not inside string)
  while ((match = keywordRegex.exec(line)) !== null) {
    const inString = matches.some(m => match!.index >= m.start && match!.index < m.end)
    if (!inString) {
      matches.push({ start: match.index, end: match.index + match[0].length, content: match[0], type: 'keyword' })
    }
  }

  // Sort matches
  matches.sort((a, b) => a.start - b.start)

  // Build output
  for (const m of matches) {
    if (m.start > lastIndex) {
      parts.push(<span key={lastIndex} className="text-foreground">{remaining.slice(lastIndex, m.start)}</span>)
    }
    if (m.type === 'string') {
      parts.push(<span key={m.start} className="text-terminal-green">{m.content}</span>)
    } else if (m.type === 'keyword') {
      parts.push(<span key={m.start} className="text-terminal-purple">{m.content}</span>)
    }
    lastIndex = m.end
  }

  if (lastIndex < remaining.length) {
    parts.push(<span key={lastIndex} className="text-foreground">{remaining.slice(lastIndex)}</span>)
  }

  return parts.length > 0 ? parts : <span className="text-foreground">{line}</span>
}
