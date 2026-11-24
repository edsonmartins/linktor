---
sidebar_position: 3
title: Atender uma conversa
description: Tutorial do atendimento pelo painel, do recebimento da mensagem até resolver e reabrir a conversa, passando por atribuição, anexos e escalação.
---

# Atender uma conversa

Este tutorial acompanha uma conversa do começo ao fim: a mensagem chega, alguém assume, responde,
encaminha se for preciso e resolve. Pressupõe que já existe pelo menos um canal conectado — se não,
comece por [Conectar um canal](./conectar-um-canal.md).

:::note
Algumas ações do menu da conversa ainda aparecem em inglês no painel (**Assign conversation**,
**Escalate conversation**, **Reopen conversation**). Os nomes abaixo são os que estão na tela.
:::

## 1. Receber

1. No menu lateral, clique em **Conversas**. O número vermelho ao lado do item é o total de
   mensagens não lidas.
2. A lista à esquerda mostra as conversas da mais recente para a mais antiga. Cada linha traz o nome
   do contato (ou o telefone, quando não há nome), o selo com o nome do canal, o selo de status e o
   começo da última mensagem.
3. Clique na conversa. O histórico abre à direita, já posicionado na mensagem mais recente.

Mensagens novas entram sozinhas, sem recarregar a página. Se aparecer a faixa **Conectando...** ou
**Desconectado - Mensagens podem atrasar** no topo do histórico, a ligação em tempo real caiu: as
mensagens continuam sendo gravadas, mas só aparecem ao clicar no botão de atualizar da lista.

## 2. Assumir ou atribuir

Uma conversa nova chega sem responsável, a menos que a atribuição automática esteja ligada (veja
[Configurações](../painel/configuracoes.md)).

1. No cabeçalho da conversa, abra o menu de três pontos.
2. Clique em **Atribuir a agente**.
3. Na janela **Assign conversation**, escolha a pessoa no campo **Agent** (para assumir, escolha você
   mesmo) e clique em **Assign**.

Só aparecem usuários ativos. Enquanto a conversa tem um responsável humano, o bot do canal deixa de
responder nela.

## 3. Responder

### Texto

1. Clique na caixa **Digite uma mensagem...**.
2. Escreva e pressione **Enter** (ou clique no botão de enviar).

O ícone ao lado do horário mostra o andamento: relógio (na fila), um tique (enviada), dois tiques
(entregue), dois tiques coloridos (lida), alerta vermelho (falhou).

### Emoji

Clique no ícone de rosto ao lado da caixa de texto e escolha o emoji; ele é inserido no fim do texto.

### Anexo

1. Clique no clipe (**Anexar arquivo**) e selecione um ou mais arquivos de até 25 MB cada.
2. Os arquivos aparecem acima da caixa de texto. Para tirar um, clique no **x** ao lado dele.
3. Escreva uma legenda, se quiser, e envie.

Imagens, vídeos e áudios recebidos abrem dentro da conversa; outros arquivos viram um botão de
download.

### Resposta pronta

As respostas prontas são cadastradas em **Respostas Rápidas**. Na versão atual a caixa de mensagem
ainda não insere a resposta ao digitar `/atalho`: abra **Respostas Rápidas** em outra aba, copie o
conteúdo e cole na conversa. Detalhes em [Respostas prontas](../painel/respostas-prontas.md).

### Modelo de mensagem (WhatsApp Business)

Fora da janela de 24 horas do WhatsApp Business, texto livre é recusado pela Meta e a mensagem
aparece com falha. A tela de conversa não tem botão para enviar modelo: o envio de modelos é feito
por [Campanhas](../painel/campanhas.md) ou pela API (veja
[Modelos e campanhas na API](../api/modelos-e-campanhas.md)).

## 4. Escalar do bot para uma pessoa

Quando um bot atende o canal, a conversa passa para a equipe de três formas: o bot escala sozinho
(por regra), o atendente assume atribuindo a conversa (passo 2), ou alguém escala manualmente:

1. No menu de três pontos, clique em **Escalate conversation**.
2. Preencha **Reason** (o motivo, obrigatório) e escolha **Priority**: **Low**, **Normal**, **High** ou
   **Urgent**. O quadro **Escalation context** ao lado mostra o resumo, a intenção detectada, o
   sentimento e as últimas mensagens, quando o bot já produziu esse contexto.
3. Clique em **Escalate**.

A conversa passa para o status **Pendentes** (aguardando uma pessoa). Se houver alguém disponível,
ela já sai atribuída a quem tem menos conversas ativas; se não, fica na fila sem responsável.

Para ler o contexto de uma conversa que chegou escalada, use **View escalation context** no mesmo
menu: resumo, prioridade, motivo, dados do cliente, quantidade de mensagens, tentativas do bot e
tempo de espera.

## 5. Resolver

1. No menu de três pontos, clique em **Resolver conversa**.
2. O selo da conversa muda para **Resolvidas**.

Resolver não apaga nada: o histórico continua disponível, e a conversa sai da fila de trabalho quando
você filtra por **Abertas**.

## 6. Reabrir

- **Automático**: se o cliente escrever de novo, a conversa volta para **Abertas** sozinha.
- **Manual**: abra a conversa resolvida e, no menu de três pontos, clique em **Reopen conversation**.

## O que fazer em seguida

- Entender cada filtro e indicador da caixa de entrada: [Conversas](../painel/conversas.md).
- Corrigir nome, telefone e tags do cliente: [Contatos](../painel/contatos.md).
- Distribuir conversas automaticamente e fechar as paradas: [Configurações](../painel/configuracoes.md).
