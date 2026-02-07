'use client'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { useState, type ReactNode } from 'react'

/**
 * Query Client Factory
 * Creates a new QueryClient with default options
 */
function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // Stale time of 1 minute for most queries
        staleTime: 60 * 1000,
        // Retry failed requests once
        retry: 1,
        // Refetch on window focus for real-time feel
        refetchOnWindowFocus: true,
      },
      mutations: {
        // Retry mutations once
        retry: 1,
      },
    },
  })
}

let browserQueryClient: QueryClient | undefined = undefined

function getQueryClient() {
  if (typeof window === 'undefined') {
    // Server: always make a new query client
    return makeQueryClient()
  } else {
    // Browser: make a new query client if we don't already have one
    if (!browserQueryClient) browserQueryClient = makeQueryClient()
    return browserQueryClient
  }
}

/**
 * QueryProvider - Wraps app with React Query context
 */
export function QueryProvider({ children }: { children: ReactNode }) {
  const [queryClient] = useState(() => getQueryClient())

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      {process.env.NODE_ENV === 'development' && (
        <ReactQueryDevtools initialIsOpen={false} />
      )}
    </QueryClientProvider>
  )
}

/**
 * Query Keys Factory - Centralized query key management
 * Following the factory pattern for type-safe query keys
 */
export const queryKeys = {
  // Auth
  auth: {
    all: ['auth'] as const,
    user: () => [...queryKeys.auth.all, 'user'] as const,
  },

  // Conversations
  conversations: {
    all: ['conversations'] as const,
    lists: () => [...queryKeys.conversations.all, 'list'] as const,
    list: (filters: Record<string, unknown>) =>
      [...queryKeys.conversations.lists(), filters] as const,
    details: () => [...queryKeys.conversations.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.conversations.details(), id] as const,
  },

  // Messages
  messages: {
    all: ['messages'] as const,
    lists: () => [...queryKeys.messages.all, 'list'] as const,
    list: (conversationId: string) =>
      [...queryKeys.messages.lists(), conversationId] as const,
  },

  // Contacts
  contacts: {
    all: ['contacts'] as const,
    lists: () => [...queryKeys.contacts.all, 'list'] as const,
    list: (filters: Record<string, unknown>) =>
      [...queryKeys.contacts.lists(), filters] as const,
    details: () => [...queryKeys.contacts.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.contacts.details(), id] as const,
  },

  // Channels
  channels: {
    all: ['channels'] as const,
    lists: () => [...queryKeys.channels.all, 'list'] as const,
    list: () => [...queryKeys.channels.lists()] as const,
    details: () => [...queryKeys.channels.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.channels.details(), id] as const,
  },

  // Dashboard
  dashboard: {
    all: ['dashboard'] as const,
    stats: () => [...queryKeys.dashboard.all, 'stats'] as const,
    activity: () => [...queryKeys.dashboard.all, 'activity'] as const,
  },

  // Knowledge Bases
  knowledgeBases: {
    all: ['knowledge-bases'] as const,
    lists: () => [...queryKeys.knowledgeBases.all, 'list'] as const,
    list: (filters: Record<string, unknown>) =>
      [...queryKeys.knowledgeBases.lists(), filters] as const,
    details: () => [...queryKeys.knowledgeBases.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.knowledgeBases.details(), id] as const,
  },

  // Knowledge Items
  knowledgeItems: {
    all: ['knowledge-items'] as const,
    lists: () => [...queryKeys.knowledgeItems.all, 'list'] as const,
    list: (kbId: string, filters: Record<string, unknown>) =>
      [...queryKeys.knowledgeItems.lists(), kbId, filters] as const,
    details: () => [...queryKeys.knowledgeItems.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.knowledgeItems.details(), id] as const,
  },

  // Flows (Conversational Decision Trees)
  flows: {
    all: ['flows'] as const,
    lists: () => [...queryKeys.flows.all, 'list'] as const,
    list: (filters: Record<string, unknown>) =>
      [...queryKeys.flows.lists(), filters] as const,
    details: () => [...queryKeys.flows.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.flows.details(), id] as const,
  },

  // Analytics
  analytics: {
    all: ['analytics'] as const,
    overview: (params: Record<string, unknown>) =>
      [...queryKeys.analytics.all, 'overview', params] as const,
    conversations: (params: Record<string, unknown>) =>
      [...queryKeys.analytics.all, 'conversations', params] as const,
    flows: (params: Record<string, unknown>) =>
      [...queryKeys.analytics.all, 'flows', params] as const,
    escalations: (params: Record<string, unknown>) =>
      [...queryKeys.analytics.all, 'escalations', params] as const,
    channels: (params: Record<string, unknown>) =>
      [...queryKeys.analytics.all, 'channels', params] as const,
  },

  // Bots
  bots: {
    all: ['bots'] as const,
    lists: () => [...queryKeys.bots.all, 'list'] as const,
    list: (filters: Record<string, unknown>) =>
      [...queryKeys.bots.lists(), filters] as const,
    details: () => [...queryKeys.bots.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.bots.details(), id] as const,
  },

  // Users
  users: {
    all: ['users'] as const,
    lists: () => [...queryKeys.users.all, 'list'] as const,
    list: (filters: Record<string, unknown>) =>
      [...queryKeys.users.lists(), filters] as const,
    details: () => [...queryKeys.users.all, 'detail'] as const,
    detail: (id: string) => [...queryKeys.users.details(), id] as const,
  },

  // AI Providers
  ai: {
    all: ['ai'] as const,
    providers: () => [...queryKeys.ai.all, 'providers'] as const,
    models: (provider: string) => [...queryKeys.ai.all, 'models', provider] as const,
  },

  // Observability
  observability: {
    all: ['observability'] as const,
    logs: (filters: Record<string, unknown>) =>
      [...queryKeys.observability.all, 'logs', filters] as const,
    queue: () => [...queryKeys.observability.all, 'queue'] as const,
    streamInfo: (streamName: string) =>
      [...queryKeys.observability.all, 'queue', streamName] as const,
    stats: (period: string) =>
      [...queryKeys.observability.all, 'stats', period] as const,
  },
}

export { getQueryClient }
