<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { onNativeFileDrop } from '../../lib/runtime';
  import { type AnalysisResult, OpenFLACFilesDialog, SelectFolderForAnalysis, ExportAnalysisReport } from '../../lib/api';
  import { startAnalyzeBatch, MAX_ANALYZE_BATCH } from '../../lib/analyzeBatch';
  import { toastStore } from '../../stores/toast';
  import DropZone from '../../components/DropZone.svelte';
  import { FileSearch, CheckCircle, AlertTriangle, XCircle } from 'lucide-svelte';
  import { t, locale } from '../../lib/i18n';

  let files: string[] = $state([]);
  type AnalysisRow = AnalysisResult & { bpm?: number; musicalKey?: string };
  let results: AnalysisRow[] = $state([]);
  let isAnalyzing = $state(false);
  let done = $state(0);
  let total = $state(0);
  let cancelled = $state(false);
  let cancelRequested = $state(false);
  let cancelBatch: (() => Promise<boolean>) | null = null;

  const okRows = $derived(results.filter((r) => r.verdict !== 'error'));
  const errorRows = $derived(results.filter((r) => r.verdict === 'error'));

  async function analyzeFiles(paths: string[]) {
    if (paths.length === 0) return;
    if (paths.length > MAX_ANALYZE_BATCH) {
      toastStore.showKey('tools.analyzer.tooMany', { count: paths.length, max: MAX_ANALYZE_BATCH }, 'error');
      return;
    }
    files = paths;
    isAnalyzing = true;
    cancelled = false;
    cancelRequested = false;
    results = [];
    done = 0;
    total = paths.length;
    try {
      const run = startAnalyzeBatch(paths, (d, t, r) => {
        done = d;
        total = t;
        results = [...results, r];
      });
      cancelBatch = run.cancel;
      const outcome = await run.done;
      results = outcome.results;
      cancelled = outcome.cancelled;
    } catch (error) {
      toastStore.showKey('tools.analyzer.failed', { error: error instanceof Error ? error.message : String(error) }, 'error');
    } finally {
      isAnalyzing = false;
      cancelBatch = null;
    }
  }

  async function cancelAnalysis() {
    cancelRequested = true;
    try {
      // false: the server does not know the batch yet, so let the user try again
      if (!(await cancelBatch?.())) cancelRequested = false;
    } catch {
      cancelRequested = false;
    }
  }

  async function exportReport(format: 'csv' | 'json') {
    try {
      await ExportAnalysisReport(results, format);
    } catch (error) {
      toastStore.showKey('tools.analyzer.failed', { error: error instanceof Error ? error.message : String(error) }, 'error');
    }
  }

  async function handleSelectFiles() {
    const paths = await OpenFLACFilesDialog();
    if (paths && paths.length > 0) {
      await analyzeFiles(paths);
    }
  }

  async function handleSelectFolder() {
    const paths = await SelectFolderForAnalysis();
    if (paths && paths.length > 0) {
      await analyzeFiles(paths);
    }
  }

  function verdictClass(verdict: string): string {
    switch (verdict) {
      case 'lossless': return 'v-lossless';
      case 'likely_upscaled': return 'v-likely';
      case 'upscaled': return 'v-upscaled';
      default: return 'v-other';
    }
  }

  function verdictLabel(verdict: string): string {
    switch (verdict) {
      case 'lossless': return $t('tools.analyzer.lossless');
      case 'likely_upscaled': return $t('tools.analyzer.likelyUpscaled');
      case 'upscaled': return $t('tools.analyzer.upscaled');
      default: return verdict;
    }
  }

  function reset() {
    files = [];
    results = [];
  }

  let unsubscribeFileDrop: () => void;

  onMount(() => {
    // Browser mode: no-op (see lib/runtime.ts), drag-and-drop needs the desktop app.
    unsubscribeFileDrop = onNativeFileDrop((_x: number, _y: number, paths: string[]) => {
      const audioPaths = paths.filter((p: string) =>
        /\.(flac|mp3|m4a|aac)$/i.test(p)
      );
      if (audioPaths.length > 0) {
        analyzeFiles(audioPaths);
      }
    }, false);
  });

  onDestroy(() => {
    unsubscribeFileDrop?.();
  });
