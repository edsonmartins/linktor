// Content script injected into Linktor admin pages. It is a thin bridge between
// the page (window.postMessage) and the extension service worker
// (chrome.runtime). The page never talks to chrome.* directly.
//
// Protocol (mirror of the WHATSMEOW-IMPLEMENTATION.md contract):
//   page -> extension  { target: 'linktor-passkey-connector', type: 'PING' | 'RUN_PASSKEY_ASSERTION' }
//   extension -> page  { source: 'linktor-passkey-connector', type: 'CONNECTOR_READY' | 'PASSKEY_ASSERTION_RESULT' }
(function () {
  const SOURCE = 'linktor-passkey-connector'
  const FROM_WORKER = ['PASSKEY_ASSERTION_RESULT']

  const guard = window
  if (guard.__linktorPasskeyConnectorBridge) return
  guard.__linktorPasskeyConnectorBridge = true

  const announce = () => {
    window.postMessage({ source: SOURCE, type: 'CONNECTOR_READY' }, '*')
  }

  // Relay results coming back from the service worker to the page.
  chrome.runtime.onMessage.addListener((msg) => {
    if (msg && typeof msg.type === 'string' && FROM_WORKER.includes(msg.type)) {
      window.postMessage({ source: SOURCE, ...msg }, '*')
    }
  })

  // Relay requests coming from the page to the service worker.
  window.addEventListener('message', (event) => {
    if (event.source !== window) return
    const data = event.data
    if (!data || data.target !== SOURCE) return

    if (data.type === 'PING') {
      announce()
    }
    if (data.type === 'RUN_PASSKEY_ASSERTION' && data.publicKey) {
      chrome.runtime.sendMessage({
        type: 'RUN_PASSKEY_ASSERTION',
        requestId: data.requestId,
        publicKey: data.publicKey,
      })
    }
  })

  announce()
})()
