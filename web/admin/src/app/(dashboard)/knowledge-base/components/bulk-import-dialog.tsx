'use client'

import { useState, useRef } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Upload, FileJson, FileSpreadsheet, AlertCircle, CheckCircle } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import { toast } from '@/hooks/use-toast'

interface BulkImportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  knowledgeBaseId: string
}

interface BulkImportResult {
  created: number
  items: unknown[]
  errors: string[]
}

interface ImportItem {
  question: string
  answer: string
  keywords?: string[]
  source?: string
}

export function BulkImportDialog({ open, onOpenChange, knowledgeBaseId }: BulkImportDialogProps) {
  const queryClient = useQueryClient()
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [activeTab, setActiveTab] = useState<'json' | 'csv'>('json')
  const [jsonInput, setJsonInput] = useState('')
  const [csvInput, setCsvInput] = useState('')
  const [parseErrors, setParseErrors] = useState<string[]>([])
  const [parsedItems, setParsedItems] = useState<ImportItem[]>([])

  const importMutation = useMutation({
    mutationFn: (items: ImportItem[]) =>
      api.post<BulkImportResult>(`/knowledge-bases/${knowledgeBaseId}/items/bulk`, { items }),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeItems.list(knowledgeBaseId, {}) })
      queryClient.invalidateQueries({ queryKey: queryKeys.knowledgeBases.detail(knowledgeBaseId) })

      toast({
        title: 'Import completed',
        description: `Successfully imported ${result.created} items.${result.errors?.length > 0 ? ` ${result.errors.length} errors.` : ''}`,
      })

      if (result.errors?.length > 0) {
        setParseErrors(result.errors)
      } else {
        handleClose()
      }
    },
    onError: (error: Error) => {
      toast({
        title: 'Import failed',
        description: error.message || 'Failed to import items',
        variant: 'error',
      })
    },
  })

  const handleClose = () => {
    setJsonInput('')
    setCsvInput('')
    setParseErrors([])
    setParsedItems([])
    onOpenChange(false)
  }

  const parseJSON = (input: string): ImportItem[] | null => {
    try {
      const data = JSON.parse(input)
      const items = data.items || data

      if (!Array.isArray(items)) {
        throw new Error('Expected an array of items')
      }

      const parsed: ImportItem[] = []
      const errors: string[] = []

      items.forEach((item, index) => {
        if (!item.question || !item.answer) {
          errors.push(`Item ${index + 1}: Missing question or answer`)
          return
        }
        parsed.push({
          question: String(item.question),
          answer: String(item.answer),
          keywords: Array.isArray(item.keywords) ? item.keywords.map(String) : undefined,
          source: item.source ? String(item.source) : undefined,
        })
      })

      if (errors.length > 0) {
        setParseErrors(errors)
      }

      return parsed
    } catch (e) {
      setParseErrors([(e as Error).message || 'Invalid JSON format'])
      return null
    }
  }

  const parseCSV = (input: string): ImportItem[] | null => {
    try {
      const lines = input.split('\n').filter((line) => line.trim())
      if (lines.length < 2) {
        throw new Error('CSV must have at least a header row and one data row')
      }

      const headers = lines[0].split(',').map((h) => h.trim().toLowerCase())
      const questionIndex = headers.indexOf('question')
      const answerIndex = headers.indexOf('answer')
      const keywordsIndex = headers.indexOf('keywords')
      const sourceIndex = headers.indexOf('source')

      if (questionIndex === -1 || answerIndex === -1) {
        throw new Error('CSV must have "question" and "answer" columns')
      }

      const parsed: ImportItem[] = []
      const errors: string[] = []

      for (let i = 1; i < lines.length; i++) {
        const values = parseCSVLine(lines[i])

        if (!values[questionIndex] || !values[answerIndex]) {
          errors.push(`Row ${i + 1}: Missing question or answer`)
          continue
        }

        parsed.push({
          question: values[questionIndex],
          answer: values[answerIndex],
          keywords: keywordsIndex >= 0 && values[keywordsIndex]
            ? values[keywordsIndex].split(';').map((k) => k.trim()).filter(Boolean)
            : undefined,
          source: sourceIndex >= 0 ? values[sourceIndex] : undefined,
        })
      }

      if (errors.length > 0) {
        setParseErrors(errors)
      }

      return parsed
    } catch (e) {
      setParseErrors([(e as Error).message || 'Invalid CSV format'])
      return null
    }
  }

  // Simple CSV line parser that handles quoted values
  const parseCSVLine = (line: string): string[] => {
    const values: string[] = []
    let current = ''
    let inQuotes = false

    for (let i = 0; i < line.length; i++) {
      const char = line[i]

      if (char === '"' && (i === 0 || line[i - 1] !== '\\')) {
        inQuotes = !inQuotes
      } else if (char === ',' && !inQuotes) {
        values.push(current.trim())
        current = ''
      } else {
        current += char
      }
    }
    values.push(current.trim())

    return values
  }

  const handleParse = () => {
    setParseErrors([])
    setParsedItems([])

    const input = activeTab === 'json' ? jsonInput : csvInput
    if (!input.trim()) {
      setParseErrors(['Please enter some data to import'])
      return
    }

    const items = activeTab === 'json' ? parseJSON(input) : parseCSV(input)
    if (items && items.length > 0) {
      setParsedItems(items)
    }
  }

  const handleImport = () => {
    if (parsedItems.length === 0) {
      handleParse()
      return
    }

    if (parsedItems.length > 100) {
      setParseErrors(['Maximum 100 items per import. Please split your data.'])
      return
    }

    importMutation.mutate(parsedItems)
  }

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const reader = new FileReader()
    reader.onload = (event) => {
      const content = event.target?.result as string
      if (file.name.endsWith('.json')) {
        setActiveTab('json')
        setJsonInput(content)
      } else if (file.name.endsWith('.csv')) {
        setActiveTab('csv')
        setCsvInput(content)
      }
      setParseErrors([])
      setParsedItems([])
    }
    reader.readAsText(file)

    // Reset file input
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[700px]">
        <DialogHeader>
          <DialogTitle>Bulk Import</DialogTitle>
          <DialogDescription>
            Import multiple items at once using JSON or CSV format.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* File Upload */}
          <div className="flex items-center gap-4">
            <input
              ref={fileInputRef}
              type="file"
              accept=".json,.csv"
              onChange={handleFileUpload}
              className="hidden"
            />
            <Button
              variant="outline"
              onClick={() => fileInputRef.current?.click()}
            >
              <Upload className="mr-2 h-4 w-4" />
              Upload File
            </Button>
            <span className="text-sm text-muted-foreground">
              Supports .json and .csv files
            </span>
          </div>

          {/* Tabs */}
          <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as 'json' | 'csv')}>
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="json" className="flex items-center gap-2">
                <FileJson className="h-4 w-4" />
                JSON
              </TabsTrigger>
              <TabsTrigger value="csv" className="flex items-center gap-2">
                <FileSpreadsheet className="h-4 w-4" />
                CSV
              </TabsTrigger>
            </TabsList>

            <TabsContent value="json" className="space-y-2">
              <Textarea
                placeholder={`{
  "items": [
    {
      "question": "How do I reset my password?",
      "answer": "Go to Settings > Security > Reset Password",
      "keywords": ["password", "reset", "security"],
      "source": "FAQ v1.0"
    }
  ]
}`}
                value={jsonInput}
                onChange={(e) => {
                  setJsonInput(e.target.value)
                  setParsedItems([])
                  setParseErrors([])
                }}
                className="font-mono text-sm min-h-[200px]"
              />
            </TabsContent>

            <TabsContent value="csv" className="space-y-2">
              <Textarea
                placeholder={`question,answer,keywords,source
"How do I reset my password?","Go to Settings > Security > Reset Password","password;reset;security","FAQ v1.0"
"What are your hours?","We're open 9am-5pm Monday to Friday","hours;schedule;availability","FAQ v1.0"`}
                value={csvInput}
                onChange={(e) => {
                  setCsvInput(e.target.value)
                  setParsedItems([])
                  setParseErrors([])
                }}
                className="font-mono text-sm min-h-[200px]"
              />
              <p className="text-xs text-muted-foreground">
                Use semicolons (;) to separate multiple keywords within the keywords column.
              </p>
            </TabsContent>
          </Tabs>

          {/* Parse Errors */}
          {parseErrors.length > 0 && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>Errors</AlertTitle>
              <AlertDescription>
                <ul className="list-disc list-inside text-sm">
                  {parseErrors.slice(0, 5).map((error, i) => (
                    <li key={i}>{error}</li>
                  ))}
                  {parseErrors.length > 5 && (
                    <li>...and {parseErrors.length - 5} more errors</li>
                  )}
                </ul>
              </AlertDescription>
            </Alert>
          )}

          {/* Parsed Preview */}
          {parsedItems.length > 0 && parseErrors.length === 0 && (
            <Alert>
              <CheckCircle className="h-4 w-4" />
              <AlertTitle>Ready to Import</AlertTitle>
              <AlertDescription>
                {parsedItems.length} item{parsedItems.length > 1 ? 's' : ''} parsed successfully.
              </AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button
            variant="outline"
            onClick={handleParse}
            disabled={!(jsonInput.trim() || csvInput.trim())}
          >
            Validate
          </Button>
          <Button
            onClick={handleImport}
            disabled={importMutation.isPending || (parsedItems.length === 0 && parseErrors.length > 0)}
          >
            {importMutation.isPending
              ? 'Importing...'
              : parsedItems.length > 0
              ? `Import ${parsedItems.length} Items`
              : 'Parse & Import'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