</script>

<div class="analyzer-page">
  <div class="page-header">
    <div class="header-title">
      <FileSearch size={28} strokeWidth={1.5} />
      <h1>{$t('tools.analyzer.title')}</h1>
    </div>
    {#if results.length > 0 && !isAnalyzing}
      <button class="btn-reset" onclick={reset}>{$t('tools.analyzer.analyzeMore')}</button>
    {/if}
  </div>

  {#if isAnalyzing || results.length > 0}
    {#if isAnalyzing}
      <div class="analyzing-state">
        <p>{$t('tools.analyzer.analyzing', { count: files.length })}</p>
        <progress class="analyze-progress" value={done} max={total} aria-valuenow={done} aria-valuemin="0" aria-valuemax={total} aria-label={$t('tools.analyzer.title')}></progress>
        <p class="progress-text" aria-live="polite">{$t('tools.analyzer.progress', { done, total })}</p>
        <button class="btn-cancel-analysis" onclick={cancelAnalysis} disabled={cancelRequested}>{$t('tools.analyzer.cancel')}</button>
      </div>
    {:else if cancelled}
      <p class="cancelled-note" role="status">{$t('tools.analyzer.cancelled', { done: results.length, total: files.length })}</p>
    {/if}

    <div class="export-bar">
      <button class="btn-export btn-export-csv" onclick={() => exportReport('csv')} disabled={results.length === 0}>{$t('tools.analyzer.export.csv')}</button>
      <button class="btn-export btn-export-json" onclick={() => exportReport('json')} disabled={results.length === 0}>{$t('tools.analyzer.export.json')}</button>
    </div>

    {#if okRows.length > 0}
    <div class="results-table">
      <div class="table-header">
        <span class="th file-col">{$t('tools.analyzer.file')}</span>
        <span class="th verdict-col">{$t('tools.analyzer.verdict')}</span>
        <span class="th confidence-col">{$t('tools.analyzer.confidence')}</span>
        <span class="th rate-col">{$t('tools.analyzer.sampleRate')}</span>
        <span class="th depth-col">{$t('tools.analyzer.bitDepth')}</span>
        <span class="th bpm-col">BPM</span>
        <span class="th key-col">{$t('tools.analyzer.key')}</span>
      </div>
      <div class="table-body">
        {#each okRows as result (result.filePath)}
          <div class="table-row">
            <span class="cell file-col" title={result.filePath}>{result.fileName}</span>
            <div class="cell verdict-col">
              <span class="verdict-badge {verdictClass(result.verdict)}">
                {#if result.verdict === 'lossless'}
                  <CheckCircle size={16} />
                {:else if result.verdict === 'likely_upscaled'}
                  <AlertTriangle size={16} />
                {:else}
                  <XCircle size={16} />
                {/if}
                {result.verdictLabel || verdictLabel(result.verdict)}
              </span>
            </div>
            <span class="cell confidence-col">{Math.round(result.confidence)}%</span>
            <span class="cell rate-col mono">{(result.sampleRate / 1000).toLocaleString($locale, { minimumFractionDigits: 1, maximumFractionDigits: 1 })} kHz</span>
            <span class="cell depth-col mono">{result.bitDepth}-bit</span>
            <span class="cell bpm-col mono">{(result.bpm ?? 0) > 0 ? Math.round(result.bpm ?? 0) : '—'}</span>
            <span class="cell key-col mono">{result.musicalKey || '—'}</span>
          </div>
        {/each}
      </div>
    </div>
    {/if}
    {#if errorRows.length > 0}
      <section class="errors-panel" aria-labelledby="analyzer-errors-title">
        <h2 id="analyzer-errors-title">{$t('tools.analyzer.errors.title')} ({errorRows.length})</h2>
        <ul>
          {#each errorRows as row (row.filePath)}
            <li class="error-row">
              <span class="error-file" title={row.filePath}>{row.fileName}</span>
              <span class="error-details">{row.details}</span>
            </li>
          {/each}
        </ul>
      </section>
    {/if}
  {:else}
    <DropZone
      supportedFormats="FLAC, MP3, M4A, AAC"
      onFilesSelected={handleSelectFiles}
      onFolderSelected={handleSelectFolder}
    />
  {/if}
</div>

<style>
  .analyzer-page {
    padding: 32px;
    max-width: 1000px;
  }

  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 32px;
  }

  .header-title {
    display: flex;
    align-items: center;
    gap: 12px;
    color: var(--color-text-primary);
  }

  .header-title h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0;
  }

  .btn-reset {
    padding: 10px 20px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border-subtle);
    color: var(--color-text-secondary);
    transition: all 0.2s;
  }

  .btn-reset:hover {
    background: var(--color-bg-tertiary);
    color: var(--color-text-primary);
  }

  .analyzing-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 80px 20px;
    color: var(--color-text-muted);
  }

  .analyzing-state p {
    margin: 0;
    font-size: 16px;
  }

  .analyze-progress {
    width: 100%;
    max-width: 480px;
    height: 10px;
  }

  .progress-text {
    margin: 0;
    color: var(--color-text-secondary);
  }

  .btn-cancel-analysis,
  .btn-export {
    padding: 8px 16px;
    border-radius: 8px;
    border: 1px solid var(--color-border-subtle);
    background: var(--color-bg-secondary);
    color: var(--color-text-primary);
    font-size: 14px;
    cursor: pointer;
  }

  .btn-cancel-analysis:disabled,
  .btn-export:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .export-bar {
    display: flex;
    gap: 8px;
    margin-bottom: 16px;
  }

  .cancelled-note {
    margin: 0 0 16px;
    color: var(--color-text-secondary);
  }

  .errors-panel {
    margin-top: 24px;
    padding: 16px;
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
  }

  .errors-panel h2 {
    margin: 0 0 12px;
    font-size: 16px;
  }

  .errors-panel ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  .error-row {
    display: flex;
    gap: 12px;
    padding: 6px 0;
    font-size: 13px;
  }

  .error-file {
    font-weight: 600;
  }

  .error-details {
    color: var(--color-text-secondary);
  }

  .results-table {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    overflow: hidden;
  }

  .table-header {
    display: grid;
    grid-template-columns: 1fr 160px 100px 110px 90px 80px 70px;
    gap: 16px;
    padding: 12px 16px;
    background: var(--color-bg-primary);
    border-bottom: 1px solid var(--color-border);
  }

  .th {
    font-size: 12px;
    font-weight: 600;
    color: var(--color-text-tertiary);
    text-transform: uppercase;
  }

  .table-body {
    max-height: calc(100vh - 300px);
    overflow-y: auto;
  }

  .table-row {
    display: grid;
    grid-template-columns: 1fr 160px 100px 110px 90px 80px 70px;
    gap: 16px;
    padding: 12px 16px;
    align-items: center;
    border-bottom: 1px solid var(--color-border);
    transition: background 0.2s;
  }

  .table-row:last-child {
    border-bottom: none;
  }

  .table-row:hover {
    background: rgba(255, 255, 255, 0.02);
  }

  .cell {
    font-size: 14px;
    color: var(--color-text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .cell.mono {
    font-family: monospace;
    font-size: 13px;
    font-variant-numeric: tabular-nums;
  }

  .v-lossless { color: #22c55e; }
  .v-likely { color: var(--color-highlight-gold, #eab308); }
  .v-upscaled { color: #ef4444; }
  .v-other { color: var(--color-text-secondary); }
  :global([data-theme="light"]) .v-lossless { color: #15803d; }
  :global([data-theme="light"]) .v-likely { color: #a16207; }
  :global([data-theme="light"]) .v-upscaled { color: #b91c1c; }

  .verdict-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-weight: 500;
    font-size: 13px;
  }
</style>
