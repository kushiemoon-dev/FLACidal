<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { onNativeFileDrop } from '../../lib/runtime';
  import { ConvertFiles, OpenFLACFilesDialog, SelectDownloadFolder } from '../../lib/api';
  import DropZone from '../../components/DropZone.svelte';
  import { SlidersHorizontal, CheckCircle, XCircle } from 'lucide-svelte';
  import { t, locale } from '../../lib/i18n';
  import { formatNumber } from '../../lib/format';

  let files: string[] = $state([]);
  let results: any[] = $state([]);
  let isResampling = $state(false);

  let sampleRate = $state('44100');
  let bitDepth = $state('16');
  let outputDir = $state('');

  async function handleSelectFiles() {
    const paths = await OpenFLACFilesDialog();
    if (paths && paths.length > 0) {
      files = paths;
    }
  }

  async function handleSelectFolder() {
    const folder = await SelectDownloadFolder();
    if (folder) {
      // Fall back to file dialog for individual paths
      await handleSelectFiles();
    }
  }

  async function selectOutputDir() {
    const folder = await SelectDownloadFolder();
    if (folder) {
      outputDir = folder;
    }
  }

  async function resample() {
    if (files.length === 0 || !outputDir) return;
    isResampling = true;
    results = [];
    try {
      const quality = `${sampleRate}:${bitDepth}`;
      results = await ConvertFiles(files, 'flac', quality, outputDir, false);
    } catch (error) {
      console.error('Resample error:', error);
    } finally {
      isResampling = false;
    }
  }

  function reset() {
    files = [];
    results = [];
  }

  function basename(path: string): string {
    return path.split('/').pop() || path.split('\\').pop() || path;
  }

  let unsubscribeFileDrop: () => void;

  onMount(() => {
    // Browser mode: no-op (see lib/runtime.ts), drag-and-drop needs the desktop app.
    unsubscribeFileDrop = onNativeFileDrop((_x: number, _y: number, paths: string[]) => {
      const flacPaths = paths.filter((p: string) => p.toLowerCase().endsWith('.flac'));
      if (flacPaths.length > 0) {
        files = flacPaths;
        results = [];
      }
    }, false);
  });

  onDestroy(() => {
    unsubscribeFileDrop?.();
  });
</script>

