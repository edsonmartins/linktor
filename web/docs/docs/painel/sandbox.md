---
sidebar_position: 15
title: Sandbox
description: Canais de teste (sandbox) do WhatsApp e a allowlist de destinatários — como criar, quem pode receber, como identificar na tela e o que fazer quando um envio é bloqueado.
---

# Sandbox

Um canal **sandbox** é um canal de teste: funciona como um canal comum, mas só entrega mensagens a
números previamente autorizados. Serve para homologar integrações, bots e campanhas sem risco de
uma mensagem de teste chegar a um cliente real. A lista de números autorizados é a **allowlist**.

## Quem pode usar

- Criar um canal sandbox: quem tem acesso à tela **Canais**.
- Gerenciar a allowlist (**Allowlist de Teste**): somente **Administrador** e o dono da conta
  (`owner`).

## Como funciona

- O ambiente é escolhido na **criação** do canal e **não pode ser alterado** depois. Para levar um
  canal de teste a produção, crie um canal novo de produção.
- Todo envio por um canal sandbox consulta a allowlist. Destinatário fora dela: a mensagem não sai e
  fica marcada como falha, com o motivo.
- **Com a allowlist vazia, todo envio sandbox é bloqueado.**
- A regra vale para qualquer origem do envio: painel, API, campanha e bot.
- Canais de produção não são afetados pela allowlist.
- Só o WhatsApp tem sandbox: os tipos **WhatsApp Business** e **WhatsApp (não oficial)**. Um canal
  sandbox de outro tipo, criado pela API, tem todos os envios bloqueados.

:::warning
A allowlist é a única barreira efetiva entre uma mensagem de teste e um destinatário real. A
declaração de "credenciais de teste" pedida na criação é só uma declaração: o Linktor não tem como
verificar se um token é de teste.
:::

## Criar um canal sandbox

1. Em **Canais**, adicione um canal **WhatsApp Business** ou **WhatsApp (não oficial)**.
2. No campo **Ambiente do canal**, escolha **Sandbox — canal de teste/homologação**.
3. Preencha a **Configuração de sandbox**:

   | Campo | Observação |
   |---|---|
   | **Declaro que estas credenciais são do número de TESTE (não de produção)** | Marcação obrigatória |
   | **phone_number_ids de teste declarados** | Só em WhatsApp Business. Lista separada por vírgula dos IDs de número de teste da Meta; o **ID do Número de Telefone** do canal precisa constar nela |

4. Preencha as credenciais do número de teste e clique em **Criar Canal**.
5. Conecte o canal, como em [Conectar um canal](../guias/conectar-um-canal.md).

No canal não oficial, use um número dedicado e descartável: a sessão roda na rede real do WhatsApp,
e o risco de bloqueio do número é o mesmo de qualquer sessão não oficial.

## Gerenciar a allowlist

No menu lateral, clique em **Allowlist de Teste**. A tela **Allowlist de destinatários (sandbox)**
tem o formulário de inclusão e a tabela dos números autorizados.

### Adicionar um destinatário

1. Em **Destinatário (E.164)**, digite o número. Qualquer formato comum é aceito e normalizado para
   o padrão internacional: `+55 11 99999-9999` e `5511999999999` são o mesmo destinatário.
2. Em **Escopo**, escolha **Todos os canais sandbox do tenant** ou um canal sandbox específico.
3. Opcional: preencha **Observação**, por exemplo "número do operador de homologação".
4. Clique em **Adicionar destinatário**.

A tabela mostra o número já normalizado, o escopo, a observação e a data de inclusão.

### Remover um destinatário

1. Clique na lixeira da linha.
2. Confirme em **Remover agora**.

O bloqueio vale a partir do **próximo envio**, imediatamente. Envios sandbox para o número removido
passam a falhar.

Inclusões e remoções ficam registradas em [Auditoria](./observabilidade.md), com autor e valor.

## Como identificar o que é sandbox

| Onde | Indicação |
|---|---|
| Cartão do canal | Selo amarelo **Sandbox** ao lado do nome |
| Lista de conversas | Selo **Sandbox** na linha |
| Conversa aberta | Selo no cabeçalho e faixa amarela fixa: "Conversa de SANDBOX — tráfego sintético de homologação; não é um cliente real" |
| Filtro de conversas | **Ambiente**: **Todos os ambientes**, **Produção** ou **Sandbox** |

A faixa da conversa não pode ser fechada, de propósito.

## Quando um envio é bloqueado

A mensagem aparece na conversa com o alerta vermelho e o quadro **Bloqueado localmente pelo
Linktor**, com o motivo:

| Motivo exibido | O que fazer |
|---|---|
| O destinatário não está na allowlist de sandbox deste tenant | Adicione o número em **Allowlist de Teste** e envie de novo |
| O destinatário não é um número E.164 válido | Corrija o número |
| Este tipo de canal sandbox ainda não tem semântica de entrega definida | Use um canal sandbox de WhatsApp |

O mesmo bloqueio aparece em **Observabilidade** > **Logs**, como "Envio bloqueado por guarda".

## Limitações atuais

- Conversas de sandbox entram nos números de [Analytics](./analytics.md) junto com as de produção.
- Não há limpeza automática dos dados de teste.
- A allowlist controla apenas o **envio**. Qualquer pessoa pode escrever para o número de teste, e a
  mensagem entra na caixa de entrada marcada como sandbox.

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| Nenhuma mensagem de teste sai | Allowlist vazia | Adicione ao menos um destinatário |
| Número adicionado, envio ainda bloqueado | Entrada com escopo em outro canal | Use **Todos os canais sandbox do tenant** ou o canal certo |
| Não consigo mudar o canal para produção | O ambiente é imutável | Crie um canal de produção |
| Erro ao criar: declaração de credencial obrigatória | A marcação não foi feita | Marque a declaração de credenciais de teste |
| Erro ao criar: phone_number_id fora da lista | O ID do canal não está entre os IDs de teste declarados | Inclua o ID na lista |
| **Allowlist de Teste** não aparece no menu | Seu perfil não é administrador | Peça a um administrador |
