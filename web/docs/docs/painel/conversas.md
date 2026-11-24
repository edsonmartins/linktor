---
sidebar_position: 1
title: Conversas
description: A caixa de entrada do painel — lista, filtros, status, atribuição, anexos, reações, chamadas, grupos e o significado de cada indicador.
---

# Conversas

**Conversas** é a caixa de entrada: a lista de todas as conversas de todos os canais à esquerda e o
histórico da conversa selecionada à direita. É onde a equipe lê, responde, atribui, escala e resolve.

Para o passo a passo de um atendimento, veja [Atender uma conversa](../guias/atender-uma-conversa.md).

## Quem pode usar

Todos os usuários autenticados veem e respondem conversas. **Excluir conversa** só aparece para os
perfis **Administrador**, **Supervisor** e para o dono da conta (`owner`).

## A lista de conversas

Cada linha mostra:

| Elemento | Significado |
|---|---|
| Nome | Nome do contato; sem nome, o telefone; sem os dois, **Contato Desconhecido** |
| Horário | Tempo desde a última mensagem |
| Selo do canal | Nome do canal por onde a conversa acontece (passe o mouse para ver o tipo) |
| Selo de status | **Abertas**, **Pendentes**, **Resolvidas** ou **Fechadas** |
| Selo **Sandbox** | A conversa pertence a um canal de teste — veja [Sandbox](./sandbox.md) |
| Prévia | Começo da última mensagem; quando não há texto, o tipo: Foto, Vídeo, Áudio, Documento, Figurinha, Localização, Contato |
| Círculo com número | Mensagens não lidas (mostra `9+` acima de nove) |
| Bolinha no avatar | Verde quando a conversa está aberta |

A lista carrega 30 conversas por vez e busca as seguintes conforme você rola. O rodapé informa
"Mostrando X de Y" enquanto houver mais para carregar. A largura da lista é ajustável: arraste a
divisória (duplo clique volta ao tamanho padrão).

### Buscar

O campo **Buscar conversas...** procura no **nome**, no **telefone** e no **e-mail** do contato. Não
procura no texto das mensagens.

### Filtrar

O botão de funil abre três grupos de filtro, que podem ser combinados. Os filtros ativos aparecem
como etiquetas abaixo da busca; clique no **×** para remover.

| Grupo | Opções |
|---|---|
| **Filtrar por Status** | **Todas**, **Abertas**, **Pendentes**, **Resolvidas**, **Adiadas** |
| **Ambiente** | **Todos os ambientes**, **Produção**, **Sandbox** |
| **Filtrar por Canal** | **Todos os canais** ou um canal específico |

:::note
O filtro **Adiadas** e a ação **Adiar** existem na tela, mas o recurso de adiar ainda não foi
implementado no servidor: a ação aparece desabilitada e o filtro não traz resultados.
:::

## Status

| Status | Quando acontece |
|---|---|
| **Abertas** | Conversa nova, ou conversa resolvida em que o cliente voltou a escrever |
| **Pendentes** | Conversa escalada, aguardando uma pessoa |
| **Resolvidas** | Alguém clicou em **Resolver conversa**, ou o fechamento automático por inatividade agiu |
| **Fechadas** | Estado previsto no servidor; não há ação no painel que leve a ele |

Uma conversa resolvida reabre sozinha quando o cliente manda nova mensagem.

## O cabeçalho da conversa

Mostra o nome do contato, o canal, o selo **Sandbox** quando for o caso e o telefone ou e-mail. À
direita há três ícones (telefone, câmera e pessoa) e o menu de três pontos.

:::note
Os ícones **Chamada de voz**, **Chamada de vídeo** e **Info do contato** ainda não executam nenhuma
ação. Para ver ou editar o contato, use a tela [Contatos](./contatos.md).
:::

### Menu de ações

| Item na tela | O que faz |
|---|---|
| **Resolver conversa** / **Reopen conversation** | Alterna entre resolver e reabrir |
| **Atribuir a agente** | Abre **Assign conversation** para escolher o responsável entre os usuários ativos |
| **Escalate conversation** | Passa a conversa para atendimento humano, com motivo e prioridade |
| **View escalation context** | Mostra o resumo e os dados gerados na escalação |
| **Adiar** | Desabilitado (não implementado) |
| **Excluir conversa** | Apaga a conversa com mensagens e anexos; não tem volta |

Parte desses rótulos e as janelas que eles abrem estão em inglês na versão atual.

### Atribuição

Uma conversa tem zero ou um responsável. Atribuir a alguém faz o bot do canal parar de responder
naquela conversa. A atribuição automática de conversas novas é configurada em **Operações** — veja
[Configurações](./configuracoes.md).

### Escalação

Escalar muda o status para **Pendentes**, grava o motivo e a prioridade e tenta entregar a conversa
ao usuário disponível com menos conversas ativas; se não houver ninguém disponível, ela fica na fila
sem responsável. O motivo informado aparece depois em
[Analytics](./analytics.md), no quadro **Motivos de Escalação**.

## O histórico

