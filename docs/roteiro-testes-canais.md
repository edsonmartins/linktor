# Roteiro de testes por canal

Atualizado em 13/10/2026. Tarefas no Jira: épico [LIN-1](https://relevantsolutions.atlassian.net/browse/LIN-1).

## Escopo

Entram 13 canais; o WhatsApp não oficial (QR) fica de fora porque já foi testado. O roteiro foi montado lendo o código, sem executar nada, então todo problema apontado aqui precisa ser confirmado no teste. Ele pressupõe um build que já contenha as correções do PR #102.

| Canal | Entrada | Saída suportada | Status que o canal devolve | Conta externa necessária |
| --- | --- | --- | --- | --- |
| WhatsApp Oficial | Webhook Meta | Texto, mídia, template, botões e lista | Enviado, entregue, lido, falhou | App Meta, WABA e número |
| Facebook Messenger | Webhook Meta | Texto, mídia | Enviado, entregue | App Meta e Página |
| Instagram | Webhook Meta | Texto, mídia | Enviado | App Meta e conta profissional |
| Telegram | Webhook registrado no connect | Texto, mídia, botões inline | Enviado | Bot do BotFather |
| WebChat | WebSocket do widget | Texto, mídia | Enviado | Nenhuma |
| Email | IMAP ou webhook do provedor | Texto, HTML, 1 anexo | Enviado; entregue e lido só em Postmark e SES | Caixa SMTP/IMAP ou provedor |
| SMS | Webhook Twilio | Texto, MMS | Enviado, entregue, falhou | Conta Twilio e número |
| RCS | Webhook do provedor | Texto; arquivo só na Zenvia | Conforme o provedor | Conta Zenvia, Infobip, Pontaltech ou Google |
| Voz | Webhook Twilio | Nenhuma | Nenhum | Número de voz Twilio |
| Microsoft Teams | Bot Framework | Texto, mídia por URL | Enviado | Azure Bot |
| Slack | Events API | Texto, upload de arquivo | Enviado | App Slack |
| Mattermost | WebSocket aberto pelo Linktor | Texto, upload de arquivo | Enviado | Servidor Mattermost e bot |
| Direto (VendaX) | Webhook | Texto, mídia por URL | Enviado | Runtime Direto ou servidor mock |

Ficam fora também: os provedores de voz Vonage, Asterisk, FreeSWITCH e Amazon Connect (o código existe, mas nada o chama) e o sandbox, que é um atributo de canal WhatsApp e não um tipo de canal.

## Ambiente e pré-requisitos

Sem URL pública HTTPS, nenhum canal de webhook recebe mensagem. Prepare isto uma vez, antes de qualquer canal. O passo a passo para subir o projeto e o `.env` completo estão na seção Instalação do `README.md`.

- **URL pública**: um túnel (ngrok, Cloudflare Tunnel) ou a VPS de homologação apontando para o backend. O proxy precisa repassar `X-Forwarded-Proto`, senão a assinatura da Twilio falha.
- **`BASE_URL`** no backend com a URL pública: é usada no registro do webhook do Telegram e no OAuth da Meta.
- **`NEXT_PUBLIC_WEBHOOK_BASE_URL`** (ou `LINKTOR_ADMIN_WEBHOOK_BASE_URL`) no admin com a mesma URL, para as telas mostrarem o webhook certo em vez de `localhost:8081`.
- **Armazenamento de mídia**: MinIO (`MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`) ou `MEDIA_UPLOAD_DIR`. Sem ele, mídia recebida fica com URL do provedor e não abre no admin.
- **Redis e NATS no ar**: NATS é obrigatório para entrada, saída e status; Redis liga a deduplicação de webhook.
- **Modo do servidor**: teste em `release` ou com `LINKTOR_REQUIRE_WEBHOOK_SECRETS=true`. Em `debug`, webhook sem segredo configurado é aceito, e o teste de assinatura não prova nada.
- **`LINKTOR_ENABLED_CHANNEL_TYPES`**: se estiver definida, precisa listar todos os tipos do teste, inclusive `direto`.
- **`LINKTOR_WS_ALLOWED_ORIGINS`** com o domínio da página de teste do WebChat; sem isso só origem `localhost` conecta.
- **Usuário**: `admin@demo.com`, com a senha definida em `SEED_ADMIN_PASSWORD` antes do primeiro boot. Sem a variável, a senha é aleatória e aparece uma única vez no log do backend.

**Conectar o canal.** O canal nasce `Desconectado`, e nesse estado toda mensagem recebida é descartada com resposta 200 ao provedor. Depois de criar cada canal, abra o menu do card (três pontos) e clique em **Conectar**; o card deve passar a mostrar **Conectado**. O mesmo pode ser feito por API:

```bash
curl -X POST "$BASE_URL/api/v1/channels/$CHANNEL_ID/connect" \
  -H "Authorization: Bearer $JWT"
```

O ID do canal está no card, com botão de copiar. Alguns passos do roteiro gravam configuração por API; eles usam `PUT /api/v1/channels/{id}` com o mesmo JWT.

**Onde olhar o resultado.** Conversas em `/conversations`; diagnóstico em `/observability`, filtrando pelo canal: lá aparecem `Mensagem recebida do canal`, `Webhook duplicado ignorado`, `Webhook recebido descartado` e `Envio falhou`.

**Reenvio de webhook.** Um webhook já aceito, reenviado com o mesmo corpo para o mesmo canal em até 6 horas, volta com `X-Webhook-Dedup: duplicate`. Para repetir uma chamada manual aceita, mude um byte do corpo ou mande um `X-Event-Id` novo. Chamada recusada (4xx) pode ser repetida igual.

## Casos comuns a todo canal

Estes 10 casos valem para cada canal; o roteiro por canal só acrescenta o que é específico. Registre em cada um: passou, falhou ou não se aplica, com print e o trecho do log de `/observability`.

| ID | Caso | Como fazer | Resultado esperado |
| --- | --- | --- | --- |
| C1 | Criar o canal | Canais > Adicionar Novo Canal, escolher o tipo e preencher as credenciais | Canal aparece no card como Habilitado e Desconectado |
| C2 | Testar conexão | Botão Testar Conexão da tela, com credencial certa e depois com credencial errada | Certa passa; errada falha com mensagem clara |
| C3 | Conectar | Menu do card > Conectar | Card muda para Conectado; Desconectar desfaz |
| C4 | Receber texto | Enviar um texto pelo aplicativo do canal | Conversa nova em Conversas, com o contato e o texto corretos |
| C5 | Responder texto | Responder pela conversa no admin | Mensagem chega no aplicativo; status vira Enviado |
| C6 | Receber mídia | Enviar imagem, áudio, vídeo e documento | Cada mídia abre no admin |
| C7 | Enviar mídia | Anexar imagem e documento na resposta | Arquivo chega no aplicativo com a legenda |
| C8 | Mesma conversa | Enviar uma segunda mensagem do mesmo contato | Cai na conversa já aberta, sem duplicar contato |
| C9 | Assinatura do webhook | Chamar o webhook por curl com assinatura ou token inválido | Resposta 401 (400 em Facebook e Instagram) e nada criado |
| C10 | Desabilitar e excluir | Desabilitar o canal, enviar mensagem, reabilitar; ao final, Excluir Canal | Desabilitado não recebe; exclusão remove o card e o webhook para de aceitar |

O botão Testar Conexão só consulta o provedor em Teams, Slack, Mattermost e Email. Em WhatsApp Oficial, Facebook, Instagram, Telegram, SMS e RCS ele apenas confere se os campos foram preenchidos, então passar em C2 não prova que o token é válido.

## Roteiro por canal

Cada canal traz a preparação no provedor e os testes específicos, que se somam aos casos C1 a C10. Os webhooks ficam todos em `{BASE_URL}/api/v1/webhooks/...`.

### WhatsApp Oficial (Cloud API)

Webhook: `/api/v1/webhooks/whatsapp/{channelId}`.

Preparação:

1. No app da Meta, anotar token de acesso, Phone Number ID, WABA ID e App Secret.
2. Criar o canal pela aba Manual Setup. No campo Business ID, colocar o WABA ID (é dele que os templates dependem). Em Webhook Secret, colocar o App Secret.
3. Conectar pelo menu do card e cadastrar na Meta o webhook e o verify token mostrados na aba Webhook, assinando o campo `messages`.

Testes específicos:

- Verificação GET da Meta devolve o challenge; com token errado ou vazio, 403. Repetir a verificação logo em seguida: deve devolver o challenge de novo.
- Receber: texto, imagem, vídeo, áudio, documento, figurinha, localização, contato, resposta citando mensagem e reação (chega como mensagem de texto com o emoji).
- Enviar: texto com link, imagem, vídeo, áudio e documento.
- Botões e lista: por API, enviar até 3 respostas rápidas (viram botões) e mais de 3 (viram lista); tocar em uma opção e ver a resposta chegar com o título.
- Status: a mensagem enviada passa por Enviado, Entregue e Lido; número inválido termina em Falhou com o código da Meta.
- Janela de 24 h: com `LINKTOR_WA_WINDOW_ENFORCEMENT=enforce`, responder a um contato parado há mais de 24 h deve falhar; template deve passar.
- Sandbox: canal criado como sandbox só entrega para números da lista em `/sandbox-allowlist`.
- Embedded Signup: completar o fluxo da aba Embedded Signup (exige `WHATSAPP_EMBEDDED_SIGNUP_APP_SECRET` no servidor) e conferir que o canal nasce Conectado.
- Ao editar o canal, conferir se o campo do verify token mostra o valor real ou `__redacted__`; se mostrar o segundo, não salvar e registrar como bug.

### WhatsApp Oficial: recursos extras

- **Templates** (`/templates`): criar, aguardar aprovação, sincronizar com a Meta, editar e excluir; enviar um template aprovado por Campanhas, pois o chat não tem seletor de template.
- **Qualidade do número**: após um evento de qualidade da Meta, conferir `quality_rating` no canal.
- **Coexistência**: em canal criado por Embedded Signup, conferir o selo de coexistência no card e enviar uma mensagem pelo app WhatsApp Business do celular.
- **Aba Operações** (pagamentos, chamadas, CTWA): abrir cada bloco e registrar o que responde. Pagamentos só funciona com `config.payment_gateway_type=mock` gravado por API.
- **Webhooks de pagamentos, chamadas e CTWA**: chamar `POST /api/v1/webhooks/payments/{channelId}`, `/calls/{channelId}` e `/ctwa/{channelId}` e registrar a resposta.
- **Flows**: `POST /api/v1/whatsapp/flows`, publicar e ver o preview; exige `config.waba_id` no canal.

### Facebook Messenger

Webhook: `/api/v1/webhooks/facebook/{channelId}`.

Preparação:

1. Ter App ID, App Secret, Page ID e Page Access Token com `pages_messaging`.
2. Criar o canal pela aba Manual.
3. Conectar pelo menu do card, cadastrar o webhook na Meta e assinar `messages`, `messaging_postbacks` e `message_deliveries`.

Testes específicos:

- Verificação GET da Meta devolve o challenge.
- Receber texto, imagem, vídeo, áudio, arquivo e localização pelo Messenger.
- Postback: tocar em um botão do menu da Página; chega como texto com o payload.
- Enviar texto e mídia; status vai até Entregue (Lido não é registrado).
- Enviar várias mídias de uma vez: só a primeira deve aparecer.
- Fluxo OAuth da aba Connect: escolher a Página, criar o canal e responder uma mensagem.

### Instagram

Webhook: `/api/v1/webhooks/instagram/{channelId}`.

Preparação:

1. Conta profissional ligada a uma Página; ter Instagram ID, token, App ID e App Secret.
2. Criar pela aba Manual preenchendo o App Secret (a tela o trata como opcional, mas sem ele todo POST é recusado).
3. Conectar pelo menu do card e assinar o webhook do Instagram na Meta manualmente.

Testes específicos:

- Verificação GET da Meta devolve o challenge.
- Receber DM de texto, imagem, vídeo e áudio.
- Responder a um story e mencionar a conta em um story: chegam como `[story reply]` e `[mentioned you in a story]`.
- Reagir a uma mensagem: verificar se a reação aparece ou some.
- Enviar texto e mídia; o status para em Enviado.
- Fluxo OAuth da aba Connect.

### Telegram

Webhook: `/api/v1/webhooks/telegram/{channelId}`, registrado sozinho no connect.

Preparação:

1. Criar um bot no @BotFather e copiar o token.
2. Criar o canal com o token e conectar pelo menu do card. O connect chama `getMe` e `setWebhook`; com token errado ou `BASE_URL` vazia ele deve falhar com mensagem clara.
3. Conferir em `https://api.telegram.org/bot<TOKEN>/getWebhookInfo` que a URL registrada é a do canal.

Testes específicos:

- Enviar 3 mensagens seguidas ao bot e conferir que as 3 chegam.
- Receber `/start`, texto, foto com legenda, documento, nota de voz e vídeo.
- Receber figurinha, GIF, contato e localização: registrar como cada um aparece.
- Editar uma mensagem já enviada: registrar se a edição aparece.
- Enviar texto, foto, vídeo, áudio e documento pelo admin.
- Botões: por API, enviar mensagem com respostas rápidas; tocar em um botão e ver a resposta chegar.
- Grupo: adicionar o bot a um grupo, escrever nele e responder pelo admin; registrar para onde a resposta vai.
- Desconectar pelo menu do card e conferir em `getWebhookInfo` que o webhook foi removido.

### WebChat

Sem webhook: o widget abre um WebSocket em `/ws/{channelId}`.

Preparação:

1. Criar o canal, conectar pelo menu do card e reabrir em Configurar para copiar o snippet da aba Embed (antes disso ele mostra `{CHANNEL_ID}`).
2. Colar o snippet em uma página HTML servida em um domínio listado em `LINKTOR_WS_ALLOWED_ORIGINS`.

Testes específicos:

- O lançador aparece, abre, e título e cor seguem a configuração.
- Visitante envia texto; a conversa nasce no admin como Visitor.
- Atendente responde; o texto chega no widget em tempo real, com indicador de digitação.
- Recarregar a página mantém a sessão; nova aba vira novo visitante.
- Passar `visitor: {name, email}` no `init` e conferir o contato criado.
- Canal desconectado ou desabilitado: o widget não conecta (503).
- Página em domínio fora da lista: conexão recusada.
- Com `widget_secret` gravado por API: sem token dá 401; com token de `POST /api/v1/webchat/{id}/widget-token` conecta.
- Atendente envia imagem e documento: o widget deve mostrar o anexo. Mensagem com botões enviada por API chega como texto.
- Mensagem de boas-vindas, mensagem offline e exigir nome ou email: registrar se têm efeito.

### Email

Entrada por IMAP (provedor SMTP) ou por webhook em `/api/v1/webhooks/email/{channelId}` (SendGrid, Mailgun, Postmark, SES).

Preparação:

1. Usar uma caixa nova e vazia: o IMAP lê todo email não lido, 25 por rodada, e marca como lido.
2. Criar o canal SMTP com host, porta, usuário, senha e o bloco IMAP (porta 993). Testar Conexão aqui é real.
3. Conectar pelo menu do card.

Testes específicos:

- Receber email de texto, email só HTML e email com acentos no assunto.
- Receber email com anexo: registrar se o anexo abre.
- Responder pelo admin e conferir, na caixa do cliente, o assunto e se a resposta entra na mesma thread.
- Enviar resposta com anexo.
- Cliente responde de novo: cai na mesma conversa.
- Credencial SMTP errada: Testar Conexão falha; envio termina em Falhou.
- Se houver conta em SendGrid, Mailgun, Postmark ou SES: repetir envio e recebimento por webhook e conferir Entregue e Lido.

### SMS (Twilio)

Webhook: `/api/v1/webhooks/twilio/{channelId}`.

Preparação:

1. Criar o canal com Account SID, Auth Token e o número (ou Messaging Service SID) e conectar pelo menu do card.
2. No console da Twilio, apontar o webhook de mensagens do número para a URL do canal, método POST.
3. Para receber status, gravar `status_callback_url` por API com a mesma URL (a tela não tem o campo).

Testes específicos:

- Receber SMS de texto e MMS com imagem.
- Enviar SMS de texto e MMS com imagem.
- Status passa de Enviado para Entregue.
- Enviar para número inválido: registrar quanto tempo leva até Falhou.
- Chamada por curl sem `X-Twilio-Signature` válida responde 401.

### RCS

Webhook: `/api/v1/webhooks/rcs/{channelId}`; a tela não mostra a URL, monte à mão.

Preparação:

1. Criar o canal Zenvia com Agent ID, API Key e Sender ID; gravar `credentials.webhook_secret` por API e conectar pelo menu do card.
2. Cadastrar o webhook no painel do provedor.

Testes específicos:

- Receber duas mensagens do mesmo número e conferir que caem na mesma conversa.
- Responder pelo admin e conferir que chega no aparelho.
- Receber e enviar imagem (só Zenvia).
- Status de entrega e leitura.
- Assinatura inválida em `X-Signature` responde 401.

### Voz (Twilio)

Webhook: `/api/v1/webhooks/voice/{channelId}`. Só a Twilio está ligada, e só para chamada recebida.

Preparação:

1. Criar o canal Voz com provedor Twilio, Account SID, Auth Token e número; conectar pelo menu do card.
2. No console da Twilio, apontar o webhook de voz do número para a URL do canal.

Testes específicos:

- Ligar para o número: ouvir a saudação e gravar um recado de até 120 s.
- No admin, a conversa mostra o áudio da gravação, a transcrição e o fim da chamada com a duração.
- Digitar teclas durante a chamada: aparecem como mensagem.
- Desligar antes de atender ou deixar ocupado: evento registrado.
- Responder na conversa de voz: registrar o erro mostrado.

### Microsoft Teams

Webhook: `/api/v1/webhooks/teams/{channelId}`. Só aceita chamada assinada pelo Bot Framework; não dá para simular por curl.

Preparação:

1. Criar um Azure Bot com App ID e senha e habilitar o canal Teams.
2. Criar o canal no Linktor, rodar Testar Conexão (real) e conectar pelo menu do card.
3. Colar a URL do webhook como Messaging endpoint do bot e instalar o bot no Teams.

Testes específicos:

- Enviar mensagem ao bot em chat 1:1; só depois dessa primeira mensagem o Linktor consegue responder.
- Responder pelo admin com texto e com imagem.
- Receber imagem e arquivo.
- Mencionar o bot em um chat de grupo: registrar como o texto chega e quem aparece como contato.
- Senha errada em Testar Conexão falha.

### Slack

Webhook: `/api/v1/webhooks/slack/{channelId}`.

Preparação:

1. Criar um app Slack com bot token (`chat:write`, `files:write`, `im:history`, `channels:history`) e copiar o Signing Secret.
2. Criar o canal, rodar Testar Conexão (real, mostra time e bot) e conectar pelo menu do card.
3. Em Event Subscriptions, colar a URL (o desafio `url_verification` deve passar) e assinar `message.im` e `message.channels`. Convidar o bot para um canal.

Testes específicos:

- Receber DM e mensagem em canal; cada canal ou DM vira uma conversa.
- Receber arquivo.
- Responder com texto e com arquivo (upload nativo no Slack).
- Editar e apagar uma mensagem no Slack: registrar se aparece mensagem vazia no admin.
- Responder dentro de uma thread no Slack: registrar onde a resposta do admin aparece.
- A resposta do próprio bot não volta como mensagem recebida.
- Assinatura inválida ou timestamp com mais de 5 minutos responde 401.

### Mattermost

Sem webhook: o Linktor abre um WebSocket em `{base_url}/api/v4/websocket`. O backend precisa alcançar o servidor Mattermost.

Preparação:

1. Criar uma conta de bot no Mattermost, gerar o token e adicionar o bot a um canal.
2. Criar o canal no Linktor com Base URL e token, rodar Testar Conexão (real) e conectar pelo menu do card; é o connect que inicia a escuta.

Testes específicos:

- Receber mensagem em canal e em DM com o bot.
- Receber arquivo.
- Responder com texto e com arquivo.
- Trocar o token por um inválido: registrar se o card continua Conectado e o que aparece no log.
- Desconectar pelo menu do card: mensagens novas deixam de chegar; reconectar retoma.
- Reiniciar o backend com o canal conectado: a escuta volta sozinha.

### Direto (VendaX)

Webhook: `/api/v1/webhooks/direto/{channelId}`, assinado em `X-Direto-Signature-256`.

Preparação:

1. Ter o runtime Direto da VendaX ou um servidor mock que responda `POST {send_url}/channel/v1/{instance_id}/messages` com `{"messages":[{"id":"..."}]}`.
2. Criar o canal com Send URL, Instance ID, API Token e Segredo de Assinatura; conectar pelo menu do card.

Testes específicos:

- Receber texto pelo Direto ou por curl assinado.
- Responder texto e mídia; conferir a chamada recebida pelo runtime.
- Evento `typing` mostra o indicador de digitação na conversa aberta.
- Receber mídia: registrar se abre.
- Criar um canal sem API Token e tentar enviar: registrar o erro.
- Assinatura inválida responde 401.

## Pontos de atenção

A leitura do código apontou 19 comportamentos que fariam um teste falhar ou enganariam quem testa. Nenhum foi reproduzido com provedor real.

### Corrigidos no PR #102, a confirmar no teste

| Canal | O que acontecia | O que o teste deve mostrar agora |
| --- | --- | --- |
| Todos | A tela não tinha como conectar o canal, e canal desconectado descarta a entrada respondendo 200 | Menu do card com Conectar e Desconectar |
| Todos com webhook | A deduplicação usava chave global e mantinha a chave mesmo após 4xx | O mesmo corpo em outro canal passa; chamada recusada pode ser repetida |
| Telegram | A deduplicação usava o cabeçalho do secret token, fixo por canal: só a primeira mensagem a cada 6 h passava | Mensagens seguidas chegam todas |
| Telegram | `cmd/server/telegram_webhook.go` ficava fora do git por causa do padrão `server` no `.gitignore` | Webhook registrado no connect |
| WhatsApp Oficial, Facebook, Instagram | O formulário grava `verify_token` em `config` e o webhook só lia de `credentials` (403) | Verificação passa com o canal criado pela tela |
| WhatsApp Oficial, Facebook, Instagram | A verificação GET passava pela deduplicação: a segunda em 6 h voltava `duplicate` | Challenge devolvido em toda verificação |
| WhatsApp Oficial | A tela e o Embedded Signup mostravam `/webhooks/whatsapp_official/{id}`, rota inexistente | URL mostrada é `/webhooks/whatsapp/{id}` |
| WhatsApp Oficial | Webhooks de pagamentos, chamadas e CTWA respondiam sempre 404 | Respondem conforme o corpo enviado |
| Facebook | Canal criado por OAuth recebia, mas não enviava | Envio funciona |
| RCS | Cada mensagem criava contato e conversa novos | Mesmo número, mesma conversa |
| WebChat | Mídia do atendente chegava só com a legenda; botões chegavam vazios | Anexo aparece; botões chegam como texto |

### Em aberto

| Canal | O que o código indica | Efeito esperado no teste | Contorno | Onde |
| --- | --- | --- | --- | --- |
| WhatsApp Oficial | Analíticos por canal mandam o ID do Linktor como Phone Number ID | Erro da Graph API | Nenhum | `internal/api/handlers/whatsapp_analytics.go` |
| Facebook, Instagram | Template e botões não são suportados no envio | Falha permanente | Não testar | `internal/adapters/facebook/sender.go`, `internal/adapters/instagram/sender.go` |
| WebChat | Boas-vindas, mensagem offline, exigir nome e email, domínios permitidos e atraso de abertura não são lidos | Opções da tela sem efeito | Nenhum | `internal/adapters/webchat/adapter.go`, `web/embed/v1/linktor.js` |
| Email | Resposta do admin sai com assunto "Message" e sem cabeçalhos de thread | Cliente recebe em thread nova | Nenhum pela tela | `internal/adapters/email/sender.go` |
| Email | Anexo recebido por IMAP guarda só nome e tipo | Anexo não abre | Nenhum | `internal/adapters/email/imap_fetcher.go` |
| Email | Entrada SendGrid e Mailgun lê só `x-www-form-urlencoded`; status do Mailgun não casa com a mensagem | Email multipart chega vazio; sem Entregue | Preferir Postmark ou IMAP | `internal/api/handlers/webhook.go` |
| Voz | Não há envio registrado para `voice`; Testar Conexão não suporta o tipo | Resposta falha; teste de conexão sempre falha | Testar só chamada recebida | `internal/outbound/resolver.go`, `internal/api/handlers/channel.go` |
| Teams, Slack, Mattermost | Contato é o canal ou chat, não a pessoa; sem threads | Grupo inteiro vira um contato | Testar em 1:1 | `internal/api/handlers/webhook.go`, `internal/adapters/mattermost/listener.go` |

## Tarefas no Jira

São 15 tarefas no projeto LIN, todas sob o épico [LIN-1](https://relevantsolutions.atlassian.net/browse/LIN-1). Cada uma traz preparação, casos comuns, testes específicos e pontos de atenção do canal. A LIN-2 vem primeiro: as demais dependem do ambiente pronto.

| Tarefa | O que cobre |
| --- | --- |
| [LIN-2](https://relevantsolutions.atlassian.net/browse/LIN-2) | Preparar o ambiente de teste dos canais |
| [LIN-3](https://relevantsolutions.atlassian.net/browse/LIN-3) | WhatsApp Oficial (Cloud API): mensagens |
| [LIN-4](https://relevantsolutions.atlassian.net/browse/LIN-4) | WhatsApp Oficial: templates, coexistência e aba Operações |
| [LIN-5](https://relevantsolutions.atlassian.net/browse/LIN-5) | Facebook Messenger |
| [LIN-6](https://relevantsolutions.atlassian.net/browse/LIN-6) | Instagram |
| [LIN-7](https://relevantsolutions.atlassian.net/browse/LIN-7) | Telegram |
| [LIN-8](https://relevantsolutions.atlassian.net/browse/LIN-8) | WebChat (widget) |
| [LIN-9](https://relevantsolutions.atlassian.net/browse/LIN-9) | Email (SMTP/IMAP e provedores) |
| [LIN-10](https://relevantsolutions.atlassian.net/browse/LIN-10) | SMS (Twilio) |
| [LIN-11](https://relevantsolutions.atlassian.net/browse/LIN-11) | RCS |
| [LIN-12](https://relevantsolutions.atlassian.net/browse/LIN-12) | Voz (Twilio, chamada recebida) |
| [LIN-13](https://relevantsolutions.atlassian.net/browse/LIN-13) | Microsoft Teams |
| [LIN-14](https://relevantsolutions.atlassian.net/browse/LIN-14) | Slack |
| [LIN-15](https://relevantsolutions.atlassian.net/browse/LIN-15) | Mattermost |
| [LIN-16](https://relevantsolutions.atlassian.net/browse/LIN-16) | Direto (VendaX) |

Cada falha encontrada vira um Bug no LIN ligado à tarefa do canal.
