import { describe, it, expect } from 'vitest';
import { get } from 'svelte/store';
import { resolveLang } from './resolve';
import { translate, type Dicts } from './translate';
import { t, locale, languagePref, setLanguage, initLanguage } from './index';

const dicts: Dicts = {
  en: {
    'a.hello': 'Hello {name}',
    'a.only_en': 'Only English',
    'a.items_one': '{count} item',
    'a.items_other': '{count} items',
  },
  fr: {
    'a.hello': 'Bonjour {name}',
    'a.items_one': '{count} élément',
    'a.items_other': '{count} éléments',
  },
  de: {},
};

describe('resolveLang', () => {
  it.each([
    ['', 'fr-CA', 'fr'],
    ['', 'de-AT', 'de'],
    ['', 'es', 'en'],
    ['', undefined, 'en'],
    ['xx', 'fr-FR', 'fr'],
    ['de', 'fr-FR', 'de'],
  ])('pref %j + nav %j -> %s', (pref, nav, expected) => {
    expect(resolveLang(pref, nav)).toBe(expected);
  });
});

describe('translate', () => {
  it('interpolates variables', () => {
    expect(translate(dicts, 'fr', 'a.hello', { name: 'Zoé' })).toBe('Bonjour Zoé');
  });
  it('leaves unknown variables untouched', () => {
    expect(translate(dicts, 'en', 'a.hello', {})).toBe('Hello {name}');
  });
  it('falls back to English', () => {
    expect(translate(dicts, 'fr', 'a.only_en')).toBe('Only English');
  });
  it('falls back to a readable id, never the full key', () => {
    expect(translate(dicts, 'fr', 'zone.some_missing_key')).toBe('some missing key');
  });
  it('pluralizes in English', () => {
    expect(translate(dicts, 'en', 'a.items', { count: 1 })).toBe('1 item');
    expect(translate(dicts, 'en', 'a.items', { count: 0 })).toBe('0 items');
    expect(translate(dicts, 'en', 'a.items', { count: 2 })).toBe('2 items');
  });
  it('pluralizes in French (0 is singular)', () => {
    expect(translate(dicts, 'fr', 'a.items', { count: 0 })).toBe('0 élément');
    expect(translate(dicts, 'fr', 'a.items', { count: 2 })).toBe('2 éléments');
  });
  it('falls back to English plural forms when the language lacks them', () => {
    expect(translate(dicts, 'de', 'a.items', { count: 3 })).toBe('3 items');
  });
});

describe('language store', () => {
  it('switches at runtime and updates <html lang>', () => {
    setLanguage('en');
    const seen: string[] = [];
    const unsub = t.subscribe((tr) => seen.push(tr('nav.home')));
    setLanguage('fr');
    unsub();
    expect(seen).toEqual(['Home', 'Accueil']);
    expect(get(locale)).toBe('fr');
    expect(get(languagePref)).toBe('fr');
    expect(document.documentElement.lang).toBe('fr');
  });
  it('auto mode resolves from navigator and keeps pref empty', () => {
    initLanguage('');
    expect(get(languagePref)).toBe('');
    expect(get(locale)).toBe(resolveLang('', navigator.language));
  });
  it('treats an invalid pref as auto', () => {
    setLanguage('xx');
    expect(get(languagePref)).toBe('');
  });
});
