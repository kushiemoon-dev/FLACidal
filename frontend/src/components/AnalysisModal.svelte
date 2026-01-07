<script lang="ts">
  import { onMount } from 'svelte';
  import { AnalyzeMultiple } from '../../wailsjs/go/main/App.js';

  export let files: string[];
  export let onClose: () => void;

  interface AnalysisResult {
    filePath: string;
    fileName: string;
    isTrueLossless: boolean;
    confidence: number;
    spectrumCutoff: number;
    expectedCutoff: number;
    verdict: string;
    verdictLabel: string;
    details: string;
    sampleRate: number;
    bitDepth: number;
  }

  let results: AnalysisResult[] = [];
  let isAnalyzing = true;
  let error = '';

  $: summary = {
    total: results.length,
    lossless: results.filter(r => r.verdict === 'lossless').length,
    likelyUpscaled: results.filter(r => r.verdict === 'likely_upscaled').length,
    upscaled: results.filter(r => r.verdict === 'upscaled').length,
    unknown: results.filter(r => r.verdict === 'unknown' || r.verdict === 'error').length
  };

  onMount(async () => {
    await analyzeFiles();
  });

  async function analyzeFiles() {
    isAnalyzing = true;
    error = '';

    try {
      results = await AnalyzeMultiple(files);
    } catch (e: any) {
      error = e.message || 'Analysis failed';
    } finally {
      isAnalyzing = false;
    }
  }

  function getVerdictColor(verdict: string): string {
    switch (verdict) {
      case 'lossless': return '#22c55e';
      case 'likely_upscaled': return '#f59e0b';
      case 'upscaled': return '#ef4444';
      default: return '#666';
    }
  }

  function getVerdictBg(verdict: string): string {
    switch (verdict) {
      case 'lossless': return 'rgba(34, 197, 94, 0.1)';
      case 'likely_upscaled': return 'rgba(245, 158, 11, 0.1)';
      case 'upscaled': return 'rgba(239, 68, 68, 0.1)';
      default: return 'rgba(102, 102, 102, 0.1)';
    }
  }

  function formatFrequency(hz: number): string {
    if (hz >= 1000) {
      return (hz / 1000).toFixed(1) + ' kHz';
    }
    return hz + ' Hz';
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget && !isAnalyzing) {
      onClose();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && !isAnalyzing) {
      onClose();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="modal-backdrop" on:click={handleBackdropClick} on:keydown={handleKeydown} role="dialog" aria-modal="true">
  <div class="modal-content">
    <div class="modal-header">
      <h2>Quality Analysis</h2>
      <span class="file-count">{files.length} file{files.length !== 1 ? 's' : ''}</span>
      <button class="close-btn" on:click={onClose} disabled={isAnalyzing}>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>

    {#if isAnalyzing}
      <div class="loading-state">
        <div class="loader"></div>
        <p>Analyzing audio files...</p>
        <span class="loading-hint">Checking spectrum and quality indicators</span>
      </div>
    {:else if error}
      <div class="error-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <h3>Analysis Failed</h3>
        <p>{error}</p>
      </div>
    {:else}
      <div class="modal-body">
        <!-- Summary -->
        <div class="summary-section">
          <div class="summary-item lossless">
            <span class="summary-count">{summary.lossless}</span>
            <span class="summary-label">Lossless</span>
          </div>
          <div class="summary-item warning">
            <span class="summary-count">{summary.likelyUpscaled}</span>
            <span class="summary-label">Likely Upscaled</span>
          </div>
          <div class="summary-item danger">
            <span class="summary-count">{summary.upscaled}</span>
            <span class="summary-label">Upscaled</span>
          </div>
          {#if summary.unknown > 0}
            <div class="summary-item unknown">
              <span class="summary-count">{summary.unknown}</span>
              <span class="summary-label">Unknown</span>
            </div>
          {/if}
        </div>

        <!-- Results List -->
        <div class="results-section">
          <span class="section-label">Results</span>
          <div class="results-list">
            {#each results as result}
              <div class="result-item">
                <div class="result-header">
                  <div class="result-icon">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M9 18V5l12-2v13"/>
                      <circle cx="6" cy="18" r="3"/>
                      <circle cx="18" cy="16" r="3"/>
                    </svg>
                  </div>
                  <span class="result-name">{result.fileName}</span>
                  <span
                    class="verdict-badge"
                    style="color: {getVerdictColor(result.verdict)}; background: {getVerdictBg(result.verdict)}"
                  >
                    {result.verdictLabel}
                  </span>
                </div>

                <div class="result-details">
                  <div class="detail-row">
                    <span class="detail-label">Confidence</span>
                    <div class="confidence-bar">
                      <div
                        class="confidence-fill"
                        style="width: {result.confidence}%; background: {getVerdictColor(result.verdict)}"
                      ></div>
                    </div>
                    <span class="confidence-value">{result.confidence}%</span>
                  </div>

                  <div class="detail-row">
                    <span class="detail-label">Spectrum</span>
                    <span class="detail-value">
                      {formatFrequency(result.spectrumCutoff)} / {formatFrequency(result.expectedCutoff)}
                    </span>
                  </div>

                  <div class="detail-row">
                    <span class="detail-label">Format</span>
                    <span class="detail-value">
                      {result.sampleRate / 1000} kHz / {result.bitDepth}-bit
                    </span>
                  </div>

                  {#if result.details}
                    <div class="detail-note">
                      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="10"/>
                        <line x1="12" y1="16" x2="12" y2="12"/>
                        <line x1="12" y1="8" x2="12.01" y2="8"/>
                      </svg>
                      {result.details}
                    </div>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        </div>

        <!-- Info -->
        <div class="info-banner">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="16" x2="12" y2="12"/>
            <line x1="12" y1="8" x2="12.01" y2="8"/>
          </svg>
          <span>Analysis detects frequency cutoffs to identify files transcoded from lossy sources (MP3, AAC, etc.)</span>
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn-primary" on:click={onClose}>Close</button>
      </div>
    {/if}
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    animation: fadeIn 0.2s ease;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .modal-content {
    background: #111;
    border: 1px solid #222;
    border-radius: 16px;
    width: 90%;
    max-width: 600px;
    max-height: 85vh;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    animation: slideIn 0.2s ease;
  }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: scale(0.95) translateY(10px);
    }
    to {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
  }

  .modal-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px 20px;
    border-bottom: 1px solid #222;
  }

  .modal-header h2 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
  }

  .file-count {
    font-size: 13px;
    color: #666;
    margin-left: auto;
    margin-right: 12px;
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    background: transparent;
    border: none;
    border-radius: 8px;
    color: #666;
    cursor: pointer;
    transition: all 0.2s;
  }

  .close-btn:hover:not(:disabled) {
    background: #222;
    color: #fff;
  }

  .close-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .modal-body {
    padding: 20px;
    overflow-y: auto;
    flex: 1;
  }

  .loading-state,
  .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: #555;
    text-align: center;
  }

  .loader {
    width: 40px;
    height: 40px;
    border: 3px solid #222;
    border-top-color: #f472b6;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin-bottom: 16px;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .loading-state p {
    margin: 0 0 4px 0;
    color: #888;
  }

  .loading-hint {
    font-size: 12px;
    color: #555;
  }

  .error-state svg {
    color: #ef4444;
    margin-bottom: 16px;
  }

  .error-state h3 {
    margin: 0 0 8px 0;
    color: #ef4444;
    font-size: 16px;
  }

  .error-state p {
    margin: 0;
    font-size: 14px;
    color: #888;
  }

  .summary-section {
    display: flex;
    gap: 12px;
    margin-bottom: 20px;
  }

  .summary-item {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 16px 12px;
    background: #0a0a0a;
    border: 1px solid #1a1a1a;
    border-radius: 12px;
  }

  .summary-item.lossless {
    border-color: rgba(34, 197, 94, 0.2);
  }

  .summary-item.warning {
    border-color: rgba(245, 158, 11, 0.2);
  }

  .summary-item.danger {
    border-color: rgba(239, 68, 68, 0.2);
  }

  .summary-item.unknown {
    border-color: rgba(102, 102, 102, 0.2);
  }

  .summary-count {
    font-size: 24px;
    font-weight: 600;
  }

  .summary-item.lossless .summary-count { color: #22c55e; }
  .summary-item.warning .summary-count { color: #f59e0b; }
  .summary-item.danger .summary-count { color: #ef4444; }
  .summary-item.unknown .summary-count { color: #666; }

  .summary-label {
    font-size: 11px;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .section-label {
    display: block;
    font-size: 12px;
    font-weight: 600;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 12px;
  }

  .results-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
    max-height: 300px;
    overflow-y: auto;
    margin-bottom: 16px;
  }

  .result-item {
    padding: 14px;
    background: #0a0a0a;
    border: 1px solid #1a1a1a;
    border-radius: 12px;
  }

  .result-header {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 12px;
  }

  .result-icon {
    color: #666;
  }

  .result-name {
    flex: 1;
    font-size: 13px;
    color: #ccc;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .verdict-badge {
    padding: 4px 10px;
    border-radius: 12px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .result-details {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-left: 26px;
  }

  .detail-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .detail-label {
    width: 80px;
    font-size: 12px;
    color: #555;
  }

  .detail-value {
    font-size: 12px;
    color: #888;
    font-family: 'JetBrains Mono', monospace;
  }

  .confidence-bar {
    flex: 1;
    height: 4px;
    background: #222;
    border-radius: 2px;
    overflow: hidden;
    max-width: 100px;
  }

  .confidence-fill {
    height: 100%;
    border-radius: 2px;
    transition: width 0.3s ease;
  }

  .confidence-value {
    width: 40px;
    font-size: 12px;
    color: #888;
    font-family: 'JetBrains Mono', monospace;
    text-align: right;
  }

  .detail-note {
    display: flex;
    align-items: flex-start;
    gap: 8px;
    padding: 8px 10px;
    background: #111;
    border-radius: 6px;
    font-size: 11px;
    color: #666;
    margin-top: 4px;
  }

  .detail-note svg {
    flex-shrink: 0;
    margin-top: 1px;
  }

  .info-banner {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 12px 14px;
    background: rgba(59, 130, 246, 0.1);
    border: 1px solid rgba(59, 130, 246, 0.2);
    border-radius: 8px;
    font-size: 12px;
    color: #60a5fa;
  }

  .info-banner svg {
    flex-shrink: 0;
    margin-top: 1px;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    padding: 16px 20px;
    border-top: 1px solid #222;
  }

  .btn-primary {
    padding: 10px 24px;
    background: #f472b6;
    border: none;
    border-radius: 8px;
    color: #000;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-primary:hover {
    background: #ec4899;
  }
</style>
