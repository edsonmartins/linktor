---
sidebar_position: 6
title: Respostas prontas
description: Cadastro de respostas reutilizáveis com atalho, título, conteúdo e tags, e o que está disponível hoje na caixa de mensagem.
---

# Respostas prontas

Respostas prontas (no painel, **Respostas Rápidas**) são textos reutilizáveis para as perguntas que
se repetem — saudação, horário de atendimento, prazo de entrega. Cada uma tem um **atalho** que a
identifica.

## Quem pode usar

Todos os usuários autenticados veem a lista. Criar, editar e excluir exigem a permissão
`canned_responses` (`create`, `update`, `delete`): **Administrador** e **Supervisor** têm por padrão;
**Agente** só lê. Veja [Usuários e papéis](./usuarios-e-papeis.md).

## A tela

No menu lateral, clique em **Respostas Rápidas**. A tabela mostra:

| Coluna | Significado |
|---|---|
| **Atalho** | O identificador, exibido com barra: `/saudacao` |
| **Título** | Descrição curta |
| **Tags** | Etiquetas para organizar |
| **Usos** | Quantas vezes a resposta foi usada |

O campo **Buscar** filtra a lista.

## Criar uma resposta

1. Clique em **Nova resposta**.
2. Preencha:

   | Campo | Obrigatório | Observação |
   |---|---|---|
   | **Atalho** | Sim | Sem a barra e sem espaços, por exemplo `saudacao` |
   | **Título** | Não | |
   | **Conteúdo** | Sim | O texto da resposta. Aceita marcadores como `{{nome}}` para o atendente completar |
   | **Tags** | Não | Separadas por vírgula |

3. Clique em **Criar**.

## Editar e excluir

Na linha da resposta, use o lápis para editar (e depois **Salvar**) ou a lixeira para excluir, com
confirmação.

## Usar em uma conversa

:::warning
A tela descreve as respostas como "inseridas com /atalho", mas a caixa de mensagem de **Conversas**
ainda não faz essa inserção: digitar `/saudacao` envia o texto literal. Por enquanto, copie o
conteúdo da resposta e cole na conversa. Pelo mesmo motivo, a coluna **Usos** só aumenta quando a
resposta é consultada pela API.
:::

Os marcadores `{{...}}` não são substituídos automaticamente: troque-os à mão antes de enviar.

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| Digitar `/atalho` não insere nada | Inserção por atalho ainda não existe na caixa de mensagem | Copie e cole o conteúdo |
| Erro 403 ao criar | Seu papel só tem leitura | Peça a um supervisor ou administrador |
| O cliente recebeu `{{nome}}` | O marcador não foi substituído | Edite o texto antes de enviar |
