'use client'

import { useEffect, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslations } from 'next-intl'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Skeleton } from '@/components/ui/skeleton'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useToast } from '@/hooks/use-toast'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { AssignmentStrategy, TenantSettings, UpdateSettingsRequest } from '@/types'

export default function OperationsPage() {
  const t = useTranslations('operations')
  const { toast } = useToast()
  const queryClient = useQueryClient()

  const { data: settings, isLoading } = useQuery({
    queryKey: queryKeys.tenantSettings.detail(),
    queryFn: () => api.get<TenantSettings>('/settings'),
  })

  const [form, setForm] = useState<UpdateSettingsRequest>({
    assignment_strategy: 'manual',
    auto_assign_enabled: false,
    sla_first_response_minutes: 0,
    sla_resolution_minutes: 0,
    auto_close_after_minutes: 0,
  })

  useEffect(() => {
    if (settings) {
      setForm({
        assignment_strategy: settings.assignment_strategy,
        auto_assign_enabled: settings.auto_assign_enabled,
        sla_first_response_minutes: settings.sla_first_response_minutes,
        sla_resolution_minutes: settings.sla_resolution_minutes,
        auto_close_after_minutes: settings.auto_close_after_minutes,
      })
    }
  }, [settings])

  const saveMutation = useMutation({
    mutationFn: (body: UpdateSettingsRequest) => api.put<TenantSettings>('/settings', body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tenantSettings.all })
      toast({ title: t('saved') })
    },
    onError: (e: Error) => toast({ title: 'Error', description: e.message, variant: 'destructive' }),
  })

  function num(v: string): number {
    const n = parseInt(v, 10)
    return Number.isFinite(n) && n >= 0 ? n : 0
  }

  return (
    <>
      <Header title={t('title')} />
      <div className="p-6">
        <p className="mb-6 text-sm text-muted-foreground">{t('subtitle')}</p>

        {isLoading ? (
          <Skeleton className="h-96 w-full max-w-2xl" />
        ) : (
          <div className="max-w-2xl space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t('assignment')}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-5">
                <div className="space-y-2">
                  <Label>{t('assignmentStrategy')}</Label>
                  <Select
                    value={form.assignment_strategy}
                    onValueChange={(v) =>
                      setForm((f) => ({ ...f, assignment_strategy: v as AssignmentStrategy }))
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="manual">{t('manual')}</SelectItem>
                      <SelectItem value="round_robin">{t('roundRobin')}</SelectItem>
                      <SelectItem value="load_balanced">{t('loadBalanced')}</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex items-center justify-between">
                  <Label htmlFor="auto-assign">{t('autoAssign')}</Label>
                  <Switch
                    id="auto-assign"
                    checked={form.auto_assign_enabled}
                    onCheckedChange={(v) => setForm((f) => ({ ...f, auto_assign_enabled: v }))}
                  />
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t('sla')}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-5">
                <SlaField
                  label={t('firstResponse')}
                  hint={t('zeroDisabled')}
                  value={form.sla_first_response_minutes}
                  onChange={(v) => setForm((f) => ({ ...f, sla_first_response_minutes: v }))}
                  num={num}
                />
                <SlaField
                  label={t('resolution')}
                  hint={t('zeroDisabled')}
                  value={form.sla_resolution_minutes}
                  onChange={(v) => setForm((f) => ({ ...f, sla_resolution_minutes: v }))}
                  num={num}
                />
                <SlaField
                  label={t('autoClose')}
                  hint={t('zeroDisabled')}
                  value={form.auto_close_after_minutes}
                  onChange={(v) => setForm((f) => ({ ...f, auto_close_after_minutes: v }))}
                  num={num}
                />
              </CardContent>
            </Card>

            <div className="flex justify-end">
              <Button onClick={() => saveMutation.mutate(form)} disabled={saveMutation.isPending}>
                {t('save')}
              </Button>
            </div>
          </div>
        )}
      </div>
    </>
  )
}

function SlaField({
  label,
  hint,
  value,
  onChange,
  num,
}: {
  label: string
  hint: string
  value: number
  onChange: (v: number) => void
  num: (v: string) => number
}) {
  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <Input
        type="number"
        min={0}
        value={value}
        onChange={(e) => onChange(num(e.target.value))}
        className="w-40"
      />
      <p className="text-xs text-muted-foreground">{hint}</p>
    </div>
  )
}
