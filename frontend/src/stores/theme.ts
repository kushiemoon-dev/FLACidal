import { writable } from 'svelte/store';

export type ThemeMode = 'dark' | 'light' | 'system';

// Helper to convert hex to RGB
function hexToRgb(hex: string): { r: number; g: number; b: number } | null {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  return result
    ? {
        r: parseInt(result[1], 16),
        g: parseInt(result[2], 16),
        b: parseInt(result[3], 16)
      }
    : null;
}

// Helper to lighten a color
function lightenColor(hex: string, percent: number): string {
  const rgb = hexToRgb(hex);
  if (!rgb) return hex;

  const r = Math.min(255, Math.floor(rgb.r + (255 - rgb.r) * percent));
  const g = Math.min(255, Math.floor(rgb.g + (255 - rgb.g) * percent));
  const b = Math.min(255, Math.floor(rgb.b + (255 - rgb.b) * percent));

  return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`;
}

function createThemeStore() {
  const { subscribe, set } = writable<ThemeMode>('system');

  function applyTheme(theme: ThemeMode) {
    const root = document.documentElement;

    if (theme === 'system') {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      root.setAttribute('data-theme', prefersDark ? 'dark' : 'light');
    } else {
      root.setAttribute('data-theme', theme);
    }
  }

  // Listen for system theme changes
  if (typeof window !== 'undefined') {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
      // Only react if current theme is 'system'
      const currentTheme = document.documentElement.getAttribute('data-theme-mode');
      if (currentTheme === 'system') {
        document.documentElement.setAttribute('data-theme', e.matches ? 'dark' : 'light');
      }
    });
  }

  return {
    subscribe,

    setTheme: (theme: ThemeMode) => {
      set(theme);
      applyTheme(theme);
      // Store the mode for system theme listener
      document.documentElement.setAttribute('data-theme-mode', theme);
    },

    initialize: (savedTheme: ThemeMode) => {
      const theme = savedTheme || 'system';
      set(theme);
      applyTheme(theme);
      document.documentElement.setAttribute('data-theme-mode', theme);
    }
  };
}

export const themeStore = createThemeStore();

// Accent color store
export const accentColor = writable<string>('#f472b6');

// Preset accent colors
export const accentPresets = [
  { name: 'Pink', color: '#f472b6' },
  { name: 'Purple', color: '#a855f7' },
  { name: 'Blue', color: '#3b82f6' },
  { name: 'Cyan', color: '#06b6d4' },
  { name: 'Green', color: '#10b981' },
  { name: 'Orange', color: '#f59e0b' },
  { name: 'Red', color: '#ef4444' }
];

// Apply accent color to CSS variables
export function applyAccentColor(color: string) {
  const root = document.documentElement;
  const rgb = hexToRgb(color);

  if (rgb) {
    root.style.setProperty('--color-accent', color);
    root.style.setProperty('--color-accent-hover', lightenColor(color, 0.2));
    root.style.setProperty('--color-accent-subtle', `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, 0.15)`);
  }
}

// Initialize accent color from config
export function initializeAccentColor(color: string) {
  const validColor = color || '#f472b6';
  accentColor.set(validColor);
  applyAccentColor(validColor);
}
