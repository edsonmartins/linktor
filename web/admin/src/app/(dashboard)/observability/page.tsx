'use client'

import { useState } from 'react'
import { Activity, Radio, Database, AlertTriangle } from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ChannelLogsViewer } from './components/channel-logs-viewer'
import { MessageQueueMonitor } from './components/message-queue-monitor'
import { SystemStatistics } from './components/system-statistics'

export default function ObservabilityPage() {
  const [activeTab, setActiveTab] = useState('logs')

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-semibold flex items-center gap-2">
          <Activity className="h-6 w-6 text-primary" />
          Observability
        </h1>
        <p className="text-sm text-muted-foreground">
          Monitor system health, logs, message queues, and statistics
        </p>
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
        <TabsList>
          <TabsTrigger value="logs" className="gap-2">
            <AlertTriangle className="h-4 w-4" />
            Logs
          </TabsTrigger>
          <TabsTrigger value="queue" className="gap-2">
            <Database className="h-4 w-4" />
            Message Queues
          </TabsTrigger>
          <TabsTrigger value="stats" className="gap-2">
            <Radio className="h-4 w-4" />
            Statistics
          </TabsTrigger>
        </TabsList>

        <TabsContent value="logs" className="space-y-4">
          <ChannelLogsViewer />
        </TabsContent>

        <TabsContent value="queue" className="space-y-4">
          <MessageQueueMonitor />
        </TabsContent>

        <TabsContent value="stats" className="space-y-4">
          <SystemStatistics />
        </TabsContent>
      </Tabs>
    </div>
  )
}
