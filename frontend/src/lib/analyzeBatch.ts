import { AnalyzeBatch, CancelAnalyzeBatch, type AnalysisResult, type AnalyzeBatchOutcome } from './api'
import { EventsOn } from './websocket'

// Mirrors core.MaxAnalyzeBatch; the server enforces it too (HTTP 413).
export const MAX_ANALYZE_BATCH = 5000

interface ProgressEvent {
  batchId: string
  done: number
  total: number
  result: AnalysisResult
}

// crypto.randomUUID only exists in secure contexts; plain http on a LAN address has getRandomValues only.
function newBatchId(): string {
  if (typeof crypto.randomUUID === 'function') return crypto.randomUUID()
  return Array.from(crypto.getRandomValues(new Uint8Array(16)), (b) => b.toString(16).padStart(2, '0')).join('')
}

/**
 * Starts a cancellable batch analysis. `onProgress` fires once per file, in
 * both the desktop app and the browser. The event subscription is made before
 * the call so no early progress is missed, and removed when the call ends.
 */
export function startAnalyzeBatch(
  paths: string[],
  onProgress: (done: number, total: number, result: AnalysisResult) => void,
): { done: Promise<AnalyzeBatchOutcome>; cancel: () => Promise<boolean> } {
  const batchId = newBatchId()
  const unsubscribe = EventsOn('analyze-progress', (e: ProgressEvent) => {
    if (e.batchId === batchId) onProgress(e.done, e.total, e.result)
  })
  const done = AnalyzeBatch(batchId, paths).finally(() => unsubscribe?.())
  return { done, cancel: () => CancelAnalyzeBatch(batchId) }
}
