---
sidebar_position: 14
title: Observabilidade e auditoria
description: Logs de canal, filas de mensagens, estatísticas, alerta de versão do cliente WhatsApp e o registro de auditoria — como investigar um problema pelo painel.
---

# Observabilidade e auditoria

**Observabilidade** responde "por que a mensagem não chegou?": mostra os logs de cada canal, a
situação das filas de envio e os números do sistema. **Auditoria** responde "quem mudou o quê?":
registra as alterações relevantes de segurança e configuração.

## Quem pode usar

Somente **Administrador** e o dono da conta (`owner`). As duas telas não aparecem no menu dos demais
perfis, e o acesso direto pelo endereço mostra **Acesso negado**.

## Observabilidade

No menu lateral, clique em **Observabilidade**. No topo fica o cartão de versão do cliente WhatsApp;
abaixo, três abas: **Logs**, **Filas de Mensagens** e **Estatísticas**.

### Versão do cliente WhatsApp

O cartão **Versão do cliente WhatsApp (canais não-oficiais)** acompanha a versão que os canais
**WhatsApp (não oficial)** apresentam ao WhatsApp. Quando essa versão envelhece, o WhatsApp passa a
recusar a conexão e o sintoma é o canal "conectar e desconectar" sem parar.

| Selo | Significado |
|---|---|
| **Em dia** | A versão em uso é a atual |
| **Desatualizada** | Existe versão mais nova e a atualização automática está desligada |
| **Recusada pelo WhatsApp** | O WhatsApp está recusando a versão (erro 405) |
| **Não verificada** | Ainda não houve consulta |

O cartão mostra a versão **Em uso**, a **Atual no WhatsApp**, a **Última verificação** e, se houve,
quando a versão foi atualizada automaticamente; caso contrário, a versão **Embutida no build**. A
tela consulta de novo a cada minuto.

Quando aparece o alerta vermelho **O WhatsApp está recusando a versão do cliente (erro 405)**:

1. Os canais WhatsApp não oficiais não conectam nem enviam enquanto durar.
2. **Não** pareie o aparelho de novo: o Linktor busca a versão nova e reconecta sozinho.
3. Clique em **Verificar agora** para forçar a consulta.
4. Se o alerta continuar depois disso, o Linktor precisa ser atualizado. Veja
   [Backup e atualização](../implantar/backup-e-atualizacao.md).

O alerta amarelo **Há uma versão mais nova do WhatsApp Web** aparece quando a atualização automática
foi desligada no servidor (`WHATSAPP_VERSION_AUTO_UPDATE=false`): os canais podem ser recusados a
qualquer momento.

### Logs

A aba **Logs** lista os registros dos canais (**Logs de Canais**), 50 por página, atualizados a cada
10 segundos.

| Coluna | Significado |
|---|---|
| **Nível** | **Info**, **Aviso** ou **Erro** |
| **Origem** | **Canal**, **Fila**, **Sistema** ou **Webhook** |
| **Canal** | Canal a que o registro se refere |
| **Mensagem** | O que aconteceu |
| **Horário** | Quando |

Filtros: **Nível do Log**, **Origem** e **Canal**.

Como investigar uma mensagem que não saiu:

1. Filtre pelo **Canal** e pelo nível **Erro**.
2. Procure o registro no horário do envio.
3. "Envio bloqueado por guarda (motivo)" indica que uma regra do próprio Linktor impediu o envio
   (allowlist de sandbox, janela de 24 h, modelo reprovado). "Envio falhou (permanente)" indica
   recusa do provedor.

Números de telefone aparecem mascarados nos logs.

### Filas de Mensagens

As mensagens passam por filas internas antes de sair e depois de chegar. A aba mostra o **Total de
Mensagens**, o **Total Pendente** e um cartão por fila, com **Tamanho**, **Primeira Seq**, **Última
Seq** e os **Consumidores**. Atualiza a cada 5 segundos.

| Número do consumidor | Significado | Atenção quando |
|---|---|---|
| **Pendente** | Mensagens aguardando processamento | Passa de 100 e continua subindo |
| **Ack Pendente** | Entregues ao consumidor, ainda sem confirmação | Passa de 50 |
| **Reentregues** | Mensagens que precisaram ser entregues de novo | Passa de 10 |

