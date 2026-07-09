<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { OnFileDrop, OnFileDropOff } from '../../../wailsjs/runtime/runtime.js';
  import { FetchAndEmbedLyricsMultiple, OpenFLACFilesDialog } from '../../../wailsjs/go/app/App.js';
  import DropZone from '../../components/DropZone.svelte';
  import { FileAudio, Music2, X, CheckCircle, AlertCircle, Loader } from 'lucide-svelte';
  import { toastStore } from '../../stores/toast';

  let files: string[] = $state([]);
  let fetching = $state(false);
  let results: { filePath: string; success: boolean; hasPlain?: boolean; hasSynced?: boolean; error?: string }[] = $state([]);

  onMount(() => {
    OnFileDrop((_x: number, _y: number, paths: string[]) => {
      const flacFiles = paths.filter(p => /\.flac$/i.test(p));
      if (flacFiles.length > 0) {
        files = [...files, ...flacFiles];
        results = [];
      }
    }, false);
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
    } catch (err: any) {
      toastStore.show(err?.message || 'Failed to select files', 'error');
    }
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

  async function fetchAndEmbed() {
    if (files.length === 0) return;
    fetching = true;
    results = [];

    try {
      const res = await FetchAndEmbedLyricsMultiple(files);
      results = Array.isArray(res) ? res.map((r: any) => ({
        filePath:  r.filePath  ?? '',
        success:   r.success   ?? false,
        hasPlain:  r.hasPlain  ?? false,
        hasSynced: r.hasSynced ?? false,
        error:     r.error,
      })) : [];
    } catch (err: any) {
      results = files.map(f => ({ filePath: f, success: false, error: err?.message || 'Failed' }));
    } finally {
      fetching = false;
    }
  }
</script>

<div class="page">
  <header class="page-header">
    <h1>Lyrics Manager</h1>
    <p class="page-subtitle">Fetch and embed lyrics into FLAC files via LRCLIB</p>
  </header>

  {#if files.length === 0}
    <DropZone
      supportedFormats="FLAC"
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

    <div class="action-bar">
      <button
        class="btn btn-accent btn-lg"
        onclick={fetchAndEmbed}
        disabled={fetching || files.length === 0}
      >
        {#if fetching}
          <Loader size={16} class="spin" />
          Fetching lyrics...
        {:else}
          <Music2 size={16} />
          Fetch & Embed Lyrics
        {/if}
      </button>
    </div>
  {/if}

  {#if results.length > 0}
    <div class="results-section">
      <h3 class="section-title">Results</h3>
      <div class="results-list">
        {#each results as result (result.filePath)}
          <div class="result-item" class:success={result.success} class:failure={!result.success}>
            {#if result.success}
              <CheckCircle size={16} />
            {:else}
              <AlertCircle size={16} />
            {/if}
            <span class="result-name">{getFileName(result.filePath)}</span>
            {#if result.success}
              <span class="result-meta">
                {result.hasSynced ? 'LRC + plain' : result.hasPlain ? 'plain' : ''}
              </span>
            {/if}
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
    max-height: 220px;
    overflow-y: auto;
    border: 1px solid var(--color-border);
    border-radius: 10px;
    padding: 8px;
    background: var(--color-bg-secondary);
  }

  .file-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    border-radius: 6px;
    font-size: 13px;
    color: var(--color-text-secondary);
  }

  .file-item:hover {
    background: var(--color-bg-tertiary);
  }

  .file-name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .action-bar {
    display: flex;
    justify-content: flex-start;
    margin-bottom: 24px;
  }

  .results-section {
    margin-top: 24px;
  }

  .section-title {
    font-size: 15px;
    font-weight: 600;
    color: var(--color-text-primary);
    margin: 0 0 12px;
  }

  .results-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .result-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    border-radius: 8px;
    font-size: 13px;
    border: 1px solid var(--color-border);
    background: var(--color-bg-secondary);
  }

  .result-item.success {
    color: var(--color-success, #4ade80);
  }

  .result-item.failure {
    color: var(--color-error, #f87171);
  }

  .result-name {
    flex: 1;
    color: var(--color-text-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .result-meta {
    font-size: 12px;
    color: var(--color-text-tertiary);
  }

  .result-error {
    font-size: 12px;
    color: var(--color-error, #f87171);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 300px;
  }
</style>