As mensagens do cliente ficam à esquerda; tudo o que saiu da empresa fica à direita, tenha sido
enviado por um atendente, por um bot, pela API ou pelo próprio celular pareado. O histórico carrega
30 mensagens por vez e busca as anteriores quando você rola para cima; a data de abertura da conversa
aparece no topo quando não há mais nada antes.

### Indicadores de cada mensagem enviada

| Ícone | Estado |
|---|---|
| Relógio | Na fila de envio |
| Um tique | Enviada ao provedor |
| Dois tiques | Entregue ao destinatário |
| Dois tiques coloridos | Lida |
| Alerta vermelho | Falhou |

Nem todo canal informa entrega e leitura; nesses, a mensagem para em "enviada".

### Mensagem com falha

Abaixo da mensagem aparece um quadro vermelho com a origem da falha:

| Título | Significado |
|---|---|
| **Bloqueado localmente pelo Linktor** | Uma regra do próprio Linktor impediu o envio, antes de chamar o provedor |
| **Rejeitado pelo provedor** | O provedor recusou; o **Detalhe técnico** traz a resposta dele |

Motivos de bloqueio local: destinatário fora da allowlist de sandbox, fora da janela de 24 h do
WhatsApp, modelo reprovado ou pausado na Meta, número inválido, ou tipo de canal sandbox sem suporte.

### Selos de origem

| Selo | Significado |
|---|---|
| **App** | Mensagem enviada pelo aplicativo WhatsApp Business no celular (modo coexistência) |
| **Histórico** | Mensagem importada do histórico do aparelho |

### Anexos

- Imagem: miniatura; clique para abrir em outra aba.
- Vídeo e áudio: tocam dentro da conversa.
- Outros arquivos: botão com o nome do arquivo para baixar.
- Quadro tracejado com **Anexo indisponível**: o canal informou o nome do arquivo, mas o conteúdo não
  foi recebido.

Para enviar, use o clipe (**Anexar arquivo**). Limite de 25 MB por arquivo; é possível anexar vários.
Se o canal de destino não aceitar o tipo, a mensagem falha com o motivo do provedor.

### Localização e contato

Uma localização recebida aparece como link com nome, endereço e coordenadas; o clique abre o mapa.

### Reações

As reações recebidas aparecem como pequenos selos com o emoji logo abaixo da mensagem. O painel não
tem botão para reagir; isso é feito pela API ([Conversas na API](../api/conversas.md)).

### Chamadas

Nos canais **WhatsApp (não oficial)** e **WhatsApp via Bridge**, chamadas de voz e vídeo entram na
conversa como um registro com o estado: **Tocando no aparelho**, **Atendida no aparelho**,
**Não atendida**, **Recusada** ou **Recusada automaticamente pelo canal**. O áudio fica no celular;
pelo painel dá para clicar em **Recusar** enquanto a chamada toca.

### Mensagens sem conteúdo

Alguns tipos de mensagem (enquetes, certos avisos do aplicativo) chegam sem texto nem anexo. Elas
aparecem como **Mensagem sem conteúdo exibível** em vez de uma bolha vazia.

### Quem está digitando

Quando outro atendente está escrevendo na mesma conversa, aparece "está digitando..." acima da caixa
de mensagem.

## Grupos

Nos canais WhatsApp não oficiais, mensagens de grupo são recebidas quando **Receber mensagens de
grupo** está ligado na configuração do canal. O painel não tem uma visão separada por grupo: a
mensagem entra na conversa do contato que a enviou, misturada às mensagens individuais dele. Para
não receber tráfego de grupo, desligue a opção no canal. Enviar para um grupo é feito pela API
([Enviar mensagens](../api/enviar-mensagens.md)).

O mesmo canal tem a opção **Ignorar status/stories**, que descarta as publicações de status.

## Conversas atendidas por bot

Se o canal tem um bot ativo, ele responde enquanto a conversa não tem responsável humano. As
respostas do bot aparecem à direita, como as dos atendentes. Para assumir, atribua a conversa a uma
pessoa. Veja [Bots](./bots.md).

## Conversas de sandbox

Conversas de canais de teste trazem o selo **Sandbox** e uma faixa amarela fixa: "Conversa de SANDBOX
— tráfego sintético de homologação; não é um cliente real".

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| Mensagem nova não aparece | Ligação em tempo real caiu (faixa **Desconectado**) | Clique em atualizar na lista; confira a rede |
| Resposta fica no relógio | Fila de envio parada ou canal desconectado | Veja [Observabilidade](./observabilidade.md) e o status em [Canais](./canais.md) |
| Falha "Fora da janela de 24h" | Regra do WhatsApp Business | Envie um modelo aprovado por campanha ou API |
| Busca não acha uma frase | A busca só olha nome, telefone e e-mail | Busque pelo contato |
| Conversa de uma pessoa cheia de mensagens de grupo | Mensagens de grupo entram no fio do remetente | Desligue **Receber mensagens de grupo** no canal |
| Não vejo **Excluir conversa** | Perfil **Agente** não pode excluir | Peça a um supervisor ou administrador |

Pela API: ver [Conversas](../api/conversas.md).
