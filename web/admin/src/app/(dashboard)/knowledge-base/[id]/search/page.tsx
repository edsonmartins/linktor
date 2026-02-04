'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { useQuery, useMutation } from '@tanstack/react-query'
import { ArrowLeft, Search, ChevronDown, ChevronUp, Sparkles } from 'lucide-react'
import { Header } from '@/components/layout/header'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Slider } from '@/components/ui/slider'
import { Skeleton } from '@/components/ui/skeleton'
import { api } from '@/lib/api'
import { queryKeys } from '@/lib/query'
import type { KnowledgeBase, KnowledgeSearchResult } from '@/types'

export default function KnowledgeBaseSearchPage() {
  const params = useParams()
  const id = params.id as string

  const [query, setQuery] = useState('')
  const [limit, setLimit] = useState(5)
  const [expandedResults, setExpandedResults] = useState<Set<string>>(new Set())

  // Fetch knowledge base details
  const { data: kb, isLoading: isLoadingKb } = useQuery({
    queryKey: queryKeys.knowledgeBases.detail(id),
    queryFn: () => api.get<KnowledgeBase>(`/knowledge-bases/${id}`),
  })

  // Search mutation
  const searchMutation = useMutation({
    mutationFn: ({ query, limit }: { query: string; limit: number }) =>
      api.post<{ results: KnowledgeSearchResult[]; query: string; count: number }>(
        `/knowledge-bases/${id}/search`,
        { query, limit }
      ),
  })

  const handleSearch = () => {
    if (!query.trim()) return
    searchMutation.mutate({ query: query.trim(), limit })
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch()
    }
  }

  const toggleExpanded = (itemId: string) => {
    const newExpanded = new Set(expandedResults)
    if (newExpanded.has(itemId)) {
      newExpanded.delete(itemId)
    } else {
      newExpanded.add(itemId)
    }
    setExpandedResults(newExpanded)
  }

  const getScoreColor = (score: number): string => {
    if (score >= 0.8) return 'text-success'
    if (score >= 0.6) return 'text-warning'
    return 'text-muted-foreground'
  }

  const getScoreLabel = (score: number): string => {
    if (score >= 0.8) return 'Excellent match'
    if (score >= 0.6) return 'Good match'
    if (score >= 0.4) return 'Partial match'
    return 'Weak match'
  }

  return (
    <div className="flex h-full flex-col">
      <Header title="Test Search" />

      <div className="flex-1 overflow-auto p-6">
        {/* Breadcrumb */}
        <Link
          href={`/knowledge-base/${id}`}
          className="mb-4 inline-flex items-center text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="mr-1 h-4 w-4" />
          Back to {kb?.name || 'Knowledge Base'}
        </Link>

        <div className="grid gap-6 lg:grid-cols-[1fr_2fr]">
          {/* Search Panel */}
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Sparkles className="h-5 w-5 text-primary" />
                  Semantic Search
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <label className="text-sm font-medium">Query</label>
                  <div className="flex gap-2">
                    <Input
                      placeholder="Enter your question..."
                      value={query}
                      onChange={(e) => setQuery(e.target.value)}
                      onKeyDown={handleKeyDown}
                    />
                    <Button onClick={handleSearch} disabled={searchMutation.isPending || !query.trim()}>
                      <Search className="h-4 w-4" />
                    </Button>
                  </div>
                </div>

                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <label className="text-sm font-medium">Results Limit</label>
                    <span className="text-sm text-muted-foreground">{limit}</span>
                  </div>
                  <Slider
                    value={[limit]}
                    onValueChange={(v) => setLimit(v[0])}
                    min={1}
                    max={20}
                    step={1}
                  />
                </div>

                {kb && (
                  <div className="rounded-lg bg-muted/50 p-3 text-sm">
                    <p className="font-medium">{kb.name}</p>
                    <p className="text-muted-foreground">
                      {kb.item_count} items · {kb.type}
                    </p>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Search Tips */}
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Search Tips</CardTitle>
              </CardHeader>
              <CardContent className="text-sm text-muted-foreground space-y-2">
                <p>
                  <strong>Natural language:</strong> Ask questions as your customers would.
                </p>
                <p>
                  <strong>Scoring:</strong> Results are ranked by semantic similarity (0-100%).
                </p>
                <p>
                  <strong>No embeddings?</strong> If items lack embeddings, regenerate them from the knowledge base page.
                </p>
              </CardContent>
            </Card>
          </div>

          {/* Results Panel */}
          <div>
            <Card>
              <CardHeader>
                <CardTitle>
                  Results
                  {searchMutation.data && (
                    <Badge variant="secondary" className="ml-2">
                      {searchMutation.data.count} found
                    </Badge>
                  )}
                </CardTitle>
              </CardHeader>
              <CardContent>
                {searchMutation.isPending ? (
                  <div className="space-y-4">
                    {Array.from({ length: 3 }).map((_, i) => (
                      <div key={i} className="space-y-2">
                        <Skeleton className="h-4 w-3/4" />
                        <Skeleton className="h-16 w-full" />
                      </div>
                    ))}
                  </div>
                ) : searchMutation.data?.results.length === 0 ? (
                  <div className="py-8 text-center text-muted-foreground">
                    <Search className="mx-auto mb-4 h-12 w-12 opacity-50" />
                    <p>No results found for your query.</p>
                    <p className="text-sm">Try different keywords or check if items have embeddings.</p>
                  </div>
                ) : searchMutation.data?.results ? (
                  <div className="space-y-4">
                    {searchMutation.data.results.map((result, index) => {
                      const isExpanded = expandedResults.has(result.item.id)
                      const scorePercent = Math.round(result.score * 100)

                      return (
                        <div
                          key={result.item.id}
                          className="rounded-lg border border-border p-4 transition-colors hover:border-primary/30"
                        >
                          <div className="flex items-start justify-between gap-4">
                            <div className="flex-1">
                              <div className="flex items-center gap-2 mb-2">
                                <span className="text-xs text-muted-foreground">#{index + 1}</span>
                                <Badge variant="outline" className={getScoreColor(result.score)}>
                                  {scorePercent}%
                                </Badge>
                                <span className="text-xs text-muted-foreground">
                                  {getScoreLabel(result.score)}
                                </span>
                              </div>

                              <h4 className="font-medium">{result.item.question}</h4>

                              <p className={`mt-2 text-sm text-muted-foreground ${isExpanded ? '' : 'line-clamp-2'}`}>
                                {result.item.answer}
                              </p>

                              {result.item.answer.length > 200 && (
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  className="mt-1 h-6 px-2"
                                  onClick={() => toggleExpanded(result.item.id)}
                                >
                                  {isExpanded ? (
                                    <>
                                      <ChevronUp className="mr-1 h-3 w-3" />
                                      Show less
                                    </>
                                  ) : (
                                    <>
                                      <ChevronDown className="mr-1 h-3 w-3" />
                                      Show more
                                    </>
                                  )}
                                </Button>
                              )}

                              {result.item.keywords && result.item.keywords.length > 0 && (
                                <div className="mt-2 flex flex-wrap gap-1">
                                  {result.item.keywords.map((keyword) => (
                                    <Badge key={keyword} variant="secondary" className="text-xs">
                                      {keyword}
                                    </Badge>
                                  ))}
                                </div>
                              )}
                            </div>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                ) : (
                  <div className="py-8 text-center text-muted-foreground">
                    <Search className="mx-auto mb-4 h-12 w-12 opacity-50" />
                    <p>Enter a query to test semantic search.</p>
                    <p className="text-sm mt-1">
                      This uses vector embeddings to find semantically similar answers.
                    </p>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </div>
    </div>
  )
}
