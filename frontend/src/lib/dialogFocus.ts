const FOCUSABLE =
  'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

/** Moves focus into a dialog on open and back to the opener on close. Tab stays inside only with `trap`. */
export function dialogFocus(node: HTMLElement, options: { trap?: boolean } = {}) {
  const opener = document.activeElement as HTMLElement | null
  const target = node.querySelector<HTMLElement>(FOCUSABLE)
  if (target) {
    target.focus()
  } else {
    if (!node.hasAttribute('tabindex')) node.setAttribute('tabindex', '-1')
    node.focus()
  }
  const onKeydown = (e: KeyboardEvent) => {
    if (e.key !== 'Tab') return
    const items = node.querySelectorAll<HTMLElement>(FOCUSABLE)
    if (items.length === 0) return
    const first = items[0]
    const last = items[items.length - 1]
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault()
      last.focus()
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault()
      first.focus()
    }
  }
  if (options.trap) node.addEventListener('keydown', onKeydown)
  return {
    destroy() {
      node.removeEventListener('keydown', onKeydown)
      if (opener?.isConnected) opener.focus()
    },
  }
}
