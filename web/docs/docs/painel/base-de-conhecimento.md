---
sidebar_position: 9
title: Base de conhecimento
description: Bases de conhecimento no painel — criar bases, cadastrar e importar itens de pergunta e resposta, testar a busca semântica e regenerar embeddings.
---

# Base de conhecimento

Uma base de conhecimento guarda pares de pergunta e resposta que o bot consulta para responder com
informação da empresa, em vez de improvisar. A busca é semântica: encontra a resposta pelo sentido da
pergunta, não só por palavras iguais.

## Quem pode usar

Todos os usuários autenticados acessam a tela; o código não restringe por papel.

## Antes de começar

A busca semântica depende de **embeddings**, gerados por um provedor de IA configurado no servidor
(por padrão, OpenAI). Sem o provedor, os itens são gravados, mas **Regenerar Embeddings** falha com
"embedding service not available" e a busca não encontra nada. Veja
[Configuração](../implantar/configuracao.md).

## A lista de bases

No menu lateral, clique em **Base de Conhecimento**. Cada cartão mostra nome, tipo, situação,
quantidade de itens e a última sincronização. Filtros: **Buscar bases de conhecimento...**, **Todos
os Tipos** e **Todos os Status**.

O menu do cartão tem **Ver Itens**, **Testar Busca**, **Editar**, **Regenerar Embeddings** e
**Excluir**.

| Status | Significado |
|---|---|
| **Ativo** | Pronta para consulta |
| **Sincronizando** | Embeddings sendo gerados |
| **Erro** | A última geração de embeddings falhou |

## Criar uma base

1. Clique em **Nova Base de Conhecimento**.
2. Preencha:

   | Campo | Observação |
   |---|---|
   | **Nome** | Obrigatório, até 100 caracteres |
   | **Descrição** | Até 500 caracteres |
   | **Tipo** | **FAQ**, **Documentos** ou **Website**. Não muda depois |

3. Clique em **Criar**.

:::note
Os três tipos aceitam o mesmo conteúdo: itens de pergunta e resposta cadastrados à mão ou
importados. Não há, na versão atual, envio de arquivos PDF/Word nem rastreamento de site; os tipos
**Documentos** e **Website** servem apenas para classificar a base.
:::

## Itens

Clique em **Ver Itens**. A tela da base (com rótulos em inglês na versão atual) mostra quatro
indicadores — **Total Items**, **With Embeddings**, **Last Synced** e **Created** — e a tabela de
itens com **Question**, **Answer**, **Keywords**, **Source** e **Embedding** (se o item já tem
embedding).

### Adicionar um item

1. Clique em **Add Item**.
2. Preencha:

   | Campo | Observação |
   |---|---|
   | **Pergunta** | Obrigatória, até 1.000 caracteres. Escreva como o cliente perguntaria |
   | **Resposta** | Obrigatória, até 10.000 caracteres |
   | **Palavras-chave (opcional)** | Digite e pressione Enter para cada uma |
   | **Fonte (opcional)** | De onde veio a informação, por exemplo "Central de Ajuda" |

3. Clique em **Adicionar Item**.

Para alterar ou remover, use as ações na linha do item.

### Importar em massa

1. Clique em **Bulk Import**.
2. Envie um arquivo `.json` ou `.csv`, ou cole o conteúdo na aba **JSON** ou **CSV**.
3. Clique em **Validar**. A tela mostra quantos itens foram reconhecidos e os erros por linha.
4. Clique em **Importar**.

| Formato | Regras |
|---|---|
| JSON | Uma lista de itens com `question` e `answer` obrigatórios; `keywords` (lista) e `source` opcionais |
| CSV | Primeira linha com os nomes das colunas; `question` e `answer` obrigatórias; `keywords` e `source` opcionais. Várias palavras-chave na mesma célula são separadas por ponto e vírgula |

O limite é de 100 itens por importação. Divida arquivos maiores.

## Testar a busca

1. Na base, clique em **Test Search** (ou **Testar Busca** no cartão).
2. Em **Consulta**, digite uma pergunta como o cliente faria.
3. Ajuste o **Limite de Resultados**, se quiser, e clique em **Buscar**.

Os resultados vêm ordenados por similaridade, de 0 a 100%. Use esta tela para descobrir perguntas
sem boa resposta e criar itens para elas.

## Regenerar embeddings

**Regenerar Embeddings** recalcula o embedding de todos os itens da base. Use quando:

- a coluna **Embedding** mostra itens sem embedding (por exemplo, itens criados enquanto o provedor
  estava indisponível);
- a busca não devolve itens que existem;
- o provedor ou o modelo de embeddings do servidor mudou.

Durante o processo a base fica em **Sincronizando**. Em bases grandes, leva alguns minutos e consome
créditos do provedor.

## Vincular a um bot

Uma base só é consultada pelo bot ao qual está vinculada. O vínculo pela aba **Conhecimento** do bot
não é gravado na versão atual; use a API. Veja [Bots](./bots.md).

## Editar e excluir uma base

**Editar** altera nome e descrição. **Excluir** remove a base **e todos os itens dela**, após
confirmação.

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| Busca não devolve nada | Itens sem embedding | **Regenerar Embeddings** |
| "embedding service not available" | Provedor de IA não configurado no servidor | Fale com quem administra a instalação |
| Base presa em **Erro** | A geração falhou (chave inválida, limite do provedor) | Corrija a causa e regenere |
| Importação CSV recusa o arquivo | Faltam as colunas `question` e `answer` no cabeçalho | Renomeie as colunas |
| "Máximo de 100 itens por importação" | Arquivo grande demais | Divida em partes |
| O bot não usa a base | Base não vinculada ao bot | Veja [Bots](./bots.md) |

Pela API: ver [Bots e IA](../api/bots-e-ia.md).
