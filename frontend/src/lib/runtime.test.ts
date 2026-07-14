import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

const wailsRuntimeMock = {
  BrowserOpenURL: vi.fn(),
  OnFileDrop: vi.fn(),
  OnFileDropOff: vi.fn(),
}
vi.mock('../../wailsjs/runtime/runtime.js', () => wailsRuntimeMock)

function setWailsRuntime() {
  ;(window as any).go = { app: { App: {} } }
  ;(window as any).runtime = {}
}

function clearWailsRuntime() {
  delete (window as any).go
  delete (window as any).runtime
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

  it('Wails mode: delegates to BrowserOpenURL', async () => {
    setWailsRuntime()
    const { OpenExternalURL } = await import('./runtime')
    OpenExternalURL('https://example.com')

    expect(wailsRuntimeMock.BrowserOpenURL).toHaveBeenCalledWith('https://example.com')
  })

  it('browser mode: opens a new tab via window.open', async () => {
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

  it('Wails mode: wraps OnFileDrop and returns a cleanup that calls OnFileDropOff', async () => {
    setWailsRuntime()
    const { onNativeFileDrop } = await import('./runtime')
    const cb = vi.fn()

    const unsubscribe = onNativeFileDrop(cb, false)

    expect(wailsRuntimeMock.OnFileDrop).toHaveBeenCalledWith(cb, false)
    expect(wailsRuntimeMock.OnFileDropOff).not.toHaveBeenCalled()

    unsubscribe()
    expect(wailsRuntimeMock.OnFileDropOff).toHaveBeenCalledOnce()
  })

  it('browser mode: never touches window.runtime and returns a no-op cleanup', async () => {
    clearWailsRuntime()
    const { onNativeFileDrop } = await import('./runtime')
    const cb = vi.fn()

    const unsubscribe = onNativeFileDrop(cb, false)

    expect(wailsRuntimeMock.OnFileDrop).not.toHaveBeenCalled()
    expect(() => unsubscribe()).not.toThrow()
    expect(wailsRuntimeMock.OnFileDropOff).not.toHaveBeenCalled()
  })
})
