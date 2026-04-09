<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { OnFileDrop, OnFileDropOff } from '../../../wailsjs/runtime/runtime.js';
  import { AnalyzeMultiple, OpenFLACFilesDialog, SelectDownloadFolder } from '../../../wailsjs/go/main/App.js';
  import DropZone from '../../components/DropZone.svelte';
  import { FileSearch, CheckCircle, AlertTriangle, XCircle } from 'lucide-svelte';

  let files: string[] = $state([]);
  let results: any[] = $state([]);
  let isAnalyzing = $state(false);

  async function analyzeFiles(paths: string[]) {
    if (paths.length === 0) return;
    files = paths;
    isAnalyzing = true;
    results = [];
    try {
      results = await AnalyzeMultiple(paths);
    } catch (error) {
      console.error('Analysis error:', error);
    } finally {
      isAnalyzing = false;
    }
  }

  async function handleSelectFiles() {
    const paths = await OpenFLACFilesDialog();
    if (paths && paths.length > 0) {
      await analyzeFiles(paths);
    }
  }

  async function handleSelectFolder() {
    const folder = await SelectDownloadFolder();
    if (folder) {
      // For now, fall back to file dialog since we need individual file paths
      await handleSelectFiles();
    }
  }

  function verdictColor(verdict: string): string {
    switch (verdict) {
      case 'lossless': return '#22c55e';
      case 'likely_upscaled': return 'var(--color-highlight-gold, #eab308)';
      case 'upscaled': return '#ef4444';
      default: return 'var(--color-text-secondary)';
    }
  }

  function verdictLabel(verdict: string): string {
    switch (verdict) {
      case 'lossless': return 'Lossless';
      case 'likely_upscaled': return 'Likely Upscaled';
      case 'upscaled': return 'Upscaled';
      default: return verdict;
    }
  }

  function reset() {
    files = [];
    results = [];
  }

  onMount(() => {
    OnFileDrop((_x: number, _y: number, paths: string[]) => {
      const audioPaths = paths.filter((p: string) =>
        /\.(flac|mp3|m4a|aac)$/i.test(p)
      );
      if (audioPaths.length > 0) {
        analyzeFiles(audioPaths);
      }
    }, false);
  });

  onDestroy(() => {
    OnFileDropOff();
  });
</script>

<div class="analyzer-page">
  <div class="page-header">
    <div class="header-title">
      <FileSearch size={28} strokeWidth={1.5} />
      <h1>Audio Quality Analyzer</h1>
    </div>
    {#if results.length > 0}
      <button class="btn-reset" onclick={reset}>Analyze More</button>
    {/if}
  </div>

  {#if isAnalyzing}
    <div class="analyzing-state">
      <div class="loader"></div>
      <p>Analyzing {files.length} file{files.length !== 1 ? 's' : ''}...</p>
    </div>
  {:else if results.length > 0}
    <div class="results-table">
      <div class="table-header">
        <span class="th file-col">File</span>
        <span class="th verdict-col">Verdict</span>
        <span class="th confidence-col">Confidence</span>
        <span class="th rate-col">Sample Rate</span>
        <span class="th depth-col">Bit Depth</span>
      </div>
      <div class="table-body">
        {#each results as result}
          <div class="table-row">
            <span class="cell file-col" title={result.filePath}>{result.fileName}</span>
            <div class="cell verdict-col">
              <span class="verdict-badge" style="color: {verdictColor(result.verdict)}">
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
            <span class="cell rate-col mono">{(result.sampleRate / 1000).toFixed(1)} kHz</span>
            <span class="cell depth-col mono">{result.bitDepth}-bit</span>
          </div>
        {/each}
      </div>
    </div>
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

  .results-table {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    overflow: hidden;
  }

  .table-header {
    display: grid;
    grid-template-columns: 1fr 160px 100px 110px 90px;
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
    grid-template-columns: 1fr 160px 100px 110px 90px;
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

  .verdict-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-weight: 500;
    font-size: 13px;
  }
</style>
