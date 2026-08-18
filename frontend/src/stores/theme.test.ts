import { describe, it, expect, beforeEach, vi } from 'vitest'
import { get } from 'svelte/store'
import { themeStore, accentColor, accentPresets, applyAccentColor, initializeAccentColor, type ThemeMode } from '../stores/theme'

describe('themeStore', () => {
  beforeEach(() => {
    themeStore.initialize('system')
    accentColor.set('#f472b6')
  })

  it('defaults to the system theme', () => {
    const currentTheme = get(themeStore)
    expect(currentTheme).toBe('system')
  })

  it('defaults to the standard accent color', () => {
    const color = get(accentColor)
    expect(color).toBe('#f472b6')
  })

  it('updates the theme', () => {
    themeStore.setTheme('dark')
    expect(get(themeStore)).toBe('dark')
  })

  it('accepts every valid theme mode', () => {
    const validThemes: ThemeMode[] = ['dark', 'light', 'system']

    validThemes.forEach((t) => {
      themeStore.setTheme(t)
      expect(get(themeStore)).toBe(t)
    })
  })
})

describe('accentPresets', () => {
  it('provides 7 preset colors', () => {
    expect(accentPresets).toHaveLength(7)
  })

  it('exposes the expected properties', () => {
    accentPresets.forEach((preset) => {
      expect(preset).toHaveProperty('name')
      expect(preset).toHaveProperty('color')
      expect(preset.color).toMatch(/^#[0-9a-f]{6}$/i)
    })
  })
})

describe('applyAccentColor', () => {
  beforeEach(() => {
    document.documentElement.style.removeProperty('--color-accent')
    document.documentElement.style.removeProperty('--color-accent-hover')
    document.documentElement.style.removeProperty('--color-accent-subtle')
  })

  it('sets the CSS variables', () => {
    applyAccentColor('#ff0000')

    const accent = document.documentElement.style.getPropertyValue('--color-accent')
    expect(accent).toBe('#ff0000')
  })

  it('handles valid hex colors correctly', () => {
    applyAccentColor('#3b82f6')

    const accent = document.documentElement.style.getPropertyValue('--color-accent')
    expect(accent).toBe('#3b82f6')
  })
})

describe('initializeAccentColor', () => {
  beforeEach(() => {
    accentColor.set('#f472b6')
  })

  it('uses the given color', () => {
    initializeAccentColor('#00ff00')
    expect(get(accentColor)).toBe('#00ff00')
  })

  it('falls back to the default when the color is empty', () => {
    initializeAccentColor('')
    expect(get(accentColor)).toBe('#f472b6')
  })
})

describe('accentColor store', () => {
  it('can be updated directly', () => {
    accentColor.set('#a855f7')
    expect(get(accentColor)).toBe('#a855f7')
  })
})
