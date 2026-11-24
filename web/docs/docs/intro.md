---
sidebar_position: 1
slug: /
title: Linktor
description: Plataforma omnichannel de conversas — um painel de atendimento e uma API única para WhatsApp, Instagram, Telegram, e-mail, SMS, WebChat e mais.
---

# Linktor

> Plataforma **omnichannel** de conversas, de código aberto. Um **painel de atendimento** para a equipe
> e **uma API** para os seus sistemas — sobre WhatsApp, Instagram, Messenger, Telegram, WebChat, e-mail,
> SMS, RCS, voz, Slack, Teams e Mattermost.

Com o Linktor você conecta cada canal uma vez e, a partir daí, trata tudo do mesmo jeito: a mensagem
do cliente vira uma **conversa** no painel, a sua aplicação recebe um **evento** no webhook, e a
resposta sai pela mesma chamada de API — qualquer que seja o canal.

- **Um contrato para todos os canais.** Enviar por WhatsApp, Telegram ou e-mail é a mesma chamada
  (`POST /messages/send`); só mudam o canal e o destinatário.
- **Painel de atendimento.** Caixa de entrada única, atribuição, respostas rápidas, templates,
  campanhas, papéis e permissões.
- **Bots e IA.** Bots com modelos de linguagem, fluxos visuais, base de conhecimento e escalação para
  humano.
- **Feito para integrar.** API HTTP, webhooks assinados, WebSocket, servidor MCP e SDKs em sete
  linguagens.
- **Sob o seu controle.** Auto-hospedado, multi-tenant, credenciais de canal cifradas.

## Para quem é esta documentação

| Você é… | Comece por |
|---|---|
| Novo no Linktor | [Primeiros passos](guias/primeiros-passos.md) e [Conceitos](conceitos/visao-geral.md) |
| Atendente ou supervisor | [Atender uma conversa](guias/atender-uma-conversa.md) e [Painel → Conversas](painel/conversas.md) |
| Administrador que vai ligar um canal | [Conectar um canal](guias/conectar-um-canal.md) e a página do canal em [Canais](canais/visao-geral.md) |
| Desenvolvedor integrando um sistema | [API → Visão geral](api/visao-geral.md), [Enviar mensagens](api/enviar-mensagens.md) e [Receber eventos](api/receber-eventos.md) |
| Desenvolvedor em uma linguagem específica | [SDKs](sdks/visao-geral.md) — TypeScript, Python, Go, Java, C#, PHP e Rust |
| Quem instala e opera o servidor | [Instalação](implantar/instalacao.md) e [Produção](implantar/producao.md) |

## Canais

| Canal | Como conecta | Página |
|---|---|---|
| WhatsApp (API oficial da Meta) | App na Meta + token, ou Embedded Signup | [WhatsApp oficial](canais/whatsapp-oficial.md) |
| WhatsApp (não oficial) | QR code ou código de pareamento | [WhatsApp não oficial](canais/whatsapp-nao-oficial.md) |
| Instagram Direct | App na Meta + conta profissional | [Instagram](canais/instagram.md) |
| Facebook Messenger | App na Meta + Página | [Facebook](canais/facebook.md) |
| Telegram | Bot criado no BotFather | [Telegram](canais/telegram.md) |
| WebChat | Snippet no seu site | [WebChat](canais/webchat.md) |
| E-mail | Provedor de e-mail | [E-mail](canais/email.md) |
| SMS | Provedor de SMS | [SMS](canais/sms.md) |
| RCS | Provedor de RCS | [RCS](canais/rcs.md) |
| Voz (parcial: só chamadas recebidas) | Provedor de telefonia | [Voz](canais/voz.md) |
| Slack | Slack App | [Slack](canais/slack.md) |
| Microsoft Teams | Bot no Azure | [Teams](canais/teams.md) |
| Mattermost | Bot no servidor Mattermost | [Mattermost](canais/mattermost.md) |
| Direto | Integração própria | [Direto](canais/direto.md) |

A comparação do que cada canal suporta está em [Canais → Visão geral](canais/visao-geral.md).

## Mapa da documentação

- **[Usar](guias/primeiros-passos.md)** — o painel: conversas, contatos, canais, templates, campanhas,
  bots, fluxos, base de conhecimento, equipe e configurações.
- **[Canais](canais/visao-geral.md)** — para cada canal: o que criar no provedor, o que preencher no
  Linktor, a URL do webhook e exemplos de código.
- **[Desenvolver](api/visao-geral.md)** — API, autenticação, envio, eventos, mídia, tempo real e MCP.
- **[SDKs](sdks/visao-geral.md)** — referência por linguagem.
- **[Implantar](implantar/arquitetura.md)** — arquitetura, instalação, produção, configuração,
  monitoramento, backup e segurança.

> **Nota de precisão:** esta documentação é escrita a partir do código. Onde uma capacidade é parcial
> ou depende de configuração, a página diz isso explicitamente.
