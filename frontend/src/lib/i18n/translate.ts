import type { Lang } from './resolve';

export type Vars = Record<string, string | number>;
export type Dicts = Record<Lang, Record<string, string>>;

/** Last key segment as readable text, never the raw full key. */
function readable(key: string): string {
  const last = key.split('.').pop() ?? key;
  return last.replace(/[_.]/g, ' ');
}

const pluralRules = new Map<string, Intl.PluralRules>();

function pluralForm(lang: string, count: number): 'one' | 'other' {
  let rules = pluralRules.get(lang);
  if (!rules) {
    rules = new Intl.PluralRules(lang);
    pluralRules.set(lang, rules);
  }
  return rules.select(count) === 'one' ? 'one' : 'other';
}

/** Picks the plural variant of `key` for one language, or undefined when it has none. */
function pluralKey(dicts: Dicts, lang: Lang, key: string, count: number): { lang: Lang; key: string } | undefined {
  const found = [`${key}_${pluralForm(lang, count)}`, `${key}_other`, key].find((c) => dicts[lang]?.[c] !== undefined);
  return found ? { lang, key: found } : undefined;
}

export function translate(dicts: Dicts, lang: Lang, key: string, vars?: Vars): string {
  let source: Lang = lang;
  let k = key;
  if (typeof vars?.count === 'number') {
    // Choose per language first, and only fall back to English as a whole.
    const hit = pluralKey(dicts, lang, key, vars.count) ?? pluralKey(dicts, 'en', key, vars.count);
    if (hit) ({ lang: source, key: k } = hit);
  }
  const raw = dicts[source]?.[k] ?? dicts[lang]?.[k] ?? dicts.en?.[k] ?? readable(key);
  if (!vars) return raw;
  return raw.replace(/\{(\w+)\}/g, (m, name) => (name in vars ? String(vars[name]) : m));
}
