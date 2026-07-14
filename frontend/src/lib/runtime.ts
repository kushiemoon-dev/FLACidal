// Dual-mode wrappers for Wails runtime capabilities that are neither
// REST-mappable App bindings (see api.ts) nor generic pub/sub events
// (see websocket.ts): opening external URLs, and native OS-level file
// drag-and-drop.

import {
  BrowserOpenURL,
  OnFileDrop as WailsOnFileDrop,
  OnFileDropOff as WailsOnFileDropOff,
} from '../../wailsjs/runtime/runtime.js'
import { isWailsRuntime } from './api'

/**
 * Opens a URL in the user's default browser (Wails) or a new tab (plain browser).
 */
export function OpenExternalURL(url: string): void {
  if (isWailsRuntime()) {
    BrowserOpenURL(url)
    return
  }
  window.open(url, '_blank', 'noopener,noreferrer')
}

/**
 * Subscribes to native OS-level file drag-and-drop, which delivers absolute
 * filesystem paths (Wails' window-level EnableFileDrop capability).
 *
 * Wails mode: wraps OnFileDrop/OnFileDropOff 1:1 and returns a cleanup
 * function that calls OnFileDropOff.
 *
 * Browser mode: a no-op that returns a no-op cleanup. A browser's HTML5 drop
 * event only ever exposes File objects (name + content), never an absolute
 * filesystem path — and none of this app's REST endpoints currently accept
 * uploads for the batch operations that consume these paths (AnalyzeMultiple,
 * ConvertFiles, FetchAndEmbedLyricsMultiple all take path arrays). So there is
 * no way to honestly wire this up in browser mode today; callers should pair
 * this with DropZone's own browser-mode messaging rather than pretend drops
 * are handled. See migration report for the known-gap writeup.
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
