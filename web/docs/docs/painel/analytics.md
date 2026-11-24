---
sidebar_position: 10
title: Analytics
description: O Dashboard inicial e a tela de Analytics — o que cada indicador, gráfico e tabela mede e como interpretar.
---

# Analytics

O painel tem duas telas de números: o **Dashboard**, que é a página inicial e resume o momento, e
**Analytics**, que mostra a evolução em um período. As duas medem o atendimento com foco em quanto o
bot resolve sozinho e quanto vai para a equipe.

## Quem pode usar

Todos os usuários autenticados.

## Dashboard

É a primeira tela depois do login. Os números do topo consideram as últimas 24 horas.

| Cartão | O que mede |
|---|---|
| **Total de Conversas** | Conversas criadas no período |
| **Taxa de Resolução** | Percentual de conversas resolvidas sem escalação (legenda "por bot") |
| **Total de Contatos** | Contatos cadastrados na base inteira |
| **Tempo Médio de Resposta** | Tempo médio até a primeira resposta |

Abaixo:

- **Conversas Recentes**: as últimas conversas abertas, com nome, canal e começo da última mensagem.
- **Status dos Canais**: os canais e a situação de conexão de cada um.
- **system.log**: um quadro decorativo com mensagens fixas; não é um log real. Os logs estão em
  [Observabilidade](./observabilidade.md).

## Analytics

No menu lateral, clique em **Analytics**.

### Período

No topo, escolha **Últimas 24 horas**, **Últimos 7 dias** (padrão) ou **Últimos 30 dias**, ou informe
**Data Início** e **Data Fim**. Todos os quadros da tela seguem o período escolhido.

### Indicadores

| Cartão | O que mede |
|---|---|
| **Total de Conversas** | Conversas criadas no período |
| **Taxa de Resolução** | Conversas resolvidas sem escalação ÷ total de conversas |
| **Tempo Médio de Resposta** | Média entre a criação da conversa e a primeira resposta |
| **Confiança Média** | Média da confiança das respostas do bot, de 0 a 1 |

A seta com percentual ao lado de um indicador compara com o período anterior de mesmo tamanho.

:::note
"Resolvida pelo bot" significa **conversa resolvida que nunca foi escalada**. Uma conversa atendida
e resolvida por uma pessoa sem passar por escalação também entra nessa conta. Em operações sem bot,
a **Taxa de Resolução** mede simplesmente o que foi resolvido.
:::

### Conversas ao Longo do Tempo

Gráfico por dia com três séries: **Total**, **Resolvidas pelo Bot** e **Escaladas**.

### Escalações

O quadro **Motivos de Escalação** mostra quantas conversas foram escaladas e por quê:

| Motivo | Origem |
|---|---|
| **Baixa Confiança** | O bot não teve confiança na resposta |
| **Solicitação do Usuário** | O cliente pediu uma pessoa |
| **Sentimento Negativo** | O cliente demonstrou insatisfação |
| **Gatilho de Palavra-chave** | Regra de palavra-chave do bot |
| **Correspondência de Intenção** | Regra de intenção do bot |
| **Desconhecido** | Escalação sem motivo classificado, ou com motivo escrito à mão |

A **Taxa de Escalação** é conversas escaladas ÷ total de conversas.

### Distribuição por Canal

| Coluna | Significado |
|---|---|
| **Canal** | Nome do canal |
| **Conversas** | Conversas do canal no período |
| **Resolvidas** | Resolvidas sem escalação |
| **Taxa** | Resolvidas ÷ conversas do canal |

### Desempenho dos Fluxos

| Coluna | Significado |
|---|---|
| **Nome do Fluxo** | O fluxo |
| **Acionados** | Vezes em que o fluxo começou |
| **Concluídos** | Vezes em que chegou ao fim |
| **Taxa** | Concluídos ÷ acionados |

Uma taxa baixa indica que os clientes abandonam o fluxo no meio ou caem em um nó sem saída. Veja
[Fluxos](./fluxos.md).

## Como ler os números

- **Taxa de Resolução** caindo com **Escaladas** subindo: o bot está deixando de responder algo.
  Veja os **Motivos de Escalação** e complete a [base de conhecimento](./base-de-conhecimento.md).
- **Tempo Médio de Resposta** alto com poucas conversas: conversas ficando sem responsável. Considere
  a atribuição automática em [Configurações](./configuracoes.md).
- **Confiança Média** baixa: prompt genérico ou base de conhecimento insuficiente.

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| "Sem dados disponíveis para o período selecionado" | Não houve conversas no período | Amplie o período |
| **Confiança Média** em 0 | Nenhum bot respondeu no período | Normal em operações sem bot |
| Tabela de fluxos vazia | Nenhum fluxo foi acionado | Confira se há fluxo ativo e bot no canal |
| Números do Dashboard e de Analytics diferentes | O Dashboard usa 24 horas; Analytics abre em 7 dias | Selecione **Últimas 24 horas** |
| Conversas de teste entram nos totais | Conversas de sandbox não são separadas dos números | Considere isso ao ler períodos de homologação |

Os dados de envio e custo do WhatsApp Business por canal estão disponíveis apenas pela API.
