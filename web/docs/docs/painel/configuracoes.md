---
sidebar_position: 13
title: Configurações
description: Perfil, idioma, senha, aparência e dados da organização em Configurações, e a tela Operações com atribuição automática, SLA e fechamento por inatividade.
---

# Configurações

Esta página cobre duas telas: **Configurações**, com as preferências de cada usuário e os dados da
organização, e **Operações**, com as regras de distribuição de conversas e de SLA que valem para toda
a equipe.

## Quem pode usar

| O quê | Quem |
|---|---|
| **Perfil**, **Segurança**, **Aparência** | Cada usuário, para a própria conta |
| **Notificações**, **Organização**, **Chaves de API** | **Administrador** e dono da conta (`owner`) |
| **Operações**: ver | Todos |
| **Operações**: salvar | **Administrador** e `owner` |

## Configurações

No rodapé do menu lateral, clique em **Configurações**. A tela tem seis seções na coluna da esquerda.

### Perfil

| Campo | Observação |
|---|---|
| **Alterar avatar** | Pede o endereço (URL) de uma imagem já publicada. Não há envio de arquivo, apesar da dica "JPG, PNG ou GIF. Máx. 2MB" |
| Nome | Campo rotulado **Perfil** na tela; é o seu nome de exibição |
| Email | Somente leitura |
| **Perfil** (selo) | Seu perfil de acesso; quem altera é um administrador, em [Equipe](./usuarios-e-papeis.md) |
| **Idioma** | **Português (Brasil)**, **Español** ou **English** |

Clique em **Salvar alterações** para gravar o nome. O **Idioma** muda na hora, vale só para o seu
navegador e é lembrado por um ano. O mesmo seletor existe no rodapé do menu lateral, ao lado do
seletor de tema.

### Notificações

Chaves para **Notificações por E-mail** (**Nova conversa atribuída**, **Conversa resolvida**,
**Resumo diário**) e **Notificações Push** (**Novas mensagens**, **Menções**).

:::warning
As preferências são gravadas, mas o servidor ainda não envia essas notificações: ligar ou desligar
as chaves não tem efeito prático na versão atual. As preferências também são gravadas na
organização, não por usuário, e salvar exige perfil de administrador.
:::

### Segurança

Para trocar a senha:

1. Preencha **Senha atual**, **Nova senha** (mínimo de 8 caracteres) e **Confirmar nova senha**.
2. Clique em **Atualizar senha**.

:::note
Os quadros **Autenticação de Dois Fatores** (**Habilitar 2FA**) e **Sessões Ativas** são apenas
ilustrativos: o botão não faz nada e a sessão exibida é um texto fixo. A autenticação em dois
fatores não está disponível.
:::

### Aparência

Escolha o **Tema**: **Claro**, **Escuro** ou **Sistema** (acompanha o dispositivo). Vale só para o
seu navegador.

### Chaves de API

Veja [Chaves de API](./chaves-de-api.md).

### Organização

| Campo | Observação |
|---|---|
| **Nome da organização** | Editável; clique em **Salvar alterações** |
| **Slug** | Identificador da organização; somente leitura |

:::note
O quadro **Assinatura** ("Plano Profissional", valor mensal, **Gerenciar plano**) é ilustrativo. O
Linktor não tem cobrança embutida e o botão não executa nenhuma ação.
:::

## Operações

No menu lateral, clique em **Operações** ("Roteamento de conversas e SLA"). As regras valem para
toda a organização.

### Atribuição automática

| Campo | Opções |
|---|---|
| **Estratégia de atribuição** | **Manual (agentes puxam)**, **Round-robin** ou **Balanceado (menos abertas)** |
| **Atribuir novas conversas automaticamente** | Liga ou desliga a atribuição |

| Estratégia | Como escolhe |
|---|---|
| **Manual (agentes puxam)** | Não atribui; alguém assume cada conversa |
| **Round-robin** | Entrega para quem recebeu uma conversa há mais tempo |
| **Balanceado (menos abertas)** | Entrega para quem tem menos conversas abertas ou pendentes |

Para ligar:

1. Escolha **Round-robin** ou **Balanceado (menos abertas)**.
2. Ligue **Atribuir novas conversas automaticamente**.
3. Clique em **Salvar alterações**.

São necessárias as duas coisas: com a estratégia em **Manual**, a chave ligada não tem efeito.

Como funciona:

- A atribuição acontece quando chega uma mensagem em uma conversa sem responsável.
- Concorrem todos os usuários **ativos**, de qualquer perfil — inclusive administradores e
  supervisores. Não há controle de presença nem de horário: quem está ativo recebe, mesmo fora do
  expediente.
- Sem ninguém elegível, a conversa fica sem responsável.
- Uma conversa atribuída a uma pessoa deixa de ser respondida pelo bot do canal. Em canais com bot,
  a atribuição automática tira o bot da conversa logo na primeira mensagem; nesses casos, prefira
  **Manual** e deixe o bot escalar.

### SLA e fechamento automático

| Campo | Efeito |
|---|---|
| **SLA de primeira resposta (minutos)** | Conversas sem primeira resposta dentro do prazo são marcadas como fora do SLA |
| **SLA de resolução (minutos)** | Valor gravado; ainda sem efeito |
| **Auto-fechar após inatividade (minutos)** | Conversas abertas ou pendentes paradas por esse tempo passam a **Resolvidas** |

Em todos, `0` desativa. A verificação roda uma vez por minuto.

:::note
A marcação de SLA de primeira resposta é registrada na conversa, mas ainda não aparece em nenhuma
tela do painel nem gera aviso. O **SLA de resolução** é apenas armazenado. O que tem efeito visível
hoje é o **Auto-fechar após inatividade**.
:::

O fechamento automático não envia mensagem ao cliente. Se ele escrever de novo, a conversa reabre.

## Versões

No rodapé do menu lateral, **Versões** mostra a versão do **Painel (admin)** e da **API (backend)**
que estão no ar e avisa quando as duas diferem — útil para conferir se uma atualização foi aplicada
por inteiro.

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| "Falha ao alterar senha" | **Senha atual** incorreta | Digite a senha atual correta |
| "Falha ao atualizar organização" ou ao salvar notificações | Seu perfil não é administrador | Peça a um administrador |
| Atribuição automática não acontece | Estratégia em **Manual**, ou chave desligada | Ajuste os dois e salve |
| Conversas vão para o administrador | Todos os usuários ativos concorrem | Desative usuários que não atendem, ou use **Manual** |
| O bot parou de responder depois de ligar a atribuição | A conversa passa a ter responsável humano | Use **Manual** nos canais com bot |
| Conversas somem da fila sozinhas | **Auto-fechar após inatividade** ligado | Aumente o prazo ou use `0` |
| O avatar não aparece | A URL não é pública ou não aponta para uma imagem | Use um endereço HTTPS acessível |
| O idioma voltou ao padrão | Cookies do navegador foram apagados | Escolha o idioma de novo |
