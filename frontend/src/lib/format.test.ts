import { describe, it, expect } from 'vitest';
import { formatNumber, formatDate } from './format';

describe('formatNumber', () => {
  it('follows the given locale', () => {
    expect(formatNumber(1234567, 'en')).toBe('1,234,567');
    expect(formatNumber(1234567, 'fr')).not.toBe('1,234,567');
    expect(formatNumber(1234567, 'de')).toBe('1.234.567');
  });
});

describe('formatDate', () => {
  it('produces distinct formats per locale', () => {
    const d = new Date(2026, 8, 30);
    const o: Intl.DateTimeFormatOptions = { year: 'numeric', month: 'long', day: 'numeric' };
    const [en, fr, de] = ['en', 'fr', 'de'].map((l) => formatDate(d, o, l));
    expect(new Set([en, fr, de]).size).toBe(3);
  });
});
