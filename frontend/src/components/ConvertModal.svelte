<script lang="ts">
  import { onMount } from 'svelte';
  import { IsConverterAvailable, GetConversionFormats, ConvertFiles, GetFFmpegInfo } from '../../wailsjs/go/main/App.js';

  let { files, onClose, onComplete }: { files: string[]; onClose: () => void; onComplete: () => void } = $props();

  interface ConversionFormat {
    id: string;
    name: string;
    extension: string;
    qualities: string[];
    description: string;
  }

  interface ConversionResult {
    sourcePath: string;
    outputPath: string;
    success: boolean;
    error?: string;
    outputSize?: number;
    sourceSize?: number;
  }

  let formats: ConversionFormat[] = $state([]);
  let selectedFormat = $state('mp3');
  let selectedQuality = $state('320k');
  let deleteSource = $state(false);
  let isLoading = $state(true);
  let isConverting = $state(false);
  let ffmpegAvailable = $state(false);
  let ffmpegVersion = $state('');
  let results: ConversionResult[] = $state([]);
  let showResults = $state(false);
  let error = $state('');

  let currentFormat = $derived(formats.find(f => f.id === selectedFormat));
  let qualities = $derived(currentFormat?.qualities || []);
  $effect(() => {
    if (currentFormat && !qualities.includes(selectedQuality)) {
      selectedQuality = qualities[0] || '';
    }
  });

  onMount(async () => {
    await checkFFmpeg();
  });

  async function checkFFmpeg() {
    isLoading = true;
    try {
      ffmpegAvailable = await IsConverterAvailable();
      if (ffmpegAvailable) {
        const info = await GetFFmpegInfo();
        ffmpegVersion = info.version || '';
        formats = await GetConversionFormats();
        if (formats.length > 0) {
          selectedFormat = formats[0].id;
          selectedQuality = formats[0].qualities[0];
        }
      }
    } catch (e: any) {
      error = e.message || 'Failed to check FFmpeg';
    } finally {
      isLoading = false;
    }
  }

  async function handleConvert() {
    if (isConverting || !ffmpegAvailable) return;

    isConverting = true;
    error = '';

    try {
      results = await ConvertFiles(files, selectedFormat, selectedQuality, '', deleteSource);
      showResults = true;
    } catch (e: any) {
      error = e.message || 'Conversion failed';
    } finally {
      isConverting = false;
    }
  }

  function handleDone() {
    onComplete();
    onClose();
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget && !isConverting) {
      onClose();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape' && !isConverting) {
      onClose();
    }
  }

  function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function getQualityLabel(quality: string): string {
    if (quality.startsWith('V')) return `VBR ${quality}`;
    if (quality.startsWith('q')) return `Quality ${quality.slice(1)}`;
    return quality;
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="modal-backdrop" onclick={handleBackdropClick} onkeydown={handleKeydown} role="dialog" aria-modal="true" tabindex="-1">
  <div class="modal-content">
    <div class="modal-header">
      <h2>Convert Files</h2>
      <span class="file-count">{files.length} file{files.length !== 1 ? 's' : ''}</span>
      <button class="close-btn" onclick={onClose} disabled={isConverting} aria-label="Close">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>

    {#if isLoading}
      <div class="loading-state">
        <div class="loader"></div>
        <p>Checking FFmpeg...</p>
      </div>
    {:else if !ffmpegAvailable}
      <div class="unavailable-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <h3>FFmpeg Not Found</h3>
        <p>Audio conversion requires FFmpeg to be installed on your system.</p>
        <div class="install-hint">
          <code>sudo pacman -S ffmpeg</code>
        </div>
      </div>
    {:else if showResults}
      <div class="modal-body">
        <div class="results-section">
          <div class="results-summary">
            {#if results.every(r => r.success)}
              <div class="success-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
                  <polyline points="22 4 12 14.01 9 11.01"/>
                </svg>
              </div>
              <p>All files converted successfully!</p>
            {:else}
              <div class="partial-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="12" y1="8" x2="12" y2="12"/>
                  <line x1="12" y1="16" x2="12.01" y2="16"/>
                </svg>
              </div>
              <p>{results.filter(r => r.success).length} of {results.length} files converted</p>
            {/if}
          </div>

          <div class="results-list">
            {#each results as result}
              <div class="result-item" class:success={result.success} class:error={!result.success}>
                <div class="result-icon">
                  {#if result.success}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="20 6 9 17 4 12"/>
                    </svg>
                  {:else}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <line x1="18" y1="6" x2="6" y2="18"/>
                      <line x1="6" y1="6" x2="18" y2="18"/>
                    </svg>
                  {/if}
                </div>
                <div class="result-info">
                  <span class="result-name">{result.sourcePath.split('/').pop()}</span>
                  {#if result.success && result.sourceSize && result.outputSize}
                    <span class="result-size">
                      {formatBytes(result.sourceSize)} → {formatBytes(result.outputSize)}
                      ({Math.round((result.outputSize / result.sourceSize) * 100)}%)
                    </span>
                  {:else if result.error}
                    <span class="result-error">{result.error}</span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button class="btn-primary" onclick={handleDone}>Done</button>
      </div>
    {:else}
      <div class="modal-body">
        <div class="ffmpeg-info">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
          </svg>
          <span>FFmpeg {ffmpegVersion.split(' ')[0] || 'available'}</span>
        </div>

        <!-- Format Selection -->
        <div class="format-section">
          <span class="section-label">Output Format</span>
          <div class="format-grid">
            {#each formats as format}
              <button
                class="format-option"
                class:selected={selectedFormat === format.id}
                onclick={() => selectedFormat = format.id}
              >
                <span class="format-name">{format.name}</span>
                <span class="format-ext">{format.extension}</span>
              </button>
            {/each}
          </div>
        </div>

        <!-- Quality Selection -->
        <div class="quality-section">
          <span class="section-label">Quality</span>
          <div class="quality-options">
            {#each qualities as quality}
              <label class="quality-option" class:selected={selectedQuality === quality}>
                <input
                  type="radio"
                  name="quality"
                  value={quality}
                  bind:group={selectedQuality}
                />
                <span>{getQualityLabel(quality)}</span>
              </label>
            {/each}
          </div>
        </div>

        <!-- Options -->
        <div class="options-section">
          <label class="option-toggle">
            <input type="checkbox" bind:checked={deleteSource} />
            <span class="toggle-switch"></span>
            <span class="option-label">Delete source files after conversion</span>
          </label>
        </div>

        {#if error}
          <div class="error-banner">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <line x1="12" y1="8" x2="12" y2="12"/>
              <line x1="12" y1="16" x2="12.01" y2="16"/>
            </svg>
            <span>{error}</span>
          </div>
        {/if}
      </div>

      <div class="modal-footer">
        <button class="btn-secondary" onclick={onClose} disabled={isConverting}>Cancel</button>
        <button class="btn-primary" onclick={handleConvert} disabled={isConverting}>
          {#if isConverting}
            <span class="spinner"></span>
            Converting...
          {:else}
            Convert to {currentFormat?.name || 'MP3'}
          {/if}
        </button>
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
    max-width: 500px;
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
  .unavailable-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: #555;
    text-align: center;
  }

  .loader {
    width: 36px;
    height: 36px;
    border: 3px solid #222;
    border-top-color: #f472b6;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin-bottom: 16px;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .unavailable-state svg {
    color: #666;
    margin-bottom: 16px;
  }

  .unavailable-state h3 {
    margin: 0 0 8px 0;
    color: #888;
    font-size: 16px;
  }

  .unavailable-state p {
    margin: 0 0 16px 0;
    font-size: 14px;
  }

  .install-hint code {
    display: block;
    padding: 10px 16px;
    background: #0a0a0a;
    border: 1px solid #222;
    border-radius: 8px;
    font-family: 'JetBrains Mono', monospace;
    font-size: 13px;
    color: #888;
  }

  .ffmpeg-info {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: rgba(34, 197, 94, 0.1);
    border: 1px solid rgba(34, 197, 94, 0.2);
    border-radius: 8px;
    color: #22c55e;
    font-size: 12px;
    margin-bottom: 20px;
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

  .format-section {
    margin-bottom: 20px;
  }

  .format-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }

  .format-option {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
    padding: 12px 8px;
    background: #0a0a0a;
    border: 1px solid #1a1a1a;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .format-option:hover {
    background: #111;
    border-color: #333;
  }

  .format-option.selected {
    background: rgba(244, 114, 182, 0.1);
    border-color: rgba(244, 114, 182, 0.3);
  }

  .format-name {
    font-size: 14px;
    font-weight: 500;
    color: #fff;
  }

  .format-ext {
    font-size: 11px;
    color: #666;
    font-family: 'JetBrains Mono', monospace;
  }

  .quality-section {
    margin-bottom: 20px;
  }

  .quality-options {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .quality-option {
    display: flex;
    align-items: center;
    padding: 8px 14px;
    background: #0a0a0a;
    border: 1px solid #1a1a1a;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .quality-option:hover {
    background: #111;
    border-color: #333;
  }

  .quality-option.selected {
    background: rgba(244, 114, 182, 0.1);
    border-color: rgba(244, 114, 182, 0.3);
  }

  .quality-option input {
    display: none;
  }

  .quality-option span {
    font-size: 13px;
    color: #ccc;
  }

  .options-section {
    margin-bottom: 20px;
  }

  .option-toggle {
    display: flex;
    align-items: center;
    gap: 12px;
    cursor: pointer;
  }

  .option-toggle input {
    display: none;
  }

  .toggle-switch {
    position: relative;
    width: 40px;
    height: 22px;
    background: #222;
    border-radius: 11px;
    transition: background 0.2s;
  }

  .toggle-switch::after {
    content: '';
    position: absolute;
    top: 3px;
    left: 3px;
    width: 16px;
    height: 16px;
    background: #666;
    border-radius: 50%;
    transition: all 0.2s;
  }

  .option-toggle input:checked + .toggle-switch {
    background: #f472b6;
  }

  .option-toggle input:checked + .toggle-switch::after {
    left: 21px;
    background: #fff;
  }

  .option-label {
    font-size: 14px;
    color: #888;
  }

  .error-banner {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 14px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: 8px;
    color: #ef4444;
    font-size: 13px;
  }

  .results-section {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .results-summary {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    padding: 20px;
    text-align: center;
  }

  .success-icon {
    color: #22c55e;
  }

  .partial-icon {
    color: #f59e0b;
  }

  .results-summary p {
    margin: 0;
    font-size: 14px;
    color: #888;
  }

  .results-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 250px;
    overflow-y: auto;
  }

  .result-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    background: #0a0a0a;
    border: 1px solid #1a1a1a;
    border-radius: 8px;
  }

  .result-item.success {
    border-color: rgba(34, 197, 94, 0.2);
  }

  .result-item.error {
    border-color: rgba(239, 68, 68, 0.2);
    background: rgba(239, 68, 68, 0.05);
  }

  .result-icon {
    flex-shrink: 0;
  }

  .result-item.success .result-icon {
    color: #22c55e;
  }

  .result-item.error .result-icon {
    color: #ef4444;
  }

  .result-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .result-name {
    font-size: 13px;
    color: #ccc;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .result-size {
    font-size: 11px;
    color: #666;
  }

  .result-error {
    font-size: 11px;
    color: #ef4444;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    padding: 16px 20px;
    border-top: 1px solid #222;
  }

  .btn-secondary {
    padding: 10px 20px;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 8px;
    color: #888;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-secondary:hover:not(:disabled) {
    background: #222;
    color: #fff;
  }

  .btn-secondary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 20px;
    background: #f472b6;
    border: none;
    border-radius: 8px;
    color: #000;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-primary:hover:not(:disabled) {
    background: #ec4899;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .spinner {
    width: 14px;
    height: 14px;
    border: 2px solid transparent;
    border-top-color: currentColor;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
</style>
