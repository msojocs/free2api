export function parseAccountExtra(extraRaw: string | undefined): Record<string, unknown> {
  if (!extraRaw) return {}
  try {
    const parsed = JSON.parse(extraRaw) as unknown
    if (parsed && typeof parsed === 'object') {
      return parsed as Record<string, unknown>
    }
    return {}
  } catch {
    return {}
  }
}

export function extractChatGPTAccountIdFromToken(accessToken: string): string {
  const parts = accessToken.split('.')
  if (parts.length < 2) return ''

  try {
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64 + '='.repeat((4 - (base64.length % 4 || 4)) % 4)
    const payload = JSON.parse(atob(padded)) as Record<string, unknown>
    const auth = payload['https://api.openai.com/auth']
    if (!auth || typeof auth !== 'object') return ''
    const id = (auth as Record<string, unknown>).chatgpt_account_id
    return typeof id === 'string' ? id : ''
  } catch {
    return ''
  }
}

export function extractJwtExp(accessToken: string): number | null {
  const parts = accessToken.split('.')
  if (parts.length < 2) return null

  try {
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64 + '='.repeat((4 - (base64.length % 4 || 4)) % 4)
    const payload = JSON.parse(atob(padded)) as Record<string, unknown>
    const exp = payload.exp
    return typeof exp === 'number' ? exp : null
  } catch {
    return null
  }
}

export function maskToken(value: string, head = 8, tail = 6): string {
  if (!value) return '-'
  if (value.length <= head + tail + 3) return value
  return `${value.slice(0, head)}...${value.slice(-tail)}`
}
