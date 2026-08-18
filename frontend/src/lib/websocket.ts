// Real-time events layer that adapts to its runtime.
//
// Wails mode: EventsOn/EventsOff wrap the native runtime one-to-one
// (../../wailsjs/runtime/runtime.js), so App.svelte/Terminal.svelte/
// Settings.svelte keep behaving exactly as before — only the import path
// changes.
//
// Browser mode: connects to the headless server's /ws WebSocket hub
// (internal/api/server.go), which broadcasts download-progress events as
// {"type":"download-progress","trackId":N,"status":"...","result":{...}}
// (see cmd/server/main.go's DownloadManager.SetProgressCallback — this is
// the only event type the hub currently emits). Messages get unwrapped and
// redispatched to 'download-progress' listeners using the exact payload
// shape Wails emits ({trackId, status, result}), so App.svelte's handler
// needs no changes.
//
// Known gap: 'queue-paused', 'endpoint-cooldown', 'log',
// 'ffmpeg-install-progress', and 'sldl-install-progress' have no
// server-side broadcaster in headless mode yet (those are Wails-app-only
// features — see migration report). Subscribing to them in browser mode is
// harmless (no error), but the callback will just never fire.

import { EventsOn as WailsEventsOn, EventsOff as WailsEventsOff } from '../../wailsjs/runtime/runtime.js'
import { isWailsRuntime } from './api'

type EventCallback = (...data: any[]) => void

const browserListeners = new Map<string, Set<EventCallback>>()

let socket: WebSocket | null = null
let reconnectAttempt = 0
let reconnectTimer: ReturnType<typeof setTimeout> | null = null

function backoffDelayMs(attempt: number): number {
  return Math.min(1000 * 2 ** attempt, 30000)
}

function socketURL(): string {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${proto}//${window.location.host}/ws`
}

function dispatch(eventName: string, payload: any): void {
  const listeners = browserListeners.get(eventName)
  if (!listeners) return
  for (const cb of listeners) {
    try {
      cb(payload)
    } catch (err) {
      console.error(`listener for websocket event '${eventName}' raised an error:`, err)
    }
  }
}

function handleMessage(event: MessageEvent): void {
  let msg: any
  try {
    msg = JSON.parse(event.data)
  } catch {
    return
  }

  if (msg?.type === 'download-progress') {
    dispatch('download-progress', { trackId: msg.trackId, status: msg.status, result: msg.result })
  }
}

function scheduleReconnect(): void {
  if (reconnectTimer) return
  const delay = backoffDelayMs(reconnectAttempt++)
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null
    connect()
  }, delay)
}

function connect(): void {
  if (typeof WebSocket === 'undefined') return
  if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) return

  socket = new WebSocket(socketURL())
  socket.onopen = () => {
    reconnectAttempt = 0
  }
  socket.onmessage = handleMessage
  socket.onerror = () => {
    socket?.close()
  }
  socket.onclose = () => {
    scheduleReconnect()
  }
}

// Lazily opens the /ws connection on first subscription; the returned
// unsubscribe function matches Wails' EventsOn contract.
export function EventsOn(eventName: string, callback: EventCallback): () => void {
  if (isWailsRuntime()) {
    return WailsEventsOn(eventName, callback)
  }

  connect()
  let listeners = browserListeners.get(eventName)
  if (!listeners) {
    listeners = new Set()
    browserListeners.set(eventName, listeners)
  }
  listeners.add(callback)

  return () => {
    browserListeners.get(eventName)?.delete(callback)
  }
}

export function EventsOff(eventName: string, ...additionalEventNames: string[]): void {
  if (isWailsRuntime()) {
    WailsEventsOff(eventName, ...additionalEventNames)
    return
  }
  for (const name of [eventName, ...additionalEventNames]) {
    browserListeners.delete(name)
  }
}
