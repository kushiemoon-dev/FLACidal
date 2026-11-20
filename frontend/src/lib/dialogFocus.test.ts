import { describe, it, expect, afterEach } from 'vitest'
import { dialogFocus } from './dialogFocus'

function setup(inner: string) {
  const opener = document.createElement('button')
  document.body.appendChild(opener)
  opener.focus()
  const dialog = document.createElement('div')
  dialog.innerHTML = inner
  document.body.appendChild(dialog)
  return { opener, dialog }
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('dialogFocus', () => {
  it('focuses the first enabled focusable element and restores the opener', () => {
    const { opener, dialog } = setup('<button disabled>x</button><button id="ok">ok</button>')
    const action = dialogFocus(dialog)
    expect(document.activeElement?.id).toBe('ok')
    action.destroy()
    expect(document.activeElement).toBe(opener)
  })

  it('falls back to the dialog itself when nothing inside is focusable', () => {
    const { dialog } = setup('<p>text</p>')
    dialogFocus(dialog)
    expect(document.activeElement).toBe(dialog)
    expect(dialog.getAttribute('tabindex')).toBe('-1')
  })

  it('does nothing on close if the opener left the DOM', () => {
    const { opener, dialog } = setup('<button id="ok">ok</button>')
    const action = dialogFocus(dialog)
    opener.remove()
    expect(() => action.destroy()).not.toThrow()
    expect(document.activeElement?.id).toBe('ok')
  })

  describe('trap option', () => {
    const tab = (shiftKey = false) =>
      new KeyboardEvent('keydown', { key: 'Tab', shiftKey, bubbles: true, cancelable: true })

    it('wraps Tab from the last element to the first', () => {
      const { dialog } = setup('<button id="a">a</button><button id="b">b</button>')
      dialogFocus(dialog, { trap: true })
      const last = dialog.querySelector<HTMLElement>('#b')!
      last.focus()
      const ev = tab()
      dialog.dispatchEvent(ev)
      expect(ev.defaultPrevented).toBe(true)
      expect(document.activeElement?.id).toBe('a')
    })

    it('wraps Shift+Tab from the first element to the last', () => {
      const { dialog } = setup('<button id="a">a</button><button id="b">b</button>')
      dialogFocus(dialog, { trap: true })
      dialog.querySelector<HTMLElement>('#a')!.focus()
      dialog.dispatchEvent(tab(true))
      expect(document.activeElement?.id).toBe('b')
    })

    it('leaves Tab alone in the middle of the dialog', () => {
      const { dialog } = setup('<button id="a">a</button><button id="b">b</button><button id="c">c</button>')
      dialogFocus(dialog, { trap: true })
      dialog.querySelector<HTMLElement>('#b')!.focus()
      const ev = tab()
      dialog.dispatchEvent(ev)
      expect(ev.defaultPrevented).toBe(false)
    })

    it('does not trap without the option', () => {
      const { dialog } = setup('<button id="a">a</button><button id="b">b</button>')
      dialogFocus(dialog)
      dialog.querySelector<HTMLElement>('#b')!.focus()
      const ev = tab()
      dialog.dispatchEvent(ev)
      expect(ev.defaultPrevented).toBe(false)
    })

    it('removes the listener on destroy', () => {
      const { dialog } = setup('<button id="a">a</button><button id="b">b</button>')
      const action = dialogFocus(dialog, { trap: true })
      action.destroy()
      dialog.querySelector<HTMLElement>('#b')!.focus()
      const ev = tab()
      dialog.dispatchEvent(ev)
      expect(ev.defaultPrevented).toBe(false)
    })
  })
})
