---
sidebar_position: 1
title: Primeiros passos
description: Do primeiro login até uma mensagem enviada e uma resposta recebida.
---

# Primeiros passos

Este tutorial leva você do primeiro acesso até uma conversa funcionando: um canal conectado, uma
mensagem recebida no painel e uma mensagem enviada pela API. Usa o **Telegram**, porque é o canal mais
rápido de criar — o roteiro é o mesmo para os outros.

Você precisa de um Linktor no ar. Se ainda não tem, siga [Instalação](../implantar/instalacao.md).

## 1. Entre no painel

Abra o endereço do seu Linktor e entre com o seu e-mail e senha. O menu lateral traz as áreas do
produto: **Conversas**, **Contatos**, **Canais**, **Bots**, **Fluxos**, **Templates**, **Campanhas**,
**Analytics** e **Configurações**, entre outras. O que aparece depende do seu papel.

## 2. Crie o bot no Telegram

1. No Telegram, abra uma conversa com **@BotFather**.
2. Envie `/newbot`, escolha um nome e um nome de usuário terminado em `bot`.
3. Copie o **token** que o BotFather devolve.

O passo a passo completo está em [Telegram](../canais/telegram.md).

## 3. Conecte o canal no Linktor

1. No painel, abra **Canais**, clique em **Adicionar Canal** e escolha **Telegram**.
2. Dê um nome e cole o token.
3. Salve e depois **conecte** o canal — todo canal nasce desconectado, e é ao conectar que o Linktor
   registra o webhook no Telegram. O canal deve aparecer como **Conectado**.

:::note O servidor precisa ser alcançável pela internet
O Telegram entrega as mensagens chamando o seu Linktor por HTTPS. Em uma instalação local, exponha o
servidor por um túnel e ajuste o endereço público — ver [Telegram](../canais/telegram.md).
:::

Detalhes e variações em [Conectar um canal](conectar-um-canal.md).

## 4. Receba a primeira mensagem

No Telegram, procure o seu bot pelo nome de usuário e envie um "oi". Em **Conversas**, uma conversa
nova aparece com a mensagem. Responda por ali — a resposta chega no Telegram.

O dia a dia do atendimento está em [Atender uma conversa](atender-uma-conversa.md).

## 5. Envie pela API

Para a sua aplicação enviar mensagens, gere uma chave:

1. Gere uma chave de API com o scope `messages:send` — o caminho na tela está em
   [Chaves de API](../painel/chaves-de-api.md).
2. Copie a chave (`lk_...`). **Ela é exibida uma única vez.**

Agora envie. `CHANNEL_ID` é o id do canal criado no passo 3; `to` é o id do chat do Telegram que
escreveu para o bot (aparece na ficha do contato).

```bash
export LINKTOR_URL="https://linktor.suaempresa.com"
export LINKTOR_API_KEY="lk_sua_chave"

curl -X POST "$LINKTOR_URL/api/v1/messages/send" \
  -H "X-API-Key: $LINKTOR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "channel_id": "CHANNEL_ID",
    "to": "123456789",
    "content_type": "text",
    "text": "Olá! Esta mensagem saiu pela API."
  }'
```

A resposta `202` significa que a mensagem foi **enfileirada**. O mesmo exemplo em TypeScript, Python,
Go, Java, C#, PHP e Rust está em [Enviar mensagens](../api/enviar-mensagens.md).

## 6. Receba eventos na sua aplicação

Para saber quando o cliente responde — e quando a sua mensagem foi entregue ou lida — configure o
**webhook de saída** do canal apontando para um endereço da sua aplicação. No canal Telegram isso é
feito pela API (`webhook_url` e `credentials.webhook_secret` do canal); a tela ainda não tem esses campos. O Linktor passa a enviar
eventos como `message.received` e `message.delivered`.

:::warning Defina o segredo do webhook
Sem um segredo configurado no canal, o Linktor **não entrega** os eventos — e não avisa. Preencha a
URL e o segredo juntos.
:::

Como configurar, o formato de cada evento e como validar a assinatura estão em
[Receber eventos](../api/receber-eventos.md).

## E agora?

- Ligar os canais de produção: [Canais](../canais/visao-geral.md).
- Convidar a equipe: [Usuários e papéis](../painel/usuarios-e-papeis.md).
- Automatizar o atendimento: [Bots](../painel/bots.md) e [Fluxos](../painel/fluxos.md).
- Integrar na sua linguagem: [SDKs](../sdks/visao-geral.md).
