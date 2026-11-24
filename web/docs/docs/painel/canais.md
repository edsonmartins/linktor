---
sidebar_position: 3
title: Canais
description: A tela de canais do painel — lista, status de conexão, habilitar e desabilitar, conectar, testar, webhook de saída por canal e exclusão.
---

# Canais

**Canais** lista as conexões da empresa com cada meio de mensagem e é onde se cria, configura,
conecta, pausa e remove um canal. Para criar o primeiro, siga
[Conectar um canal](../guias/conectar-um-canal.md); os campos de cada provedor estão na área
[Canais](../canais/visao-geral.md).

## Quem pode usar

A tela não é restrita por papel: qualquer usuário autenticado consegue criar, alterar, conectar e
excluir canais. Gerar o token do **WhatsApp via Bridge** exige **Administrador**. Toda alteração de
canal fica registrada em [Auditoria](./observabilidade.md).

## A tela

- **Canais Ativos**: um cartão por canal existente, com o botão **Atualizar** para recarregar.
- **Adicionar Novo Canal**: um cartão por tipo de canal disponível.

### O cartão do canal

| Elemento | Significado |
|---|---|
| Nome e descrição | Nome dado ao canal e a descrição do tipo |
| Selo **Sandbox** | Canal de teste — veja [Sandbox](./sandbox.md) |
| **Habilitado** / **Desabilitado** | Se o Linktor usa o canal |
| Selo de conexão | Situação da ligação com o provedor (tabela abaixo) |
| Selo de coexistência | Só em **WhatsApp Business** ligado ao aplicativo do celular: situação da coexistência |
| Tipo | Nome do tipo, à direita |
| **ID do canal** | Identificador do canal, com botão **Copiar ID do canal** |

O **ID do canal** é o valor usado nas URLs de webhook, nas chamadas de API e ao pedir suporte.

### Status de conexão

| Selo | Significado |
|---|---|
| **Conectado** | Credenciais aceitas; o canal troca mensagens |
| **Conectando** | Conexão em andamento (por exemplo, aguardando leitura do QR Code) |
| **Desconectado** | Canal criado mas não conectado, ou desconectado manualmente |
| **Erro** | A última tentativa de conexão falhou |

Um canal só envia e recebe quando está **Habilitado** e **Conectado**. Com qualquer um dos dois
faltando, o envio é recusado e o que chega do provedor é ignorado.

## Ações do canal

No menu de três pontos do cartão (**Ações do canal**):

| Item | O que faz |
|---|---|
| **Configurar** | Abre o painel lateral com o formulário do canal |
| **Habilitar** / **Desabilitar** | Liga ou pausa o uso do canal, sem mexer na conexão |
| **Conectar** / **Desconectar** | Abre ou encerra a ligação com o provedor |
| **Excluir Canal** | Remove o canal |

**Conectar** não aparece em **WhatsApp (não oficial)**: nesse tipo a conexão é feita por QR Code
dentro de **Configurar**, na aba **Conexão**.

### Habilitar e desabilitar

Use **Desabilitar** para pausar um canal sem perder nada: conversas e histórico permanecem, e basta
**Habilitar** para voltar. Enquanto desabilitado, as mensagens que o provedor entregar não entram na
caixa de entrada.

### Testar a conexão

Dentro de **Configurar**, os tipos WhatsApp Business, Telegram, SMS, Facebook Messenger, Instagram,
E-mail, RCS, Voz, Teams, Slack e Mattermost têm um botão de teste (**Testar Conexão**). Ele valida as credenciais digitadas
junto ao provedor e mostra **Conexão bem sucedida** ou **Falha na conexão**, sem salvar nada.

### Editar

1. Clique em **Configurar**.
2. Altere o que for preciso. Campos de segredo aparecem vazios: deixar em branco mantém o valor
   atual; digitar substitui.
3. Clique em **Atualizar Canal**.

### Excluir

**Excluir Canal** pede confirmação e apaga o canal **junto com todas as conversas e mensagens
dele**. A ação não pode ser desfeita.

## Webhook de saída por canal

O bloco **Webhook de saída**, presente no formulário de todos os tipos, faz o Linktor avisar um
sistema seu a cada evento do canal. É opcional e independente do webhook de entrada (a URL que o
provedor chama).

| Campo | O que preencher |
|---|---|
| **URL do Consumidor** | Endereço HTTPS do seu sistema que recebe os eventos |
| **Segredo de Assinatura** | Chave compartilhada usada para assinar cada evento, para o seu sistema conferir a origem |
| **Eventos a entregar** | Uma chave por evento; **com nenhuma ligada, todos os eventos são entregues** |

Eventos disponíveis:

| Rótulo | Evento |
|---|---|
| **Mensagem recebida (inbound)** | `message.received` |
| **Mensagem enviada** | `message.sent` |
| **Mensagem entregue** | `message.delivered` |
| **Mensagem lida** | `message.read` |
| **Falha no envio** | `message.failed` |
| **Contato criado** | `contact.created` |
| **Conversa criada** | `conversation.created` |
| **Conversa atribuída** | `conversation.assigned` |
| **Conversa resolvida** | `conversation.resolved` |
| **Conversa reaberta** | `conversation.reopened` |
| **Conversa escalada** | `conversation.escalated` |
| **Chamada atendida** | `call.answered` |
| **Chamada não atendida** | `call.missed` |
| **Chamada recusada** | `call.rejected` |

O segredo nunca é exibido de volta: ao editar, o campo aparece vazio e só é alterado se você digitar
um novo valor. O formato do evento e a verificação da assinatura estão em
[Receber eventos](../api/receber-eventos.md).

## Ambiente do canal

Nos tipos **WhatsApp Business** e **WhatsApp (não oficial)**, o formulário de criação tem o campo
**Ambiente do canal**: **Produção — tráfego real de clientes** ou **Sandbox — canal de
teste/homologação**. A escolha é definitiva; não dá para mudar depois. Veja [Sandbox](./sandbox.md).

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| Canal novo não recebe nada | Está **Desconectado** | **Ações do canal** > **Conectar** |
| Selo **Erro** | Credencial inválida ou provedor fora do ar | Abra **Configurar**, use **Testar Conexão**, corrija e conecte de novo |
| **Conectado** e **Habilitado**, mas nada chega | Webhook de entrada não cadastrado no provedor | Copie a **URL do Webhook** da aba **Webhook** e cadastre no provedor |
| Meu sistema não recebe eventos | **URL do Consumidor** vazia ou inacessível pelo servidor | Confira a URL e os logs em [Observabilidade](./observabilidade.md) |
| Meu sistema recebe eventos que não quero | Nenhum evento ligado significa todos | Ligue apenas os desejados |
| WhatsApp não oficial conecta e desconecta | Versão do cliente recusada pelo WhatsApp | Veja o alerta em [Observabilidade](./observabilidade.md) |
| Erro "channel type ... is not enabled for this deployment" | Tipo desligado no servidor | Fale com quem administra a instalação |

Pela API: ver [Canais](../api/canais.md).
