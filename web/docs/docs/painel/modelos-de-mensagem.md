---
sidebar_position: 4
title: Modelos de mensagem
description: Modelos (templates) do WhatsApp Business no painel — criar, usar a biblioteca da Meta, sincronizar, acompanhar a aprovação, editar e excluir.
---

# Modelos de mensagem

Modelos (no painel, **Templates**) são mensagens pré-aprovadas pela Meta, exigidas para iniciar uma
conversa no WhatsApp Business ou responder fora da janela de 24 horas. A tela **Templates do
WhatsApp** cria modelos, envia para aprovação e mantém a lista local igual à da Meta.

Só existe para canais do tipo **WhatsApp Business** (API oficial). Sem um canal desse tipo, a tela
avisa: "Você precisa configurar um canal do WhatsApp oficial antes de criar templates". Veja
[WhatsApp oficial](../canais/whatsapp-oficial.md).

## Quem pode usar

Todos os usuários autenticados acessam a tela; o código não restringe por papel.

## A lista

No menu lateral, clique em **Templates**. Cada linha mostra o nome técnico, o idioma, a categoria, a
subcategoria (quando houver), a **Qualidade** informada pela Meta e o selo de status. Clique na linha
para abrir o detalhe; o ícone de lixeira exclui.

Filtros: **Buscar por nome...**, **Todos os canais**, **Todos os status** (Aprovado, Pendente,
Rejeitado, Pausado, Desativado) e **Todas as categorias**.

### Status de aprovação

| Status | Significado |
|---|---|
| **Pendente** | Enviado; a Meta ainda está analisando |
| **Aprovado** | Pode ser usado em envios |
| **Rejeitado** | Recusado; o **Motivo da rejeição** aparece no detalhe |
| **Pausado** | A Meta pausou por baixa qualidade; não pode ser enviado enquanto estiver assim |
| **Desativado** | A Meta desativou o modelo |
| **Em recurso** | Há uma contestação em andamento |
| **Reativado** | Voltou a valer depois de pausa ou recurso |
| **Limite excedido** | A conta atingiu o limite de modelos |
| **Remoção pendente** / **Excluído** / **Arquivado** | Em remoção, removido ou arquivado na Meta |

O status não muda sozinho na tela: use **Sincronizar com Meta** ou **Atualizar** no detalhe.

## Criar um modelo

1. Clique em **Novo template**.
2. Preencha a seção **Identificação**:

   | Campo | Observação |
   |---|---|
   | **Canal** | O canal WhatsApp Business dono do modelo |
   | **Nome técnico** | Só minúsculas, números e underscore (`order_confirmation`). Não muda depois |
   | **Idioma** | Código no formato da Meta: `pt_BR`, `en_US`, `es_MX`. Não muda depois |
   | **Categoria** | **Utilitário**, **Marketing** ou **Autenticação** |
   | **Subcategoria** | Só em Utilitário: `ORDER_DETAILS`, `ORDER_STATUS`, `RICH_ORDER_STATUS` |
   | **Formato de parâmetros** | **Posicional** (`{{1}}`, `{{2}}`) ou **Nomeado** (`{{customer_name}}`) |
   | **TTL da mensagem (segundos)** | Tempo máximo de tentativa de entrega; `0` ou vazio usa o padrão da Meta |
   | **Permitir reclassificação** | Deixa a Meta trocar a categoria se o conteúdo não corresponder |

3. **Cabeçalho (opcional)**: escolha o **Formato** — **Sem cabeçalho**, **Texto**, **Imagem**,
   **Vídeo**, **Documento** ou **Localização**. Texto aceita até 60 caracteres.
4. **Corpo da mensagem**: escreva o **Texto** com as variáveis. Para cada variável aparece um campo de
   exemplo; a Meta exige um exemplo por variável.
5. **Rodapé**: até 60 caracteres, sem variáveis.
6. **Botões (opcional)**: até 10, dos tipos **Resposta rápida**, **Link**, **Ligar**, **Copiar
   código**, **OTP** e **Flow**. Texto do botão com até 25 caracteres.
7. Opcionais avançados: **Oferta por tempo limitado** (banner de até 16 caracteres, com ou sem
   expiração) e **Carrossel** (de 1 a 10 cards, cada um com imagem ou vídeo, texto e botões).
8. Clique em **Enviar para aprovação**. O modelo entra na lista como **Pendente**.

:::note
Para cabeçalho de imagem, vídeo ou documento, o formulário pede o **Handle da Meta (resumable
upload)**: o identificador devolvido pela API de upload da Meta. O painel não faz esse upload; o
arquivo serve apenas de exemplo para a aprovação.
:::

## Usar a biblioteca da Meta

A biblioteca traz modelos prontos, que a Meta aprova automaticamente.

1. Em **Templates**, clique em **Biblioteca**.
2. Escolha o **Canal**. Refine com **Busca**, **Idioma**, **Tópico**, **Caso de uso** e **Setor**.
3. No modelo desejado, clique em **Usar este template**.
4. Informe o **Nome técnico** (único na sua conta) e o **Idioma** e clique em **Criar**.

## Sincronizar com a Meta

**Sincronizar com Meta** traz para o Linktor os modelos e os status que estão na conta do WhatsApp
Business — inclusive os criados direto no gerenciador da Meta. Com um canal selecionado no filtro,
sincroniza só ele; com **Todos os canais**, sincroniza um por um.

Sincronize depois de conectar um canal que já tinha modelos, depois de criar um modelo (para ver a
aprovação) e sempre que um envio falhar por status de modelo.

## Ver o detalhe

O detalhe mostra a **Pré-visualização** (cabeçalho, corpo, rodapé e botões em formato de balão) e os
**Metadados**: **ID Meta**, **Pontuação de qualidade**, **Formato de parâmetros**, **TTL da
mensagem**, **Criado em** e **Última sincronização**. O botão **Atualizar** consulta a Meta só para
aquele modelo.

A pré-visualização é simplificada: cabeçalhos de mídia aparecem como um marcador, e carrossel e
oferta por tempo limitado não são desenhados.

## Editar

1. No detalhe, clique em **Editar**.
2. Altere categoria, conteúdo ou TTL. **Nome técnico** e **Idioma** são imutáveis.
3. Clique em **Salvar alterações**. A Meta revisa o modelo de novo, e ele volta a **Pendente**.

## Excluir

Clique na lixeira (na lista) ou em **Excluir** (no detalhe) e confirme. O modelo é removido do
Linktor **e da Meta**; não dá para desfazer.

## Problemas comuns

| Sintoma | Causa | O que fazer |
|---|---|---|
| **Sincronizar com Meta** desabilitado | Não há canal WhatsApp Business | Crie o canal primeiro |
| Sincronização falha | Token do canal sem permissão ou **ID da Conta WhatsApp Business** errado | Corrija em [Canais](./canais.md) |
| Modelo **Rejeitado** | Conteúdo fora das regras da Meta | Leia o **Motivo da rejeição**, edite e reenvie |
| Status não muda | A tela não atualiza sozinha | Clique em **Atualizar** ou **Sincronizar com Meta** |
| Envio falha com "template reprovado/pausado" | O modelo não está **Aprovado** | Sincronize e use um modelo aprovado |
| Erro ao criar com nome repetido | Nome e idioma já existem na conta | Use outro **Nome técnico** |

Para enviar um modelo, use [Campanhas](./campanhas.md) ou a API.

Pela API: ver [Modelos e campanhas](../api/modelos-e-campanhas.md).
