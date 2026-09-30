import { describe, it, expect, afterEach, vi } from 'vitest'
import { get } from 'svelte/store'
import { mount, unmount, flushSync } from 'svelte'
import { toastStore } from './toast'
import Toast from '../components/Toast.svelte'
import { setLanguage, tm } from '../lib/i18n'

// jsdom has no Web Animations API; Toast uses fly/fade transitions.
Element.prototype.animate = (() => ({ cancel() {}, finished: Promise.resolve(), onfinish: null })) as unknown as Element['animate']

describe('toastStore', () => {
  afterEach(() => {
    vi.useRealTimers()
    setLanguage('en')
  })

  it('keeps raw text as is', () => {
    toastStore.show('Core said no', 'error')
    const last = get(toastStore).at(-1)!
    expect(get(tm)(last.message)).toBe('Core said no')
  })

  it('showMsg ignores empty messages and shows the fallback key', () => {
    const before = get(toastStore).length
    toastStore.showMsg('', 'error')
    expect(get(toastStore).length).toBe(before)
    toastStore.showMsg({ key: 'settings.toast.saveError' }, 'error')
    expect(get(toastStore).length).toBe(before + 1)
  })

  it('renders a key toast in the language active at render time', () => {
    vi.useFakeTimers()
    setLanguage('en')
    const target = document.createElement('div')
    document.body.appendChild(target)
    const component = mount(Toast, { target })

    toastStore.showKey('nav.home', {}, 'info')
    flushSync()
    expect(target.textContent).toContain('Home')

    setLanguage('fr')
    flushSync()
    expect(target.textContent).toContain('Accueil')
    expect(target.textContent).not.toContain('Home')

    unmount(component)
    target.remove()
    vi.runAllTimers()
  })

  it('does not translate raw toast text on a language switch', () => {
    vi.useFakeTimers()
    setLanguage('en')
    const target = document.createElement('div')
    document.body.appendChild(target)
    const component = mount(Toast, { target })

    toastStore.show('Raw Core text', 'error')
    flushSync()
    setLanguage('de')
    flushSync()
    expect(target.textContent).toContain('Raw Core text')

    unmount(component)
    target.remove()
    vi.runAllTimers()
  })
})
