---
sidebar_position: 12
title: Chaves de API
description: Gerar, guardar e revogar chaves de API no painel, com a lista de permissões (scopes) que cada chave pode receber.
---

# Chaves de API

Uma chave de API permite que um sistema externo use o Linktor sem login de usuário: enviar
mensagens, ler conversas, criar canais. Cada chave pertence à sua organização e carrega uma lista de
permissões (scopes) que limita o que ela alcança.

## Quem pode usar

Somente **Administrador** e o dono da conta (`owner`). Para os demais perfis a seção aparece, mas a
lista não carrega e gerar chave retorna erro de permissão.

## Gerar uma chave

1. No menu lateral, clique em **Configurações** e depois em **Chaves de API**.
2. Em **Nome da chave**, descreva o uso, por exemplo "Integração de produção". Sem nome, a chave
   recebe "API Key" seguido da data.
3. Em **Permissões**, escolha:
   - **Acesso total** ligado: a chave recebe `*` e pode fazer tudo; ou
   - **Acesso total** desligado: ligue apenas as permissões necessárias. É preciso pelo menos uma.
4. Clique em **Gerar nova chave**.
5. A chave aparece em um quadro, começando por `lk_`. Clique em **Copiar** e guarde em um cofre de
   senhas.

:::warning
A chave completa é exibida **uma única vez**. Depois de sair da tela, só o prefixo fica visível. Se
perder a chave, revogue e gere outra.
:::

## Permissões disponíveis

| Permissão | Libera |
|---|---|
| `channels:read` | Listar e consultar canais |
| `channels:write` | Criar, alterar, conectar, desconectar e excluir canais |
| `messages:send` | Enviar mensagens, reações e anexos |
| `conversations:read` | Listar conversas e ler mensagens |
| `conversations:write` | Criar, atribuir, resolver, reabrir, escalar e excluir conversas |
| `contacts:read` | Listar e consultar contatos |
| `contacts:write` | Criar, alterar e excluir contatos e identidades |
| `bots:read` / `bots:write` | Consultar / gerenciar bots |
| `flows:read` / `flows:write` | Consultar / gerenciar fluxos |
| `templates:read` / `templates:write` | Consultar / gerenciar modelos de mensagem |
| `knowledge:read` / `knowledge:write` | Consultar / gerenciar bases de conhecimento |
| `orders:read` / `orders:write` | Consultar / alterar pedidos |
| `analytics:read` | Ler os números de analytics |
| `ai:use` | Usar os recursos de IA (completar, classificar intenção, analisar sentimento) |

Regras:

- Uma permissão `:write` inclui a `:read` do mesmo recurso.
- **Acesso total** equivale a `*`.
- Uma chave para um sistema que só envia mensagens precisa de `messages:send`. Para responder dentro
  de uma conversa existente ou enviar anexo por ela, some `conversations:write`.

:::note
As permissões seguem o princípio do menor privilégio, mas não isolam completamente: algumas rotas do
servidor não têm verificação de permissão e ficam acessíveis a qualquer chave válida. Para separar
de fato dois sistemas, use organizações diferentes. As áreas administrativas (equipe, papéis,
auditoria, observabilidade, chaves de API) não aceitam chave de API. Criar, iniciar, reenviar e
cancelar campanhas e gravar respostas prontas também não: essas ações dependem de permissão de papel
de usuário, que uma chave de API não tem.
:::

## Chaves existentes

Abaixo do formulário, cada chave aparece com o nome, o **Prefixo**, a data de criação e a lista de
permissões. Use o prefixo para descobrir qual chave um sistema está usando.

## Revogar

Clique em **Revoke** na linha da chave. A revogação é imediata e **não pede confirmação**: o sistema
que usa a chave passa a receber erro de autenticação na chamada seguinte.

Para trocar uma chave sem parar a integração:

1. Gere a chave nova.
2. Atualize o sistema externo para usá-la.
3. Confirme que está funcionando.
4. Revogue a antiga.

## Boas práticas

- Uma chave por sistema, com nome que diga quem usa.
- Só as permissões necessárias; evite **Acesso total** em produção.
- Nunca coloque a chave em código-fonte, em página web ou em aplicativo distribuído a clientes.
- Revogue na hora qualquer chave que tenha aparecido em log, conversa ou repositório.

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| "Não foi possível gerar a chave de API" | Seu perfil não é administrador | Peça a um administrador |
| **Gerar nova chave** desabilitado | **Acesso total** desligado e nenhuma permissão marcada | Marque pelo menos uma |
| Integração recebe 401 | Chave revogada, digitada errada ou enviada no cabeçalho errado | Confira o cabeçalho `X-API-Key` |
| Integração recebe 403 | A chave não tem a permissão da rota | Gere outra com a permissão que falta |
| Perdi a chave | Ela só é exibida na criação | Revogue e gere outra |

Pela API: ver [Autenticação](../api/autenticacao.md).
