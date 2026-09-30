import { describe, it, expect } from 'vitest';
import { parse } from 'svelte/compiler';

const sources = import.meta.glob(['/src/**/*.svelte', '/src/stores/*.ts', '/src/lib/*.ts'], { query: '?raw', import: 'default', eager: true }) as Record<string, string>;

// Paths relative to src/. Each i18n PR appends the files it migrated.
const MIGRATED: string[] = [
  'App.svelte',
  'components/Sidebar.svelte', 'components/Toast.svelte', 'components/ConfirmDialog.svelte',
  'components/ContextMenu.svelte', 'components/TabBar.svelte', 'components/DropZone.svelte',
  'components/UpdateRequiredScreen.svelte',
  'pages/Home.svelte', 'pages/Search.svelte', 'pages/Queue.svelte', 'components/QueuePanel.svelte',
  'pages/History.svelte', 'pages/Files.svelte', 'pages/Settings.svelte',
  'components/AnalysisModal.svelte', 'components/ConvertModal.svelte',
  'components/MetadataModal.svelte', 'components/RenameModal.svelte',
  'pages/Terminal.svelte', 'pages/About.svelte', 'components/IssueReporterModal.svelte',
  'pages/tools/AudioConverter.svelte', 'pages/tools/FileManager.svelte',
  'pages/tools/AudioResampler.svelte', 'pages/tools/LyricsManager.svelte',
  'pages/tools/AudioQualityAnalyzer.svelte',
];

// Full R6 scope. MIGRATED must equal this once the migration is finished (PR6).
const ALL_TARGETS: string[] = [
  'App.svelte',
  'pages/Home.svelte', 'pages/Search.svelte', 'pages/Queue.svelte', 'pages/History.svelte',
  'pages/Files.svelte', 'pages/Settings.svelte', 'pages/Terminal.svelte', 'pages/About.svelte',
  'pages/tools/AudioConverter.svelte', 'pages/tools/FileManager.svelte',
  'pages/tools/AudioResampler.svelte', 'pages/tools/LyricsManager.svelte',
  'pages/tools/AudioQualityAnalyzer.svelte',
  'components/Sidebar.svelte', 'components/Toast.svelte', 'components/ConfirmDialog.svelte',
  'components/ContextMenu.svelte', 'components/TabBar.svelte', 'components/DropZone.svelte',
  'components/QueuePanel.svelte', 'components/IssueReporterModal.svelte',
  'components/UpdateRequiredScreen.svelte', 'components/AnalysisModal.svelte',
  'components/ConvertModal.svelte', 'components/MetadataModal.svelte', 'components/RenameModal.svelte',
];

// Non-component code that can produce user-visible text. lib/api.ts (browser-mode developer errors)
// and lib/websocket.ts (console diagnostics) are intentionally out of scope.
const TS_TARGETS: string[] = [
  'stores/audio.ts', 'stores/queue.ts', 'stores/theme.ts', 'stores/toast.ts',
  'lib/format.ts', 'lib/navHistory.ts', 'lib/runtime.ts',
];

// Proper nouns and product names stay untranslated (longest first so "Amazon Music" wins over "Amazon").
const PROPER_NOUNS = [
  'FLACidal Mobile', 'Amazon Music', 'Plus Jakarta Sans', 'Bricolage Grotesque', 'OpenDrop', 'FLACidal', 'YouFLAC', 'Tidal', 'Qobuz',
  'Soulseek', 'Jellyfin', 'Bandcamp', 'English', 'Français', 'Deutsch', 'Amazon', 'HiFi', 'ReplayGain', 'FFmpeg',
  'Nicotine+', 'Ko-fi', 'BPM', 'Outfit', 'VJ',
];
// Untranslated technical tokens: units, codecs, literal shell commands, log level tags, filename separators.
const TECH_TERMS = new RegExp(
  [
    String.raw`\b(?:kHz|Hz|kbps|ISRC|ms|sldl|ch|VBR|feat\.|sans-serif)\b`,
    String.raw`\b[KMG]?B\b`,
    String.raw`\b(?:MP3|AAC|OGG|Opus|Vorbis|ALAC|WAV|AIFF|FLAC)\b`,
    String.raw`-?\bbit\b`,
    String.raw`sudo pacman -S ffmpeg`,
    String.raw`\[(?:ERROR|WARN|OK|INFO|LOG)\]`,
  ].join('|'),
  'g'
);
// URLs, `{placeholder}` template variables, absolute paths and CSS media queries carry no prose.
const NON_PROSE = /https?:\/\/\S*|\{\w+\}|(?:^|\s)\/[\w.\-/]+|^\([^)]*\)$/g;
// Strings that are identifiers, not shown text. Keep this list short and reasoned.
const ALLOWED_STRINGS: Record<string, string[]> = {
  '*': ['v'], // version prefix in `v${version}`
  // Preset ids; the visible label comes from labelKey via $t.
  'stores/theme.ts': ['Pink', 'Purple', 'Blue', 'Cyan', 'Green', 'Orange', 'Red'],
};
// Attributes whose value is shown to the user or read by assistive tech.
const VISIBLE_ATTRS = ['title', 'placeholder', 'aria-label', 'alt', 'label', 'aria-description', 'aria-valuetext', 'aria-placeholder'];
// Names of variables/properties that hold user-facing text, so even a lowercase word assigned to them is flagged.
const TEXT_NAME = /^(?!cssText$)(?:error|err|message|msg|label|title|text|description|desc|hint|tooltip|placeholder|\w+(?:Error|Message|Msg|Label|Title|Text|Result))$/;
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

