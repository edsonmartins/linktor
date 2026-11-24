---
sidebar_position: 16
title: VRE
description: A tela VRE (respostas visuais) — pré-visualizar e renderizar templates de imagem, enviar por um canal, configurar a identidade visual e limpar o cache.
---

# VRE

A VRE gera imagens a partir de templates e dados — um menu de opções, um card de produto, o status
de um pedido, uma cobrança PIX — para enviar em canais que não mostram botões ou cards nativos. A
tela **VRE** serve para testar os templates, enviar uma imagem avulsa por um canal e ajustar a
identidade visual usada em todas elas.

## Quem pode usar

Todos os usuários autenticados; o código não restringe por papel.

:::note
A VRE depende de um serviço de renderização no servidor. Se ele não iniciar, o item **VRE** continua
no menu, mas a tela não carrega templates e as ações falham. Veja
[Configuração](../implantar/configuracao.md).
:::

## Runtime

O quadro **Runtime** monta e dispara uma renderização.

| Campo | O que informar |
|---|---|
| **Template** | O template de imagem |
| **Canal** | O canal por onde enviar (necessário só para enviar) |
| **Formato** | `jpeg`, `png` ou `webp` |
| **Legenda** | Texto enviado junto da imagem (opcional) |
| **Texto de acompanhamento** | Texto enviado depois da imagem (opcional) |
| **Dados de renderização (JSON)** | Os dados que preenchem o template |
| **Enviar para** | Destinatário (necessário só para enviar) |

Três ações:

| Botão | O que faz |
|---|---|
| **Pré-visualizar** | Renderiza o template com dados de exemplo |
| **Renderizar** | Renderiza com os seus dados, sem enviar |
| **Renderizar e enviar** | Renderiza e envia a imagem ao destinatário pelo canal |

### Testar um template

1. Escolha o **Template**.
2. Clique em **Pré-visualizar** para ver o template com dados de exemplo.
3. Ajuste os **Dados de renderização (JSON)** e clique em **Renderizar**.
4. Confira o resultado no quadro **Pré-visualização e saída**.

### Enviar uma imagem

1. Escolha **Template** e **Canal** e preencha os dados.
2. Em **Enviar para**, informe o destinatário no formato do canal (para WhatsApp, o telefone em
   formato internacional, por exemplo `+5511999999999`).
3. Clique em **Renderizar e enviar**.

O aviso **Enviado** confirma que a mensagem entrou na fila do canal; **Renderizado sem envio** indica
que a imagem foi gerada, mas não entregue. As regras do canal continuam valendo (canal conectado e
habilitado, janela de 24 h do WhatsApp Business, allowlist em canais sandbox).

## Pré-visualização e saída

Mostra a **Pré-visualização do template** e a **Última renderização**, cada uma com **Formato**,
**Tamanho**, **Bytes**, **Tempo de renderização** e, quando houver, **Legenda** e **Acompanhamento**.
O selo **cache hit** indica que a imagem veio do cache; **entregue**, que foi enviada.

## Templates e cache

Lista os templates disponíveis no servidor.

| Botão | O que faz |
|---|---|
| **Atualizar templates** | Recarrega a lista |
| **Invalidar cache** | Descarta as imagens em cache, para que as próximas sejam geradas de novo |

Use **Invalidar cache** depois de mudar a marca ou de o administrador atualizar um template no
servidor. O painel não cria nem edita templates: eles são arquivos do servidor.

Os templates usados pelo nó **VRE Visual** dos [fluxos](./fluxos.md) são **Opções de Menu**, **Card
de Produto**, **Status do Pedido**, **Lista de Produtos**, **Confirmação** e **Pagamento PIX**.

## Configuração de marca

Define a identidade visual aplicada a todos os templates da organização.

| Campo | Observação |
|---|---|
| **Nome da marca** | Exibido nas imagens |
| **URL do logo** | Endereço público da imagem do logo |
| **Cor primária**, **Cor secundária**, **Cor de destaque** | Cores principais |
| **Fundo**, **Cor do texto**, **Cor suave** | Cores de base |
| **Família da fonte** | Fonte usada nos textos |
| **Raio da borda** | Arredondamento dos cantos |

Clique em **Salvar configuração de marca**. Para ver o efeito em imagens já geradas, use **Invalidar
cache**.

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| Lista de templates vazia | Serviço de renderização não iniciou, ou pasta de templates vazia no servidor | Fale com quem administra a instalação |
| "Dados de renderização inválidos" | JSON malformado | Corrija o JSON (aspas duplas, vírgulas) |
| "Destinatário obrigatório" | **Enviar para** vazio | Preencha antes de **Renderizar e enviar** |
| "Renderizado sem envio" | Canal não escolhido, desconectado ou envio recusado | Confira o canal e veja [Observabilidade](./observabilidade.md) |
| A marca nova não aparece | Imagem antiga em cache | **Invalidar cache** |
| O logo não aparece | A URL não é acessível pelo servidor | Use um endereço HTTPS público |
