# Linktor Passkey Connector (extensão de navegador)

Extensão **Manifest V3** que autoriza o vínculo de um canal **WhatsApp não-oficial**
(whatsmeow) do Linktor quando a conta está sob o novo regime de **passkey**
("Shortcake") do WhatsApp.

Contas com passkey **não podem** ser vinculadas por QR a partir de um cliente
headless: o servidor exige uma *assertion* WebAuthn assinada pelo autenticador do
dono da conta. O backend do Linktor dirige todo o handshake de pareamento; esta
extensão delega **a única coisa** que não dá pra fazer headless — a assinatura da
passkey — ao navegador do dono, e devolve a *assertion*. O resultado é um
**dispositivo novo vinculado**, não uma cópia da sessão do dono. Não lê nem move
a sessão do WhatsApp.

## Como funciona

1. O admin do Linktor detecta a extensão (`PING` → `CONNECTOR_READY`).
2. Quando um canal precisa de passkey, o admin busca o desafio no backend e envia
   `RUN_PASSKEY_ASSERTION { requestId, publicKey }` via `window.postMessage`.
3. A extensão abre o `web.whatsapp.com`, roda `navigator.credentials.get(desafio)`
   no *MAIN world* (o dono aprova com biometria/PIN), e devolve a *assertion* em
   `PASSKEY_ASSERTION_RESULT { requestId, assertion }`.
4. O admin faz `POST` da *assertion* pro backend, que chama `SendPasskeyResponse`.

Protocolo completo em `docs` do projeto de referência (ver Créditos).

## Instalação (dev, sem build)

Esta extensão é JS puro — **não precisa de build**.

1. Chrome/Edge → `chrome://extensions` → ative o **Modo do desenvolvedor**.
2. **Carregar sem compactação** → selecione esta pasta (`web/passkey-extension`).

## Configuração de origem

A extensão só conversa com as origens do admin declaradas em dois lugares (mantenha
os dois em sincronia):

- `manifest.json` → `host_permissions` **e** `content_scripts[].matches`
- `background.js` → `APP_HOST_PATTERNS`

Padrão: `https://*.linktor.dev/*`, `http://localhost/*`, `http://127.0.0.1/*`.
Adicione o domínio do seu admin se for diferente.

## Ícones

Sem ícones o navegador usa um placeholder. Para publicar, adicione
`icons/16.png`, `48.png`, `128.png` e uma chave `"icons"` no `manifest.json`.

## Permissões

`scripting`, `tabs`, `activeTab`, `storage` + `host_permissions` para
`web.whatsapp.com` (onde a assinatura roda) e as origens do admin. **Não** lê a
sessão do WhatsApp e não usa `browsingData`.

## Distribuição

Automatiza um `navigator.credentials.get` no `web.whatsapp.com` — pode tocar os
Termos do WhatsApp e políticas de loja de extensões. Sempre com **consentimento
explícito do dono da conta**; prefira distribuição privada/unlisted ou
empresarial a uma listagem pública.

## Créditos

Adaptado de [takeflow-oficial/wa-passkey-connector](https://github.com/takeflow-oficial/wa-passkey-connector)
(The Unlicense / domínio público). A função de assertion (`runPasskeyAssertionInPage`)
é mantida verbatim para casar com o `RawURLEncoding` estrito do whatsmeow no Go.
