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
 * Native file drop is a Wails feature: a browser's HTML5 drop event only
 * exposes File objects (name plus content), never an absolute filesystem
 * path, and the batch operations take server paths. Browser mode returns a
 * no-op cleanup; DropZone says so and its buttons open the server file
 * browser (components/FileBrowserDialog.svelte) instead.
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
