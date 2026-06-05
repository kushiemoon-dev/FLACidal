<script lang="ts">
  import { FileSearch, CheckCircle, AlertTriangle, XCircle, Upload } from 'lucide-svelte';

  // --- state ------------------------------------------------------------------
  let selectedFile: File | null = $state(null);
  let isDragging = $state(false);
  let isAnalyzing = $state(false);
  let result: AnalyzeResult | null = $state(null);
  let errorMsg: string | null = $state(null);

  // --- types ------------------------------------------------------------------
  interface AnalyzeResult {
    isUpscaled: boolean;
    realBitrate: number;
    spectralCutoff: number;
    format: string;
    message: string;
    confidence: number;
    verdict: string;
    verdictLabel: string;
    fileName: string;
    sampleRate: number;
    bitDepth: number;
  }

  // --- helpers ----------------------------------------------------------------
  function reset() {
    selectedFile = null;
    result = null;
    errorMsg = null;
  }

  function onFileInput(e: Event) {
    const input = e.target as HTMLInputElement;
    if (input.files && input.files[0]) {
      selectedFile = input.files[0];
      result = null;
      errorMsg = null;
    }
  }

  function onDragOver(e: DragEvent) {
    e.preventDefault();
    isDragging = true;
  }

  function onDragLeave() {
    isDragging = false;
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    isDragging = false;
    const files = e.dataTransfer?.files;
    if (files && files[0]) {
      selectedFile = files[0];
      result = null;
      errorMsg = null;
    }
  }

  async function analyze() {
    if (!selectedFile) return;
    isAnalyzing = true;
    result = null;
    errorMsg = null;

    try {
      const form = new FormData();
      form.append('file', selectedFile);

      const res = await fetch('/api/analyze', { method: 'POST', body: form });
      const data = await res.json();

      if (!res.ok) {
        errorMsg = data.error ?? 'Erreur inconnue';
        return;
      }

      result = data as AnalyzeResult;
    } catch (err) {
      errorMsg = err instanceof Error ? err.message : 'Network error';
    } finally {
      isAnalyzing = false;
    }
  }

  // --- derived display --------------------------------------------------------
  function statusColor(r: AnalyzeResult): string {
    if (r.verdict === 'lossless') return 'var(--color-success)';
    if (r.verdict === 'upscaled') return 'var(--color-error)';
    return 'var(--color-warning)';
  }

  function statusBg(r: AnalyzeResult): string {
    if (r.verdict === 'lossless') return 'rgba(16, 185, 129, 0.08)';
    if (r.verdict === 'upscaled') return 'rgba(239, 68, 68, 0.08)';
    return 'rgba(245, 158, 11, 0.08)';
  }
</script>

