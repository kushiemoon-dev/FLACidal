// Runtime-detecting real-time events layer.
//
// Wails mode: EventsOn/EventsOff wrap the native runtime 1:1 (../../wailsjs/
// runtime/runtime.js), so App.svelte/Terminal.svelte/Settings.svelte keep
// their exact existing behavior — only the import path changes.
//
// Browser mode: connects to the headless server's /ws WebSocket hub
// (internal/api/server.go), which broadcasts download-progress events as
// {"type":"download-progress","trackId":N,"status":"...","result":{...}}
// (see cmd/server/main.go's DownloadManager.SetProgressCallback -> this is
// the only event type the hub currently carries). Messages are unwrapped
// and redispatched to 'download-progress' listeners with the exact same
// payload shape Wails emits ({trackId, status, result}), so App.svelte's
// handler works unchanged.
//
// Known gap: 'queue-paused', 'endpoint-cooldown', 'log',
// 'ffmpeg-install-progress' and 'sldl-install-progress' have no server-side
// broadcaster in headless mode today (those are Wails-app-only features —
// see migration report). Subscribing to them in browser mode is safe (no
// error) but the callback will simply never fire.

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
      console.error(`websocket listener for '${eventName}' threw:`, err)
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

/**
 * Subscribes to a real-time event by name. Wraps Wails' EventsOn in Wails
 * mode; lazily opens the /ws connection on first subscription in browser
 * mode. Returns an unsubscribe function (same contract as Wails' EventsOn).
 */
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

/** Unsubscribes all listeners for the given event name(s). Mirrors Wails' EventsOff. */
export function EventsOff(eventName: string, ...additionalEventNames: string[]): void {
  if (isWailsRuntime()) {
    WailsEventsOff(eventName, ...additionalEventNames)
    return
  }
  for (const name of [eventName, ...additionalEventNames]) {
    browserListeners.delete(name)
  }
}
