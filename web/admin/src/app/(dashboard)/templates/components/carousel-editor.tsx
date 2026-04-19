'use client'

import { useTranslations } from 'next-intl'
import { Plus, Trash2, GripVertical } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { TemplateCarouselCard, TemplateComponent } from '@/types'

/**
 * CarouselEditor drives the CAROUSEL template component: 1-10 cards, each
 * with its own HEADER (image/video only per Meta), BODY with its own
 * placeholders, and an optional row of QUICK_REPLY or URL buttons. The
 * component operates on the canonical `TemplateCarouselCard[]` shape so
 * it can be serialised straight into the create/edit payload.
 */
export function CarouselEditor({
  cards,
  onChange,
}: {
  cards: TemplateCarouselCard[]
  onChange: (cards: TemplateCarouselCard[]) => void
}) {
  const t = useTranslations('templates.form.carousel')

  const addCard = () => {
    if (cards.length >= 10) return
    onChange([
      ...cards,
      {
        components: [
          { type: 'HEADER', format: 'IMAGE', example: { header_handle: [''] } },
          { type: 'BODY', text: '' },
        ],
      },
    ])
  }

  const updateCard = (index: number, card: TemplateCarouselCard) => {
    onChange(cards.map((c, i) => (i === index ? card : c)))
  }

  const removeCard = (index: number) => {
    onChange(cards.filter((_, i) => i !== index))
  }

  const moveCard = (index: number, direction: -1 | 1) => {
    const target = index + direction
    if (target < 0 || target >= cards.length) return
    const next = [...cards]
    ;[next[index], next[target]] = [next[target], next[index]]
    onChange(next)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <span>
            {t('title')} <span className="text-sm text-muted-foreground">({cards.length}/10)</span>
          </span>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={addCard}
            disabled={cards.length >= 10}
          >
            <Plus className="mr-1 h-3 w-3" />
            {t('addCard')}
          </Button>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {cards.length === 0 ? (
          <p className="text-sm text-muted-foreground">{t('empty')}</p>
        ) : (
          cards.map((card, i) => (
            <CardEditor
              key={i}
              card={card}
              index={i}
              total={cards.length}
              onChange={(next) => updateCard(i, next)}
              onRemove={() => removeCard(i)}
              onMove={(dir) => moveCard(i, dir)}
            />
          ))
        )}
      </CardContent>
    </Card>
  )
}

const POSITIONAL = /\{\{\s*(\d+)\s*\}\}/g
function posCount(s: string): number {
  let max = 0
  for (const m of s.matchAll(POSITIONAL)) {
    const n = Number(m[1])
    if (Number.isFinite(n) && n > max) max = n
  }
  return max
}

