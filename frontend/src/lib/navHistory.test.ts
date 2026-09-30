import { describe, it, expect } from 'vitest'
import {
  createNavHistory,
  navigateTo,
  goBack,
  goForward,
  canGoBack,
  canGoForward,
  currentPage,
} from './navHistory'

describe('navHistory', () => {
  it('starts with Back and Forward disabled', () => {
    const h = createNavHistory('home')
    expect(currentPage(h)).toBe('home')
    expect(canGoBack(h)).toBe(false)
    expect(canGoForward(h)).toBe(false)
  })

  it('walks back and forward through A, B, C', () => {
    let h = createNavHistory('A')
    h = navigateTo(h, 'B')
    h = navigateTo(h, 'C')
    h = goBack(goBack(h))
    expect(currentPage(h)).toBe('A')
    expect(canGoBack(h)).toBe(false)
    h = goForward(h)
    expect(currentPage(h)).toBe('B')
    expect(canGoForward(h)).toBe(true)
  })

  it('ignores navigation to the current page', () => {
    const h = createNavHistory('A')
    expect(navigateTo(h, 'A')).toBe(h)
  })

  it('clears Forward when navigating after Back', () => {
    let h = navigateTo(navigateTo(createNavHistory('A'), 'B'), 'C')
    h = goBack(h)
    h = navigateTo(h, 'D')
    expect(currentPage(h)).toBe('D')
    expect(canGoForward(h)).toBe(false)
    expect(h.stack).toEqual(['A', 'B', 'D'])
  })

  it('returns the same state at the extremes', () => {
    const h = navigateTo(createNavHistory('A'), 'B')
    expect(goForward(h)).toBe(h)
    const start = createNavHistory('A')
    expect(goBack(start)).toBe(start)
  })

  it('does not mutate the previous state', () => {
    const h = navigateTo(createNavHistory('A'), 'B')
    const before = [...h.stack]
    navigateTo(goBack(h), 'C')
    expect(h.stack).toEqual(before)
    expect(h.index).toBe(1)
  })
})
