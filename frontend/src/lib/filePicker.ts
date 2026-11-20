import { get, writable } from 'svelte/store'

/** 'anywhere' browses the whole server filesystem (folders only); 'tools' stays inside the allowed folders. */
export type PickScope = 'tools' | 'anywhere'

export interface PickRequest {
  kind: 'files' | 'folder'
  scope: PickScope
  title?: string
  resolve: (paths: string[] | null) => void
}

export const pickRequest = writable<PickRequest | null>(null)

/** Opens the server file browser. Resolves with the chosen paths, or null when cancelled. */
export function requestPick(opts: Omit<PickRequest, 'resolve'>): Promise<string[] | null> {
  return new Promise((resolve) => {
    get(pickRequest)?.resolve(null)
    pickRequest.set({ ...opts, resolve })
  })
}

/** Ends the open request with the user's choice (null = cancelled). */
export function closePick(paths: string[] | null): void {
  const current = get(pickRequest)
  if (!current) return
  pickRequest.set(null)
  current.resolve(paths)
}
