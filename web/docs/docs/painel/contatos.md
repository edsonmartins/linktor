---
sidebar_position: 2
title: Contatos
description: Cadastro de contatos do painel — criar, editar, etiquetar, excluir e gerenciar as identidades de cada contato por canal.
---

# Contatos

**Contatos** reúne as pessoas com quem a empresa conversa. Um contato é criado automaticamente na
primeira mensagem recebida de alguém novo, e também pode ser cadastrado à mão. Cada contato pode ter
várias **identidades**: o identificador dele em cada canal.

## Quem pode usar

Todos os usuários autenticados podem ver, criar, editar e excluir contatos. Toda alteração fica
registrada em [Auditoria](./observabilidade.md).

## A tela

No topo há o campo **Buscar contatos...**, o botão de atualizar e **Adicionar Contato**. Abaixo, três
contadores e os cartões dos contatos.

| Contador | O que conta |
|---|---|
| **Total de Contatos** | Contatos exibidos na tela |
| **Com Telefone** | Entre os exibidos, os que têm telefone |
| **Com E-mail** | Entre os exibidos, os que têm e-mail |

:::note
Os contadores consideram apenas os contatos carregados na tela, não a base inteira. O total real da
base aparece no **Dashboard**, em **Total de Contatos**.
:::

Cada cartão mostra nome, e-mail, telefone, um selo por canal em que o contato tem identidade, as tags
e a data de criação.

## Adicionar um contato

1. Clique em **Adicionar Contato**.
2. Preencha os campos e clique em **Adicionar Contato**.

| Campo | Obrigatório | Observação |
|---|---|---|
| **Nome** | Sim | |
| **E-mail** | Não | |
| **Telefone** | Não | Use o formato internacional, por exemplo `+5511999999999` |
| **Tags** | Não | Separadas por vírgula, por exemplo `vip, lead` |

## Ver, editar e excluir

No cartão, abra o menu de três pontos:

| Item | O que faz |
|---|---|
| **Ver detalhes** | Abre a ficha com dados, tags e identidades |
| **Iniciar conversa** | Sem ação na versão atual |
| **Editar Contato** | Abre o formulário com os mesmos campos da criação |
| **Excluir** | Pede confirmação e remove o contato |

## Identidades por canal

Uma identidade liga o contato a um endereço em um canal: o número no WhatsApp, o ID do usuário no
Telegram, o e-mail, o ID de visitante do Web Chat. É por ela que o Linktor reconhece que uma mensagem
nova pertence a um contato que já existe.

Para gerenciar:

1. Abra **Ver detalhes** do contato.
2. Na seção **Identities**, cada linha mostra o tipo de canal e o identificador externo.
3. Para adicionar, escolha o tipo de canal na lista, digite o identificador no campo **Identifier** e
   clique em **Add**.
4. Para remover, clique em **Excluir** ao lado da identidade.

Tipos de canal disponíveis na lista: `webchat`, `whatsapp`, `whatsapp_official`, `telegram`, `sms`,
`instagram`, `facebook`, `rcs`, `email`, `voice`, `teams`, `slack`, `mattermost`.

| Canal | Formato do identificador |
|---|---|
| WhatsApp, SMS, Voz | Telefone em formato internacional |
| E-mail | Endereço de e-mail |
| Telegram, Instagram, Facebook, Slack, Teams, Mattermost | ID do usuário na plataforma (não o nome de usuário) |
| Web Chat | ID de visitante gerado pelo widget |

:::warning
Remover uma identidade faz a próxima mensagem daquele endereço criar um contato novo. Adicionar uma
identidade que pertence a outra pessoa mistura as conversas das duas.
:::

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| A mesma pessoa aparece duas vezes | Escreveu por dois canais, e cada um criou um contato | Mantenha um contato, adicione nele a identidade do outro canal e exclua o duplicado |
| Contato aparece só com o telefone | O canal não informou o nome | Edite e preencha o **Nome** |
| **Iniciar conversa** não faz nada | Ação ainda não implementada no painel | Inicie a conversa pela API ([Enviar mensagens](../api/enviar-mensagens.md)) |
| Contadores não batem com a base | Contam só os contatos exibidos | Consulte o **Dashboard** |

Pela API: ver [Contatos](../api/contatos.md).
