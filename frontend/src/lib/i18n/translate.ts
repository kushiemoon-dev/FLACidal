import type { Lang } from './resolve';

export type Vars = Record<string, string | number>;
export type Dicts = Record<Lang, Record<string, string>>;

/** Last key segment as readable text, never the raw full key. */
function readable(key: string): string {
  const last = key.split('.').pop() ?? key;
  return last.replace(/[_.]/g, ' ');
}

export function translate(dicts: Dicts, lang: Lang, key: string, vars?: Vars): string {
  let k = key;
  if (typeof vars?.count === 'number') {
    const form = new Intl.PluralRules(lang).select(vars.count) === 'one' ? 'one' : 'other';
    const has = (l: Lang, kk: string) => dicts[l]?.[kk] !== undefined;
    k = [`${key}_${form}`, `${key}_other`, key].find((c) => has(lang, c) || has('en', c)) ?? key;
  }
  const raw = dicts[lang]?.[k] ?? dicts.en?.[k] ?? readable(key);
  if (!vars) return raw;
  return raw.replace(/\{(\w+)\}/g, (m, name) => (name in vars ? String(vars[name]) : m));
}
