---
sidebar_position: 2
title: Conectar um canal
description: Passo a passo genérico para adicionar um canal no painel, testar as credenciais, conectar, habilitar e conferir com uma mensagem de teste.
---

# Conectar um canal

Um canal só envia e recebe mensagens quando está **Habilitado** e **Conectado** ao mesmo tempo. Criar
o canal o deixa habilitado, mas ainda **Desconectado**: falta o passo de conectar. Este guia mostra o
caminho comum a todos os tipos; os campos de cada provedor estão na página do canal, listada no final.

## Antes de começar

- Tenha em mãos as credenciais do provedor (token, ID do número, conta Twilio, etc.). O que criar em
  cada provedor está na página do canal.
- O endereço do Linktor precisa ser acessível pela internet em HTTPS para os canais que recebem por
  webhook (WhatsApp Business, Instagram, Facebook Messenger, Telegram, SMS, RCS, Teams, Slack).
- A tela **Canais** não é restrita por papel: qualquer usuário autenticado consegue criar e alterar
  canais. A exceção é gerar o token do **WhatsApp via Bridge**, que exige administrador.

## 1. Abrir a tela de canais

1. No menu lateral, clique em **Canais**.
2. A tela tem duas seções: **Canais Ativos** (os que já existem) e **Adicionar Novo Canal** (um cartão
   por tipo disponível).

## 2. Escolher o tipo

Em **Adicionar Novo Canal**, clique no cartão do tipo desejado. Um painel lateral abre com o título
**Adicionar** seguido do nome do tipo.

| Cartão | Para quê |
|---|---|
| **Web Chat** | Widget de chat no seu site |
| **WhatsApp Business** | API oficial da Meta (Cloud API) |
| **WhatsApp (não oficial)** | Sessão do WhatsApp no servidor, pareada por QR Code |
| **WhatsApp via Bridge (não oficial)** | Sessão do WhatsApp no computador do atendente |
| **Telegram** | Bot do Telegram |
| **SMS** | SMS via Twilio |
| **Facebook Messenger** | Mensagens de uma Página do Facebook |
| **Instagram** | Mensagens diretas de uma conta profissional |
| **E-mail** | SMTP/IMAP, SendGrid, Mailgun, AWS SES ou Postmark |
| **RCS** | RCS Business Messaging por um provedor |
| **Voz** | Telefonia (Twilio, Vonage, Amazon Connect, Asterisk, FreeSWITCH) |
| **Microsoft Teams**, **Slack**, **Mattermost** | Bots em ferramentas de trabalho |
| **Direto (VendaX)** | Integração com o canal Direto da VendaX |

:::note
Se ao salvar aparecer o erro `channel type "..." is not enabled for this deployment`, o tipo foi
desligado pelo administrador do servidor (variável `LINKTOR_ENABLED_CHANNEL_TYPES`). Veja
[Configuração](../implantar/configuracao.md).
:::

## 3. Preencher o formulário

O formulário é dividido em abas, que variam por tipo. As mais comuns:

| Aba | O que tem |
|---|---|
| **Credenciais** (ou **Configuração**) | **Nome do Canal** e as chaves do provedor |
| **Webhook** | A **URL do Webhook** que você cadastra no provedor e, quando existe, o **Token de Verificação** |
| **Guia** | Resumo dos passos no console do provedor |

1. Dê um **Nome do Canal** que diferencie canais do mesmo tipo (por exemplo, "WhatsApp Vendas"). É
   esse nome que aparece na caixa de entrada.
2. Preencha as credenciais conforme a página do canal.
3. Opcional: preencha o bloco **Webhook de saída** se um sistema seu precisa receber os eventos deste
   canal. Os campos estão descritos em [Canais](../painel/canais.md).

:::note
A **URL do Webhook** de entrada depende do ID do canal. Em vários tipos ela só aparece depois de
salvar ("Será gerada após a criação"): salve, reabra o canal em **Configurar** e copie a URL.
:::

## 4. Testar a conexão

Nos tipos que têm o botão **Testar Conexão** (WhatsApp Business, Telegram, SMS, Facebook Messenger,
Instagram, E-mail, RCS, Voz, Teams, Slack, Mattermost), clique nele antes de salvar. O Linktor chama o provedor
com as credenciais digitadas e mostra **Conexão bem sucedida** ou **Falha na conexão** com o motivo.