**Pendente** alto e crescendo significa que o processamento está mais lento que a chegada, ou
parado. "Sem consumidores ativos" em uma fila com mensagens significa que o serviço que a processa
não está rodando.

:::warning
**Resetar Consumidor** devolve o consumidor ao estado inicial e faz as mensagens pendentes serem
processadas de novo — o que pode reenviar mensagens a clientes. A ação não pode ser desfeita e as
filas são do servidor inteiro, não só da sua organização. Use apenas com orientação de quem
administra a instalação.
:::

### Estatísticas

Escolha o período: **Última Hora**, **Últimas 24 Horas** ou **Últimos 7 Dias**. Atualiza a cada 30
segundos.

| Indicador | Significado |
|---|---|
| **Total de Mensagens** | Mensagens no período |
| **Tempo Médio de Resposta** | Tempo médio de resposta no período |
| **Canais Conectados** | Canais conectados, com a quantidade de desconectados abaixo |
| **Erros (24h)** | Erros nas últimas 24 horas, com os da última hora abaixo |
| **Conversas Ativas** | Conversas em andamento |
| **Resolvidas Hoje** | Conversas resolvidas no dia |
| **Pendente** | Conversas aguardando atendimento |

Os gráficos **Mensagens por Hora**, **Mensagens por Canal** e **Erros por Origem (Últimas 24h)**
mostram onde está o volume e de onde vêm os erros.

## Auditoria

No menu lateral, clique em **Auditoria** ("Mudanças relevantes de segurança"). A tabela mostra 50
registros por página, do mais recente para o mais antigo.

| Coluna | Significado |
|---|---|
| **Quando** | Data e hora |
| **Autor** | E-mail, nome ou identificador de quem fez |
| **Ação** | Nome da ação, no formato `recurso.ação` |
| **Recurso** | Tipo do recurso e o começo do identificador |
| **Detalhes** | Valores relevantes da alteração, sem segredos |
| **IP** | Endereço de origem |

Filtros: **Filtrar por autor**, **Filtrar por ação**, **Filtrar por recurso** e as datas **De** e
**Até**.

### O que é registrado

Só alterações bem-sucedidas entram; consultas e tentativas recusadas não.

| Área | Exemplos de ação |
|---|---|
| Canais | `channel.create`, `channel.update`, `channel.delete`, `channel.connect`, `channel.disconnect`, `channel.enabled` |
| Contatos | `contact.create`, `contact.update`, `contact.delete`, `contact.identities` |
| Usuários | `user.create`, `user.update`, `user.delete` |
| Papéis | `role.create`, `role.update`, `role.delete` |
| Respostas prontas | `canned_response.create`, `canned_response.update` |
| Organização e operações | `tenant.update`, `settings.update`, `conversation.auto_assign` |
| Sandbox | Inclusão e remoção de destinatários da allowlist |

Para os filtros, digite o texto da ação ou do recurso como aparece na tabela — por exemplo,
`channel.delete` em **Filtrar por ação** ou `channel` em **Filtrar por recurso**.

Credenciais nunca aparecem em **Detalhes**. Não há exportação nem exclusão de registros pelo painel.

## Operações

A tela **Operações**, apesar do nome, não é de monitoramento: ela define a atribuição automática e
os prazos de SLA. Está descrita em [Configurações](./configuracoes.md).

## Problemas comuns

| Sintoma | Onde olhar | Causa provável |
|---|---|---|
| Mensagem enviada fica no relógio | **Filas de Mensagens** | Consumidor parado ou fila acumulada |
| Mensagem falha | **Logs**, nível **Erro**, no canal | Recusa do provedor ou bloqueio local |
| Nada chega em um canal | **Logs**, origem **Webhook** | Webhook não cadastrado no provedor, ou assinatura inválida |
| WhatsApp não oficial conecta e cai | Cartão de versão | Versão recusada (erro 405) |
| "Falha ao consultar web.whatsapp.com" no cartão | Rede do servidor | Servidor sem saída para a internet |
| "Nenhuma stream encontrada" | **Filas de Mensagens** | Serviço de filas indisponível |
| Não encontro quem excluiu um canal | **Auditoria**, ação `channel.delete` | — |

Para métricas do servidor, alertas e painéis externos, veja
[Monitoramento](../implantar/monitoramento.md).
