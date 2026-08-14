/**
 * Copia texto para a área de transferência, com fallback.
 *
 * `navigator.clipboard` só existe em contexto seguro (HTTPS ou localhost). A
 * instalação on-premises é servida por HTTP puro numa rede interna, então lá a
 * API simplesmente não existe e todo botão de copiar falhava em silêncio — o
 * usuário clicava para copiar a URL do webhook e nada acontecia, sem erro.
 *
 * Retorna se a cópia funcionou, para quem chama não anunciar sucesso que não
 * houve.
 */
export async function copyText(text: string): Promise<boolean> {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // Permissão negada ou contexto inseguro — cai no fallback abaixo.
    }
  }

  try {
    const field = document.createElement('textarea')
    field.value = text
    field.setAttribute('readonly', '')
    field.style.position = 'fixed'
    field.style.opacity = '0'
    document.body.appendChild(field)
    field.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(field)
    return ok
  } catch {
    return false
  }
}
