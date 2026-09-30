import { describe, it, expect } from 'vitest';
import { en } from './messages/en';
import { fr } from './messages/fr';
import { de } from './messages/de';

const vars = (s: string) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort().join(',');

describe.each([
  ['fr', fr],
  ['de', de],
] as const)('%s dictionary', (_name, dict) => {
  const enKeys = Object.keys(en);
  const keys = Object.keys(dict);

  it('has exactly the same keys as en', () => {
    const missing = enKeys.filter((k) => !keys.includes(k));
    const extra = keys.filter((k) => !enKeys.includes(k));
    expect({ missing, extra }).toEqual({ missing: [], extra: [] });
  });
  it('uses the same {variables} as en', () => {
    const bad = enKeys.filter((k) => vars((dict as Record<string, string>)[k] ?? '') !== vars((en as Record<string, string>)[k]));
    expect(bad).toEqual([]);
  });
  it('has no empty values', () => {
    expect(Object.entries(dict).filter(([, v]) => !v.trim()).map(([k]) => k)).toEqual([]);
  });
});
