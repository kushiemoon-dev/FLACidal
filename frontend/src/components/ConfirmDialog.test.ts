import { describe, it, expect, vi } from 'vitest'

vi.setConfig({ testTimeout: 30000 })

async function setup() {
  vi.resetModules()
  const { mount, unmount, flushSync } = await import('svelte')
  const { default: ConfirmDialog } = await import('./ConfirmDialog.svelte')
  const opener = document.createElement('button')
  document.body.appendChild(opener)
  opener.focus()
  const target = document.createElement('div')
  document.body.appendChild(target)
  const onCancel = vi.fn()
  const component = mount(ConfirmDialog, {
    target,
    props: { title: 'Delete it?', message: 'Gone for good', onConfirm: vi.fn(), onCancel },
  })
  flushSync()
  return {
    opener,
    target,
    onCancel,
    destroy: () => {
      unmount(component)
      flushSync()
    },
    cleanup: () => {
      target.remove()
      opener.remove()
    },
  }
}

describe('ConfirmDialog', () => {
  it('is a named modal dialog and takes focus', async () => {
    const { target, cleanup, destroy } = await setup()
    const dialog = target.querySelector('[role="dialog"]') as HTMLElement
    expect(dialog.getAttribute('aria-modal')).toBe('true')
    const title = document.getElementById(dialog.getAttribute('aria-labelledby') ?? '')
    expect(title?.textContent).toBe('Delete it?')
    expect(dialog.contains(document.activeElement)).toBe(true)
    destroy()
    cleanup()
  })

  it('calls onCancel on Escape', async () => {
    const { onCancel, cleanup, destroy } = await setup()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    expect(onCancel).toHaveBeenCalledTimes(1)
    destroy()
    cleanup()
  })

  it('returns focus to the opener when destroyed', async () => {
    const { opener, cleanup, destroy } = await setup()
    destroy()
    expect(document.activeElement).toBe(opener)
    cleanup()
  })
})
