---
sidebar_position: 5
title: Campanhas
description: Envio em massa por modelo de mensagem — criar a campanha, informar destinatários, iniciar, acompanhar, reenviar falhas e cancelar.
---

# Campanhas

Uma campanha envia o mesmo modelo de mensagem para uma lista de telefones, com os parâmetros de cada
destinatário. A tela **Campanhas** cria a campanha, dispara o envio e mostra o andamento por
destinatário.

## Quem pode usar

Qualquer usuário autenticado vê as campanhas. Criar exige a permissão `campaigns` / `create`;
iniciar, reenviar falhas e cancelar exigem `campaigns` / `update`. Por padrão, **Administrador** e
**Supervisor** têm essas permissões e **Agente** não. Veja [Usuários e papéis](./usuarios-e-papeis.md).

## Antes de começar

- Um canal **WhatsApp Business** ou **Telegram** conectado e habilitado. O formulário só lista esses
  dois tipos.
- Para WhatsApp Business: um modelo **Aprovado**, com o nome técnico e o idioma em mãos. Veja
  [Modelos de mensagem](./modelos-de-mensagem.md).

:::warning
O envio é montado como modelo do WhatsApp (nome, idioma e parâmetros do corpo). O formulário aceita
canais do Telegram, mas o Telegram não tem modelos; valide com uma campanha de um destinatário antes
de usar em volume.
:::

## Criar uma campanha

1. No menu lateral, clique em **Campanhas** e depois em **Nova campanha**.
2. Preencha:

   | Campo | O que informar |
   |---|---|
   | **Nome** | Nome interno da campanha |
   | **Canal** | Canal por onde sai o envio |
   | **Template** | Nome técnico do modelo, digitado exatamente como na Meta (`welcome_v1`) |
   | **Idioma** | Código do idioma do modelo; padrão `pt_BR` |
   | **Destinatários** | Um por linha, no formato `telefone,param1,param2` |

3. Clique em **Criar campanha**. A campanha é criada em rascunho e a tela de detalhe abre.

### Formato dos destinatários

- Um destinatário por linha.
- O primeiro valor é o telefone em formato internacional, por exemplo `+5511999999999`.
- Os valores seguintes, separados por vírgula, preenchem as variáveis do **corpo** do modelo, na
  ordem: o primeiro vira `{{1}}`, o segundo `{{2}}`, e assim por diante.
- O contador ao lado de **Destinatários** mostra quantas linhas foram reconhecidas. Linhas sem
  telefone são ignoradas, com o aviso "linha(s) sem telefone foram ignoradas".

:::note
O nome do modelo é digitado à mão e não é conferido na criação. Um nome ou idioma errado só aparece
como falha no envio. A lista não remove telefones repetidos, e um parâmetro não pode conter vírgula.
Variáveis de cabeçalho e de botão não são preenchidas pela campanha.
:::

## Iniciar

Na tela de detalhe, clique em **Iniciar**. O status passa a `processing` e as mensagens entram na
fila de envio em lotes. O botão só aparece enquanto a campanha está em `draft`.

:::warning
Iniciar dispara mensagens cobradas pela Meta e não há confirmação adicional. Confira o modelo e a
lista antes.
:::

## Acompanhar

A lista de campanhas atualiza sozinha a cada 5 segundos; o detalhe, a cada 2 segundos enquanto a
campanha está processando.

### Status da campanha

| Status | Significado |
|---|---|
| `draft` | Criada, ainda não iniciada |
| `scheduled` | Agendada. O estado existe, mas o agendamento ainda não está implementado: nenhuma campanha chega a ele hoje |
| `processing` | Envio em andamento |
| `completed` | Todos os destinatários terminaram em enviado ou falha |
| `failed` | A campanha não pôde ser processada |
| `cancelled` | Cancelada |

### Contadores

| Contador | Significado |
|---|---|
| **Total** | Destinatários da campanha |
| **Enviadas** | Aceitas pelo provedor |
| **Entregues** | Com confirmação de entrega |
| **Lidas** | Com confirmação de leitura |
| **Falhas** | Não enviadas |

A barra de **Progresso** é (enviadas + falhas) ÷ total.

### Destinatários

A tabela **Destinatários** lista telefone, status, número de tentativas (coluna `#`) e o **Erro**
quando há. Use o filtro para ver só um status; são 50 linhas por página.

| Status | Significado |
|---|---|
| `pending` | Ainda não entrou na fila |
| `queued` | Na fila de envio |
| `sent` | Aceita pelo provedor |
| `delivered` | Entregue |
| `read` | Lida |
| `failed` | Falhou; a coluna **Erro** traz o motivo |

Um destinatário que fica em `queued` por mais de 15 minutos sem confirmação é marcado como `failed`
por uma rotina que roda a cada 5 minutos.

## Reenviar falhas

Com pelo menos uma falha e a campanha fora de `processing`, aparece **Reenviar falhas**. Ele recoloca
na fila apenas os destinatários com `failed`. Corrija a causa antes (modelo, canal, número).

## Cancelar

Durante o processamento, clique em **Cancelar** e confirme em **Sim, cancelar campanha**. Os
destinatários ainda não enviados são interrompidos; o que já saiu não é desfeito. Uma campanha
concluída ou já cancelada não pode ser cancelada.

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| Canal não aparece no formulário | Só WhatsApp Business e Telegram são listados | Use um canal desses tipos |
| **Criar campanha** desabilitado | Falta nome, canal, modelo ou nenhum destinatário válido | Confira os campos e o contador de destinatários |
| Todas falham | Modelo inexistente, não aprovado ou idioma errado | Veja a coluna **Erro**; sincronize os modelos |
| Falha com número de parâmetros | Quantidade de valores na linha difere das variáveis do modelo | Ajuste a lista e crie outra campanha |
| Falhas de sandbox / allowlist | Canal de teste e destinatário fora da allowlist | Veja [Sandbox](./sandbox.md) |
| Erro 403 ao criar ou iniciar | Seu papel não tem a permissão de campanhas | Peça a um administrador |
| Campanha parada em `processing` | Fila de envio parada | Veja **Filas de Mensagens** em [Observabilidade](./observabilidade.md) |

Pela API: ver [Modelos e campanhas](../api/modelos-e-campanhas.md).
