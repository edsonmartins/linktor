'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { Radio, Database, AlertTriangle } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ChannelLogsViewer } from './components/channel-logs-viewer'
import { MessageQueueMonitor } from './components/message-queue-monitor'
import { SystemStatistics } from './components/system-statistics'

export default function ObservabilityPage() {
  const t = useTranslations('observability')
  const [activeTab, setActiveTab] = useState('logs')

  return (
    <div className="flex flex-col h-full">
      <Header title={t('title')} />

      <div className="p-6 space-y-6 overflow-auto">
        {/* Tabs */}
        <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
          <TabsList>
            <TabsTrigger value="logs" className="gap-2">
              <AlertTriangle className="h-4 w-4" />
              {t('logs')}
            </TabsTrigger>
            <TabsTrigger value="queue" className="gap-2">
              <Database className="h-4 w-4" />
              {t('messageQueues')}
            </TabsTrigger>
            <TabsTrigger value="stats" className="gap-2">
              <Radio className="h-4 w-4" />
              {t('statistics')}
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
    </div>
  )
}
