import { describe, it, expect, afterEach, vi } from 'vitest'

vi.setConfig({ testTimeout: 30000 })

async function setup() {
  vi.resetModules()
  // jsdom has no Web Animations API; Svelte transitions call element.animate
  Element.prototype.animate = vi.fn(() => ({ cancel() {}, finish() {}, onfinish: null })) as unknown as Element['animate']
  const { mount, unmount, flushSync } = await import('svelte')
  const { toastStore } = await import('../stores/toast')
  const { default: Toast } = await import('./Toast.svelte')
  const target = document.createElement('div')
  document.body.appendChild(target)
  vi.useFakeTimers()
  const component = mount(Toast, { target })
  flushSync()
  return {
    toastStore,
    target,
    flush: flushSync,
    cleanup: () => {
      unmount(component)
      target.remove()
    },
  }
}

describe('Toast', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders both live regions before any toast', async () => {
    const { target, cleanup } = await setup()
    expect(target.querySelector('[role="status"]')).not.toBeNull()
    expect(target.querySelector('[role="alert"]')).not.toBeNull()
    cleanup()
  })

  it('announces errors as alert and success or info as status', async () => {
    const { toastStore, target, flush, cleanup } = await setup()
    toastStore.show('boom', 'error')
    toastStore.show('done', 'success')
    toastStore.show('fyi', 'info')
    flush()
    expect(target.querySelector('[role="alert"]')?.textContent).toContain('boom')
    const status = target.querySelector('[role="status"]')?.textContent ?? ''
    expect(status).toContain('done')
    expect(status).toContain('fyi')
    expect(status).not.toContain('boom')
    cleanup()
  })

  it('hides decorative icons from assistive technology', async () => {
    const { toastStore, target, flush, cleanup } = await setup()
    toastStore.show('boom', 'error')
    toastStore.show('done', 'success')
    flush()
    const icons = target.querySelectorAll('.toast svg')
    expect(icons.length).toBe(2)
    icons.forEach((svg) => expect(svg.getAttribute('aria-hidden')).toBe('true'))
    cleanup()
  })
})
