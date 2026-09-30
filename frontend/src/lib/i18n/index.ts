import { derived, writable } from 'svelte/store';
import { en } from './messages/en';
import { fr } from './messages/fr';
import { de } from './messages/de';
import { resolveLang, type Lang, type LanguagePref } from './resolve';
import { translate, type Vars } from './translate';

export type { Lang, LanguagePref };
export type MessageKey = keyof typeof en;
/** Base of a `_one`/`_other` pair, callable with `t(base, { count })`. */
export type PluralKey = MessageKey extends infer K ? (K extends `${infer B}_one` ? B : never) : never;

const dicts = { en, fr, de } as Record<Lang, Record<string, string>>;

export const languagePref = writable<LanguagePref>('');
export const locale = writable<Lang>('en');

export const t = derived(
  locale,
  (l) => (key: MessageKey | PluralKey, vars?: Vars) => translate(dicts, l, key, vars)
);

/** Applies a preference: stores it, resolves the active language, updates <html lang>. */
export function setLanguage(pref: string): void {
  const resolved = resolveLang(pref, typeof navigator !== 'undefined' ? navigator.language : undefined);
  languagePref.set(pref === 'en' || pref === 'fr' || pref === 'de' ? pref : '');
  locale.set(resolved);
  if (typeof document !== 'undefined') document.documentElement.lang = resolved;
}

export const initLanguage = setLanguage;
