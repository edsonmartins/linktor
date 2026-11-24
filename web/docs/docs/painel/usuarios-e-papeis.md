---
sidebar_position: 11
title: Usuários e papéis
description: Gestão da equipe no painel — criar e desativar usuários, os perfis de acesso, o que cada um pode fazer e a tela de papéis com o catálogo de permissões.
---

# Usuários e papéis

A tela **Equipe** cadastra as pessoas que entram no painel e define o perfil de cada uma. A tela
**Papéis** mostra e edita os conjuntos de permissões. As duas ficam no menu lateral e só aparecem
para administradores.

## Quem pode usar

Somente os perfis **Administrador** e o dono da conta (`owner`). Quem não é administrador e digita o
endereço dessas telas vê **Acesso negado**.

## Perfis

Todo usuário tem um perfil. O painel oferece três; um quarto, `owner`, pertence ao dono da conta
criado junto com a organização e tem acesso irrestrito.

| Perfil | Descrição na tela |
|---|---|
| **Administrador** | Acesso total a todos os recursos |
| **Supervisor** | Gerenciar equipe e ver relatórios |
| **Agente** | Atender conversas |

### O que cada perfil alcança

| Área | Agente | Supervisor | Administrador / owner |
|---|---|---|---|
| Conversas, contatos, canais, bots, fluxos, base de conhecimento, modelos, analytics, VRE | Sim | Sim | Sim |
| Excluir conversa | Não | Sim | Sim |
| Criar, editar e excluir respostas prontas | Não | Sim | Sim |
| Criar, iniciar, reenviar e cancelar campanhas | Não | Sim | Sim |
| Tela **Operações** | Ver | Ver | Ver e salvar |
| **Equipe**, **Papéis**, **Auditoria**, **Observabilidade**, **Allowlist de Teste** | Não | Não | Sim |
| **Chaves de API** e dados da **Organização** | Não | Não | Sim |
| Gerar token do WhatsApp via Bridge | Não | Não | Sim |

:::note
Esta tabela descreve o que o servidor efetivamente bloqueia hoje. As telas de canais, bots, fluxos,
modelos e base de conhecimento não têm restrição por perfil: um **Agente** consegue, por exemplo,
excluir um canal. Leve isso em conta ao distribuir acessos.
:::

## Equipe

No menu lateral, clique em **Equipe**. No topo há quatro contadores (**Total de Membros**, **Ativo**,
**Administradores**, **Agentes**), a busca **Buscar usuários...** e o filtro **Filtrar por perfil**.
Cada cartão mostra nome, e-mail, perfil e situação (**Ativo** ou **Inativo**).

### Adicionar um membro

1. Clique em **Adicionar Membro**.
2. Preencha:

   | Campo | Observação |
   |---|---|
   | **Nome Completo** | |
   | **E-mail** | É o login; precisa ser único |
   | **Senha** | Mínimo de 8 caracteres. **Gerar Senha** cria uma senha forte |
   | **Confirmar Senha** | |
   | **Perfil** | **Administrador**, **Supervisor** ou **Agente** (padrão) |

3. Clique em **Adicionar Usuário**.

O indicador de força avalia cinco critérios: 8 caracteres, uma maiúscula, uma minúscula, um número e
um caractere especial.

:::note
Apesar do texto "Convidar um novo membro", nenhum e-mail de convite é enviado. O administrador define
a senha e a repassa à pessoa, que pode trocá-la em **Configurações** > **Segurança**.
:::

### Editar, desativar e excluir

No menu do cartão:

| Item | O que faz |
|---|---|
| **Editar** | Altera nome e perfil. O campo **E-mail** aparece no formulário, mas a alteração dele não é gravada pelo servidor. A senha não é alterada por aqui |
| **Desativar** / **Ativar** | Bloqueia ou libera o acesso sem apagar o usuário |
| **Excluir** | Remove o usuário, após confirmação |

Prefira **Desativar** quando alguém sai da equipe: o histórico de atendimentos continua ligado ao
nome. Usuários inativos não aparecem na lista de atribuição de conversas.

Não há redefinição de senha de outro usuário, nem pelo painel nem pela API: cada pessoa troca a
própria senha em **Configurações** > **Segurança**, informando a atual. Se alguém esquecer a senha,
a alternativa hoje é excluir e criar o usuário de novo, ou pedir a quem administra o servidor.

## Papéis

No menu lateral, clique em **Papéis**. A tabela lista cada papel com **Nome**, **Descrição** e a
quantidade de **Permissões**. Os quatro papéis de sistema (`owner`, `admin`, `supervisor`, `agent`)
trazem o selo **Sistema** e só podem ser consultados: "Papéis de sistema não podem ser editados".

### Catálogo de permissões

Uma permissão é um par recurso + ação. Ao abrir um papel, a tela mostra uma grade com um recurso por
linha e uma ação por coluna.

| Ação | Significado |
|---|---|
| `read` | Ver |
| `create` | Criar |
| `update` | Alterar |
| `delete` | Excluir |
| `manage` | Todas as ações do recurso |

Recursos: `users`, `roles`, `channels`, `contacts`, `conversations`, `messages`, `bots`, `flows`,
`templates`, `campaigns`, `canned_responses`, `analytics`, `knowledge`, `audit`, `settings`.

### Permissões padrão dos papéis de sistema

| Papel | Permissões |
|---|---|
| `owner` | Tudo |
| `admin` | `manage` em todos os recursos |
| `supervisor` | `manage` em canais, contatos, conversas, mensagens, modelos, campanhas, respostas prontas, bots e fluxos; `read` em analytics e usuários |
| `agent` | `read` e `update` em conversas e contatos; `read` e `create` em mensagens; `read` em respostas prontas |

### Criar um papel

1. Clique em **Novo papel**.
2. Informe **Nome** e **Descrição** e marque as permissões na grade.
3. Clique em **Criar papel**.

Papéis criados podem ser editados (lápis) e excluídos (lixeira). Ao excluir, "usuários com este papel
voltam aos padrões internos".

:::warning
A grade de permissões tem efeito limitado na versão atual:

- Só duas áreas consultam essa grade: **campanhas** (criar, iniciar, reenviar, cancelar) e
  **respostas prontas** (criar, editar, excluir). As telas administrativas são liberadas pelo nome do
  perfil (`admin` / `owner`), e as demais não têm restrição.
- O formulário de usuário só permite escolher **Administrador**, **Supervisor** ou **Agente**. Não há
  como atribuir um papel personalizado a um usuário pelo painel.

Na prática, use os três perfis de sistema. Papéis personalizados ainda não substituem esses perfis.
:::

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| "Este e-mail já está em uso" | Já existe usuário com o e-mail | Use outro, ou reative o existente |
| **Equipe** não aparece no menu | Seu perfil não é administrador | Peça a um administrador |
| Usuário não consegue entrar | Está **Inativo** | **Ativar** no menu do cartão |
| Agente recebe erro 403 em campanhas | O perfil não tem a permissão | Altere o perfil para **Supervisor** |
| Papel novo não aparece no formulário de usuário | Papéis personalizados não são atribuíveis pelo painel | Use um dos três perfis |

As ações desta tela ficam registradas em [Auditoria](./observabilidade.md).
