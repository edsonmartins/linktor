# Documentação do Linktor

Site de documentação (Docusaurus 3, pt-BR), organizado por público: **Usar**, **Canais**,
**Desenvolver**, **SDKs** e **Implantar**.

```bash
npm install
npm start        # http://localhost:3000
npm run build    # valida links e âncoras (falha se algum quebrar)
npm run serve    # serve o build local
```

| Caminho | Conteúdo |
|---|---|
| `docs/` | Páginas |
| `sidebars.ts` | Navegação por público |
| `src/theme/MDXComponents.tsx` | Componentes globais (`Tabs`, `TabItem`) |
| `src/components/McpPlayground.jsx` | Playground do servidor MCP |

Convenções de escrita, molde das páginas de canal e formato dos exemplos multi-linguagem:
[`docs/contribuir/escrever-paginas.md`](docs/contribuir/escrever-paginas.md).
