/**
 * WebSocket Client Hook
 * Real-time connection for admin panel
 */

import { useEffect, useRef, useCallback, useState } from 'react'
import { tokenStorage } from '@/lib/api'

const WS_BASE_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8081/api/v1/ws'

// WebSocket event types (match backend)
export const WSEventTypes = {
  NEW_MESSAGE: 'new_message',
  MESSAGE_UPDATED: 'message_updated',
  CONVERSATION_UPDATED: 'conversation_updated',
  CONVERSATION_CREATED: 'conversation_created',
  TYPING: 'typing',
  PRESENCE: 'presence',
  ERROR: 'error',
  CONNECTED: 'connected',
} as const

export type WSEventType = typeof WSEventTypes[keyof typeof WSEventTypes]

// Payload types
export interface WSMessage<T = unknown> {
  type: WSEventType
  payload: T
}

export interface WSNewMessagePayload {
  conversation_id: string
  message: {
    id: string
    content: string
    content_type: string
    sender_type: string
    sender_id: string
    created_at: string
  }
}

export interface WSTypingPayload {
  conversation_id: string
  user_id: string
  user_name: string
  is_typing: boolean
}

export interface WSPresencePayload {
  user_id: string
  status: 'online' | 'offline' | 'away'
  last_seen?: string
}

export interface WSConnectedPayload {
  user_id: string
  tenant_id: string
  online_users: string[]
}

export interface WSConversationUpdatedPayload {
  id: string
  status?: string
  assigned_to?: string
  last_message_at?: string
  unread_count?: number
}

// Connection states
export type ConnectionState = 'connecting' | 'connected' | 'disconnected' | 'reconnecting'

// Event handlers
export type WSEventHandler<T = unknown> = (payload: T) => void

interface UseWebSocketOptions {
  autoConnect?: boolean
  reconnectAttempts?: number
  reconnectInterval?: number
  onConnect?: () => void
  onDisconnect?: () => void
  onError?: (error: Event) => void
}

interface UseWebSocketReturn {
  connectionState: ConnectionState
  onlineUsers: string[]
  connect: () => void
  disconnect: () => void
  sendTyping: (conversationId: string, isTyping: boolean) => void
  subscribe: <T>(event: WSEventType, handler: WSEventHandler<T>) => () => void
}

export function useWebSocket(options: UseWebSocketOptions = {}): UseWebSocketReturn {
  const {
    autoConnect = true,
    reconnectAttempts = 5,
    reconnectInterval = 3000,
    onConnect,
    onDisconnect,
    onError,
  } = options

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectCountRef = useRef(0)
  const reconnectTimerRef = useRef<NodeJS.Timeout | null>(null)
  const handlersRef = useRef<Map<WSEventType, Set<WSEventHandler>>>(new Map())

  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected')
  const [onlineUsers, setOnlineUsers] = useState<string[]>([])

  // Subscribe to events
  const subscribe = useCallback(<T,>(event: WSEventType, handler: WSEventHandler<T>) => {
    if (!handlersRef.current.has(event)) {
      handlersRef.current.set(event, new Set())
    }
    handlersRef.current.get(event)!.add(handler as WSEventHandler)

    // Return unsubscribe function
    return () => {
      handlersRef.current.get(event)?.delete(handler as WSEventHandler)
    }
  }, [])

  // Emit event to handlers
  const emit = useCallback((event: WSEventType, payload: unknown) => {
    const handlers = handlersRef.current.get(event)
    if (handlers) {
      handlers.forEach((handler) => handler(payload))
    }
  }, [])

  // Send message
  const send = useCallback((message: WSMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message))
    }
  }, [])

  // Send typing indicator
  const sendTyping = useCallback((conversationId: string, isTyping: boolean) => {
    send({
      type: WSEventTypes.TYPING,
      payload: {
        conversation_id: conversationId,
        is_typing: isTyping,
      },
    })
  }, [send])

  // Connect to WebSocket
  const connect = useCallback(() => {
    const token = tokenStorage.getAccessToken()
    if (!token) {
      console.warn('[WebSocket] No auth token available')
      return
    }

    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    setConnectionState('connecting')
    const url = `${WS_BASE_URL}?token=${encodeURIComponent(token)}`

    try {
      wsRef.current = new WebSocket(url)

      wsRef.current.onopen = () => {
        console.log('[WebSocket] Connected')
        setConnectionState('connected')
        reconnectCountRef.current = 0
        onConnect?.()
      }

      wsRef.current.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data)

          // Handle connected event
          if (message.type === WSEventTypes.CONNECTED) {
            const payload = message.payload as WSConnectedPayload
            setOnlineUsers(payload.online_users || [])
          }

          // Handle presence updates
          if (message.type === WSEventTypes.PRESENCE) {
            const payload = message.payload as WSPresencePayload
            setOnlineUsers((prev) => {
              if (payload.status === 'online') {
                return prev.includes(payload.user_id) ? prev : [...prev, payload.user_id]
              } else {
                return prev.filter((id) => id !== payload.user_id)
              }
            })
          }

          // Emit to subscribers
          emit(message.type, message.payload)
        } catch (err) {
          console.error('[WebSocket] Failed to parse message:', err)
        }
      }

      wsRef.current.onclose = () => {
        console.log('[WebSocket] Disconnected')
        setConnectionState('disconnected')
        onDisconnect?.()

        // Attempt reconnect
        if (reconnectCountRef.current < reconnectAttempts) {
          setConnectionState('reconnecting')
          reconnectCountRef.current++
          console.log(`[WebSocket] Reconnecting (${reconnectCountRef.current}/${reconnectAttempts})...`)

          reconnectTimerRef.current = setTimeout(() => {
            connect()
          }, reconnectInterval)
        }
      }

      wsRef.current.onerror = (error) => {
        // Only log error if we have a token (user is authenticated)
        const token = tokenStorage.getAccessToken()
        if (token) {
          console.warn('[WebSocket] Connection error')
          onError?.(error)
        }
      }
    } catch (err) {
      console.error('[WebSocket] Failed to connect:', err)
      setConnectionState('disconnected')
    }
  }, [reconnectAttempts, reconnectInterval, onConnect, onDisconnect, onError, emit])

  // Disconnect from WebSocket
  const disconnect = useCallback(() => {
    if (reconnectTimerRef.current) {
      clearTimeout(reconnectTimerRef.current)
      reconnectTimerRef.current = null
    }
    reconnectCountRef.current = reconnectAttempts // Prevent reconnect

    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }

    setConnectionState('disconnected')
  }, [reconnectAttempts])

  // Auto connect on mount (only if token exists)
  useEffect(() => {
    if (autoConnect) {
      const token = tokenStorage.getAccessToken()
      if (token) {
        connect()
      }
    }

    return () => {
      disconnect()
    }
  }, [autoConnect, connect, disconnect])

  return {
    connectionState,
    onlineUsers,
    connect,
    disconnect,
    sendTyping,
    subscribe,
  }
}

// Context for sharing WebSocket across components
import { createContext, useContext, type ReactNode } from 'react'

interface WebSocketContextValue extends UseWebSocketReturn {}

const WebSocketContext = createContext<WebSocketContextValue | null>(null)

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const ws = useWebSocket()

  return (
    <WebSocketContext.Provider value={ws}>
      {children}
    </WebSocketContext.Provider>
  )
}

export function useWebSocketContext(): WebSocketContextValue {
  const context = useContext(WebSocketContext)
  if (!context) {
    throw new Error('useWebSocketContext must be used within a WebSocketProvider')
  }
  return context
}
