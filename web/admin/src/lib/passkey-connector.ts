/**
 * Client for the Linktor Passkey Connector browser extension.
 *
 * When a WhatsApp (unofficial) account is passkey-locked ("Shortcake"), the QR
 * handshake can't proceed and the account owner must sign a WebAuthn challenge
 * in their own browser. The extension runs `navigator.credentials.get` on the
 * `web.whatsapp.com` origin and hands the assertion back. This module speaks the
 * small `window.postMessage` protocol to it.
 *
 * page -> extension  { target: 'linktor-passkey-connector', type: 'PING' | 'RUN_PASSKEY_ASSERTION' }
 * extension -> page  { source: 'linktor-passkey-connector', type: 'CONNECTOR_READY' | 'PASSKEY_ASSERTION_RESULT' }
 */

const SOURCE = 'linktor-passkey-connector'

/** The server-issued WebAuthn challenge, mirror of Go's types.WebAuthnPublicKey. */
export interface PasskeyPublicKey {
  challenge: string
  timeout?: number
  rpId: string
  allowCredentials?: Array<{ id: string; type: string; transports?: string[] }>
  userVerification?: string
  extensions?: Record<string, unknown>
}

/** The browser assertion, mirror of Go's types.WebAuthnResponse. Forward verbatim. */
export type PasskeyAssertion = Record<string, unknown>

/**
 * Resolves true if the extension announces itself within timeoutMs. Pings a few
 * times to cover the content script attaching a beat late.
 */
export function detectPasskeyConnector(timeoutMs = 1500): Promise<boolean> {
  if (typeof window === 'undefined') return Promise.resolve(false)
  return new Promise((resolve) => {
    let done = false
    const onMsg = (e: MessageEvent) => {
      if (
        e.source === window &&
        e.data?.source === SOURCE &&
        e.data?.type === 'CONNECTOR_READY'
      ) {
        finish(true)
      }
    }
    const finish = (result: boolean) => {
      if (done) return
      done = true
      clearInterval(iv)
      clearTimeout(to)
      window.removeEventListener('message', onMsg)
      resolve(result)
    }
    window.addEventListener('message', onMsg)
    const ping = () => window.postMessage({ target: SOURCE, type: 'PING' }, '*')
    ping()
    const iv = setInterval(ping, 300)
    const to = setTimeout(() => finish(false), timeoutMs)
  })
}

/**
 * Drives one passkey assertion through the extension. Resolves with the assertion
 * (to be POSTed verbatim to the backend) or rejects with an error/timeout. The OS
 * passkey prompt can sit open a while, hence the generous default timeout.
 */
export function runPasskeyAssertion(
  publicKey: PasskeyPublicKey,
  timeoutMs = 120000,
): Promise<PasskeyAssertion> {
  return new Promise((resolve, reject) => {
    if (typeof window === 'undefined') {
      reject(new Error('no window'))
      return
    }
    const requestId =
      typeof crypto !== 'undefined' && 'randomUUID' in crypto
        ? crypto.randomUUID()
        : String(Date.now()) + Math.random().toString(16).slice(2)

    const onMsg = (e: MessageEvent) => {
      if (e.source !== window || e.data?.source !== SOURCE) return
      if (e.data?.type !== 'PASSKEY_ASSERTION_RESULT' || e.data?.requestId !== requestId) return
      cleanup()
      if (e.data.assertion) {
        resolve(e.data.assertion as PasskeyAssertion)
      } else {
        reject(new Error(e.data.error || 'passkey_assertion_failed'))
      }
    }
    const cleanup = () => {
      window.removeEventListener('message', onMsg)
      clearTimeout(t)
    }
    const t = setTimeout(() => {
      cleanup()
      reject(new Error('timeout'))
    }, timeoutMs)

    window.addEventListener('message', onMsg)
    window.postMessage({ target: SOURCE, type: 'RUN_PASSKEY_ASSERTION', requestId, publicKey }, '*')
  })
}
