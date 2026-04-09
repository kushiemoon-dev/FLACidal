<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { OnFileDrop, OnFileDropOff } from '../../../wailsjs/runtime/runtime.js';
  import { ConvertFiles, OpenFLACFilesDialog, SelectDownloadFolder, GetDownloadFolder } from '../../../wailsjs/go/main/App.js';
  import DropZone from '../../components/DropZone.svelte';
  import { FileAudio, FolderOpen, X, CheckCircle, AlertCircle, Loader } from 'lucide-svelte';

  let files: string[] = $state([]);
  let outputFormat = $state('MP3');
  let quality = $state('320k');
  let outputDir = $state('');
  let converting = $state(false);
  let results: { file: string; success: boolean; error?: string }[] = $state([]);

  const formatOptions = ['MP3', 'AAC', 'OGG', 'Opus', 'ALAC', 'WAV'];

  const qualityOptions: Record<string, string[]> = {
    MP3: ['320k', '256k', '192k', '128k', 'V0', 'V2'],
    AAC: ['256k', '192k', '128k'],
    OGG: ['320k', '256k', '192k', '128k'],
    Opus: ['256k', '192k', '128k', '96k', '64k'],
    ALAC: [],
    WAV: [],
  };

  $effect(() => {
    const opts = qualityOptions[outputFormat];
    if (opts && opts.length > 0) {
      quality = opts[0];
    } else {
      quality = '';
    }
  });

  onMount(async () => {
    try {
      const folder = await GetDownloadFolder();
      if (folder) outputDir = folder;
    } catch {}

    OnFileDrop((_x: number, _y: number, paths: string[]) => {
      const audioFiles = paths.filter(p =>
        /\.(flac|mp3|wav|ogg|opus|aac|m4a|alac|wma|aiff)$/i.test(p)
      );
      if (audioFiles.length > 0) {
        files = [...files, ...audioFiles];
        results = [];
      }
    }, true);
  });

  onDestroy(() => {
    OnFileDropOff();
  });

  async function selectFiles() {
    try {
      const selected = await OpenFLACFilesDialog();
      if (selected && selected.length > 0) {
        files = [...files, ...selected];
        results = [];
      }
    } catch {}
  }

  async function selectOutputFolder() {
    try {
      const folder = await SelectDownloadFolder();
      if (folder) outputDir = folder;
    } catch {}
  }

  function removeFile(index: number) {
    files = files.filter((_, i) => i !== index);
  }

  function clearFiles() {
    files = [];
    results = [];
  }

  function getFileName(path: string): string {
    return path.split('/').pop()?.split('\\').pop() || path;
  }

  async function convert() {
    if (files.length === 0 || !outputDir) return;
    converting = true;
    results = [];

    try {
      const res = await ConvertFiles(files, outputFormat.toLowerCase(), quality, outputDir, false);
      if (Array.isArray(res)) {
        results = res.map((r: any, i: number) => ({
          file: files[i] || '',
          success: r.success ?? !r.error,
          error: r.error,
        }));
      } else {
        results = files.map(f => ({ file: f, success: true }));
      }
    } catch (err: any) {
      results = files.map(f => ({ file: f, success: false, error: err?.message || 'Conversion failed' }));
    } finally {
      converting = false;
    }
  }
</script>

