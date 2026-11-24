---
sidebar_position: 7
title: Bots
description: Bots de IA no painel — criar, escolher provedor e modelo, escrever o prompt, vincular a canais e bases de conhecimento, regras de escalação e teste.
---

# Bots

Um bot responde automaticamente às conversas dos canais a que está vinculado, usando um modelo de
linguagem (OpenAI, Anthropic ou Ollama). Ele atende enquanto a conversa não tem responsável humano e
pode passar o atendimento para a equipe.

## Quem pode usar

Todos os usuários autenticados acessam a tela; o código não restringe por papel.

## Antes de começar

O provedor de IA é configurado no servidor, não no painel: quem administra a instalação define a
chave da OpenAI, da Anthropic ou o endereço do Ollama. Sem isso o bot é criado, mas não responde.
Veja [Configuração](../implantar/configuracao.md).

## A tela

No menu lateral, clique em **Bots**. Cada cartão mostra o nome, o provedor e o modelo, o selo
**Ativo** ou **Inativo** e a quantidade de canais vinculados. O menu do cartão tem **Configurar**,
**Ativar** / **Desativar** e **Excluir**. Abaixo, o quadro **Tipos de Bot** descreve os quatro tipos.

## Criar um bot

1. Clique em **Criar Bot**.
2. Preencha:

   | Campo | Opções |
   |---|---|
   | **Nome do Bot** | Nome interno, por exemplo "Bot de Suporte" |
   | **Tipo do Bot** | **Atendimento ao Cliente**, **Vendas**, **FAQ** ou **Personalizado** |
   | **Provedor de IA** | **OpenAI**, **Anthropic** ou **Ollama (Local)** |
   | **Modelo** | Lista fixa por provedor (tabela abaixo) |

3. Clique em **Criar Bot**. O bot nasce **Inativo**.

| Provedor | Modelos oferecidos na tela |
|---|---|
| OpenAI | `gpt-4`, `gpt-4-turbo`, `gpt-3.5-turbo` |
| Anthropic | `claude-3-opus`, `claude-3-sonnet`, `claude-3-haiku` |
| Ollama (Local) | `llama2`, `mistral`, `codellama` |

:::note
Provedor e modelo são escolhidos na criação e aparecem bloqueados na tela de configuração. Para usar
um modelo que não está na lista, crie o bot ou altere o modelo pela API
([Bots e IA](../api/bots-e-ia.md)). O provedor não pode ser trocado depois: crie outro bot.
:::

## Configurar

Clique em **Configurar** no cartão. A tela tem cinco abas; as alterações só valem depois de
**Salvar Alterações** (o botão fica desabilitado enquanto não há mudança).

### Geral

| Campo | Significado |
|---|---|
| **Temperatura** | De 0 a 1. Valores baixos deixam as respostas mais previsíveis; altos, mais variadas. Padrão 0,7 |
| **Máx. Tokens** | Tamanho máximo de cada resposta. Padrão 1024 |
| **Limite de Confiança** | De 0 a 1. Abaixo desse valor a resposta é considerada de baixa confiança. Padrão 0,7 |

### Prompts

| Campo | Significado |
|---|---|
| **Prompt do Sistema** | Instruções fixas: quem o bot é, o que pode e o que não pode responder, tom de voz |
| **Mensagem de Boas-vindas** | Primeira mensagem enviada em conversas novas |
| **Mensagem de Fallback** | Enviada quando o bot não entende |

Um bom prompt diz o nome da empresa, o escopo do atendimento, o que fazer quando não souber (por
exemplo, oferecer um atendente) e o que nunca prometer.

### Canais

Lista todos os canais. Clique em **Atribuir** para o bot atender o canal, ou em **Remover** para
parar. A mudança vale na hora, sem **Salvar Alterações**.

Um canal tem no máximo um bot: atribuir um canal que já está em outro bot o tira do bot anterior.

### Conhecimento

Lista as bases de conhecimento, com **Vincular** e **Desvincular**.

:::warning
Na versão atual o vínculo feito nesta aba não é gravado: a tela envia uma lista de bases, e o
servidor espera uma única base em outro campo. Para vincular uma base a um bot, use a API
([Bots e IA](../api/bots-e-ia.md)).
:::

### Escalação

As **Regras de Escalação** definem quando o bot passa a conversa para a equipe.

| Tipo de Regra | Quando escala |
|---|---|
| **Baixa Confiança** | A confiança da resposta fica abaixo do limite |
| **Sentimento Negativo** | O cliente demonstra insatisfação |
| **Palavra-chave** | A mensagem contém uma das palavras informadas em **Valor** |
| **Intenção** | A intenção detectada é uma das informadas em **Valor** |
| **Solicitação do Usuário** | O cliente pede para falar com uma pessoa |

:::warning
Na versão atual, salvar o bot com regras criadas nesta aba falha com "Invalid request body": o
formato enviado pela tela não é o que o servidor aceita. Além disso, a aba abre vazia mesmo quando o
bot já tem regras, e salvar outra alteração nessas condições apaga as regras existentes. Enquanto
isso não é corrigido, cadastre as regras pela API ([Bots e IA](../api/bots-e-ia.md)) e evite salvar
este bot pelo painel.
:::

## Ativar e desativar

Use **Ativar** no cartão ou na tela de configuração. Para o bot responder, é preciso que:

1. o bot esteja **Ativo**;
2. o canal esteja atribuído a ele, habilitado e conectado;
3. a conversa **não** tenha responsável humano.

Atribuir uma conversa a uma pessoa tira o bot daquela conversa. Veja [Conversas](./conversas.md).

Se houver um fluxo ativo cujo gatilho combine com a mensagem, o bot executa o fluxo em vez de gerar
uma resposta por IA. Veja [Fluxos](./fluxos.md).

## Testar

1. Na tela de configuração, clique em **Testar Bot**.
2. Digite uma mensagem como um cliente escreveria e envie.
3. O resultado mostra a **Resposta**, a **Confiança**, a **Intenção** detectada e **Escalar?**.

O teste não envia nada a clientes e não cria conversa.

## Excluir

Em **Zona de Perigo**, no fim da tela de configuração, ou em **Excluir** no cartão. A exclusão é
permanente.

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| Bot ativo não responde | Canal não atribuído, canal desconectado, ou conversa com responsável | Confira a aba **Canais** e a atribuição da conversa |
| Bot não responde em nenhum canal | Provedor de IA sem chave no servidor | Fale com quem administra a instalação |
| **Testar Bot** retorna erro | Chave inválida, modelo indisponível no provedor ou Ollama fora do ar | Confira o provedor e o modelo |
| Erro "Invalid request body" ao salvar | Há regra de escalação criada pela tela | Remova as regras da aba e cadastre pela API |
| O bot respondia e parou | A conversa foi atribuída a uma pessoa, ou o canal foi atribuído a outro bot | Verifique os dois |
| Respostas fora do assunto | **Prompt do Sistema** vago | Delimite o escopo e diga o que recusar |

Pela API: ver [Bots e IA](../api/bots-e-ia.md).