O teste valida as credenciais; ele não envia mensagem e não cadastra o webhook.

## 5. Salvar

Clique em **Criar Canal**. O painel fecha e o canal aparece em **Canais Ativos** com os selos
**Habilitado** e **Desconectado**.

## 6. Conectar

1. No cartão do canal, abra o menu de três pontos (**Ações do canal**).
2. Clique em **Conectar**. O selo muda para **Conectado** e aparece o aviso "O canal já pode receber
   mensagens".

Exceções:

- **WhatsApp (não oficial)**: a conexão é feita dentro do próprio canal. Abra **Configurar**, vá à
  aba **Conexão** e use **Conectar com QR Code** (ou **Obter Código de Pareamento**).
- **WhatsApp via Bridge**: a conexão acontece quando o aplicativo Linktor Bridge, no computador do
  atendente, se autentica com o token do canal.

Se o selo ficar em **Erro**, a mensagem do aviso traz o motivo devolvido pelo provedor. Corrija a
credencial em **Configurar** e conecte de novo.

## 7. Cadastrar o webhook no provedor

Para os canais que recebem por webhook, copie a **URL do Webhook** da aba **Webhook** e cadastre no
console do provedor, com o **Token de Verificação** quando houver. O Telegram é exceção: o Linktor
registra o webhook sozinho. Mattermost, E-mail por IMAP e os dois WhatsApp não oficiais
não usam webhook de entrada.

## 8. Conferir se está habilitado

O selo **Habilitado** precisa estar verde. Se estiver **Desabilitado**, abra o menu do cartão e clique
em **Habilitar**. Um canal desabilitado continua conectado, mas o Linktor recusa envio e ignora o que
chega.

## 9. Enviar uma mensagem de teste

1. De um aparelho ou conta que **não** seja a do próprio canal, envie uma mensagem para o canal
   (para o número do WhatsApp, para o bot do Telegram, pelo widget do site…).
2. Abra **Conversas**. A conversa aparece no topo da lista, com o nome do canal no selo.
3. Responda pela caixa **Digite uma mensagem...** e confira se chega no aparelho.
4. Olhe o ícone ao lado do horário da sua resposta: relógio (na fila), um tique (enviada), dois
   tiques (entregue), dois tiques coloridos (lida) ou alerta vermelho (falhou, com o motivo abaixo
   da mensagem).

Se nada chegar, veja **Observabilidade** > **Logs** filtrando pelo canal. Detalhes em
[Observabilidade](../painel/observabilidade.md).

## Problemas comuns

| Sintoma | Causa provável | O que fazer |
|---|---|---|
| Canal criado, mas nada chega | Canal ainda **Desconectado** | Menu do cartão > **Conectar** |
| **Conectado**, mas nada chega | Webhook não cadastrado no provedor, ou URL do Linktor inacessível de fora | Recopie a **URL do Webhook** e confira no console do provedor |
| Mensagens chegam, respostas falham | Credencial sem permissão de envio, ou regra do provedor (janela de 24 h do WhatsApp) | Leia o motivo abaixo da mensagem com falha |
| **Testar Conexão** falha | Token expirado, ID trocado, espaço em branco colado junto | Gere o token de novo e cole sem espaços |
| Canal some depois de **Excluir Canal** junto com as conversas | Comportamento esperado: excluir apaga conversas e mensagens do canal | Para pausar sem perder histórico, use **Desabilitar** |

## Página de cada canal

- [Visão geral dos canais](../canais/visao-geral.md)
- [WhatsApp oficial](../canais/whatsapp-oficial.md)
- [WhatsApp não oficial](../canais/whatsapp-nao-oficial.md)
- [Instagram](../canais/instagram.md)
- [Facebook Messenger](../canais/facebook.md)
- [Telegram](../canais/telegram.md)
- [Web Chat](../canais/webchat.md)
- [E-mail](../canais/email.md)
- [SMS](../canais/sms.md)
- [RCS](../canais/rcs.md)
- [Voz](../canais/voz.md)
- [Slack](../canais/slack.md)
- [Microsoft Teams](../canais/teams.md)
- [Mattermost](../canais/mattermost.md)
- [Direto](../canais/direto.md)
