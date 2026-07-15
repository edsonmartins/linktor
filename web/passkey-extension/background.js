// MV3 service worker for the Linktor Passkey Connector.
//
// On RUN_PASSKEY_ASSERTION { requestId, publicKey } from a Linktor admin tab, it
// opens web.whatsapp.com in a focused tab, runs navigator.credentials.get with
// the server-issued challenge in the page's MAIN world (the account owner
// confirms with their passkey + 2FA PIN if any), and returns the assertion
// (PASSKEY_ASSERTION_RESULT) back to the originating tab. It reads nothing else
// and stores nothing.
//
// The assertion-running function (runPasskeyAssertionInPage) is kept verbatim
// from the reference implementation (takeflow-oficial/wa-passkey-connector,
// Unlicense) because its base64url encoding must match whatsmeow's strict
// RawURLEncoding on the Go side.

const WA_ORIGIN = 'https://web.whatsapp.com'
// Origins where the Linktor admin runs. Mirror this with the manifest
// content_scripts / host_permissions when deploying to other hosts.
const APP_HOST_PATTERNS = [
  'https://*.linktor.dev/*',
  'http://localhost/*',
  'http://127.0.0.1/*',
]

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (msg?.type === 'RUN_PASSKEY_ASSERTION' && msg.publicKey) {
    handlePasskeyAssertion(msg.publicKey, sender.tab?.id, msg.requestId).then(sendResponse)
    return true // async response
  }
  if (msg?.type === 'IS_CONNECTOR_INSTALLED') {
    sendResponse({ installed: true })
    return false
  }
  return false
})

// Inject the content bridge into admin tabs that were already open when the
// extension was installed/updated (new page loads get it via content_scripts).
chrome.runtime.onInstalled.addListener(() => {
  injectBridgeIntoOpenTabs()
})

async function injectBridgeIntoOpenTabs() {
  try {
    const tabs = await chrome.tabs.query({ url: APP_HOST_PATTERNS })
    for (const tab of tabs) {
      if (tab.id == null) continue
      try {
        await chrome.scripting.executeScript({ target: { tabId: tab.id }, files: ['content.js'] })
      } catch {}
    }
  } catch {}
}

async function handlePasskeyAssertion(publicKey, originTabId, requestId) {
  const reply = (payload) => {
    if (originTabId == null) return
    chrome.tabs
      .sendMessage(originTabId, { type: 'PASSKEY_ASSERTION_RESULT', requestId, ...payload })
      .catch(() => {})
  }

  let tabId
  try {
    const tab = await chrome.tabs.create({ url: `${WA_ORIGIN}/`, active: true })
    tabId = tab.id
    if (tabId == null) {
      reply({ error: 'tab_open_failed' })
      return { ok: false }
    }
    await waitForTabComplete(tabId)
    const [inj] = await chrome.scripting.executeScript({
      target: { tabId },
      world: 'MAIN',
      func: runPasskeyAssertionInPage,
      args: [publicKey],
    })
    const result = inj?.result
    if (result?.assertion) {
      reply({ assertion: result.assertion })
    } else {
      reply({ error: result?.error || 'assertion_failed' })
    }
    return { ok: true }
  } catch (error) {
    reply({ error: error instanceof Error ? error.message : 'assertion_exception' })
    return { ok: false }
  } finally {
    if (tabId != null) chrome.tabs.remove(tabId).catch(() => {})
  }
}

function waitForTabComplete(tabId) {
  return new Promise((resolve) => {
    const done = () => {
      chrome.tabs.onUpdated.removeListener(onUpdated)
      resolve()
    }
    const onUpdated = (id, info, tab) => {
      if (id === tabId && info.status === 'complete' && tab.url?.startsWith(`${WA_ORIGIN}/`)) {
        done()
      }
    }
    chrome.tabs.onUpdated.addListener(onUpdated)
    chrome.tabs
      .get(tabId)
      .then((tab) => {
        if (tab.status === 'complete' && tab.url?.startsWith(`${WA_ORIGIN}/`)) done()
      })
      .catch(() => {})
  })
}

// Runs in the web.whatsapp.com page MAIN world. Verbatim WebAuthn assertion:
// decodes the base64url challenge, calls navigator.credentials.get, and returns
// credential.toJSON() (already unpadded base64url, matching whatsmeow).
function runPasskeyAssertionInPage(inputPublicKey) {
  function base64UrlToBuffer(value) {
    const normalized = value.replace(/-/g, '+').replace(/_/g, '/')
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
    const binary = atob(padded)
    const bytes = new Uint8Array(binary.length)
    for (let i = 0; i < binary.length; i += 1) bytes[i] = binary.charCodeAt(i)
    return bytes.buffer
  }
  function bufferToBase64Url(value) {
    const bytes = new Uint8Array(value)
    let binary = ''
    bytes.forEach((byte) => {
      binary += String.fromCharCode(byte)
    })
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
  }
  async function run() {
    const publicKeyOptions = {
      challenge: base64UrlToBuffer(inputPublicKey.challenge),
      timeout: inputPublicKey.timeout,
      rpId: inputPublicKey.rpId,
      allowCredentials: (inputPublicKey.allowCredentials || []).map((credential) => ({
        id: base64UrlToBuffer(credential.id),
        type: 'public-key',
        transports: credential.transports,
      })),
      userVerification: inputPublicKey.userVerification,
      extensions: inputPublicKey.extensions,
    }

    const credential = await navigator.credentials.get({ publicKey: publicKeyOptions })
    if (!credential || credential.type !== 'public-key') {
      throw new Error('Passkey assertion was not completed')
    }
    if (typeof credential.toJSON === 'function') {
      return credential.toJSON()
    }
    const response = credential.response
    return {
      id: credential.id,
      rawId: bufferToBase64Url(credential.rawId),
      type: credential.type,
      response: {
        clientDataJSON: bufferToBase64Url(response.clientDataJSON),
        authenticatorData: bufferToBase64Url(response.authenticatorData),
        signature: bufferToBase64Url(response.signature),
        userHandle: response.userHandle ? bufferToBase64Url(response.userHandle) : null,
      },
    }
  }
  return run()
    .then((assertion) => ({ assertion }))
    .catch((error) => ({ error: error && error.message ? error.message : String(error) }))
}
