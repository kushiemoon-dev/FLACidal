// Token handling for the headless server's optional AUTH_TOKEN.
//
// Kept free of imports from api.ts: websocket.ts already imports api.ts, and
// api.ts needs this module, so a dependency back would be circular.

const STORAGE_KEY = 'flacidal.authToken'

export function getAuthToken(): string {
  try {
    return localStorage.getItem(STORAGE_KEY) ?? ''
  } catch {
    return ''
  }
}

export function setAuthToken(token: string): void {
  try {
    localStorage.setItem(STORAGE_KEY, token)
  } catch {
    // storage blocked (private mode): the token just won't survive a reload
  }
}

export function clearAuthToken(): void {
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    // nothing to clear when storage is unavailable
  }
}

export function authHeaders(): Record<string, string> {
  const token = getAuthToken()
  return token ? { Authorization: `Bearer ${token}` } : {}
}

const authListeners = new Set<() => void>()

export function onAuthRequired(cb: () => void): () => void {
  authListeners.add(cb)
  return () => {
    authListeners.delete(cb)
  }
}

export function notifyAuthRequired(): void {
  for (const cb of authListeners) cb()
}

// A browser WebSocket never sees the 401 of a refused upgrade, so a socket
// that closes before opening is followed by a plain request to find out.
export async function probeAuth(): Promise<void> {
  try {
    const res = await fetch('/api/version', { headers: authHeaders() })
    if (res.status === 401) notifyAuthRequired()
  } catch {
    // network errors are not an auth problem
  }
}
