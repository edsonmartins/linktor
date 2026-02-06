---
sidebar_position: 1
title: Knowledge Base Overview
---

# Knowledge Base Overview

The Knowledge Base allows you to train your AI bots with your own data. Upload documents, FAQs, and content to enable bots to provide accurate, company-specific answers through semantic search.

## What is a Knowledge Base?

A **Knowledge Base** in Linktor is a collection of documents and content that your AI bots can reference when answering customer questions. Instead of relying solely on the AI model's general knowledge, bots can search your specific documentation to provide accurate, up-to-date answers.

### Key Benefits

| Benefit | Description |
|---------|-------------|
| **Accurate Answers** | Bots respond with your actual product information |
| **Reduced Hallucination** | AI grounds responses in verified content |
| **Easy Updates** | Update documents without retraining models |
| **Source Citations** | Optionally show customers where answers come from |
| **Multi-language** | Works with content in any language |

## How It Works

### Architecture Overview

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Customer     │     │    Linktor      │     │   AI Provider   │
│    Question     │────►│    Bot Engine   │────►│   (GPT, Claude) │
└─────────────────┘     └────────┬────────┘     └────────┬────────┘
                                 │                       │
                        ┌────────▼────────┐              │
                        │  Vector Search  │              │
                        │  (Knowledge DB) │◄─────────────┘
                        └────────┬────────┘
                                 │
                        ┌────────▼────────┐
                        │  Your Documents │
                        │  FAQs, Guides   │
                        └─────────────────┘
```

### RAG Pipeline (Retrieval-Augmented Generation)

1. **Customer asks a question**: "How do I reset my password?"

2. **Semantic search**: Linktor searches your knowledge base for relevant content using vector similarity

3. **Context retrieval**: The most relevant document chunks are retrieved

4. **Prompt augmentation**: Retrieved content is added to the AI prompt as context

5. **AI generation**: The AI generates a response based on the provided context

6. **Response delivery**: Customer receives an accurate, grounded answer

## Supported Content Types

### Documents

| Format | Extension | Notes |
|--------|-----------|-------|
| PDF | `.pdf` | Text and scanned (OCR) |
| Word | `.docx`, `.doc` | Full formatting preserved |
| Plain Text | `.txt` | Simple text files |
| Markdown | `.md` | Headers and formatting retained |
| HTML | `.html` | Web pages and exports |
| Rich Text | `.rtf` | Basic formatting |

### Structured Data

| Format | Extension | Notes |
|--------|-----------|-------|
| JSON | `.json` | Structured data |
| CSV | `.csv` | Tabular data |
| YAML | `.yaml`, `.yml` | Configuration files |

### Web Content

- **Web pages**: Crawl and index website content
- **Sitemaps**: Import entire sites via sitemap.xml
- **RSS feeds**: Keep content updated automatically

### Integrations

- **Notion**: Sync Notion pages and databases
- **Confluence**: Import Confluence spaces
- **Google Docs**: Connect Google Drive folders
- **GitHub**: Index repository documentation

## Creating a Knowledge Base

### Via Dashboard

1. Navigate to **Knowledge Base** in the sidebar
2. Click **Create Knowledge Base**
3. Enter a name and description
4. Configure chunking and embedding settings
5. Start adding content

### Via API

```typescript
import { Linktor } from '@linktor/sdk'

const client = new Linktor({ apiKey: 'YOUR_API_KEY' })

// Create a knowledge base
const kb = await client.knowledgeBases.create({
  name: 'Product Documentation',
  description: 'Technical docs and user guides',
  settings: {
    chunkSize: 500,
    chunkOverlap: 50,
    embeddingModel: 'text-embedding-3-small'
  }
})

// Upload a document
await client.knowledgeBases.uploadDocument(kb.id, {
  file: './docs/user-guide.pdf',
  metadata: {
    category: 'user-guides',
    product: 'main-app',
    version: '2.0'
  }
})
```

## Adding Content

### File Upload

```typescript
// Upload single file
await client.knowledgeBases.uploadDocument(kbId, {
  file: './document.pdf',
  metadata: { category: 'general' }
})

// Upload multiple files
await client.knowledgeBases.uploadDocuments(kbId, {
  files: ['./doc1.pdf', './doc2.pdf', './doc3.pdf'],
  metadata: { batch: 'initial-import' }
})
```

### Text Content

```typescript
// Add text directly
await client.knowledgeBases.addContent(kbId, {
  type: 'text',
  title: 'Refund Policy',
  content: `Our refund policy allows returns within 30 days of purchase...`,
  metadata: { category: 'policies' }
})
```

### FAQ Import

```typescript
// Import FAQ pairs
await client.knowledgeBases.importFAQ(kbId, {
  faqs: [
    {
      question: 'How do I reset my password?',
      answer: 'Visit settings > security > reset password...'
    },
    {
      question: 'What payment methods do you accept?',
      answer: 'We accept Visa, Mastercard, PayPal, and bank transfer...'
    }
  ]
})
```

### Web Crawling

```typescript
// Crawl a website
await client.knowledgeBases.crawlWebsite(kbId, {
  url: 'https://docs.example.com',
  maxPages: 100,
  includePatterns: ['/docs/*', '/guides/*'],
  excludePatterns: ['/blog/*', '/news/*'],
  respectRobotsTxt: true
})
```

## Connecting to Bots

Link knowledge bases to your bots:

```typescript
// Connect knowledge base to bot
await client.bots.update('bot_123', {
  knowledgeBaseIds: ['kb_docs', 'kb_faq'],
  knowledgeSettings: {
    maxChunks: 5,
    minRelevanceScore: 0.7,
    citeSources: true,
    sourceFormat: 'inline'  // 'inline' | 'footnote' | 'hidden'
  }
})
```

### Knowledge Base Priority

When multiple knowledge bases are connected:

```typescript
{
  knowledgeBaseIds: ['kb_product_docs', 'kb_faq', 'kb_policies'],
  knowledgePriority: {
    'kb_product_docs': 10,  // Highest priority
    'kb_faq': 5,
    'kb_policies': 1
  }
}
```

## Search and Retrieval

### Manual Search

Search your knowledge base programmatically:

```typescript
const results = await client.knowledgeBases.search(kbId, {
  query: 'How to cancel subscription',
  limit: 5,
  minScore: 0.7,
  filters: {
    category: 'billing'
  }
})