function strip(raw: string): string {
  let s = raw;
  for (const n of PROPER_NOUNS) s = s.split(n).join('');
  return s.replace(TECH_TERMS, '').replace(NON_PROSE, '');
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

function calleeName(callee: any): string {
  if (!callee) return '';
  if (callee.type === 'Identifier') return callee.name;
  if (callee.type === 'MemberExpression' && !callee.computed) return `${calleeName(callee.object)}.${callee.property.name}`;
  if (callee.type === 'CallExpression') return `${calleeName(callee.callee)}()`;
  if (callee.type === 'ChainExpression') return calleeName(callee.expression);
  return '';
}

function targetName(node: any): string {
  if (!node) return '';
  if (node.type === 'Identifier') return node.name;
  if (node.type === 'MemberExpression' && !node.computed) return node.property.name;
  if (node.type === 'Literal') return String(node.value);
  return '';
}

type Mode = 'skip' | 'strict' | 'loose';

/**
 * Decides how strictly a string literal is checked by looking at where it is used.
 * strict: shown as-is (template text, visible attribute, toast/alert/confirm argument, state named error/message/label...).
 * loose: plain script code, only sentence-like strings count.
 * skip: comparison operands, translation keys, imports, types, selectors, event names, object keys.
 */
function modeFor(node: any, anc: any[]): Mode {
  let child = node;
  for (let i = anc.length - 1; i >= 0; i--) {
    const p = anc[i];
    const type: string = p.type;
    if (type.startsWith('TS') && !PASS_THROUGH.has(type)) return 'skip';
    if (type === 'ImportDeclaration' || type === 'ExportAllDeclaration' || type === 'ExportNamedDeclaration') return 'skip';
    if (type === 'ExpressionStatement' && p.directive) return 'skip';
    if (type === 'SwitchCase') return child === p.test ? 'skip' : 'loose';
    if (type === 'BinaryExpression') {
      if (COMPARISON.has(p.operator)) return 'skip';
      if (p.operator !== '+') return 'skip';
    } else if (type === 'ConditionalExpression') {
      if (child === p.test) return 'skip';
    } else if (type === 'MemberExpression') {
      if (p.computed && child === p.property) return 'skip';
      return 'loose';
    } else if (type === 'Property') {
      if (child === p.key) return 'skip';
      return TEXT_NAME.test(targetName(p.key)) ? 'strict' : 'loose';
    } else if (type === 'AssignmentExpression') {
      return TEXT_NAME.test(targetName(p.left)) ? 'strict' : 'loose';
    } else if (type === 'VariableDeclarator') {
      return TEXT_NAME.test(targetName(p.id)) ? 'strict' : 'loose';
    } else if (type === 'CallExpression' || type === 'NewExpression') {
      const name = calleeName(p.callee);
      if (name === '$t' || name.startsWith('get(t)') || name === 't') return 'skip';
      if (name.startsWith('console.') || MACHINE_CALLS.has(name) || MACHINE_METHODS.test(name)) return 'skip';
      if (USER_CALLS.has(name)) return p.arguments[0] === child ? 'strict' : 'skip';
      return 'loose';
    } else if (type === 'ExpressionTag') {
      // Template expression. Inside an attribute only the visible ones matter.
      const attr = anc.slice(0, i).reverse().find((a) => a.type === 'Attribute' || a.type.endsWith('Directive'));
      if (!attr) return 'strict';
      if (attr.type === 'Attribute') return VISIBLE_ATTRS.includes(attr.name) ? 'strict' : 'skip';
      return attr.type === 'OnDirective' ? 'loose' : 'skip';
    } else if (type === 'Attribute') {
      return VISIBLE_ATTRS.includes(p.name) ? 'strict' : 'skip';
    } else if (type === 'ArrowFunctionExpression' || type === 'FunctionExpression' || type === 'FunctionDeclaration' || type === 'ReturnStatement') {
      return 'loose';
    } else if (!PASS_THROUGH.has(type) && type !== 'ConditionalExpression' && type !== 'BinaryExpression') {
      return 'loose';
    }
    child = p;
  }
  return 'loose';
}

function scanNode(root: any, file: string, lineOf: (pos: number) => number, out: string[]) {
  const allowed = [...(ALLOWED_STRINGS['*'] ?? []), ...(ALLOWED_STRINGS[file] ?? [])];
  const visit = (node: any, anc: any[]) => {
    if (!node || typeof node !== 'object') return;
    if (Array.isArray(node)) {
      node.forEach((c) => visit(c, anc));
      return;
    }
    if (typeof node.type !== 'string') return;

    if (node.type === 'Text') {
      // Static text: only real text children and visible attributes count (class, href, id... do not).
      const inAttr = anc.some((a) => a.type === 'Attribute');
      if (!inAttr && visibleText(node.data)) out.push(`${file}:${lineOf(node.start)} text "${node.data.trim().slice(0, 40)}"`);
      return;
    }
    if (node.type === 'Attribute') {
      const vals = Array.isArray(node.value) ? node.value : node.value === true ? [] : [node.value];
      for (const v of vals) {
        if (v?.type === 'Text') {
          if (VISIBLE_ATTRS.includes(node.name) && visibleText(v.data)) {
            out.push(`${file}:${lineOf(v.start)} attr ${node.name}="${v.data.slice(0, 40)}"`);
          }
        } else visit(v, [...anc, node]);
      }
      return;
    }

    const str = node.type === 'Literal' && typeof node.value === 'string' ? node.value
      : node.type === 'TemplateElement' ? node.value.cooked : null;
    if (str != null) {
      if (!allowed.includes(str.trim())) {
        const mode = modeFor(node, anc);
        const bad = mode === 'strict' ? strictText(str) : mode === 'loose' ? proseText(str) : false;
        if (bad) out.push(`${file}:${lineOf(node.start)} ${mode} string ${JSON.stringify(str.slice(0, 40))}`);
      }
      return;
    }
    for (const key of Object.keys(node)) {
      if (key === 'parent' || key === 'metadata') continue;
      const val = node[key];
      if (val && typeof val === 'object') visit(val, [...anc, node]);
    }
  };
  visit(root, []);
}

/** Returns one finding per hardcoded user-visible string in a component (.svelte) or module (.ts). */
export function scanSource(file: string, source: string): string[] {
  const isTs = file.endsWith('.ts');
  const code = isTs ? `<script lang="ts">${source}</script>` : source;
  const ast: any = parse(code, { modern: true });
  const lineOf = (pos: number) => code.slice(0, pos).split('\n').length;
  const found: string[] = [];
  scanNode(ast.fragment, file, lineOf, found);
  scanNode(ast.instance?.content, file, lineOf, found);
  scanNode(ast.module?.content, file, lineOf, found);
  return found;
}

function scan(file: string): string[] {
  return scanSource(file, sources[`/src/${file}`]);
}

describe('no hardcoded UI strings in migrated files', () => {
  it.each(MIGRATED)('%s', (file) => {
    expect(scan(file)).toEqual([]);
  });
  it.each(TS_TARGETS)('%s', (file) => {
    expect(scan(file)).toEqual([]);
  });
  it('covers every store module', () => {
    const stores = Object.keys(sources)
      .map((p) => p.replace('/src/', ''))
      .filter((p) => /^stores\/[^/]+\.ts$/.test(p) && !p.endsWith('.test.ts'));
    expect(stores.sort()).toEqual(TS_TARGETS.filter((f) => f.startsWith('stores/')).sort());
  });
  it('MIGRATED only lists known targets', () => {
    expect(MIGRATED.filter((f) => !ALL_TARGETS.includes(f))).toEqual([]);
  });
});

describe('scanner catches each class of hardcoded string', () => {
  const wrap = (script: string, markup = '') => `<script lang="ts">\n${script}\n</script>\n${markup}`;
  const cases: [string, string][] = [
    ['static text', wrap('', '<p>Hello world</p>')],
    ['static visible attribute', wrap('', '<button title="Close it">x</button>')],
    ['ternary in text', wrap('let ok = true;', "<span>{ok ? 'Yes' : 'No'}</span>")],
    ['lowercase ternary in text', wrap('let ok = true;', "<span>{ok ? 'yes' : 'no'}</span>")],
    ['logical fallback in text', wrap('let v = \'\';', "<span>{v || 'Unavailable'}</span>")],
    ['template literal in text', wrap('let n = 1;', '<span>{`${n} files selected`}</span>')],
    ['title expression', wrap('', "<button title={'Close it'}>x</button>")],
    ['aria-label conditional', wrap('let o = true;', "<button aria-label={o ? 'Collapse' : 'Expand'}>x</button>")],
    ['placeholder expression', wrap('', "<input placeholder={'Type here'} />")],
    ['state assignment', wrap("let error = $state(''); function f() { error = 'Failed to load'; }")],
    ['lowercase state assignment', wrap("let message = $state(''); function f() { message = 'failed'; }")],
    ['fallback assignment', wrap("let error = $state(''); function f(e: any) { error = e.message || 'Something broke'; }")],
    ['label array', wrap("const opts = [{ value: 'a', label: 'All files' }];")],
    ['toast call', wrap("import { toastStore } from '../stores/toast'; toastStore.show('Saved');")],
    ['toast call with fallback', wrap("import { toastStore } from '../stores/toast'; function f(e: any) { toastStore.show(e.message || 'Failed', 'error'); }")],
    ['toast template literal', wrap("import { toastStore } from '../stores/toast'; toastStore.show(`Added ${1} items`);")],
    ['confirm call', wrap("function f() { return confirm('Delete everything?'); }")],
    ['alert call', wrap("function f() { alert('Oops'); }")],
    ['thrown error text', wrap("function f() { throw new Error('Nothing selected'); }")],
    ['event handler toast', wrap("import { toastStore } from '../stores/toast';", "<button onclick={() => toastStore.show('Copied')}>x</button>")],
  ];
  it.each(cases)('flags: %s', (_name, source) => {
    expect(scanSource('x.svelte', source)).not.toEqual([]);
  });

  it('flags a hardcoded string in a store module', () => {
    expect(scanSource('stores/x.ts', "export const msg = { text: 'Something failed' };")).not.toEqual([]);
    expect(scanSource('stores/x.ts', "let error = ''; export function f() { error = 'boom'; }")).not.toEqual([]);
  });

  const clean: [string, string][] = [
    ['translated text', wrap('import { t } from "../lib/i18n";', "<p>{$t('a.b')}</p>")],
    ['comparisons', wrap("let s = 'idle'; function f(e: KeyboardEvent) { return s === 'idle' && e.key === 'Enter'; }")],
    ['class and id attributes', wrap('let on = true;', "<div id=\"main\" class=\"btn primary {on ? 'active' : 'idle'}\" class:open={on}>{$t('a.b')}</div>")],
    ['console output', wrap("console.error('Load failed:', 1); console.warn(`Retrying ${1} times`);")],
    ['event and storage names', wrap("window.addEventListener('keydown', () => {}); localStorage.setItem('flacidal-region', 'US');")],
    ['keys, enums and object keys', wrap("const q = { status: 'error', kind: 'album' }; const k = 'nav.home';")],
    ['translation keys in stores', wrap("import { get } from 'svelte/store'; const s = get(t)('shell.downloadFailed');")],
    ['showKey', wrap("import { toastStore } from '../stores/toast'; toastStore.showKey('a.b', { n: 1 }, 'error');")],
    ['key messages', wrap("let error = $state<any>(''); function f(e: any) { error = e.message || { key: 'a.b' }; }")],
    ['units and proper nouns', wrap('', '<span>{n} kHz / 24-bit</span><b>FLACidal</b><i>Qobuz</i>')],
    ['imports', wrap("import x from 'some-package';")],
    ['data URLs and css', wrap("el.style.cssText = `left: ${1}px;`; const u = `data:image/png;base64,${1}`;")],
  ];
  it.each(clean)('allows: %s', (_name, source) => {
    expect(scanSource('x.svelte', source)).toEqual([]);
  });
});

describe('migration progress', () => {
  it('reports remaining files', () => {
    const remaining = ALL_TARGETS.filter((f) => !MIGRATED.includes(f));
    if (remaining.length) console.warn(`i18n: ${remaining.length} files left to migrate`);
    expect(true).toBe(true);
  });
  it('MIGRATED equals ALL_TARGETS', () => {
    expect([...MIGRATED].sort()).toEqual([...ALL_TARGETS].sort());
  });
});
