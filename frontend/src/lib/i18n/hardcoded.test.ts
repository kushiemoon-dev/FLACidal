import { describe, it, expect } from 'vitest';
import { parse } from 'svelte/compiler';

const sources = import.meta.glob('/src/**/*.svelte', { query: '?raw', import: 'default', eager: true }) as Record<string, string>;

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

const PROPER_NOUNS = ['FLACidal', 'Tidal', 'Qobuz', 'Soulseek', 'Jellyfin', 'English', 'Français', 'Deutsch', 'Amazon', 'HiFi', 'ReplayGain', 'FFmpeg', 'Nicotine+', 'Ko-fi', 'BPM'];
// Untranslated technical units and literal shell commands.
const TECH_TERMS = /\b(?:kHz|Hz|kbps|ISRC|ms|sldl)\b|-?\bbit\b|sudo pacman -S ffmpeg/g;
const ATTRS = ['title', 'placeholder', 'aria-label', 'alt'];

function visibleText(raw: string): boolean {
  let s = raw;
  for (const n of PROPER_NOUNS) s = s.split(n).join('');
  s = s.replace(TECH_TERMS, '');
  return /\p{L}/u.test(s);
}

function walk(node: any, out: string[]) {
  if (!node || typeof node !== 'object') return;
  if (node.type === 'Text' && visibleText(node.data)) out.push(`text "${node.data.trim().slice(0, 40)}"`);
  if (node.type === 'Attribute') {
    // Static attribute values are Text nodes but only ATTRS are user-visible (class, href... are not).
    if (ATTRS.includes(node.name)) {
      const vals = Array.isArray(node.value) ? node.value : [node.value];
      for (const v of vals) {
        if (v && v.type === 'Text' && visibleText(v.data)) out.push(`attr ${node.name}="${v.data.slice(0, 40)}"`);
      }
    }
    return;
  }
  for (const key of Object.keys(node)) {
    if (key === 'parent' || key === 'metadata') continue;
    const val = node[key];
    if (Array.isArray(val)) val.forEach((c) => walk(c, out));
    else if (val && typeof val === 'object' && typeof val.type === 'string') walk(val, out);
  }
}

function scan(file: string): string[] {
  const source = sources[`/src/${file}`];
  const ast: any = parse(source, { modern: true });
  const found: string[] = [];
  walk(ast.fragment, found);
  const script = ast.instance ? source.slice(ast.instance.start, ast.instance.end) : '';
  const literalCall = /(?:toastStore\.show|confirm)\(\s*(?:'[^']*'|"[^"]*"|`[^`$]*`)/g;
  for (const m of script.matchAll(literalCall)) found.push(`call ${m[0].slice(0, 50)}`);
  return found;
}

describe('no hardcoded UI strings in migrated files', () => {
  it.each(MIGRATED)('%s', (file) => {
    expect(scan(file)).toEqual([]);
  });
  it('MIGRATED only lists known targets', () => {
    expect(MIGRATED.filter((f) => !ALL_TARGETS.includes(f))).toEqual([]);
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
