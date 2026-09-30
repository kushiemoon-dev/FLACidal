import { parse } from 'svelte/compiler';

/*
 * Static scanner behind hardcoded.test.ts. It parses a component (.svelte) or module (.ts) and reports every
 * string literal that would reach the UI without going through $t / t().
 *
 * Modes, decided from the position of each string literal:
 *   skip    comparison operands, translation keys, imports, types, selectors, event names, object keys
 *   loose   plain script code: only sentence-like strings (capitalised word, several words) count
 *   display the value flows into something the template prints ({x}, {x.y}, {fn()}): any word counts
 *   strict  shown as-is (template text, visible attribute, toast argument, state named error/label...): any word counts
 */

// Proper nouns and product names stay untranslated. Matched as whole tokens only, never inside a longer word.
export const PROPER_NOUNS = [
  'FLACidal Mobile', 'Amazon Music', 'Plus Jakarta Sans', 'Bricolage Grotesque', 'OpenDrop', 'FLACidal', 'YouFLAC', 'Tidal', 'Qobuz',
  'Soulseek', 'Jellyfin', 'Bandcamp', 'English', 'Français', 'Deutsch', 'Amazon', 'HiFi', 'ReplayGain', 'FFmpeg',
  'Nicotine+', 'Ko-fi', 'BPM', 'Outfit', 'VJ',
];
// Untranslated technical tokens (units, codecs, log tags), each matched as an exact whole token.
export const TECH_TERMS = [
  'kHz', 'Hz', 'kbps', 'ISRC', 'ms', 'sldl', 'ch', 'VBR', 'feat.', 'sans-serif',
  'B', 'KB', 'MB', 'GB',
  'MP3', 'AAC', 'OGG', 'Opus', 'Vorbis', 'ALAC', 'WAV', 'AIFF', 'FLAC',
];

const TOKEN_BEFORE = String.raw`(?<![\p{L}\p{N}_])`;
const TOKEN_AFTER = String.raw`(?![\p{L}\p{N}_])`;
const escapeRe = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
const tokenRegex = (words: string[]) =>
  new RegExp(`${TOKEN_BEFORE}(?:${[...words].sort((a, b) => b.length - a.length).map(escapeRe).join('|')})${TOKEN_AFTER}`, 'gu');

const NOUN_RE = tokenRegex(PROPER_NOUNS);
const TECH_RE = tokenRegex(TECH_TERMS);
// Bit depth ("24-bit", "16 bit") and literal shell / log-tag strings.
const TECH_PHRASES = new RegExp(
  [String.raw`(?:^[-\s]*|(?<=\p{N}[-\s]?))bit${TOKEN_AFTER}`, escapeRe('sudo pacman -S ffmpeg'), String.raw`\[(?:ERROR|WARN|OK|INFO|LOG)\]`].join('|'),
  'gu'
);
// URLs, lowercase `{placeholder}` variables, absolute file paths and whole-string CSS media queries carry no prose.
const NON_PROSE = /https?:\/\/\S*|\{[a-z]\w*\}|(?:^|\s)\/[\w.-]+(?:\/[\w.-]*)+|^\((?:min|max)-[\w-]+\s*:[^)]*\)$|^\(prefers-[\w-]+\s*:[^)]*\)$/g;
// Strings that are identifiers, not shown text. Exact match after trim. Keep this list short and reasoned.
export const ALLOWED_STRINGS: Record<string, string[]> = {
  '*': ['v'], // version prefix in `v${version}`
  // Preset ids; the visible label comes from labelKey via $t.
  // Source ids; printed only as the fallback of sourceLabels[id] and equal to the brand names.
  'pages/Settings.svelte': ['tidal', 'qobuz', 'amazon', 'bandcamp', 'soulseek'],
  'stores/theme.ts': ['Pink', 'Purple', 'Blue', 'Cyan', 'Green', 'Orange', 'Red'],
};
// CSS units and keywords that legitimately sit next to an interpolation in a template literal.
const CSS_WORDS = new Set([
  'px', 'em', 'rem', 'vh', 'vw', 'vmin', 'vmax', 'deg', 'rad', 'turn', 'fr', 'ms', 'auto', 'none', 'solid', 'dashed', 'inherit',
  'calc', 'var', 'rgb', 'rgba', 'hsl', 'hsla', 'url',
]);

