export type Lang = 'en' | 'fr' | 'de';
export type LanguagePref = '' | Lang;

export const LANGS: readonly Lang[] = ['en', 'fr', 'de'];

const isLang = (v: string): v is Lang => (LANGS as readonly string[]).includes(v);

/** An explicit valid preference wins; anything else means auto (browser language, else en). */
export function resolveLang(pref: string | undefined, navLang: string | undefined): Lang {
  if (pref && isLang(pref)) return pref;
  const base = (navLang ?? '').toLowerCase().split('-')[0];
  return isLang(base) ? base : 'en';
}
