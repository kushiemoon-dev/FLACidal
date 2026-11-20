import { describe, it, expect, vi } from 'vitest'

vi.setConfig({ testTimeout: 30000 })

const root = {
  path: '/music',
  parent: '',
  truncated: false,
  entries: [
    { name: 'Albums', path: '/music/Albums', isDir: true },
    { name: 'a.flac', path: '/music/a.flac', isDir: false },
    { name: 'b.flac', path: '/music/b.flac', isDir: false },
  ],
}
const albums = { path: '/music/Albums', parent: '/music', truncated: false, entries: [] }

async function setup(kind: 'files' | 'folder' = 'files', scope: 'tools' | 'anywhere' = 'tools') {
  vi.resetModules()
  const BrowseFolder = vi.fn(async (path: string) => (path === '/music/Albums' ? albums : root))
  vi.doMock('../lib/api', () => ({ BrowseFolder }))
  const { mount, unmount, flushSync } = await import('svelte')
  const { default: Dialog } = await import('./FileBrowserDialog.svelte')
  const opener = document.createElement('button')
  document.body.appendChild(opener)
  opener.focus()
  const target = document.createElement('div')
  document.body.appendChild(target)
  const onDone = vi.fn()
  const component = mount(Dialog, { target, props: { kind, scope, onDone } })
  flushSync()
  await vi.waitFor(() => expect(target.querySelector('ul li')).not.toBeNull())
  return {
    target,
    onDone,
    BrowseFolder,
    flush: async () => { flushSync(); await Promise.resolve(); flushSync() },
    cleanup: () => {
      unmount(component)
      flushSync()
      target.remove()
      opener.remove()
    },
  }
}

const button = (root: HTMLElement, text: string) =>
  [...root.querySelectorAll('button')].find((b) => b.textContent?.includes(text)) as HTMLButtonElement

describe('FileBrowserDialog', () => {
  it('is a named modal dialog listing folders as buttons and files as checkboxes', async () => {
    const { target, BrowseFolder, cleanup } = await setup('files')
    const dialog = target.querySelector('[role="dialog"]') as HTMLElement
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    expect(document.getElementById(dialog.getAttribute('aria-labelledby') ?? '')).not.toBeNull()
    expect(BrowseFolder).toHaveBeenCalledWith('', 'tools', true)
    expect(button(target, 'Albums')).toBeTruthy()
    expect(target.querySelectorAll('input[type="checkbox"]').length).toBe(2)
    cleanup()
  })

  it('asks for folders only and the whole server in folder/anywhere mode', async () => {
    const { BrowseFolder, cleanup } = await setup('folder', 'anywhere')
    expect(BrowseFolder).toHaveBeenCalledWith('', 'folders', false)
    cleanup()
  })

  it('cancels on Escape', async () => {
    const { onDone, cleanup } = await setup()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(onDone).toHaveBeenCalledWith(null)
    cleanup()
  })

  it('keeps Select disabled until a file is ticked, then returns the ticked paths', async () => {
    const { target, onDone, flush, cleanup } = await setup('files')
    const select = button(target, 'Select')
    expect(select.disabled).toBe(true)
    const boxes = target.querySelectorAll<HTMLInputElement>('input[type="checkbox"]')
    boxes[1].click()
    await flush()
    expect(select.disabled).toBe(false)
    select.click()
    expect(onDone).toHaveBeenCalledWith(['/music/b.flac'])
    cleanup()
  })

  it('opens a folder on click and returns the current folder in folder mode', async () => {
    const { target, onDone, BrowseFolder, flush, cleanup } = await setup('folder', 'anywhere')
    button(target, 'Albums').click()
    await vi.waitFor(() => expect(BrowseFolder).toHaveBeenCalledWith('/music/Albums', 'folders', false))
    await flush()
    button(target, 'Choose this folder').click()
    expect(onDone).toHaveBeenCalledWith(['/music/Albums'])
    cleanup()
  })

  it('goes to the parent folder, disabled at the top', async () => {
    const { target, BrowseFolder, flush, cleanup } = await setup('folder', 'tools')
    const up = target.querySelector<HTMLButtonElement>('[data-testid="up"]')!
    expect(up.disabled).toBe(true)
    button(target, 'Albums').click()
    await vi.waitFor(() => expect(BrowseFolder).toHaveBeenCalledTimes(2))
    await flush()
    expect(up.disabled).toBe(false)
    up.click()
    await vi.waitFor(() => expect(BrowseFolder).toHaveBeenLastCalledWith('/music', 'tools', false))
    cleanup()
  })

  it('navigates to a typed path on Enter', async () => {
    const { target, BrowseFolder, flush, cleanup } = await setup('folder', 'tools')
    const input = target.querySelector<HTMLInputElement>('input[type="text"]')!
    input.value = '/music/Albums'
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await flush()
    input.closest('form')!.dispatchEvent(new Event('submit', { bubbles: true, cancelable: true }))
    await vi.waitFor(() => expect(BrowseFolder).toHaveBeenLastCalledWith('/music/Albums', 'tools', false))
    cleanup()
  })

  it('traps Tab inside the dialog', async () => {
    const { target, cleanup } = await setup('files')
    const dialog = target.querySelector('[role="dialog"]') as HTMLElement
    const focusables = dialog.querySelectorAll<HTMLElement>('button:not([disabled]), input:not([disabled])')
    focusables[focusables.length - 1].focus()
    const ev = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
    dialog.dispatchEvent(ev)
    expect(ev.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(focusables[0])
    cleanup()
  })

  it('shows a translated notice when the list is truncated', async () => {
    vi.resetModules()
    vi.doMock('../lib/api', () => ({ BrowseFolder: vi.fn(async () => ({ ...root, truncated: true })) }))
    const { mount, unmount, flushSync } = await import('svelte')
    const { default: Dialog } = await import('./FileBrowserDialog.svelte')
    const target = document.createElement('div')
    document.body.appendChild(target)
    const c = mount(Dialog, { target, props: { kind: 'folder', scope: 'tools', onDone: vi.fn() } })
    await vi.waitFor(() => expect(target.querySelector('[aria-live]')?.textContent).toContain('2000'))
    unmount(c)
    flushSync()
    target.remove()
  })
})
