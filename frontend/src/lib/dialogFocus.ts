const FOCUSABLE =
  'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

/** Moves focus into a dialog on open and back to the opener on close. No focus trap. */
export function dialogFocus(node: HTMLElement) {
  const opener = document.activeElement as HTMLElement | null
  const target = node.querySelector<HTMLElement>(FOCUSABLE)
  if (target) {
    target.focus()
  } else {
    if (!node.hasAttribute('tabindex')) node.setAttribute('tabindex', '-1')
    node.focus()
  }
  return {
    destroy() {
      if (opener?.isConnected) opener.focus()
    },
  }
}
