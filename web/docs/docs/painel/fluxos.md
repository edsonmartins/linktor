---
sidebar_position: 8
title: Fluxos
description: Fluxos conversacionais no painel — criar, editar no canvas, tipos de nó, transições, gatilhos, ativar e testar.
---

# Fluxos

Um fluxo é um roteiro de conversa em forma de árvore: o bot envia uma mensagem, faz uma pergunta,
segue um caminho conforme a resposta e executa ações (etiquetar, atribuir, escalar). Use fluxos para
menus, triagem e coleta de dados, onde a resposta precisa ser sempre a mesma.

:::note
Fluxos são executados pelo bot do canal. Um fluxo ativo só roda em conversas de canais que têm um
[bot](./bots.md) ativo atribuído e sem responsável humano.
:::

## Quem pode usar

Todos os usuários autenticados acessam a tela; o código não restringe por papel.

## A lista

No menu lateral, clique em **Fluxos**. A tela **Fluxos Conversacionais** mostra um cartão por fluxo,
com gatilho e situação. Filtros: **Buscar fluxos...**, **Todos os Gatilhos** e **Todos os Status**
(**Ativo**, **Inativo**). Cada cartão permite editar, testar, ativar ou desativar e excluir.

## Criar um fluxo

1. Clique em **Novo Fluxo**.
2. Em **Criar Novo Fluxo**, preencha **Nome**, **Descrição (opcional)** e **Gatilho**.
3. Para os gatilhos **Palavra-chave** e **Intenção**, informe o valor no campo que aparece.
4. Clique em **Criar e Editar**. O editor abre com o fluxo **Inativo**.

### Gatilhos

| Gatilho | Quando o fluxo começa |
|---|---|
| **Mensagem de Boas-vindas** | Na primeira mensagem de uma conversa sem contexto anterior |
| **Palavra-chave** | Quando a mensagem **contém** a palavra informada (sem diferenciar maiúsculas) |
| **Intenção** | Quando a intenção detectada pela IA tem o nome informado |
| **Manual** | Não começa sozinho |

Um fluxo novo não começa enquanto outro está em andamento na mesma conversa.

:::note
O gatilho **Palavra-chave** compara o campo inteiro com a mensagem: informe uma única palavra ou
expressão. Escrever `suporte, ajuda` só dispara se o cliente digitar exatamente esse trecho, com a
vírgula.
:::

## O editor

O editor tem três áreas: a barra **Nodes** à esquerda (um botão por tipo de nó), o canvas no centro e
o painel do nó selecionado à direita. No topo ficam o nome do fluxo, o selo de situação e os botões
**Test**, **Activate** / **Deactivate** e **Save**.

:::note
Os rótulos do topo do editor e da barra de nós estão em inglês na versão atual. O painel do nó está
em português.
:::

1. Clique em um tipo na barra **Nodes** para adicionar o nó ao canvas.
2. Arraste os nós para organizar e ligue um ao outro para criar transições.
3. Clique em um nó para abrir o painel e editar o conteúdo.
4. Clique em **Save**. O selo **Unsaved changes** indica alterações não gravadas.

### Tipos de nó

| Nó | O que faz | Campos no painel |
|---|---|---|
| **Mensagem** | Envia um texto e segue adiante | **Conteúdo** |
| **Pergunta** | Envia um texto com opções e espera a resposta | **Conteúdo**, **Respostas Rápidas** (texto de cada botão) |
| **Condição** | Escolhe o caminho conforme a resposta | **Transições** |
| **Ação** | Executa ações sem falar com o cliente | **Ações** |
| **VRE Visual** | Envia uma imagem gerada (menu, card de produto, PIX…) | **Template VRE**, **Legenda (opcional)**, **Texto de Acompanhamento (opcional)** |
| **Fim** | Encerra o fluxo | — |

Em canais que não mostram botões, as opções de uma **Pergunta** saem como lista numerada, e o
cliente pode responder com o número.

### Transições

Cada transição tem um nó de destino e uma condição:

| Condição | Segue quando |
|---|---|
| **Padrão (sempre)** | Nenhuma outra condição do nó foi atendida |
| **Resposta igual a** | A resposta é igual ao valor, ao texto de uma opção ou ao número dela |
| **Contém** | A resposta contém o valor |
| **Expressão regular** | A resposta casa com a expressão |

As comparações de **Resposta igual a** e **Contém** não diferenciam maiúsculas. Sem nenhuma
transição que sirva, o fluxo para naquele nó — deixe sempre uma transição **Padrão (sempre)**.

### Ações

| Ação | O que faz |
|---|---|
| **Adicionar Tag** | Marca a conversa com a tag informada |
| **Atribuir a Usuário** | Atribui a conversa |
| **Escalar** | Passa para atendimento humano, com **Prioridade** **Baixa**, **Normal**, **Alta** ou **Urgente** |
| **Definir Entidade** | Guarda um valor para usar adiante |
| **Chamada HTTP** | Chama um endereço externo |

:::warning
**Definir Entidade** e **Chamada HTTP** podem ser configuradas no editor, mas não encontramos a
execução delas no servidor: só **Adicionar Tag**, **Atribuir a Usuário** e **Escalar** têm efeito
confirmado. Teste antes de depender delas.
:::

### Nó VRE Visual

Escolha o **Template VRE**: **Opções de Menu**, **Card de Produto**, **Status do Pedido**, **Lista de
Produtos**, **Confirmação** ou **Pagamento PIX**. Para preencher os campos do template com dados
coletados no fluxo, use a sintaxe `{{variavel}}`. O nó depende do serviço de renderização — veja
[VRE](./vre.md).

## Testar

1. No editor, clique em **Test** (ou em testar, no cartão da lista).
2. No painel **Testar Fluxo**, digite como o cliente escreveria. O painel mostra as respostas do bot
   e o **Nó atual**.
3. Ao chegar a um nó **Fim**, aparece **Fluxo finalizado**. Use **Recomeçar** ou **Resetar** para
   testar outro caminho.

O teste roda a versão **salva** do fluxo: clique em **Save** antes de testar. Nada é enviado a
clientes.

## Ativar e desativar

Clique em **Activate** no editor ou em ativar no cartão. Só fluxos ativos são avaliados quando chega
uma mensagem. Para tirar um fluxo do ar sem perdê-lo, use **Deactivate**.

Se mais de um fluxo ativo combinar com a mesma mensagem, apenas um é iniciado. Evite palavras-chave
que se sobreponham e mais de um fluxo de boas-vindas ativo.

## Excluir

No cartão, escolha excluir e confirme. A exclusão é definitiva.

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| Fluxo ativo nunca começa | O canal não tem bot ativo, ou a conversa tem responsável | Atribua um bot ao canal; veja [Bots](./bots.md) |
| Palavra-chave não dispara | O campo tem várias palavras separadas por vírgula | Use uma palavra por fluxo |
| O fluxo trava em um nó | Nenhuma transição serve para a resposta | Adicione uma transição **Padrão (sempre)** |
| O teste não reflete a alteração | Alterações não salvas | Clique em **Save** e teste de novo |
| Boas-vindas dispara no lugar de outro fluxo | Dois fluxos combinam com a primeira mensagem | Mantenha um só fluxo de boas-vindas ativo |
| Nó **VRE Visual** não envia imagem | Serviço de renderização indisponível | Veja [VRE](./vre.md) |

Os números de execução de cada fluxo aparecem em [Analytics](./analytics.md), em **Desempenho dos
Fluxos**.
