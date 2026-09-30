import { describe, it, expect } from 'vitest';
import { scanSource } from './hardcodedScanner';

const sources = import.meta.glob(['/src/**/*.svelte', '/src/stores/*.ts', '/src/lib/*.ts'], { query: '?raw', import: 'default', eager: true }) as Record<string, string>;

// Paths relative to src/. Every .svelte file under src/ must be listed either here or in EXCLUDED.
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

// Svelte files deliberately not scanned, each with the reason. Keep this list tiny; a new component belongs in MIGRATED.
const EXCLUDED: Record<string, string> = {};

// Non-component code that can produce user-visible text. Every src/stores/*.ts and src/lib/*.ts module is scanned
// (tests aside) except the ones below.
const TS_EXCLUDED: Record<string, string> = {
  'lib/api.ts': 'browser-mode developer errors, never shown as product UI',
  'lib/websocket.ts': 'console diagnostics only',
};

// TODO(i18n): real leftovers the stricter scanner found in the tree. Fix them in the components, then delete the entry.
// Format: the finding without its line number, exactly as reported (`file kind detail`).
const KNOWN_LEFTOVERS: string[] = [];

const stripLine = (finding: string) => finding.replace(/^([^:]+):\d+ /, '$1 ');
const srcPath = (p: string) => p.replace(/^\/src\//, '');

const svelteFiles = Object.keys(sources).map(srcPath).filter((p) => p.endsWith('.svelte')).sort();
const tsFiles = Object.keys(sources)
  .map(srcPath)
  .filter((p) => /^(?:stores|lib)\/[^/]+\.ts$/.test(p) && !/\.(?:test|spec)\.ts$/.test(p) && !(p in TS_EXCLUDED))
  .sort();

function scan(file: string): string[] {
  return scanSource(file, sources[`/src/${file}`]).filter((f) => !KNOWN_LEFTOVERS.includes(stripLine(f)));
}

const scanned = svelteFiles.filter((f) => !(f in EXCLUDED));

describe(`no hardcoded UI strings (${scanned.length} svelte + ${tsFiles.length} ts files scanned)`, () => {
  it.each(scanned)('%s', (file) => {
    expect(scan(file)).toEqual([]);
  });
  it.each(tsFiles)('%s', (file) => {
    expect(scan(file)).toEqual([]);
  });
});

describe('scan coverage', () => {
  it('every .svelte file under src/ is MIGRATED or EXCLUDED (add new components to MIGRATED)', () => {
    const known = [...MIGRATED, ...Object.keys(EXCLUDED)];
    expect(svelteFiles.filter((f) => !known.includes(f))).toEqual([]);
  });
  it('MIGRATED and EXCLUDED only list existing, distinct files', () => {
    expect(MIGRATED.filter((f) => !svelteFiles.includes(f))).toEqual([]);
    expect(Object.keys(EXCLUDED).filter((f) => !svelteFiles.includes(f))).toEqual([]);
    expect(MIGRATED.filter((f) => f in EXCLUDED)).toEqual([]);
    expect(new Set(MIGRATED).size).toBe(MIGRATED.length);
  });
  it('every EXCLUDED entry states a reason', () => {
    expect([...Object.values(EXCLUDED), ...Object.values(TS_EXCLUDED)].filter((r) => r.trim().length < 5)).toEqual([]);
  });
  it('scans every store and lib module', () => {
    expect(tsFiles).toEqual(expect.arrayContaining(['stores/toast.ts', 'stores/queue.ts', 'lib/format.ts']));
    expect(tsFiles.some((f) => f in TS_EXCLUDED)).toBe(false);
  });
});

describe('scanner catches each class of hardcoded string', () => {
  const wrap = (script: string, markup = '') => `<script lang="ts">\n${script}\n</script>\n${markup}`;
  const toast = "import { toastStore } from '../stores/toast';";
  const violations: [string, string][] = [
    // Static markup and attributes
    ['static text', wrap('', '<p>Hello world</p>')],
    ['single lowercase text', wrap('', '<p>hello</p>')],
    ['svelte:head title', wrap('', '<svelte:head><title>My Player</title></svelte:head>')],
    ['title attribute', wrap('', '<button title="Close it">x</button>')],
    ['aria-label attribute', wrap('', '<button aria-label="Close">x</button>')],
    ['aria-description attribute', wrap('', '<div aria-description="Drop files here"></div>')],
    ['aria-roledescription attribute', wrap('', '<div aria-roledescription="slider"></div>')],
    ['aria-placeholder attribute', wrap('', '<div aria-placeholder="Search"></div>')],
    ['data-tooltip attribute', wrap('', '<span data-tooltip="Copy path">x</span>')],
    ['optgroup label attribute', wrap('', '<select><optgroup label="Lossless"></optgroup></select>')],
    ['option text', wrap('', '<select><option value="a">Alpha</option></select>')],
    ['value-less option shows its value', wrap('', '<select><option value="Alpha"></option></select>')],
    ['submit input value', wrap('', '<input type="submit" value="Save" />')],
    ['component text prop', wrap('', '<Foo title="Close" />')],
    ['component expression prop', wrap('', "<Foo message={'Hi there'} />")],
    ['text in snippet body', wrap('', '{#snippet row()}<span>Hello</span>{/snippet}')],
    ['expression in snippet body', wrap('', "{#snippet row()}<span>{'Done'}</span>{/snippet}")],
    ['@const body', wrap('', "{#if true}{@const t = 'Loading'}<span>{t}</span>{/if}")],
    ['each over literal list', wrap('', "{#each ['alpha', 'beta'] as x}<li>{x}</li>{/each}")],
    // Expressions in markup
    ['ternary in text', wrap('let ok = true;', "<span>{ok ? 'Yes' : 'No'}</span>")],
    ['lowercase ternary in text', wrap('let ok = true;', "<span>{ok ? 'yes' : 'no'}</span>")],
    ['logical fallback in text', wrap("let v = '';", "<span>{v || 'Unavailable'}</span>")],
    ['template literal in text', wrap('let n = 1;', '<span>{`${n} files selected`}</span>')],
    ['title expression', wrap('', "<button title={'Close it'}>x</button>")],
    ['aria-label conditional', wrap('let o = true;', "<button aria-label={o ? 'Collapse' : 'Expand'}>x</button>")],
    ['placeholder expression', wrap('', "<input placeholder={'Type here'} />")],
    // Script values that reach the template
    ['state assignment', wrap("let error = $state(''); function f() { error = 'Failed to load'; }")],
    ['lowercase state assignment', wrap("let message = $state(''); function f() { message = 'failed'; }")],
    ['fallback assignment', wrap("let error = $state(''); function f(e: any) { error = e.message || 'Something broke'; }")],
    ['$state initial text', wrap("let error = $state('Not ready');")],
    ['printed const, lowercase word', wrap("const phase = 'loading';", '<p>{phase}</p>')],
    ['printed $derived ternary', wrap("let ok = true; const phase = $derived(ok ? 'ready' : 'broken');", '<p>{phase}</p>')],
    ['printed assignment', wrap("let phase = $state(''); function f() { phase = 'working'; }", '<p>{phase}</p>')],
    ['printed array of words', wrap("const names = ['alpha', 'beta'];", '{#each names as n}<li>{n}</li>{/each}')],
    ['printed function return', wrap("function phase() { return 'ready'; }", '<p>{phase()}</p>')],
    ['printed $derived.by return', wrap("let ok = true; const phase = $derived.by(() => { return ok ? 'ready' : 'broken'; });", '<p>{phase}</p>')],
    ['property label', wrap("const opts = [{ value: 'a', label: 'All files' }];")],
    ['property lowercase label', wrap("const opts = [{ value: 'a', label: 'files' }];")],
    ['property caption', wrap("const c = { caption: 'files' };")],
    ['property tooltip', wrap("const c = { tooltip: 'copy' };")],
    ['property hint', wrap("const c = { hint: 'optional' };")],
    ['property name, capitalised', wrap("const c = { name: 'Alpha' };")],
    ['property status, phrase', wrap("const c = { status: 'Ready to go' };")],
    ['prop default', wrap("let { title = 'Untitled' } = $props();")],
    ['concatenation', wrap("let n = 1; const s = 'Loaded ' + n + ' files';")],
    ['lowercase concatenation', wrap("let n = 1; const s = n + ' files';")],
    ['template literal with words', wrap('let n = 1; const s = `${n} files`;')],
    // Calls
    ['toast call', wrap(`${toast} toastStore.show('Saved');`)],
    ['toast call with fallback', wrap(`${toast} function f(e: any) { toastStore.show(e.message || 'Failed', 'error'); }`)],
    ['toast template literal', wrap(`${toast} toastStore.show(\`Added \${1} items\`);`)],
    ['confirm call', wrap("function f() { return confirm('Delete everything?'); }")],
    ['alert call', wrap("function f() { alert('Oops'); }")],
    ['thrown error text', wrap("function f() { throw new Error('Nothing selected'); }")],
    ['event handler toast', wrap(`${toast}`, "<button onclick={() => toastStore.show('Copied')}>x</button>")],
    // Allowlists work on whole tokens only
    ['proper noun inside a longer word', wrap('', '<p>Bandcampus</p>')],
    ['tech term inside a longer word', wrap('', '<p>ffmpegs and mss</p>')],
    ['parenthesised prose is not a media query', wrap('', '<p>(Delete everything)</p>')],
    ['prose next to a proper noun', wrap('', '<p>Open Qobuz</p>')],
    ['prose next to a unit', wrap('', '<p>Sample rate in kHz</p>')],
  ];
  it.each(violations)('flags: %s', (_name, source) => {
    expect(scanSource('x.svelte', source)).not.toEqual([]);
  });
  it('has at least 30 injected violations', () => {
    expect(violations.length).toBeGreaterThanOrEqual(30);
  });

  it('flags a hardcoded string in a store module', () => {
    expect(scanSource('stores/x.ts', "export const msg = { text: 'Something failed' };")).not.toEqual([]);
    expect(scanSource('stores/x.ts', "let error = ''; export function f() { error = 'boom'; }")).not.toEqual([]);
    expect(scanSource('stores/x.ts', "export function f(n: number) { return n + ' files'; }")).not.toEqual([]);
  });

  const clean: [string, string][] = [
    ['translated text', wrap('import { t } from "../lib/i18n";', "<p>{$t('a.b')}</p>")],
    ['comparisons', wrap("let s = 'idle'; function f(e: KeyboardEvent) { return s === 'idle' && e.key === 'Enter'; }")],
    ['class and id attributes', wrap('let on = true;', "<div id=\"main\" class=\"btn primary {on ? 'active' : 'idle'}\" class:open={on}>{$t('a.b')}</div>")],
    ['console output', wrap("console.error('Load failed:', 1); console.warn(`Retrying ${1} times`);")],
    ['event and storage names', wrap("window.addEventListener('keydown', () => {}); localStorage.setItem('flacidal-region', 'US');")],
    ['keys, enums and object keys', wrap("const q = { status: 'error', kind: 'album' }; const k = 'nav.home';")],
    ['translation keys in stores', wrap("import { get } from 'svelte/store'; const s = get(t)('shell.downloadFailed');")],
    ['showKey', wrap(`${'import { toastStore } from "../stores/toast";'} toastStore.showKey('a.b', { n: 1 }, 'error');`)],
    ['key messages', wrap("let error = $state<any>(''); function f(e: any) { error = e.message || { key: 'a.b' }; }")],
    ['units and proper nouns', wrap('', '<span>{n} kHz / 24-bit</span><b>FLACidal</b><i>Qobuz</i>')],
    ['codec and size tokens', wrap('', '<span>FLAC / MP3 / AAC 320 kbps, 4.2 MB</span>')],
    ['imports', wrap("import x from 'some-package';")],
    ['data URLs and css', wrap("el.style.cssText = `left: ${1}px;`; const u = `data:image/png;base64,${1}`;")],
    ['css values built from parts', wrap('let x = 1; let w = 2; const a = `translate(${x}px, ${x}px)`; const b = `calc(100% - ${w}px)`;')],
    ['path built from parts', wrap('let dir = ""; let name = ""; const p = `${dir}/${name}.flac`;')],
    ['svelte:head with translated title', wrap('', "<svelte:head><title>{$t('app.title')}</title></svelte:head>")],
    ['option with translated text and machine value', wrap('', "<select><option value=\"flac\">{$t('fmt.flac')}</option></select>")],
    ['input value bound to state', wrap("let q = $state('');", '<input type="text" bind:value={q} />')],
    ['submit input with expression', wrap('', "<input type=\"submit\" value={$t('a.save')} />")],
    ['machine attributes', wrap('', '<a href="/settings" role="button" type="button" data-testid="save-btn" aria-labelledby="title-id">{$t("a")}</a>')],
    ['component props from translations', wrap('', "<Foo title={$t('a')} kind=\"primary\" size=\"lg\" />")],
    ['printed translated derived', wrap("let ok = true; const phase = $derived(ok ? $t('a') : $t('b'));", '<p>{phase}</p>')],
    ['state only compared in the template', wrap("let phase = $state('idle');", "{#if phase === 'idle'}<p>{$t('k')}</p>{/if}")],
    ['lowercase state that is never printed', wrap("let open = $state('closed'); function f() { open = open === 'closed' ? 'open' : 'closed'; }")],
    ['printed number', wrap('let n = $state(0);', '<p>{n}</p>')],
    ['snippet using translations', wrap('', "{#snippet row(item)}<span>{$t(item.key)}</span>{/snippet}")],
    ['const enum-like uppercase', wrap("const MODE = 'FLAC';")],
    ['printed option labels via keys', wrap("const opts = [{ value: 'a', labelKey: 'a.b' }];", '{#each opts as o}<option value={o.value}>{$t(o.labelKey)}</option>{/each}')],
    ['proper noun as a whole token', wrap('', '<p>Soulseek</p><p>Nicotine+</p><p>Ko-fi</p>')],
  ];
  it.each(clean)('allows: %s', (_name, source) => {
    expect(scanSource('x.svelte', source)).toEqual([]);
  });
  it('has at least 20 clean snippets', () => {
    expect(clean.length).toBeGreaterThanOrEqual(20);
  });
});
