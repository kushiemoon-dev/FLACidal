// Wrappers for the Wails runtime capabilities that don't fit neatly into
// either the REST-mappable App bindings (see api.ts) or the generic pub/sub
// events (see websocket.ts): opening external URLs and handling native
// OS-level file drag-and-drop.

import {
  BrowserOpenURL,
  OnFileDrop as WailsOnFileDrop,
  OnFileDropOff as WailsOnFileDropOff,
} from '../../wailsjs/runtime/runtime.js'
import { isWailsRuntime } from './api'

export function OpenExternalURL(url: string): void {
  if (isWailsRuntime()) {
    BrowserOpenURL(url)
    return
  }
  window.open(url, '_blank', 'noopener,noreferrer')
}

/**
 * Browser mode does nothing and returns a no-op cleanup. A browser's HTML5
 * drop event only ever exposes File objects (name plus content), never an
 * absolute filesystem path, and none of this app's REST endpoints accept
 * uploads for the batch operations that consume these paths (AnalyzeMultiple,
 * ConvertFiles, and FetchAndEmbedLyricsMultiple all take arrays of paths).
 * There's no honest way to support this in browser mode right now; callers
 * should rely on DropZone's own browser-mode messaging instead of pretending
 * drops work. See the migration report for the known-gap writeup.
 */
export function onNativeFileDrop(
  callback: (x: number, y: number, paths: string[]) => void,
  useDropTarget = true
): () => void {
  if (!isWailsRuntime()) {
    return () => {}
  }
  WailsOnFileDrop(callback, useDropTarget)
  return () => WailsOnFileDropOff()
}