// Attributes whose value is shown to the user or read by assistive tech.
export const VISIBLE_ATTRS = [
  'title', 'placeholder', 'alt', 'label', 'summary',
  'aria-label', 'aria-description', 'aria-valuetext', 'aria-placeholder', 'aria-roledescription',
  'aria-braillelabel', 'aria-brailleroledescription',
  'data-tooltip', 'data-title', 'data-label', 'data-hint', 'data-tip', 'data-text', 'data-placeholder', 'data-caption',
];
// Names of variables/properties that hold user-facing text, so even a lowercase word assigned to them is flagged.
export const TEXT_NAME = /^(?!cssText$)(?:error|err|message|msg|label|title|text|description|desc|hint|tooltip|placeholder|caption|subtitle|heading|\w+(?:Error|Message|Msg|Label|Title|Text|Result|Caption|Hint|Tooltip))$/;
// Calls whose first argument is shown to the user.
const USER_CALLS = new Set(['toastStore.show', 'toastStore.showMsg', 'alert', 'confirm', 'prompt', 'window.alert', 'window.confirm', 'window.prompt']);
// Calls whose string arguments are keys, selectors, event names or other machine tokens.
const MACHINE_CALLS = new Set([
  'toastStore.showKey',
  'addEventListener', 'removeEventListener', 'querySelector', 'querySelectorAll', 'getElementById', 'closest',
  'setAttribute', 'getAttribute', 'removeAttribute', 'matchMedia', 'createElement', 'dispatchEvent',
  'localStorage.getItem', 'localStorage.setItem', 'localStorage.removeItem',
  'sessionStorage.getItem', 'sessionStorage.setItem', 'sessionStorage.removeItem',
  'document.documentElement.style.setProperty', 'document.documentElement.style.removeProperty',
  'EventsOn', 'EventsOff', 'EventsEmit', 'window.open', 'open', 'fetch', 'RegExp',
]);
// Method names that take a search/format token rather than displayed text.
const MACHINE_METHODS = /(?:^|\.)(?:includes|startsWith|endsWith|indexOf|lastIndexOf|split|join|replace|replaceAll|match|test|padStart|padEnd|concat|toFixed|toLocaleString|toLocaleDateString|localeCompare|on|off|emit|once|has|get|set|delete|add|remove|toggle|contains|getPropertyValue|setProperty|removeProperty|append|scrollIntoView|focus|assign|keys|hasOwnProperty)$/;
const COMPARISON = new Set(['===', '!==', '==', '!=', '<', '>', '<=', '>=', 'in', 'instanceof']);
const PASS_THROUGH = new Set(['LogicalExpression', 'TemplateLiteral', 'ChainExpression', 'ParenthesizedExpression', 'TSAsExpression', 'TSNonNullExpression', 'TSSatisfiesExpression', 'AwaitExpression']);
const RUNES = new Set(['$state', '$state.raw', '$derived', '$bindable']);
const FUNCTIONS = new Set(['ArrowFunctionExpression', 'FunctionExpression', 'FunctionDeclaration']);
const COMPONENT_NODES = new Set(['Component', 'SvelteComponent', 'SvelteSelf']);

function strip(raw: string): string {
  return raw.replace(NOUN_RE, '').replace(TECH_RE, '').replace(TECH_PHRASES, '').replace(NON_PROSE, '');
}

function visibleText(raw: string): boolean {
  return /\p{L}/u.test(strip(raw));
}

/** Any letter counts, unless the text is an ALL_CAPS constant (enum value, acronym). */
function strictText(raw: string): boolean {
  const s = strip(raw);
  return /\p{L}/u.test(s) && /\p{Ll}/u.test(s);
}

