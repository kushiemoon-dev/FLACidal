import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

type WailsWindow = { go?: unknown; runtime?: unknown }

const wailsRuntimeMock = {
  EventsOn: vi.fn(),
  EventsOff: vi.fn(),
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

class MockWebSocket {
  static OPEN = 1
  static CONNECTING = 0
  static CLOSED = 3
  static instances: MockWebSocket[] = []

  readyState = MockWebSocket.CONNECTING
  url: string
  onopen: (() => void) | null = null
  onmessage: ((event: { data: string }) => void) | null = null
  onerror: (() => void) | null = null
  onclose: (() => void) | null = null

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }

  close() {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.()
  }

  emit(data: unknown) {
    this.readyState = MockWebSocket.OPEN
    this.onmessage?.({ data: JSON.stringify(data) })
  }
}

describe('EventsOn/EventsOff over websocket (Wails mode)', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    setWailsRuntime()
  })
  afterEach(() => {
    clearWailsRuntime()
  })

  it('EventsOn forwards directly to the Wails runtime binding', async () => {
    const unsub = vi.fn()
    wailsRuntimeMock.EventsOn.mockReturnValue(unsub)

    const { EventsOn } = await import('./websocket')
    const cb = vi.fn()
    const result = EventsOn('download-progress', cb)

    expect(wailsRuntimeMock.EventsOn).toHaveBeenCalledWith('download-progress', cb)
    expect(result).toBe(unsub)
  })

  it('EventsOff forwards directly to the Wails runtime binding', async () => {
    const { EventsOff } = await import('./websocket')
    EventsOff('ffmpeg-install-progress')

    expect(wailsRuntimeMock.EventsOff).toHaveBeenCalledWith('ffmpeg-install-progress')
  })
})

describe('EventsOn/EventsOff over websocket (browser mode)', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    clearWailsRuntime()
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket)
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('opens a WebSocket connection to /ws on the first subscription', async () => {
    const { EventsOn } = await import('./websocket')
    EventsOn('download-progress', vi.fn())

    expect(MockWebSocket.instances).toHaveLength(1)
    expect(MockWebSocket.instances[0].url).toMatch(/\/ws$/)
  })

  it('delivers a download-progress message to matching listeners with the "type" wrapper stripped off', async () => {
    const { EventsOn } = await import('./websocket')
    const cb = vi.fn()
    EventsOn('download-progress', cb)

    const socket = MockWebSocket.instances[0]
    socket.emit({ type: 'download-progress', trackId: 42, status: 'completed', result: { filePath: '/music/a.flac' } })

    expect(cb).toHaveBeenCalledWith({ trackId: 42, status: 'completed', result: { filePath: '/music/a.flac' } })
  })

  it('skips listeners for event names the /ws hub never broadcasts', async () => {
    const { EventsOn } = await import('./websocket')
    const cb = vi.fn()
    EventsOn('queue-paused', cb)

    const socket = MockWebSocket.instances[0]
    socket.emit({ type: 'download-progress', trackId: 1, status: 'queued', result: null })

    expect(cb).not.toHaveBeenCalled()
  })

  it('calling the unsubscribe function from EventsOn halts further dispatch', async () => {
    const { EventsOn } = await import('./websocket')
    const cb = vi.fn()
    const unsubscribe = EventsOn('download-progress', cb)
    unsubscribe()

    const socket = MockWebSocket.instances[0]
    socket.emit({ type: 'download-progress', trackId: 1, status: 'queued', result: null })

    expect(cb).not.toHaveBeenCalled()
  })

  it('EventsOff clears every listener registered under that event name', async () => {
    const { EventsOn, EventsOff } = await import('./websocket')
    const cb1 = vi.fn()
    const cb2 = vi.fn()
    EventsOn('download-progress', cb1)
    EventsOn('download-progress', cb2)
    EventsOff('download-progress')

    const socket = MockWebSocket.instances[0]
    socket.emit({ type: 'download-progress', trackId: 1, status: 'queued', result: null })

    expect(cb1).not.toHaveBeenCalled()
    expect(cb2).not.toHaveBeenCalled()
  })

  it('reuses the existing socket for a second subscription rather than opening a new one', async () => {
    const { EventsOn } = await import('./websocket')
    const socket = MockWebSocket.instances
    EventsOn('download-progress', vi.fn())
    MockWebSocket.instances[0].readyState = MockWebSocket.OPEN
    EventsOn('queue-paused', vi.fn())

    expect(socket).toHaveLength(1)
  })
})

describe('websocket auth token (browser mode)', () => {
  function stubStorage(token?: string) {
    const data = new Map<string, string>(token ? [['flacidal.authToken', token]] : [])
    vi.stubGlobal('localStorage', {
      getItem: (k: string) => data.get(k) ?? null,
      setItem: (k: string, v: string) => void data.set(k, v),
      removeItem: (k: string) => void data.delete(k),
    })
  }

  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    clearWailsRuntime()
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket)
    // Keep the reconnect backoff timer from firing into later tests.
    vi.useFakeTimers({ toFake: ['setTimeout'] })
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('connects to /ws without a query when no token is stored', async () => {
    stubStorage()
    const { EventsOn } = await import('./websocket')
    EventsOn('download-progress', vi.fn())
    expect(MockWebSocket.instances[0].url).toMatch(/^wss?:\/\/[^?]+\/ws$/)
  })

  it('appends the URL-encoded token as ?token=', async () => {
    stubStorage('s3cret/x y')
    const { EventsOn } = await import('./websocket')
    EventsOn('download-progress', vi.fn())
    expect(MockWebSocket.instances[0].url.endsWith('/ws?token=s3cret%2Fx%20y')).toBe(true)
  })

  it('probes /api/version and notifies when the socket closes before ever opening', async () => {
    stubStorage()
    const fetchMock = vi.fn().mockResolvedValue({ status: 401 })
    vi.stubGlobal('fetch', fetchMock)
    const { onAuthRequired } = await import('./auth')
    const cb = vi.fn()
    onAuthRequired(cb)
    const { EventsOn } = await import('./websocket')
    EventsOn('download-progress', vi.fn())

    MockWebSocket.instances[0].close()
    await vi.advanceTimersByTimeAsync(0)

    expect(fetchMock).toHaveBeenCalledOnce()
    expect(cb).toHaveBeenCalledOnce()
  })

  it('does not probe when the socket had opened before closing', async () => {
    stubStorage()
    const fetchMock = vi.fn().mockResolvedValue({ status: 401 })
    vi.stubGlobal('fetch', fetchMock)
    const { EventsOn } = await import('./websocket')
    EventsOn('download-progress', vi.fn())

    const socket = MockWebSocket.instances[0]
    socket.onopen?.()
    socket.close()
    await vi.advanceTimersByTimeAsync(0)

    expect(fetchMock).not.toHaveBeenCalled()
  })
})

describe('websocket auth token (Wails mode)', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    setWailsRuntime()
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket)
  })
  afterEach(() => {
    clearWailsRuntime()
    vi.unstubAllGlobals()
  })

  it('opens no socket and sends no probe', async () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    const { EventsOn } = await import('./websocket')
    EventsOn('download-progress', vi.fn())
    expect(MockWebSocket.instances).toHaveLength(0)
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