results.forEach(result => {
  console.log(`Score: ${result.score}`)
  console.log(`Content: ${result.content}`)
  console.log(`Source: ${result.metadata.source}`)
})
```

### Search Filters

Filter results by metadata:

```typescript
{
  filters: {
    category: 'support',
    product: { in: ['app-v2', 'app-v3'] },
    lastUpdated: { gte: '2024-01-01' },
    language: 'en'
  }
}
```

## Content Management

### Updating Content

```typescript
// Update document content
await client.knowledgeBases.updateDocument(kbId, docId, {
  file: './updated-document.pdf'
})

// Update metadata only
await client.knowledgeBases.updateDocumentMetadata(kbId, docId, {
  metadata: { version: '2.1', reviewed: true }
})
```

### Deleting Content

```typescript
// Delete single document
await client.knowledgeBases.deleteDocument(kbId, docId)

// Delete by filter
await client.knowledgeBases.deleteDocuments(kbId, {
  filter: { category: 'outdated' }
})
```

### Syncing External Sources

```typescript
// Set up automatic sync
await client.knowledgeBases.createSync(kbId, {
  source: 'notion',
  config: {
    pageId: 'notion-page-id',
    includeSubpages: true
  },
  schedule: '0 0 * * *'  // Daily at midnight
})
```

## Analytics

### Knowledge Base Stats

```typescript
const stats = await client.knowledgeBases.getStats(kbId)

console.log(stats.documentCount)      // Total documents
console.log(stats.chunkCount)         // Total chunks
console.log(stats.totalTokens)        // Total tokens
console.log(stats.storageUsed)        // Storage in bytes
console.log(stats.lastUpdated)        // Last content update
```

### Search Analytics

```typescript
const analytics = await client.knowledgeBases.getSearchAnalytics(kbId, {
  dateRange: { from: '2024-01-01', to: '2024-01-31' }
})

console.log(analytics.totalSearches)
console.log(analytics.avgRelevanceScore)
console.log(analytics.topQueries)         // Most common queries
console.log(analytics.noResultQueries)    // Queries with no matches
console.log(analytics.lowScoreQueries)    // Queries with poor matches
```

### Identifying Gaps

Find questions your knowledge base can't answer:

```typescript
const gaps = await client.knowledgeBases.identifyGaps(kbId, {
  dateRange: { from: '2024-01-01', to: '2024-01-31' },
  minOccurrences: 5
})

gaps.forEach(gap => {
  console.log(`Query: ${gap.query}`)
  console.log(`Occurrences: ${gap.count}`)
  console.log(`Avg Score: ${gap.avgScore}`)
})
```

## Best Practices

### Content Quality

1. **Keep content up-to-date**: Outdated information leads to wrong answers
2. **Use clear language**: Write as if explaining to a customer
3. **Structure with headers**: Helps chunking and retrieval
4. **Include common variations**: Different ways customers ask the same question

### Organization

1. **Use meaningful metadata**: Categories, tags, and versions help filtering
2. **Separate knowledge bases**: Different KBs for products, policies, and FAQs
3. **Regular reviews**: Audit content periodically for accuracy

### Performance

1. **Optimal chunk size**: 300-500 tokens typically works well
2. **Relevance threshold**: Start with 0.7 and adjust based on results
3. **Limit retrieved chunks**: 3-5 chunks is usually sufficient

## Troubleshooting

### Poor Search Results

| Issue | Solution |
|-------|----------|
| No results | Lower `minRelevanceScore` or add more content |
| Irrelevant results | Increase `minRelevanceScore` or improve content |
| Missing context | Increase `maxChunks` or `chunkOverlap` |
| Outdated answers | Update or remove old documents |

### Content Processing Issues

| Issue | Solution |
|-------|----------|
| PDF not processing | Check if PDF is scanned (enable OCR) |
| Wrong encoding | Convert to UTF-8 before upload |
| Large file fails | Split into smaller files |
| Web crawl incomplete | Check `excludePatterns` and robots.txt |

## Next Steps

- [Embeddings Guide](/knowledge-base/embeddings) - Deep dive into how embeddings work
- [Bot Configuration](/bots/configuration) - Connect knowledge bases to bots
- [API Reference](/api/overview) - Full API documentation
