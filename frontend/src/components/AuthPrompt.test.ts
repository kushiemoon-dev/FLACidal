import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import type { flushSync as FlushSync } from 'svelte'

type WailsWindow = { go?: unknown; runtime?: unknown }

function stubStorage(token?: string) {
  const data = new Map<string, string>(token ? [['flacidal.authToken', token]] : [])
  vi.stubGlobal('localStorage', {
    getItem: (k: string) => data.get(k) ?? null,
    setItem: (k: string, v: string) => void data.set(k, v),
    removeItem: (k: string) => void data.delete(k),
  })
  return data
}

async function setup(opts: { wails?: boolean; token?: string } = {}) {
  vi.resetModules()
  const data = stubStorage(opts.token)
  if (opts.wails) {
    ;(window as unknown as WailsWindow).go = { app: { App: {} } }
    ;(window as unknown as WailsWindow).runtime = {}
  }
  // svelte is re-imported with the component so both share one runtime copy.
  const { mount, unmount, flushSync } = await import('svelte')
  const auth = await import('../lib/auth')
  const { default: AuthPrompt } = await import('./AuthPrompt.svelte')
  const target = document.createElement('div')
  document.body.appendChild(target)
  const component = mount(AuthPrompt, { target })
  flushSync()
  return {
    flush: flushSync as typeof FlushSync,
    auth,
    data,
    target,
    cleanup: () => {
      unmount(component)
      target.remove()
    },
  }
}

describe('AuthPrompt', () => {
  beforeEach(() => {
    document.body.innerHTML = ''
  })
  afterEach(() => {
    delete (window as unknown as WailsWindow).go
    delete (window as unknown as WailsWindow).runtime
    vi.unstubAllGlobals()
  })

  it('renders nothing until a 401 is reported', async () => {
    const { target, auth, flush, cleanup } = await setup()
    expect(target.querySelector('input[type="password"]')).toBeNull()
    auth.notifyAuthRequired()
    flush()
    expect(target.querySelector('input[type="password"]')).not.toBeNull()
    expect(target.querySelector('[data-testid="auth-rejected"]')).toBeNull()
    cleanup()
  })

  it('stays invisible under Wails even after a notification', async () => {
    const { target, auth, flush, cleanup } = await setup({ wails: true })
    auth.notifyAuthRequired()
    flush()
    expect(target.querySelector('input[type="password"]')).toBeNull()
    cleanup()
  })

  it('drops a rejected stored token and says so', async () => {
    const { target, auth, data, flush, cleanup } = await setup({ token: 'old' })
    auth.notifyAuthRequired()
    flush()
    expect(data.has('flacidal.authToken')).toBe(false)
    expect(target.querySelector('[data-testid="auth-rejected"]')).not.toBeNull()
    cleanup()
  })

  it('stores the trimmed token and reloads on submit', async () => {
    const reload = vi.fn()
    vi.stubGlobal('location', { ...window.location, reload })
    const { target, auth, data, flush, cleanup } = await setup()
    auth.notifyAuthRequired()
    flush()

    const input = target.querySelector('input[type="password"]') as HTMLInputElement
    input.value = '  s3cret  '
    input.dispatchEvent(new Event('input', { bubbles: true }))
    ;(target.querySelector('form') as HTMLFormElement).dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    flush()

    expect(data.get('flacidal.authToken')).toBe('s3cret')
    expect(reload).toHaveBeenCalledOnce()
    cleanup()
  })

  it('ignores an empty submit', async () => {
    const reload = vi.fn()
    vi.stubGlobal('location', { ...window.location, reload })
    const { target, auth, data, flush, cleanup } = await setup()
    auth.notifyAuthRequired()
    flush()
    ;(target.querySelector('form') as HTMLFormElement).dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    flush()
    expect(data.size).toBe(0)
    expect(reload).not.toHaveBeenCalled()
    cleanup()
  })
})
