import { describe, it, expect } from 'vitest'
import { get } from 'svelte/store'
import { requestPick, closePick, pickRequest } from './filePicker'

describe('filePicker store', () => {
  it('resolves with the selection and clears the request', async () => {
    const p = requestPick({ kind: 'files', scope: 'tools' })
    expect(get(pickRequest)?.kind).toBe('files')
    closePick(['/a.flac'])
    expect(await p).toEqual(['/a.flac'])
    expect(get(pickRequest)).toBeNull()
  })

  it('resolves null when cancelled', async () => {
    const p = requestPick({ kind: 'folder', scope: 'anywhere' })
    closePick(null)
    expect(await p).toBeNull()
  })

  it('cancels the previous request when a new one opens', async () => {
    const first = requestPick({ kind: 'files', scope: 'tools' })
    const second = requestPick({ kind: 'folder', scope: 'tools' })
    expect(await first).toBeNull()
    expect(get(pickRequest)?.kind).toBe('folder')
    closePick(['/x'])
    expect(await second).toEqual(['/x'])
  })

  it('ignores closePick when nothing is open', () => {
    expect(() => closePick(null)).not.toThrow()
  })
})
