// Real-time events layer that adapts to its runtime.
//
// Wails mode: EventsOn/EventsOff wrap the native runtime one-to-one
// (../../wailsjs/runtime/runtime.js), so App.svelte/Terminal.svelte/
// Settings.svelte keep behaving exactly as before, only the import path
// changes.
//
// Browser mode: connects to the headless server's /ws WebSocket hub
// (internal/api/server.go). Messages get unwrapped and redispatched under the
// exact payload shape Wails emits, so listeners need no change:
//   download-progress      {type, trackId, status, result}
//   log                    {type, timestamp, level, message}
//   ffmpeg-install-progress {type, stage, percent, error}
// ws-reconnected is local: fired when the socket opens again after a drop,
// so a listener can reload what it may have missed (Terminal reloads its logs).
//
// Known gap: 'queue-paused' and 'endpoint-cooldown' have no server-side
// broadcaster in headless mode yet. Subscribing to them in browser mode is
// harmless (no error), but the callback will just never fire.

import { EventsOn as WailsEventsOn, EventsOff as WailsEventsOff } from '../../wailsjs/runtime/runtime.js'
import { isWailsRuntime } from './api'
import { getAuthToken, probeAuth } from './auth'

// `never[]` params let callers pass handlers typed for their own payload.
type EventCallback = (...data: never[]) => void

const browserListeners = new Map<string, Set<EventCallback>>()

let socket: WebSocket | null = null
let reconnectAttempt = 0
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let hasConnectedBefore = false

function backoffDelayMs(attempt: number): number {
  return Math.min(1000 * 2 ** attempt, 30000)
}

function socketURL(): string {
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = getAuthToken()
  const query = token ? `?token=${encodeURIComponent(token)}` : ''
  return `${proto}//${window.location.host}/ws${query}`
}

function dispatch(eventName: string, payload: unknown): void {
  const listeners = browserListeners.get(eventName)
  if (!listeners) return
  for (const cb of listeners) {
    try {
      ;(cb as (data: unknown) => void)(payload)
    } catch (err) {
      console.error(`listener for websocket event '${eventName}' raised an error:`, err)
    }
  }
}

function handleMessage(event: MessageEvent): void {
  let msg: {
    type?: string
    trackId?: number
    status?: string
    result?: unknown
    timestamp?: string
    level?: string
    message?: string
    stage?: string
    percent?: number
    error?: string
  } | null
  try {
    msg = JSON.parse(event.data)
  } catch {
    return
  }

  if (msg?.type === 'download-progress') {
    dispatch('download-progress', { trackId: msg.trackId, status: msg.status, result: msg.result })
  } else if (msg?.type === 'log') {
    dispatch('log', { timestamp: msg.timestamp, level: msg.level, message: msg.message })
  } else if (msg?.type === 'ffmpeg-install-progress') {
    dispatch('ffmpeg-install-progress', { stage: msg.stage, percent: msg.percent, error: msg.error })
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

  let opened = false
  socket = new WebSocket(socketURL())
  socket.onopen = () => {
    opened = true
    reconnectAttempt = 0
    if (hasConnectedBefore) dispatch('ws-reconnected', null)
    hasConnectedBefore = true
  }
  socket.onmessage = handleMessage
  socket.onerror = () => {
    socket?.close()
  }
  socket.onclose = () => {
    // A refused upgrade (401) is invisible to the browser, so ask over HTTP.
    if (!opened) void probeAuth()
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