/** Looks like a sentence or label: capitalised word, or several words. Filters ids, keys and CSS in plain script. */
function proseText(raw: string): boolean {
  const s = strip(raw).trim();
  return /^\p{Lu}\p{Ll}/u.test(s) || /\p{L}{2,}\s+\p{L}{2,}/u.test(s);
}

// Inline style fragments built from parts: `left: ${x}px; width: ${w}px`.
const CSS_DECLARATION = /(?:^|[;\s])(?:left|right|top|bottom|width|height|min-\w+|max-\w+|margin[\w-]*|padding[\w-]*|transform|opacity|color|background[\w-]*|border[\w-]*|flex[\w-]*|gap|display|position|z-index|font-[\w-]+|grid[\w-]*|--[\w-]+)\s*:/;

/** A whitespace-delimited English word next to an interpolation or `+`, e.g. `${n} files` or `x + ' items'`. */
function concatText(raw: string): boolean {
  const s = strip(raw);
  if (!/\s/.test(s) || CSS_DECLARATION.test(s)) return false;
  return s.split(/\s+/).some((tok) => {
    const word = tok.replace(/^[^\p{L}]+|[^\p{L}]+$/gu, '');
    return /^\p{L}[\p{L}'’-]+$/u.test(word) && !CSS_WORDS.has(word.toLowerCase());
  });
}

/** Loose view of the ESTree / Svelte AST nodes this scanner walks; only the fields it reads are declared. */
interface AstNode {
  type: string;
  start: number;
  name?: string;
  data?: string;
  operator?: string;
  computed?: boolean;
  directive?: string;
  value?: string | boolean | AstNode | AstNode[] | { cooked?: string } | null;
  test?: AstNode | null;
  key?: AstNode;
  left?: AstNode;
  id?: AstNode;
  context?: AstNode;
  callee?: AstNode;
  arguments?: AstNode[];
  object?: AstNode;
  property?: AstNode;
  expression?: AstNode;
  consequent?: AstNode;
  alternate?: AstNode;
  right?: AstNode;
  expressions?: AstNode[];
  declaration?: AstNode;
  declarations?: AstNode[];
  init?: AstNode;
  fragment?: { nodes: AstNode[] };
  attributes?: AstNode[];
  body?: AstNode;
}

type Mode = 'skip' | 'strict' | 'display' | 'loose';

function calleeName(callee: AstNode | null | undefined): string {
  if (!callee) return '';
  if (callee.type === 'Identifier') return callee.name ?? '';
  if (callee.type === 'MemberExpression' && !callee.computed) return `${calleeName(callee.object)}.${callee.property?.name}`;
  if (callee.type === 'CallExpression') return `${calleeName(callee.callee)}()`;
  if (callee.type === 'ChainExpression') return calleeName(callee.expression);
  return '';
}

function targetName(node: AstNode | null | undefined): string {
  if (!node) return '';
  if (node.type === 'Identifier') return node.name ?? '';
  if (node.type === 'MemberExpression' && !node.computed) return node.property?.name ?? '';
  if (node.type === 'Literal') return String(node.value);
  return '';
}

function isTranslateCall(name: string): boolean {
  return name === '$t' || name === 't' || name.startsWith('get(t)');
}

/** Names whose value ends up printed by the template ({x}, {x.y}, {x ? a : b}, {fn()}, {`${x}`}). */
function collectDisplayed(fragment: AstNode | undefined): Set<string> {
  const names = new Set<string>();
  const eachLinks: [string, AstNode][] = [];

  const value = (e: AstNode | null | undefined): void => {
    if (!e) return;
    switch (e.type) {
      case 'Identifier': names.add(e.name ?? ''); return;
      case 'MemberExpression': value(e.object); return;
      case 'ChainExpression': case 'TSAsExpression': case 'TSNonNullExpression': case 'AwaitExpression': case 'ParenthesizedExpression':
        value(e.expression); return;
      case 'ConditionalExpression': value(e.consequent); value(e.alternate); return;
      case 'LogicalExpression': value(e.left); value(e.right); return;
      case 'BinaryExpression': if (e.operator === '+') { value(e.left); value(e.right); } return;
      case 'TemplateLiteral': (e.expressions ?? []).forEach(value); return;
      case 'CallExpression': {
        if (isTranslateCall(calleeName(e.callee))) return;
        if (e.callee?.type === 'Identifier') names.add(e.callee.name ?? '');
        return;
      }
      default:
    }
  };

  const walk = (node: unknown, insideAttr: boolean): void => {
    if (!node || typeof node !== 'object') return;
    if (Array.isArray(node)) { node.forEach((n) => walk(n, insideAttr)); return; }
    const n = node as AstNode;
    if (typeof n.type !== 'string') return;
    if (n.type === 'ExpressionTag' && !insideAttr) { value(n.expression); return; }
    if (n.type === 'RenderTag') { (n.expression?.arguments ?? []).forEach(value); return; }
    if (n.type === 'EachBlock') {
      const ctx = n.context ? targetName(n.context) : '';
      if (ctx && n.expression) eachLinks.push([ctx, n.expression]);
    }
    if (n.type === 'Attribute') {
      // Only the visible attributes of the element print their expressions; handled through the parent element below.
      return;
    }
    if (n.type === 'RegularElement' || COMPONENT_NODES.has(n.type) || n.type === 'SvelteElement') {
      for (const a of n.attributes ?? []) {
        if (a.type !== 'Attribute' || !isVisibleAttr(n.type, a.name ?? '')) continue;
        const vals = Array.isArray(a.value) ? a.value : a.value && a.value !== true ? [a.value as AstNode] : [];
        vals.forEach((v) => v.type === 'ExpressionTag' && value(v.expression));
      }
    }
    for (const key of Object.keys(n)) {
      if (key === 'attributes' || key === 'parent' || key === 'metadata') continue;
      const val = (n as unknown as Record<string, unknown>)[key];
      if (val && typeof val === 'object') walk(val, insideAttr);
    }
  };
  walk(fragment, false);

  // {#each list as item}{item.label}: when the context variable is printed, the list is printed too.
  let changed = true;
  while (changed) {
    changed = false;
    for (const [ctx, expr] of eachLinks) {
      if (!names.has(ctx)) continue;
      const before = names.size;
      value(expr);
      if (names.size !== before) changed = true;
    }
  }
  names.delete('');
  return names;
}

function isVisibleAttr(elementType: string | undefined, name: string): boolean {
  if (VISIBLE_ATTRS.includes(name)) return true;
  return !!elementType && COMPONENT_NODES.has(elementType) && TEXT_NAME.test(name);
}

function nameMode(name: string, displayed: Set<string>): Mode {
  if (TEXT_NAME.test(name)) return 'strict';
  return displayed.has(name) ? 'display' : 'loose';
}

/** Mode for a value returned by the function at anc[fnIndex], judged by the name that function is bound to. */
function functionMode(anc: AstNode[], fnIndex: number, displayed: Set<string>): Mode {
  const fn = anc[fnIndex];
  if (fn.type === 'FunctionDeclaration') return nameMode(targetName(fn.id), displayed);
  let parent = anc[fnIndex - 1];
  // const f = $derived.by(() => ...)
  if (parent?.type === 'CallExpression' && calleeName(parent.callee) === '$derived.by') parent = anc[fnIndex - 2];
  if (parent?.type === 'VariableDeclarator') return nameMode(targetName(parent.id), displayed);
  if (parent?.type === 'Property') return nameMode(targetName(parent.key), displayed);
  return 'loose';
}

/** Decides how strictly a string literal is checked by looking at where it is used. */
function modeFor(node: AstNode, anc: AstNode[], displayed: Set<string>): Mode {
  let child = node;
  for (let i = anc.length - 1; i >= 0; i--) {
    const p = anc[i];
    const type: string = p.type;
    if (type.startsWith('TS') && !PASS_THROUGH.has(type)) return 'skip';
    if (type === 'ImportDeclaration' || type === 'ExportAllDeclaration' || type === 'ExportNamedDeclaration') return 'skip';
    if (type === 'ExpressionStatement' && p.directive) return 'skip';
    if (type === 'SwitchCase') return child === p.test ? 'skip' : 'loose';
    if (type === 'BinaryExpression') {
      if (COMPARISON.has(p.operator ?? '')) return 'skip';
      if (p.operator !== '+') return 'skip';
    } else if (type === 'ConditionalExpression') {
      if (child === p.test) return 'skip';
    } else if (type === 'MemberExpression') {
      if (p.computed && child === p.property) return 'skip';
      return 'loose';
    } else if (type === 'Property') {
      if (child === p.key) return 'skip';
      return TEXT_NAME.test(targetName(p.key)) ? 'strict' : 'loose';
    } else if (type === 'AssignmentPattern') {
      // Destructuring default: let { title = 'Untitled' } = $props()
      return child === p.left ? 'skip' : nameMode(targetName(p.left), displayed);
    } else if (type === 'AssignmentExpression') {
      if (child === p.left) return 'skip';
      const direct = targetName(p.left);
      return TEXT_NAME.test(direct) ? 'strict' : p.left?.type === 'Identifier' && displayed.has(direct) ? 'display' : 'loose';
    } else if (type === 'VariableDeclarator') {
      return nameMode(targetName(p.id), displayed);
    } else if (type === 'ArrayExpression') {
      // Elements inherit the mode of whatever the array is assigned to (const tabs = ['a', 'b']).
    } else if (type === 'EachBlock') {
      if (child !== p.expression) return 'loose';
      return displayed.has(targetName(p.context)) ? 'display' : 'loose';
    } else if (type === 'CallExpression' || type === 'NewExpression') {
      const name = calleeName(p.callee);
      if (RUNES.has(name)) { child = p; continue; }
      if (isTranslateCall(name)) return 'skip';
      if (name.startsWith('console.') || MACHINE_CALLS.has(name) || MACHINE_METHODS.test(name)) return 'skip';
      if (USER_CALLS.has(name)) return p.arguments?.[0] === child ? 'strict' : 'skip';
      return 'loose';
    } else if (type === 'ExpressionTag') {
      // Template expression. Inside an attribute only the visible ones matter.
      const idx = anc.slice(0, i).map((a) => a.type === 'Attribute' || a.type.endsWith('Directive')).lastIndexOf(true);
      if (idx < 0) return 'strict';
      const attr = anc[idx];
      if (attr.type === 'Attribute') return isVisibleAttr(anc[idx - 1]?.type, attr.name ?? '') ? 'strict' : 'skip';
      return attr.type === 'OnDirective' ? 'loose' : 'skip';
    } else if (type === 'Attribute') {
      return isVisibleAttr(anc[i - 1]?.type, p.name ?? '') ? 'strict' : 'skip';
    } else if (type === 'ReturnStatement') {
      for (let j = i - 1; j >= 0; j--) if (FUNCTIONS.has(anc[j].type)) return functionMode(anc, j, displayed);
      return 'loose';
    } else if (FUNCTIONS.has(type)) {
      // Expression-bodied arrow: () => 'text'
      return child === p.body ? functionMode(anc, i, displayed) : 'loose';
    } else if (!PASS_THROUGH.has(type)) {
      return 'loose';
    }
    child = p;
  }
  return 'loose';
}

function isConcatPart(anc: AstNode[]): boolean {
  const p = anc[anc.length - 1];
  if (!p) return false;
  if (p.type === 'TemplateLiteral') return (p.expressions ?? []).length > 0;
  return p.type === 'BinaryExpression' && p.operator === '+';
}

const textOf = (n: AstNode): string => (n.data ?? '').trim();

/** <option> without content shows its value; <input type=submit|button|reset> always does. */
function implicitValueText(n: AstNode): AstNode | null {
  const attrs = n.attributes ?? [];
  const valueAttr = attrs.find((a) => a.type === 'Attribute' && a.name === 'value');
  const valueNode = valueAttr && Array.isArray(valueAttr.value) ? valueAttr.value.find((v) => v.type === 'Text') : null;
  if (!valueNode) return null;
  if (n.name === 'option') {
    const children = n.fragment?.nodes ?? [];
    return children.every((c) => c.type === 'Text' && !textOf(c)) ? valueNode : null;
  }
  if (n.name === 'input') {
    const typeAttr = attrs.find((a) => a.type === 'Attribute' && a.name === 'type');
    const t = typeAttr && Array.isArray(typeAttr.value) ? typeAttr.value.find((v) => v.type === 'Text')?.data : undefined;
    return t === 'submit' || t === 'button' || t === 'reset' ? valueNode : null;
  }
  return null;
}

function scanNode(root: AstNode | undefined, file: string, displayed: Set<string>, lineOf: (pos: number) => number, out: string[]) {
  const allowed = [...(ALLOWED_STRINGS['*'] ?? []), ...(ALLOWED_STRINGS[file] ?? [])];
  const visit = (node: AstNode | AstNode[] | null | undefined, anc: AstNode[]) => {
    if (!node || typeof node !== 'object') return;
    if (Array.isArray(node)) {
      node.forEach((c) => visit(c, anc));
      return;
    }
    if (typeof node.type !== 'string') return;

    if (node.type === 'Text') {
      // Static text: only real text children and visible attributes count (class, href, id... do not).
      const inAttr = anc.some((a) => a.type === 'Attribute');
      if (!inAttr && visibleText(node.data ?? '')) out.push(`${file}:${lineOf(node.start)} text "${textOf(node).slice(0, 40)}"`);
      return;
    }
    if (node.type === 'RegularElement') {
      const implicit = implicitValueText(node);
      if (implicit && visibleText(implicit.data ?? '')) out.push(`${file}:${lineOf(implicit.start)} implicit value "${(implicit.data ?? '').slice(0, 40)}"`);
    }
    if (node.type === 'Attribute') {
      const element = anc[anc.length - 1];
      const visible = isVisibleAttr(element?.type, node.name ?? '');
      const vals = (Array.isArray(node.value) ? node.value : node.value === true ? [] : [node.value]) as AstNode[];
      for (const v of vals) {
        if (v?.type === 'Text') {
          if (visible && visibleText(v.data ?? '')) out.push(`${file}:${lineOf(v.start)} attr ${node.name}="${(v.data ?? '').slice(0, 40)}"`);
        } else visit(v, [...anc, node]);
      }
      return;
    }

    const str = node.type === 'Literal' && typeof node.value === 'string' ? node.value
      : node.type === 'TemplateElement' ? (node.value as { cooked?: string }).cooked ?? null : null;
    if (str != null) {
      if (!allowed.includes(str.trim())) {
        const mode = modeFor(node, anc, displayed);
        let bad = false;
        if (mode === 'strict' || mode === 'display') bad = strictText(str);
        else if (mode === 'loose') bad = proseText(str) || (isConcatPart(anc) && concatText(str));
        if (bad) out.push(`${file}:${lineOf(node.start)} ${mode} string ${JSON.stringify(str.slice(0, 40))}`);
      }
      return;
    }
    for (const key of Object.keys(node)) {
      if (key === 'parent' || key === 'metadata') continue;
      const val = (node as unknown as Record<string, unknown>)[key];
      if (val && typeof val === 'object') visit(val as AstNode, [...anc, node]);
    }
  };
  visit(root, []);
}

/** Returns one finding per hardcoded user-visible string in a component (.svelte) or module (.ts). */
export function scanSource(file: string, source: string): string[] {
  const isTs = file.endsWith('.ts');
  const code = isTs ? `<script lang="ts">${source}</script>` : source;
  const ast = parse(code, { modern: true }) as unknown as { fragment: AstNode; instance?: { content: AstNode }; module?: { content: AstNode } };
  const lineOf = (pos: number) => code.slice(0, pos).split('\n').length;
  const displayed = collectDisplayed(ast.fragment);
  const found: string[] = [];
  scanNode(ast.fragment, file, displayed, lineOf, found);
  scanNode(ast.instance?.content, file, displayed, lineOf, found);
  scanNode(ast.module?.content, file, displayed, lineOf, found);
  return found;
}
