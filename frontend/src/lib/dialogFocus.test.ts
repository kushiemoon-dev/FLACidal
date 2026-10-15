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
})