<div class="resampler-page">
  <div class="page-header">
    <div class="header-title">
      <SlidersHorizontal size={28} strokeWidth={1.5} />
      <h1>{$t('tools.resampler.title')}</h1>
    </div>
    {#if files.length > 0}
      <button class="btn-reset" onclick={reset}>{$t('tools.resampler.startOver')}</button>
    {/if}
  </div>

  {#if isResampling}
    <div class="resampling-state">
      <div class="loader"></div>
      <p>{$t('tools.resampler.resampling', { count: files.length })}</p>
    </div>
  {:else if results.length > 0}
    <div class="results-section">
      <h2>{$t('tools.resampler.results')}</h2>
      <div class="results-list">
        {#each results as result}
          <div class="result-row" class:success={result.success} class:error={!result.success}>
            {#if result.success}
              <CheckCircle size={16} color="#22c55e" />
            {:else}
              <XCircle size={16} color="#ef4444" />
            {/if}
            <span class="result-file">{basename(result.sourcePath)}</span>
            {#if result.error}
              <span class="result-error">{result.error}</span>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {:else if files.length > 0}
    <div class="options-section">
      <div class="file-summary">
        <p>{$t('tools.resampler.filesSelected', { count: files.length })}</p>
      </div>

      <div class="options-grid">
        <div class="option">
          <label for="sample-rate">{$t('tools.resampler.sampleRate')}</label>
          <select id="sample-rate" bind:value={sampleRate}>
            <option value="44100">{formatNumber(44100, $locale)} Hz</option>
            <option value="48000">{formatNumber(48000, $locale)} Hz</option>
            <option value="88200">{formatNumber(88200, $locale)} Hz</option>
            <option value="96000">{formatNumber(96000, $locale)} Hz</option>
            <option value="192000">{formatNumber(192000, $locale)} Hz</option>
          </select>
        </div>

        <div class="option">
          <label for="bit-depth">{$t('tools.resampler.bitDepth')}</label>
          <select id="bit-depth" bind:value={bitDepth}>
            <option value="16">16-bit</option>
            <option value="24">24-bit</option>
          </select>
        </div>

        <div class="option output-option">
          <span class="option-label">{$t('tools.resampler.outputFolder')}</span>
          <div class="output-row">
            <span class="output-path">{outputDir || $t('tools.resampler.notSelected')}</span>
            <button class="btn-browse" onclick={selectOutputDir}>{$t('tools.resampler.browse')}</button>
          </div>
        </div>
      </div>

      <button
        class="btn-resample"
        onclick={resample}
        disabled={!outputDir}
      >
        {$t('tools.resampler.resample')}
      </button>
    </div>
  {:else}
    <DropZone
      supportedFormats="FLAC"
      onFilesSelected={handleSelectFiles}
      onFolderSelected={handleSelectFolder}
    />
  {/if}
</div>

<style>
  .resampler-page {
    padding: 32px;
    max-width: 800px;
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

  .resampling-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 80px 20px;
    color: var(--color-text-muted);
  }

  .resampling-state p {
    margin: 0;
    font-size: 16px;
  }

  .loader {
    width: 40px;
    height: 40px;
    border: 3px solid var(--color-border-subtle);
    border-top-color: var(--color-accent);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin-bottom: 16px;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .options-section {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 24px;
  }

  .file-summary {
    margin-bottom: 24px;
  }

  .file-summary p {
    margin: 0;
    font-size: 15px;
    color: var(--color-text-secondary);
    font-weight: 500;
  }

  .options-grid {
    display: flex;
    flex-direction: column;
    gap: 20px;
    margin-bottom: 24px;
  }

  .option label,
  .option .option-label {
    display: block;
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-tertiary);
    text-transform: uppercase;
    margin-bottom: 8px;
  }

  .option select {
    width: 100%;
    max-width: 280px;
    padding: 10px 12px;
    background: var(--color-bg-primary);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    color: var(--color-text-primary);
    font-size: 14px;
    cursor: pointer;
  }

  .output-option {
    width: 100%;
  }

  .output-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .output-path {
    flex: 1;
    padding: 10px 12px;
    background: var(--color-bg-primary);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    color: var(--color-text-secondary);
    font-size: 14px;
    font-family: monospace;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .btn-browse {
    padding: 10px 16px;
    border-radius: 8px;
    font-size: 14px;
    cursor: pointer;
    background: var(--color-bg-primary);
    border: 1px solid var(--color-border);
    color: var(--color-text-secondary);
    transition: all 0.2s;
    white-space: nowrap;
  }

  .btn-browse:hover {
    border-color: var(--color-text-tertiary);
    color: var(--color-text-primary);
  }

  .btn-resample {
    padding: 12px 28px;
    border-radius: 8px;
    font-size: 15px;
    font-weight: 600;
    cursor: pointer;
    background: var(--color-accent);
    border: 1px solid var(--color-accent);
    color: #000;
    transition: all 0.2s;
  }

  .btn-resample:hover:not(:disabled) {
    opacity: 0.9;
  }

  .btn-resample:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .results-section h2 {
    font-size: 20px;
    font-weight: 600;
    margin: 0 0 16px 0;
    color: var(--color-text-primary);
  }

  .results-list {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    overflow: hidden;
  }

  .result-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 16px;
    border-bottom: 1px solid var(--color-border);
    font-size: 14px;
  }

  .result-row:last-child {
    border-bottom: none;
  }

  .result-file {
    color: var(--color-text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .result-error {
    color: #ef4444;
    font-size: 13px;
    margin-left: auto;
  }
</style>