<div class="page">
  <header class="page-header">
    <h1>Audio Converter</h1>
    <p class="page-subtitle">Convert audio files between formats</p>
  </header>

  {#if files.length === 0}
    <DropZone
      supportedFormats="FLAC, MP3, WAV, OGG, Opus, AAC, ALAC"
      onFilesSelected={selectFiles}
    />
  {:else}
    <div class="file-list-section">
      <div class="file-list-header">
        <span class="file-count">{files.length} file{files.length !== 1 ? 's' : ''} selected</span>
        <div class="file-list-actions">
          <button class="btn btn-outline btn-sm" onclick={selectFiles}>Add More</button>
          <button class="btn btn-outline btn-sm" onclick={clearFiles}>Clear All</button>
        </div>
      </div>

      <div class="file-list">
        {#each files as file, i (file + i)}
          <div class="file-item">
            <FileAudio size={16} strokeWidth={1.5} />
            <span class="file-name">{getFileName(file)}</span>
            <button class="btn-icon" onclick={() => removeFile(i)} aria-label="Remove file">
              <X size={14} />
            </button>
          </div>
        {/each}
      </div>
    </div>

    <div class="options-section">
      <h3 class="section-title">Conversion Options</h3>
      <div class="options-grid">
        <div class="option-group">
          <label class="option-label" for="format-select">Output Format</label>
          <select id="format-select" class="select" bind:value={outputFormat}>
            {#each formatOptions as fmt}
              <option value={fmt}>{fmt}</option>
            {/each}
          </select>
        </div>

        {#if qualityOptions[outputFormat]?.length > 0}
          <div class="option-group">
            <label class="option-label" for="quality-select">Quality</label>
            <select id="quality-select" class="select" bind:value={quality}>
              {#each qualityOptions[outputFormat] as q}
                <option value={q}>{q}</option>
              {/each}
            </select>
          </div>
        {/if}

        <div class="option-group option-wide">
          <label class="option-label">Output Folder</label>
          <div class="folder-row">
            <input type="text" class="input" value={outputDir} readonly placeholder="Select output folder..." />
            <button class="btn btn-accent" onclick={selectOutputFolder}>
              <FolderOpen size={16} />
              Browse
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="action-bar">
      <button
        class="btn btn-accent btn-lg"
        onclick={convert}
        disabled={converting || files.length === 0 || !outputDir}
      >
        {#if converting}
          <Loader size={16} class="spin" />
          Converting...
        {:else}
          Convert
        {/if}
      </button>
    </div>
  {/if}

  {#if results.length > 0}
    <div class="results-section">
      <h3 class="section-title">Results</h3>
      <div class="results-list">
        {#each results as result (result.file)}
          <div class="result-item" class:success={result.success} class:failure={!result.success}>
            {#if result.success}
              <CheckCircle size={16} />
            {:else}
              <AlertCircle size={16} />
            {/if}
            <span class="result-name">{getFileName(result.file)}</span>
            {#if result.error}
              <span class="result-error">{result.error}</span>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .page {
    padding: 32px 40px;
    max-width: 900px;
  }

  .page-header {
    margin-bottom: 32px;
  }

  .page-header h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0 0 6px;
    color: var(--color-text-primary);
  }

  .page-subtitle {
    margin: 0;
    font-size: 14px;
    color: var(--color-text-tertiary);
  }

  .file-list-section {
    margin-bottom: 24px;
  }

  .file-list-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .file-count {
    font-size: 14px;
    color: var(--color-text-secondary);
    font-weight: 500;
  }

  .file-list-actions {
    display: flex;
    gap: 8px;
  }

  .file-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 200px;
    overflow-y: auto;
    border: 1px solid var(--color-border);
    border-radius: 10px;
    padding: 8px;
    background: var(--color-bg-secondary);
  }

  .file-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 6px 8px;
    border-radius: 6px;
    color: var(--color-text-secondary);
    font-size: 13px;
  }

  .file-item:hover {
    background: var(--color-bg-hover);
  }

  .file-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .btn-icon {
    background: none;
    border: none;
    color: var(--color-text-tertiary);
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    display: flex;
    align-items: center;
  }

  .btn-icon:hover {
    color: var(--color-error);
    background: rgba(239, 68, 68, 0.1);
  }

  .options-section {
    margin-bottom: 24px;
  }

  .section-title {
    font-size: 15px;
    font-weight: 600;
    margin: 0 0 16px;
    color: var(--color-text-primary);
  }

  .options-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .option-wide {
    grid-column: 1 / -1;
  }

  .option-group {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .option-label {
    font-size: 13px;
    font-weight: 500;
    color: var(--color-text-secondary);
  }

  .select, .input {
    padding: 9px 12px;
    border-radius: 8px;
    border: 1px solid var(--color-border);
    background: var(--color-bg-secondary);
    color: var(--color-text-primary);
    font-family: var(--font-family);
    font-size: 14px;
  }

  .select:focus, .input:focus {
    outline: none;
    border-color: var(--color-accent);
  }

  .input {
    flex: 1;
    min-width: 0;
  }

  .folder-row {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    font-family: var(--font-family);
    cursor: pointer;
    transition: all 0.15s;
    border: 1px solid transparent;
  }

  .btn-sm {
    padding: 5px 12px;
    font-size: 13px;
  }

  .btn-lg {
    padding: 10px 28px;
    font-size: 15px;
  }

  .btn-accent {
    background: var(--color-accent);
    color: #000;
    border-color: var(--color-accent);
  }

  .btn-accent:hover:not(:disabled) {
    opacity: 0.9;
  }

  .btn-accent:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-outline {
    background: transparent;
    border: 1px solid var(--color-border);
    color: var(--color-text-secondary);
  }

  .btn-outline:hover {
    border-color: var(--color-text-tertiary);
    color: var(--color-text-primary);
  }

  .action-bar {
    display: flex;
    gap: 12px;
    margin-bottom: 24px;
  }

  .results-section {
    margin-top: 8px;
  }

  .results-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .result-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px;
    border-radius: 8px;
    font-size: 13px;
    background: var(--color-bg-secondary);
  }

  .result-item.success {
    color: var(--color-success);
  }

  .result-item.failure {
    color: var(--color-error);
  }

  .result-name {
    flex: 1;
    color: var(--color-text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .result-error {
    font-size: 12px;
    color: var(--color-error);
    opacity: 0.8;
  }

  :global(.spin) {
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    from { transform: rotate(0deg); }
    to { transform: rotate(360deg); }
  }
</style>
