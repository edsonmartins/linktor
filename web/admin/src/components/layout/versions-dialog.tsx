'use client'

import { useState } from 'react'
import { useTranslations } from 'next-intl'
import { useQuery } from '@tanstack/react-query'
import { CheckCircle2, AlertTriangle, RefreshCw, Copy, Check, Info } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { copyText } from '@/lib/clipboard'
import { getAdminVersion, getApiOrigin } from '@/lib/runtime-config'

/**
 * Mostra qual build de cada parte está no ar.
 *
 * Existe porque "subiu ou não?" era respondido entrando na VPS e rodando
 * `docker ps`. Pior: um deploy pode aplicar só metade — o admin novo com a API
 * velha, ou o contrário — e a interface não dava nenhum sinal disso.
 *
 * O admin conhece a própria versão pela config de runtime (mesma imagem serve
 * qualquer host); a da API vem do /health, que a reporta desde que o compose
 * injeta LINKTOR_VERSION.
 */
export function VersionsDialog({
  open,
  onOpenChange,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const t = useTranslations('versions')
  const [copied, setCopied] = useState(false)

  const adminVersion = getAdminVersion()

  const { data, isFetching, refetch, isError } = useQuery({
    queryKey: ['versao-api'],
    // Sem cache: a pergunta é sempre "o que está no ar AGORA".
    staleTime: 0,
    gcTime: 0,
    enabled: open,
    queryFn: async () => {
      const res = await fetch(`${getApiOrigin()}/health`, { credentials: 'include' })
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      return (await res.json()) as { version?: string; status?: string }
    },
  })

  const apiVersion = data?.version || ''
  // Só compara quando as duas são conhecidas: build antiga da API não reporta
  // versão, e alarmar nesse caso seria ruído.
  const conhecidas = Boolean(adminVersion && apiVersion)
  const combinam = conhecidas && adminVersion === apiVersion

  const copiar = async () => {
    const resumo = `admin: ${adminVersion || '—'}\napi: ${apiVersion || '—'}`
    if (!(await copyText(resumo))) return
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t('title')}</DialogTitle>
          <DialogDescription>{t('description')}</DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <Linha rotulo={t('admin')} valor={adminVersion} ausente={t('unknown')} />
          <Linha
            rotulo={t('api')}
            valor={isError ? '' : apiVersion}
            ausente={isError ? t('unreachable') : t('unknown')}
            carregando={isFetching}
          />

          {conhecidas && (
            <div
              className={`flex items-start gap-2 rounded-md border p-3 text-sm ${
                combinam ? 'text-muted-foreground' : 'border-yellow-500/40 text-yellow-500'
              }`}
            >
              {combinam ? (
                <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
              ) : (
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              )}
              <span>{combinam ? t('inSync') : t('outOfSync')}</span>
            </div>
          )}

          {!conhecidas && !isFetching && (
            <div className="flex items-start gap-2 rounded-md border p-3 text-sm text-muted-foreground">
              <Info className="mt-0.5 h-4 w-4 shrink-0" />
              <span>{t('noVersionReported')}</span>
            </div>
          )}
        </div>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" size="sm" onClick={copiar}>
            {copied ? <Check className="mr-2 h-4 w-4" /> : <Copy className="mr-2 h-4 w-4" />}
            {t('copy')}
          </Button>
          <Button type="button" size="sm" onClick={() => refetch()} disabled={isFetching}>
            <RefreshCw className={`mr-2 h-4 w-4 ${isFetching ? 'animate-spin' : ''}`} />
            {t('refresh')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}

function Linha({
  rotulo,
  valor,
  ausente,
  carregando,
}: {
  rotulo: string
  valor: string
  ausente: string
  carregando?: boolean
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-sm text-muted-foreground">{rotulo}</span>
      {carregando ? (
        <RefreshCw className="h-4 w-4 animate-spin text-muted-foreground" />
      ) : valor ? (
        <Badge variant="outline" className="font-mono text-xs select-all">
          {valor}
        </Badge>
      ) : (
        <span className="text-xs text-muted-foreground">{ausente}</span>
      )}
    </div>
  )
}
