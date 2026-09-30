export type NavHistory = { readonly stack: readonly string[]; readonly index: number }

export function createNavHistory(initial: string): NavHistory {
  return { stack: [initial], index: 0 }
}

export function currentPage(h: NavHistory): string {
  return h.stack[h.index]
}

export function canGoBack(h: NavHistory): boolean {
  return h.index > 0
}

export function canGoForward(h: NavHistory): boolean {
  return h.index < h.stack.length - 1
}

export function navigateTo(h: NavHistory, page: string): NavHistory {
  if (page === currentPage(h)) return h
  return { stack: [...h.stack.slice(0, h.index + 1), page], index: h.index + 1 }
}

export function goBack(h: NavHistory): NavHistory {
  return canGoBack(h) ? { ...h, index: h.index - 1 } : h
}

export function goForward(h: NavHistory): NavHistory {
  return canGoForward(h) ? { ...h, index: h.index + 1 } : h
}
