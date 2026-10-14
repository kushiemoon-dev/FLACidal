import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

// The Node version running vitest may expose a non-functional global
// localStorage, so tests install their own.
function stubStorage(overrides: Partial<Storage> = {}) {
  const data = new Map<string, string>()
  vi.stubGlobal('localStorage', {
    getItem: (k: string) => data.get(k) ?? null,
    setItem: (k: string, v: string) => void data.set(k, v),
    removeItem: (k: string) => void data.delete(k),
    ...overrides,
  })
}

describe('auth token storage', () => {
  beforeEach(() => {
    vi.resetModules()
    stubStorage()
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('stores, reads and clears the token', async () => {
    const { getAuthToken, setAuthToken, clearAuthToken } = await import('./auth')
    expect(getAuthToken()).toBe('')
    setAuthToken('s3cret')
    expect(getAuthToken()).toBe('s3cret')
    clearAuthToken()
    expect(getAuthToken()).toBe('')
  })

  it('authHeaders is empty without a token and a Bearer header with one', async () => {
    const { authHeaders, setAuthToken } = await import('./auth')
    expect(authHeaders()).toEqual({})
    setAuthToken('s3cret')
    expect(authHeaders()).toEqual({ Authorization: 'Bearer s3cret' })
  })

  it('does not crash when localStorage throws', async () => {
    const blocked = () => {
      throw new Error('blocked')
    }
    stubStorage({ getItem: blocked, setItem: blocked, removeItem: blocked })
    const { getAuthToken, setAuthToken, clearAuthToken, authHeaders } = await import('./auth')
    expect(getAuthToken()).toBe('')
    expect(() => setAuthToken('x')).not.toThrow()
    expect(() => clearAuthToken()).not.toThrow()
    expect(authHeaders()).toEqual({})
  })
})

describe('onAuthRequired', () => {
  beforeEach(() => {
    vi.resetModules()
  })

  it('calls every listener and stops after unsubscribe', async () => {
    const { onAuthRequired, notifyAuthRequired } = await import('./auth')
    const a = vi.fn()
    const b = vi.fn()
    const offA = onAuthRequired(a)
    onAuthRequired(b)

    notifyAuthRequired()
    expect(a).toHaveBeenCalledOnce()
    expect(b).toHaveBeenCalledOnce()

    offA()
    notifyAuthRequired()
    expect(a).toHaveBeenCalledOnce()
    expect(b).toHaveBeenCalledTimes(2)
  })
})

describe('probeAuth', () => {
  beforeEach(() => {
    vi.resetModules()
    stubStorage()
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('notifies when /api/version answers 401', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ status: 401 }))
    const { probeAuth, onAuthRequired } = await import('./auth')
    const cb = vi.fn()
    onAuthRequired(cb)
    await probeAuth()
    expect(cb).toHaveBeenCalledOnce()
  })

  it('stays silent on 200 and on network errors', async () => {
    const { probeAuth, onAuthRequired } = await import('./auth')
    const cb = vi.fn()
    onAuthRequired(cb)
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ status: 200 }))
    await probeAuth()
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('offline')))
    await probeAuth()
    expect(cb).not.toHaveBeenCalled()
  })

  it('sends the stored token', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ status: 200 })
    vi.stubGlobal('fetch', fetchMock)
    const { probeAuth, setAuthToken } = await import('./auth')
    setAuthToken('s3cret')
    await probeAuth()
    expect(fetchMock).toHaveBeenCalledWith('/api/version', { headers: { Authorization: 'Bearer s3cret' } })
  })
})
