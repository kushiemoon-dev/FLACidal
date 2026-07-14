import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

const wailsRuntimeMock = {
  EventsOn: vi.fn(),
  EventsOff: vi.fn(),
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

  // Test helper: simulate the server pushing a message.
  emit(data: unknown) {
    this.readyState = MockWebSocket.OPEN
    this.onmessage?.({ data: JSON.stringify(data) })
  }
}

describe('websocket EventsOn/EventsOff — Wails mode', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.clearAllMocks()
    setWailsRuntime()
  })
  afterEach(() => {
    clearWailsRuntime()
  })

  it('EventsOn delegates straight to the Wails runtime binding', async () => {
    const unsub = vi.fn()
    wailsRuntimeMock.EventsOn.mockReturnValue(unsub)

    const { EventsOn } = await import('./websocket')
    const cb = vi.fn()
    const result = EventsOn('download-progress', cb)

    expect(wailsRuntimeMock.EventsOn).toHaveBeenCalledWith('download-progress', cb)
    expect(result).toBe(unsub)
  })

  it('EventsOff delegates straight to the Wails runtime binding', async () => {
    const { EventsOff } = await import('./websocket')
    EventsOff('ffmpeg-install-progress')

    expect(wailsRuntimeMock.EventsOff).toHaveBeenCalledWith('ffmpeg-install-progress')
  })
})

describe('websocket EventsOn/EventsOff — browser mode', () => {
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

  it('opens a WebSocket to /ws on first subscription', async () => {
    const { EventsOn } = await import('./websocket')
    EventsOn('download-progress', vi.fn())

    expect(MockWebSocket.instances).toHaveLength(1)
    expect(MockWebSocket.instances[0].url).toMatch(/\/ws$/)
  })

  it('dispatches a download-progress message to matching listeners, stripped of the "type" wrapper', async () => {
    const { EventsOn } = await import('./websocket')
    const cb = vi.fn()
    EventsOn('download-progress', cb)

    const socket = MockWebSocket.instances[0]
    socket.emit({ type: 'download-progress', trackId: 42, status: 'completed', result: { filePath: '/music/a.flac' } })

    expect(cb).toHaveBeenCalledWith({ trackId: 42, status: 'completed', result: { filePath: '/music/a.flac' } })
  })

  it('never fires listeners for event names the /ws hub does not broadcast', async () => {
    const { EventsOn } = await import('./websocket')
    const cb = vi.fn()
    EventsOn('queue-paused', cb)

    const socket = MockWebSocket.instances[0]
    socket.emit({ type: 'download-progress', trackId: 1, status: 'queued', result: null })

    expect(cb).not.toHaveBeenCalled()
  })

  it('the unsubscribe function returned by EventsOn stops further dispatch', async () => {
    const { EventsOn } = await import('./websocket')
    const cb = vi.fn()
    const unsubscribe = EventsOn('download-progress', cb)
    unsubscribe()

    const socket = MockWebSocket.instances[0]
    socket.emit({ type: 'download-progress', trackId: 1, status: 'queued', result: null })

    expect(cb).not.toHaveBeenCalled()
  })

  it('EventsOff removes all listeners registered for that event name', async () => {
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

  it('reuses the existing socket for a second subscription instead of opening another one', async () => {
    const { EventsOn } = await import('./websocket')
    const socket = MockWebSocket.instances
    EventsOn('download-progress', vi.fn())
    MockWebSocket.instances[0].readyState = MockWebSocket.OPEN
    EventsOn('queue-paused', vi.fn())

    expect(socket).toHaveLength(1)
  })
})
