import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

type WailsWindow = { go?: unknown; runtime?: unknown }

const wailsRuntimeMock = {
  BrowserOpenURL: vi.fn(),
  OnFileDrop: vi.fn(),
  OnFileDropOff: vi.fn(),
}
vi.mock('../../wailsjs/runtime/runtime.js', () => wailsRuntimeMock)

function setWailsRuntime() {
  ;(window as unknown as WailsWindow).go = { app: { App: {} } }
  ;(window as unknown as WailsWindow).runtime = {}
}

function clearWailsRuntime() {
  delete (window as unknown as WailsWindow).go
  delete (window as unknown as WailsWindow).runtime
}

describe('OpenExternalURL', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
  })
  afterEach(() => {
    clearWailsRuntime()
    vi.unstubAllGlobals()
  })

  it('Wails mode: hands off to BrowserOpenURL', async () => {
    setWailsRuntime()
    const { OpenExternalURL } = await import('./runtime')
    OpenExternalURL('https://example.com')

    expect(wailsRuntimeMock.BrowserOpenURL).toHaveBeenCalledWith('https://example.com')
  })

  it('browser mode: opens a new tab through window.open', async () => {
    clearWailsRuntime()
    const openSpy = vi.fn()
    vi.stubGlobal('open', openSpy)

    const { OpenExternalURL } = await import('./runtime')
    OpenExternalURL('https://example.com')

    expect(openSpy).toHaveBeenCalledWith('https://example.com', '_blank', 'noopener,noreferrer')
    expect(wailsRuntimeMock.BrowserOpenURL).not.toHaveBeenCalled()
  })
})

describe('onNativeFileDrop', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
  })
  afterEach(() => {
    clearWailsRuntime()
  })

  it('Wails mode: wraps OnFileDrop and returns a cleanup that invokes OnFileDropOff', async () => {
    setWailsRuntime()
    const { onNativeFileDrop } = await import('./runtime')
    const cb = vi.fn()

    const unsubscribe = onNativeFileDrop(cb, false)

    expect(wailsRuntimeMock.OnFileDrop).toHaveBeenCalledWith(cb, false)
    expect(wailsRuntimeMock.OnFileDropOff).not.toHaveBeenCalled()

    unsubscribe()
    expect(wailsRuntimeMock.OnFileDropOff).toHaveBeenCalledOnce()
  })

  it('browser mode: leaves window.runtime untouched and returns a no-op cleanup', async () => {
    clearWailsRuntime()
    const { onNativeFileDrop } = await import('./runtime')
    const cb = vi.fn()

    const unsubscribe = onNativeFileDrop(cb, false)

    expect(wailsRuntimeMock.OnFileDrop).not.toHaveBeenCalled()
    expect(() => unsubscribe()).not.toThrow()
    expect(wailsRuntimeMock.OnFileDropOff).not.toHaveBeenCalled()
  })
})