<div class="analyzer-page">
  <div class="page-header">
    <div class="header-title">
      <FileSearch size={28} strokeWidth={1.5} />
      <h1>Analyzer FLAC</h1>
    </div>
    {#if result || selectedFile}
      <button class="btn-secondary" onclick={reset}>Nouveau fichier</button>
    {/if}
  </div>

  <!-- Drop zone -->
  {#if !result}
    <div
      class="drop-zone"
      class:dragging={isDragging}
      class:has-file={selectedFile !== null}
      role="button"
      tabindex="0"
      aria-label="FLAC file drop zone"
      ondragover={onDragOver}
      ondragleave={onDragLeave}
      ondrop={onDrop}
      onkeydown={(e) => e.key === 'Enter' && document.getElementById('file-input')?.click()}
    >
      <Upload size={36} strokeWidth={1.2} class="drop-icon" />
      {#if selectedFile}
        <p class="drop-filename">{selectedFile.name}</p>
        <p class="drop-hint">{(selectedFile.size / (1024 * 1024)).toFixed(1)} MB</p>
      {:else}
        <p class="drop-label">Glissez un fichier FLAC ici</p>
        <p class="drop-hint">ou</p>
      {/if}

      <label class="btn-pick" for="file-input">
        {selectedFile ? 'Changer de fichier' : 'Parcourir…'}
      </label>
      <input
        id="file-input"
        type="file"
        accept=".flac,audio/flac"
        onchange={onFileInput}
        style="display:none"
      />
    </div>

    {#if selectedFile}
      <div class="analyze-row">
        <button class="btn-primary" onclick={analyze} disabled={isAnalyzing}>
          {#if isAnalyzing}
            <span class="spinner"></span>
            Analyse en cours…
          {:else}
            Analyser
          {/if}
        </button>
      </div>
    {/if}

    {#if errorMsg}
      <div class="error-banner">
        <XCircle size={16} />
        {errorMsg}
      </div>
    {/if}
  {/if}

  <!-- Result cards -->
  {#if result}
    <div class="results">
      <!-- Main verdict card -->
      <div class="card verdict-card" style="border-color:{statusColor(result)};background:{statusBg(result)}">
        <div class="verdict-icon">
          {#if result.verdict === 'lossless'}
            <CheckCircle size={40} color="var(--color-success)" strokeWidth={1.5} />
          {:else if result.verdict === 'upscaled'}
            <XCircle size={40} color="var(--color-error)" strokeWidth={1.5} />
          {:else}
            <AlertTriangle size={40} color="var(--color-warning)" strokeWidth={1.5} />
          {/if}
        </div>
        <div class="verdict-text">
          <h2 style="color:{statusColor(result)}">{result.verdictLabel ?? result.verdict}</h2>
          <p class="verdict-msg">{result.message}</p>
          <span class="confidence-badge">Confiance : {result.confidence}%</span>
        </div>
      </div>

      <!-- Detail cards -->
      <div class="detail-grid">
        <div class="card detail-card">
          <span class="detail-label">Fichier</span>
          <span class="detail-value mono">{result.fileName}</span>
        </div>
        <div class="card detail-card">
          <span class="detail-label">Format</span>
          <span class="detail-value">{result.format}</span>
        </div>
        <div class="card detail-card">
          <span class="detail-label">Sample Rate</span>
          <span class="detail-value mono">{(result.sampleRate / 1000).toFixed(1)} kHz</span>
        </div>
        <div class="card detail-card">
          <span class="detail-label">Bit Depth</span>
          <span class="detail-value mono">{result.bitDepth}-bit</span>
        </div>
        <div class="card detail-card">
          <span class="detail-label">Spectral Cutoff</span>
          <span class="detail-value mono">{result.spectralCutoff.toLocaleString()} Hz</span>
        </div>
        {#if result.realBitrate > 0}
          <div class="card detail-card">
            <span class="detail-label">Estimated Real Bitrate</span>
            <span class="detail-value mono">{result.realBitrate} kbps</span>
          </div>
        {/if}
      </div>

      <button class="btn-secondary" onclick={reset}>Analyser un autre fichier</button>
    </div>
  {/if}
</div>

<style>
  .analyzer-page {
    padding: 32px;
    max-width: 720px;
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

  /* Drop zone */
  .drop-zone {
    border: 2px dashed var(--color-border);
    border-radius: 16px;
    padding: 48px 32px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    transition: border-color 0.2s, background 0.2s;
    color: var(--color-text-secondary);
    background: var(--color-bg-secondary);
  }

  .drop-zone.dragging {
    border-color: var(--color-accent);
    background: var(--color-accent-subtle);
  }

  .drop-zone.has-file {
    border-color: var(--color-success);
    border-style: solid;
  }

  .drop-label {
    margin: 0;
    font-size: 16px;
    font-weight: 500;
    color: var(--color-text-primary);
  }

  .drop-hint {
    margin: 0;
    font-size: 13px;
    color: var(--color-text-tertiary);
  }

  .drop-filename {
    margin: 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text-primary);
    word-break: break-all;
    text-align: center;
  }

  .btn-pick {
    margin-top: 8px;
    padding: 8px 18px;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    background: var(--color-bg-tertiary);
    border: 1px solid var(--color-border-subtle);
    color: var(--color-text-secondary);
    transition: background 0.2s, color 0.2s;
  }

  .btn-pick:hover {
    background: var(--color-bg-elevated);
    color: var(--color-text-primary);
  }

  /* Analyze row */
  .analyze-row {
    display: flex;
    justify-content: center;
    margin-top: 20px;
  }

  /* Buttons */
  .btn-primary {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    padding: 12px 32px;
    border-radius: 10px;
    font-size: 15px;
    font-weight: 600;
    cursor: pointer;
    background: var(--color-accent);
    border: none;
    color: #fff;
    transition: background 0.2s, opacity 0.2s;
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--color-accent-hover);
  }

  .btn-primary:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .btn-secondary {
    padding: 10px 20px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border-subtle);
    color: var(--color-text-secondary);
    transition: background 0.2s, color 0.2s;
  }

  .btn-secondary:hover {
    background: var(--color-bg-tertiary);
    color: var(--color-text-primary);
  }

  /* Spinner */
  .spinner {
    width: 16px;
    height: 16px;
    border: 2px solid rgba(255, 255, 255, 0.4);
    border-top-color: #fff;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
    flex-shrink: 0;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* Error banner */
  .error-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 16px;
    padding: 12px 16px;
    border-radius: 10px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.3);
    color: var(--color-error);
    font-size: 14px;
  }

  /* Results */
  .results {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .card {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 20px;
  }

  /* Verdict card */
  .verdict-card {
    display: flex;
    align-items: center;
    gap: 20px;
    border-width: 1.5px;
  }

  .verdict-icon {
    flex-shrink: 0;
  }

  .verdict-text {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .verdict-text h2 {
    margin: 0;
    font-size: 22px;
    font-weight: 700;
  }

  .verdict-msg {
    margin: 0;
    font-size: 14px;
    color: var(--color-text-secondary);
  }

  .confidence-badge {
    display: inline-block;
    margin-top: 6px;
    font-size: 12px;
    font-weight: 600;
    padding: 3px 10px;
    border-radius: 20px;
    background: var(--color-bg-tertiary);
    color: var(--color-text-tertiary);
  }

  /* Detail grid */
  .detail-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 12px;
  }

  .detail-card {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 14px 16px;
  }

  .detail-label {
    font-size: 11px;
    font-weight: 600;
    color: var(--color-text-tertiary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .detail-value {
    font-size: 15px;
    font-weight: 500;
    color: var(--color-text-primary);
  }

  .detail-value.mono {
    font-family: monospace;
    font-variant-numeric: tabular-nums;
  }
</style>
