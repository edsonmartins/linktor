---
sidebar_position: 1
title: Escrever páginas
description: Como a documentação é organizada, que tipo de página escrever, o molde das páginas de canal e dos exemplos multi-linguagem.
---

# Escrever páginas

A documentação vive em `web/docs` (Docusaurus) e é versionada junto com o código: uma mudança de
comportamento e a página que a descreve entram no mesmo pull request.

## Para quem é cada área

A navegação é por público. Antes de escrever, decida **quem lê**:

| Área | Pasta | Leitor | Pergunta que a página responde |
|---|---|---|---|
| **Usar** | `guias/`, `conceitos/`, `painel/` | Quem atende e administra pelo painel | "Como faço X na tela?" |
| **Canais** | `canais/` | Quem conecta um canal | "O que preciso criar no provedor e o que preencho no Linktor?" |
| **Desenvolver** | `api/`, `mcp/` | Quem integra um sistema | "Qual chamada faço e o que volta?" |
| **SDKs** | `sdks/` | Quem integra usando uma linguagem específica | "Como faço isso em Java / C# / Go…?" |
| **Implantar** | `implantar/` | Quem instala e opera o servidor | "Como subo, configuro e mantenho?" |

## Que tipo de página

Cada página faz **uma** destas coisas — não misture:

- **Tutorial** — leva um iniciante do zero a um resultado ("Primeiros passos"). Passos numerados, um
  caminho só, sem alternativas.
- **Guia de tarefa** — resolve um problema específico de quem já usa ("Conectar um canal"). Título
  começa com verbo.
- **Referência** — descreve com precisão (campos, rotas, variáveis). Tabelas, sem narrativa.
- **Explicação** — dá o porquê (arquitetura, segurança). É onde entram os diagramas.

## Regras de escrita

- **A fonte da verdade é o código.** Rota, campo, variável de ambiente ou método de SDK só entra na
  página depois de conferido no código (`cmd/server/main.go`, `internal/`, `sdks/`, `web/admin/`). O
  que está parcial ou atrás de configuração é sinalizado com um aviso (`:::note`, `:::warning`), nunca
  descrito como pronto.
- Comece pela resposta; contexto depois.
- Diga o nome exato do que está na tela, em **negrito**, como aparece no painel em português
  (`web/admin/src/i18n/locales/pt-BR.json`): "clique em **Novo canal**".
- Um exemplo real vale mais que uma descrição: mostre o JSON, o campo preenchido, a resposta.
- Nunca coloque segredo, telefone ou e-mail reais em exemplo. Use `lk_sua_chave`, `+5511999999999`,
  `cliente@exemplo.com`.

## Valores de exemplo padronizados

Todas as páginas usam os mesmos nomes, para o leitor copiar de uma página e colar em outra:

| O quê | Como aparece |
|---|---|
| Endereço do servidor | `https://linktor.suaempresa.com` (variável `LINKTOR_URL`) |
| Base da API | `https://linktor.suaempresa.com/api/v1` |
| Chave de API | `lk_sua_chave` (variável `LINKTOR_API_KEY`) |
| Id de canal | `CHANNEL_ID` |
| Telefone | `+5511999999999` |

## Molde de página de canal

Toda página em `canais/` segue as mesmas seções, nesta ordem:

1. **Resumo** — o que o canal faz, o valor de `type` na API e uma tabela do que é suportado (texto,
   mídia, botões, modelos, grupos, confirmação de leitura).
2. **Antes de começar** — contas, aprovações e acessos necessários no provedor.
3. **Configurar na origem** — passo a passo no console do provedor (Meta, BotFather, Twilio…), com
   o nome dos menus como aparecem lá.
4. **Configurar no Linktor** — passo a passo no painel e a tabela de campos: rótulo na tela, chave
   em `config` ou `credentials`, obrigatório ou não.
5. **Webhook** — a URL exata que o provedor deve chamar e como a assinatura é validada.
6. **Criar o canal pela API** — exemplo multi-linguagem.
7. **Enviar e receber** — exemplo de envio e o que chega no webhook da sua aplicação.
8. **Limites e particularidades** — janelas de atendimento, tipos de mídia, comportamentos do
   provedor.
9. **Solução de problemas** — sintoma → causa → o que fazer.

## Exemplos multi-linguagem

`Tabs` e `TabItem` estão registrados globalmente (não precisa de `import`). Use sempre o mesmo
`groupId` e os mesmos `value`, para a escolha de linguagem do leitor valer no site todo:

````mdx
<Tabs groupId="linguagem">
<TabItem value="curl" label="cURL">

```bash
curl -X POST "$LINKTOR_URL/api/v1/messages/send" -H "X-API-Key: $LINKTOR_API_KEY" ...
```

</TabItem>
<TabItem value="typescript" label="TypeScript">

```typescript
// ...
```

</TabItem>
</Tabs>
````

Ordem e valores das abas: `curl` (cURL), `typescript` (TypeScript), `python` (Python), `go` (Go),
`java` (Java), `csharp` (C#), `php` (PHP), `rust` (Rust).

Regra para o conteúdo de cada aba:

- Use o **SDK oficial** quando o método existe em `sdks/<linguagem>` e o corpo que ele envia bate com
  o que o backend espera.
- Quando o SDK ainda não cobre a operação, use o **cliente HTTP nativo** da linguagem (`fetch`,
  `requests`, `net/http`, `java.net.http`, `HttpClient`, Guzzle, `reqwest`) — o exemplo precisa
  funcionar, não parecer bonito.
- Deixe uma linha em branco depois de `<TabItem ...>` e antes de `</TabItem>`; sem ela o bloco de
  código não é interpretado.

## Cuidados com MDX

As páginas `.md` são processadas como MDX. Fora de bloco de código:

- `<` e `{` precisam estar entre crases (`` `<host>` ``, `` `{id}` ``) ou escapados (`\<`, `\{`).
- Links entre páginas são relativos e com extensão: `[Telegram](../canais/telegram.md)`.
- Âncoras explícitas usam `## Título {#ancora}`.

## Rodar o site

```bash
cd web/docs
npm install
npm start        # http://localhost:3000
npm run build    # valida links e âncoras (falha se algum quebrar)
```

| Caminho | Conteúdo |
|---|---|
| `docs/` | Páginas |
| `sidebars.ts` | Navegação por público |
| `src/theme/MDXComponents.tsx` | Componentes globais (`Tabs`, `TabItem`) |
| `static/img/` | Imagens |
