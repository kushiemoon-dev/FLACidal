import type { MessageKey, UiMsg } from './i18n'

const CODE_KEYS: Record<string, MessageKey> = {
  'folder.relative': 'errors.folder.relative',
  'folder.root': 'errors.folder.root',
  'folder.missing': 'errors.folder.missing',
  'folder.not_dir': 'errors.folder.notDir',
  'folder.not_writable': 'errors.folder.notWritable',
  'queue.no_output_dir': 'errors.queue.noOutputDir',
  'queue.not_ready': 'errors.queue.notReady',
}

const CODED = /^\[([a-z_.]+)\]\s*(.*)$/s
const NETWORK = /failed to fetch|networkerror|load failed/i

/**
 * Turns an error thrown by Wails or `apiFetch` into a translated message.
 * Core errors read "[code] detail"; the raw detail is only ever a variable of
 * a translated text, never shown alone.
 */
export function toUiError(err: unknown): NonNullable<UiMsg> {
  const message = err == null ? '' : err instanceof Error ? err.message : String(err)

  const coded = CODED.exec(message)
  if (coded && CODE_KEYS[coded[1]]) {
    return { key: CODE_KEYS[coded[1]], vars: { detail: coded[2] } }
  }
  if (err instanceof TypeError || NETWORK.test(message)) return { key: 'errors.unreachable' }
  if (/^401\b/.test(message)) return { key: 'errors.unauthorized' }
  return { key: 'errors.generic', vars: { detail: message } }
}
