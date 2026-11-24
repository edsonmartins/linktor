---
sidebar_position: 2
title: Glossário
description: Termos usados no painel, na API e nesta documentação.
---

# Glossário

| Termo | Significado |
|---|---|
| **Adaptador** | O componente do Linktor que fala o protocolo de um canal específico. |
| **Allowlist de teste** | Lista de destinatários autorizados a receber mensagens enquanto o ambiente está em modo sandbox. Ver [Sandbox](../painel/sandbox.md). |
| **API key** | Ver *Chave de API*. |
| **Atribuição** | Definir qual usuário é o responsável por uma conversa. |
| **Base de conhecimento** | Conjunto de conteúdos que os bots consultam para responder. Ver [Base de conhecimento](../painel/base-de-conhecimento.md). |
| **Bot** | Atendente automático baseado em modelo de linguagem. Ver [Bots](../painel/bots.md). |
| **Campanha** | Envio da mesma mensagem para uma lista de destinatários. Ver [Campanhas](../painel/campanhas.md). |
| **Canal** | Conexão com um meio de comunicação (um número, um bot, uma caixa de e-mail). Ver [Canais](../canais/visao-geral.md). |
| **Chave de API** | Credencial de sistema, no formato `lk_...`, enviada no cabeçalho `X-API-Key`. Ver [Chaves de API](../painel/chaves-de-api.md). |
| **Coexistência** | Uso simultâneo do aplicativo WhatsApp Business e da API oficial no mesmo número. Ver [WhatsApp oficial](../canais/whatsapp-oficial.md). |
| **`config`** | Parte não secreta da configuração de um canal. |
| **Contato** | A pessoa atendida. Reúne uma ou mais identidades. |
| **Conversa** | Fio de mensagens entre um contato e a empresa em um canal. |
| **`credentials`** | Parte secreta da configuração de um canal (tokens, segredos). É cifrada ao gravar e nunca é devolvida pela API. |
| **Envio direto** | Enviar informando canal e destinatário, sem precisar conhecer a conversa. Ver [Enviar mensagens](../api/enviar-mensagens.md). |
| **Escalação** | Passagem de uma conversa do bot para um atendente humano. |
| **Evento** | Aviso enviado pelo Linktor à sua aplicação quando algo acontece. Ver [Receber eventos](../api/receber-eventos.md). |
| **Fluxo** | Roteiro visual de atendimento. Ver [Fluxos](../painel/fluxos.md). |
| **Identidade** | O identificador de um contato em um canal: telefone, usuário, e-mail. |
| **Idempotência** | Garantia de que repetir a mesma chamada não envia a mensagem duas vezes (`idempotency_key`). |
| **Janela de atendimento** | Período, definido pelo provedor, em que se pode responder livremente a um contato. No WhatsApp oficial são 24 horas desde a última mensagem do cliente; fora dela só templates. |
| **MCP** | Model Context Protocol — permite que assistentes de IA operem o Linktor por ferramentas. Ver [MCP](../mcp/visao-geral.md). |
| **Papel** | Conjunto de permissões de um usuário. |
| **Resposta rápida** | Texto pronto acionado por atalho durante o atendimento. Ver [Respostas rápidas](../painel/respostas-prontas.md). |
| **Sandbox** | Modo de teste em que os envios são restritos à allowlist. |
| **Scope** | Permissão de uma chave de API, no formato `recurso:ação` (por exemplo `messages:send`). |
| **Template** | Mensagem pré-aprovada pela Meta, obrigatória para iniciar conversa no WhatsApp oficial fora da janela de atendimento. Ver [Templates](../painel/modelos-de-mensagem.md). |
| **Tenant** | A organização dona dos dados. Isola canais, contatos, conversas e usuários. |
| **VRE** | Visual Response Engine — gera respostas visuais (imagens) a partir de modelos. |
| **Webhook de entrada** | Chamada do provedor para o Linktor. |
| **Webhook de saída** | Chamada do Linktor para a sua aplicação. |
