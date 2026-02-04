'use client'

import { useState, useRef, useEffect } from 'react'
import { useMutation } from '@tanstack/react-query'
import { X, Send, RotateCcw, Bot, User } from 'lucide-react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { api } from '@/lib/api'
import type { FlowTestResult, QuickReply } from '@/types'

interface FlowTestPanelProps {
  flowId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface ChatMessage {
  id: string
  type: 'bot' | 'user'
  content: string
  quickReplies?: QuickReply[]
  nodeId?: string
  timestamp: Date
}

export function FlowTestPanel({ flowId, open, onOpenChange }: FlowTestPanelProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [inputValue, setInputValue] = useState('')
  const [currentNodeId, setCurrentNodeId] = useState<string | null>(null)
  const [flowEnded, setFlowEnded] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  // Test mutation
  const testMutation = useMutation({
    mutationFn: (inputs: string[]) =>
      api.post<{ results: FlowTestResult[] }>(`/flows/${flowId}/test`, { inputs }),
    onSuccess: (data) => {
      const results = data.results || []
      const newMessages: ChatMessage[] = []

      results.forEach((result) => {
        if (result.content) {
          newMessages.push({
            id: `bot-${Date.now()}-${Math.random()}`,
            type: 'bot',
            content: result.content,
            quickReplies: result.quick_replies,
            nodeId: result.node_id,
            timestamp: new Date(),
          })
        }

        setCurrentNodeId(result.next_node_id || null)
        setFlowEnded(result.flow_ended)
      })

      setMessages((prev) => [...prev, ...newMessages])
    },
  })

  // Auto-scroll to bottom
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages])

  // Start flow when panel opens
  useEffect(() => {
    if (open && messages.length === 0) {
      handleReset()
    }
  }, [open])

  const handleSend = (value?: string) => {
    const message = value || inputValue.trim()
    if (!message || flowEnded) return

    // Add user message
    setMessages((prev) => [
      ...prev,
      {
        id: `user-${Date.now()}`,
        type: 'user',
        content: message,
        timestamp: new Date(),
      },
    ])

    setInputValue('')

    // Send to API
    testMutation.mutate([message])
  }

  const handleQuickReply = (reply: QuickReply) => {
    handleSend(reply.value || reply.title)
  }

  const handleReset = () => {
    setMessages([])
    setCurrentNodeId(null)
    setFlowEnded(false)
    // Start flow with empty input to get first node
    testMutation.mutate([''])
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-[400px] sm:w-[450px] flex flex-col p-0">
        <SheetHeader className="p-4 border-b">
          <div className="flex items-center justify-between">
            <SheetTitle className="flex items-center gap-2">
              <Bot className="h-5 w-5" />
              Test Flow
            </SheetTitle>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={handleReset}>
                <RotateCcw className="h-3 w-3 mr-1" />
                Reset
              </Button>
            </div>
          </div>
          {currentNodeId && (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>Current node:</span>
              <code className="bg-muted px-1.5 py-0.5 rounded text-xs">
                {currentNodeId}
              </code>
            </div>
          )}
        </SheetHeader>

        {/* Chat Messages */}
        <ScrollArea className="flex-1 p-4" ref={scrollRef}>
          <div className="space-y-4">
            {messages.map((message) => (
              <div
                key={message.id}
                className={`flex ${message.type === 'user' ? 'justify-end' : 'justify-start'}`}
              >
                <div
                  className={`max-w-[80%] rounded-lg p-3 ${
                    message.type === 'user'
                      ? 'bg-primary text-primary-foreground'
                      : 'bg-muted'
                  }`}
                >
                  <div className="flex items-center gap-2 mb-1">
                    {message.type === 'bot' ? (
                      <Bot className="h-3 w-3" />
                    ) : (
                      <User className="h-3 w-3" />
                    )}
                    <span className="text-xs opacity-70">
                      {message.type === 'bot' ? 'Bot' : 'You'}
                    </span>
                    {message.nodeId && (
                      <Badge variant="outline" className="text-[10px] h-4">
                        {message.nodeId}
                      </Badge>
                    )}
                  </div>
                  <p className="text-sm whitespace-pre-wrap">{message.content}</p>

                  {/* Quick Replies */}
                  {message.quickReplies && message.quickReplies.length > 0 && (
                    <div className="flex flex-wrap gap-2 mt-2">
                      {message.quickReplies.map((reply) => (
                        <Button
                          key={reply.id}
                          variant="secondary"
                          size="sm"
                          className="h-7 text-xs"
                          onClick={() => handleQuickReply(reply)}
                          disabled={flowEnded || testMutation.isPending}
                        >
                          {reply.title}
                        </Button>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            ))}

            {testMutation.isPending && (
              <div className="flex justify-start">
                <div className="bg-muted rounded-lg p-3">
                  <div className="flex items-center gap-2">
                    <Bot className="h-3 w-3" />
                    <span className="text-xs opacity-70">Bot is typing...</span>
                  </div>
                  <div className="flex gap-1 mt-2">
                    <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '0ms' }} />
                    <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '150ms' }} />
                    <span className="w-2 h-2 bg-gray-400 rounded-full animate-bounce" style={{ animationDelay: '300ms' }} />
                  </div>
                </div>
              </div>
            )}

            {flowEnded && (
              <div className="text-center py-4">
                <Badge variant="secondary">Flow ended</Badge>
                <p className="text-sm text-muted-foreground mt-2">
                  The conversation flow has completed.
                </p>
                <Button variant="outline" size="sm" className="mt-2" onClick={handleReset}>
                  <RotateCcw className="h-3 w-3 mr-1" />
                  Start Over
                </Button>
              </div>
            )}
          </div>
        </ScrollArea>

        {/* Input */}
        <div className="p-4 border-t">
          <div className="flex gap-2">
            <Input
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={flowEnded ? 'Flow ended' : 'Type a message...'}
              disabled={flowEnded || testMutation.isPending}
            />
            <Button
              onClick={() => handleSend()}
              disabled={!inputValue.trim() || flowEnded || testMutation.isPending}
            >
              <Send className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </SheetContent>
    </Sheet>
  )
}