function CardEditor({
  card,
  index,
  total,
  onChange,
  onRemove,
  onMove,
}: {
  card: TemplateCarouselCard
  index: number
  total: number
  onChange: (card: TemplateCarouselCard) => void
  onRemove: () => void
  onMove: (direction: -1 | 1) => void
}) {
  const t = useTranslations('templates.form.carousel')

  const header = card.components.find((c) => c.type === 'HEADER')
  const body = card.components.find((c) => c.type === 'BODY')
  const buttonsRow = card.components.find((c) => c.type === 'BUTTONS')

  const updateComponent = (type: TemplateComponent['type'], patch: Partial<TemplateComponent>) => {
    const existing = card.components.find((c) => c.type === type)
    if (!existing && patch) {
      onChange({
        components: [...card.components, { type, ...patch } as TemplateComponent],
      })
      return
    }
    onChange({
      components: card.components.map((c) => (c.type === type ? { ...c, ...patch } : c)),
    })
  }

  const removeComponent = (type: TemplateComponent['type']) => {
    onChange({ components: card.components.filter((c) => c.type !== type) })
  }

  // Body placeholders drive the per-card example inputs. We default to
  // positional — named templates still work because the parent form
  // controls parameter_format; Meta reads the same examples block.
  const bodyVarCount = body?.text ? posCount(body.text) : 0
  const bodyExamples = body?.example?.body_text?.[0] ?? []
  const trimmed = bodyExamples.slice(0, bodyVarCount)
  while (trimmed.length < bodyVarCount) trimmed.push('')

  const updateBodyExamples = (values: string[]) => {
    updateComponent('BODY', {
      text: body?.text ?? '',
      example: { body_text: [values] },
    })
  }

  const addButton = (type: 'QUICK_REPLY' | 'URL') => {
    const current = buttonsRow?.buttons ?? []
    if (current.length >= 2) return
    const next = [...current, { type, text: '' } as const]
    updateComponent('BUTTONS', { buttons: next })
  }

  return (
    <div className="rounded-lg border p-4">
      <div className="mb-3 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <GripVertical className="h-4 w-4 text-muted-foreground" />
          <span className="font-medium">
            {t('card')} {index + 1}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onMove(-1)}
            disabled={index === 0}
          >
            ↑
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onMove(1)}
            disabled={index === total - 1}
          >
            ↓
          </Button>
          <Button type="button" variant="ghost" size="icon" onClick={onRemove}>
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Header — IMAGE or VIDEO only on carousel cards per Meta */}
      <div className="mb-3 space-y-2">
        <div className="flex items-center justify-between">
          <Label>{t('header')}</Label>
          <Select
            value={header?.format ?? 'IMAGE'}
            onValueChange={(v) =>
              updateComponent('HEADER', {
                format: v as 'IMAGE' | 'VIDEO',
                example: { header_handle: [header?.example?.header_handle?.[0] ?? ''] },
              })
            }
          >
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="IMAGE">{t('headerImage')}</SelectItem>
              <SelectItem value="VIDEO">{t('headerVideo')}</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <Input
          value={header?.example?.header_handle?.[0] ?? ''}
          onChange={(e) =>
            updateComponent('HEADER', {
              format: header?.format ?? 'IMAGE',
              example: { header_handle: [e.target.value] },
            })
          }
          placeholder={t('headerHandlePlaceholder')}
        />
      </div>

      {/* Body — text + per-card examples */}
      <div className="mb-3 space-y-2">
        <Label>{t('body')}</Label>
        <Textarea
          rows={3}
          value={body?.text ?? ''}
          onChange={(e) =>
            updateComponent('BODY', {
              text: e.target.value,
              example: body?.example,
            })
          }
          placeholder={t('bodyPlaceholder')}
        />
        {bodyVarCount > 0 && (
          <div className="space-y-2 rounded-md border border-dashed p-2">
            <Label className="text-xs">{t('bodyExamples')}</Label>
            {trimmed.map((val, i) => (
              <div key={i} className="flex items-center gap-2">
                <span className="w-12 shrink-0 font-mono text-xs text-muted-foreground">
                  {`{{${i + 1}}}`}
                </span>
                <Input
                  value={val}
                  onChange={(e) => {
                    const next = [...trimmed]
                    next[i] = e.target.value
                    updateBodyExamples(next)
                  }}
                />
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Buttons — QUICK_REPLY or URL, max 2 per Meta */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <Label>{t('buttons')}</Label>
          <div className="flex gap-1">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => addButton('QUICK_REPLY')}
              disabled={(buttonsRow?.buttons?.length ?? 0) >= 2}
            >
              <Plus className="mr-1 h-3 w-3" />
              {t('addQuickReply')}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => addButton('URL')}
              disabled={(buttonsRow?.buttons?.length ?? 0) >= 2}
            >
              <Plus className="mr-1 h-3 w-3" />
              {t('addURL')}
            </Button>
          </div>
        </div>
        {(buttonsRow?.buttons ?? []).length === 0 ? (
          <p className="text-xs text-muted-foreground">{t('buttonsEmpty')}</p>
        ) : (
          (buttonsRow?.buttons ?? []).map((btn, i) => (
            <div key={i} className="flex items-center gap-2">
              <span className="rounded bg-muted px-2 py-0.5 font-mono text-xs">
                {btn.type}
              </span>
              <Input
                value={btn.text}
                onChange={(e) => {
                  const next = [...(buttonsRow?.buttons ?? [])]
                  next[i] = { ...next[i], text: e.target.value }
                  updateComponent('BUTTONS', { buttons: next })
                }}
                placeholder={t('buttonTextPlaceholder')}
                maxLength={25}
              />
              {btn.type === 'URL' && (
                <Input
                  value={btn.url ?? ''}
                  onChange={(e) => {
                    const next = [...(buttonsRow?.buttons ?? [])]
                    next[i] = { ...next[i], url: e.target.value }
                    updateComponent('BUTTONS', { buttons: next })
                  }}
                  placeholder="https://example.com"
                />
              )}
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => {
                  const next = (buttonsRow?.buttons ?? []).filter((_, idx) => idx !== i)
                  if (next.length === 0) removeComponent('BUTTONS')
                  else updateComponent('BUTTONS', { buttons: next })
                }}
              >
                <Trash2 className="h-3 w-3" />
              </Button>
            </div>
          ))
        )}
      </div>
    </div>
  )
}
