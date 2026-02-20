import { describe, it, expect, beforeEach, vi } from 'vitest'
import { get } from 'svelte/store'
import { themeStore, accentColor, accentPresets, applyAccentColor, initializeAccentColor, type ThemeMode } from '../stores/theme'

describe('themeStore', () => {
  beforeEach(() => {
    themeStore.initialize('system')
    accentColor.set('#f472b6')
  })

  it('should have default system theme', () => {
    const currentTheme = get(themeStore)
    expect(currentTheme).toBe('system')
  })

  it('should have default accent color', () => {
    const color = get(accentColor)
    expect(color).toBe('#f472b6')
  })

  it('should update theme', () => {
    themeStore.setTheme('dark')
    expect(get(themeStore)).toBe('dark')
  })

  it('should accept valid themes', () => {
    const validThemes: ThemeMode[] = ['dark', 'light', 'system']

    validThemes.forEach((t) => {
      themeStore.setTheme(t)
      expect(get(themeStore)).toBe(t)
    })
  })
})

describe('accentPresets', () => {
  it('should have 7 preset colors', () => {
    expect(accentPresets).toHaveLength(7)
  })

  it('should have required properties', () => {
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

  it('should set CSS variables', () => {
    applyAccentColor('#ff0000')

    const accent = document.documentElement.style.getPropertyValue('--color-accent')
    expect(accent).toBe('#ff0000')
  })

  it('should handle valid hex colors', () => {
    applyAccentColor('#3b82f6')

    const accent = document.documentElement.style.getPropertyValue('--color-accent')
    expect(accent).toBe('#3b82f6')
  })
})

describe('initializeAccentColor', () => {
  beforeEach(() => {
    accentColor.set('#f472b6')
  })

  it('should use provided color', () => {
    initializeAccentColor('#00ff00')
    expect(get(accentColor)).toBe('#00ff00')
  })

  it('should use default for empty color', () => {
    initializeAccentColor('')
    expect(get(accentColor)).toBe('#f472b6')
  })
})

describe('accentColor store', () => {
  it('should be updatable', () => {
    accentColor.set('#a855f7')
    expect(get(accentColor)).toBe('#a855f7')
  })
})
