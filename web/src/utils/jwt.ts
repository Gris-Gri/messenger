export function parseUserIdFromAccessToken(token: string): number | null {
  try {
    const payload = token.split('.')[1]
    if (!payload) {
      return null
    }

    const normalized = payload.replace(/-/g, '+').replace(/_/g, '/')
    const decoded = JSON.parse(atob(normalized)) as { sub?: string }
    if (typeof decoded.sub !== 'string') {
      return null
    }

    const userId = Number.parseInt(decoded.sub, 10)
    return Number.isFinite(userId) ? userId : null
  } catch {
    return null
  }
}
