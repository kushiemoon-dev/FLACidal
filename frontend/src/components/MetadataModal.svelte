<script lang="ts">
  import { onMount } from 'svelte';
  import { GetFileMetadata, GetFileCoverArt } from '../../wailsjs/go/main/App.js';

  let { filePath, onClose }: { filePath: string; onClose: () => void } = $props();

  interface FLACMetadata {
    path: string;
    title: string;
    artist: string;
    album: string;
    trackNumber: string;
    date: string;
    genre: string;
    isrc: string;
    comment: string;
    size: number;
    duration: number;
    sampleRate: number;
    bitDepth: number;
    channels: number;
    bitrate: number;
    hasCover: boolean;
    coverMime?: string;
    coverSize?: number;
    lyrics?: string;
    syncedLyrics?: string;
    hasLyrics: boolean;
  }

  let metadata: FLACMetadata | null = $state(null);
  let coverArt: string | null = $state(null);
  let loading = $state(true);
  let error = $state('');
  let showLyrics = $state(false);

  onMount(async () => {
    await loadMetadata();
  });

  async function loadMetadata() {
    loading = true;
    error = '';
    try {
      metadata = await GetFileMetadata(filePath);

      // Load cover art if available
      if (metadata?.hasCover) {
        try {
          const coverData = await GetFileCoverArt(filePath);
          coverArt = `data:${coverData.mimeType};base64,${coverData.data}`;
        } catch {
          // Cover art loading failed, ignore
        }
      }
    } catch (e: any) {
      error = e.message || 'Failed to load metadata';
    } finally {
      loading = false;
    }
  }

  function formatDuration(seconds: number): string {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }

  function formatSize(bytes: number): string {
    if (bytes >= 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
    } else if (bytes >= 1024 * 1024) {
      return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
    } else if (bytes >= 1024) {
      return `${(bytes / 1024).toFixed(2)} KB`;
    }
    return `${bytes} B`;
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      onClose();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onClose();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="modal-backdrop" onclick={handleBackdropClick} onkeydown={handleKeydown} role="dialog" aria-modal="true" tabindex="-1">
  <div class="modal-content">
    <div class="modal-header">
      <h2>File Metadata</h2>
      <button class="close-btn" onclick={onClose} aria-label="Close">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>

    {#if loading}
      <div class="loading-state">
        <div class="loader"></div>
        <p>Loading metadata...</p>
      </div>
    {:else if error}
      <div class="error-state">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <p>{error}</p>
      </div>
    {:else if metadata}
      <div class="modal-body">
        <!-- Cover & Basic Info -->
        <div class="info-header">
          {#if coverArt}
            <img src={coverArt} alt="Cover Art" class="cover-art" />
          {:else}
            <div class="cover-placeholder">
              <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                <path d="M9 18V5l12-2v13"/>
                <circle cx="6" cy="18" r="3"/>
                <circle cx="18" cy="16" r="3"/>
              </svg>
            </div>
          {/if}
          <div class="basic-info">
            <h3>{metadata.title || 'Unknown Title'}</h3>
            <p class="artist">{metadata.artist || 'Unknown Artist'}</p>
            <p class="album">{metadata.album || 'Unknown Album'}</p>
          </div>
        </div>

        <!-- Tags Section -->
        <div class="section">
          <h4>Tags</h4>
          <div class="metadata-grid">
            {#if metadata.title}
              <div class="meta-item">
                <span class="meta-label">Title</span>
                <span class="meta-value">{metadata.title}</span>
              </div>
            {/if}
            {#if metadata.artist}
              <div class="meta-item">
                <span class="meta-label">Artist</span>
                <span class="meta-value">{metadata.artist}</span>
              </div>
            {/if}
            {#if metadata.album}
              <div class="meta-item">
                <span class="meta-label">Album</span>
                <span class="meta-value">{metadata.album}</span>
              </div>
            {/if}
            {#if metadata.trackNumber}
              <div class="meta-item">
                <span class="meta-label">Track</span>
                <span class="meta-value">{metadata.trackNumber}</span>
              </div>
            {/if}
            {#if metadata.date}
              <div class="meta-item">
                <span class="meta-label">Date</span>
                <span class="meta-value">{metadata.date}</span>
              </div>
            {/if}
            {#if metadata.genre}
              <div class="meta-item">
                <span class="meta-label">Genre</span>
                <span class="meta-value">{metadata.genre}</span>
              </div>
            {/if}
            {#if metadata.isrc}
              <div class="meta-item">
                <span class="meta-label">ISRC</span>
                <span class="meta-value mono">{metadata.isrc}</span>
              </div>
            {/if}
          </div>
        </div>

        <!-- Audio Properties Section -->
        <div class="section">
          <h4>Audio Properties</h4>
          <div class="metadata-grid">
            <div class="meta-item">
              <span class="meta-label">Duration</span>
              <span class="meta-value">{formatDuration(metadata.duration)}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">Sample Rate</span>
              <span class="meta-value">{metadata.sampleRate.toLocaleString()} Hz</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">Bit Depth</span>
              <span class="meta-value">{metadata.bitDepth} bit</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">Channels</span>
              <span class="meta-value">{metadata.channels === 2 ? 'Stereo' : metadata.channels === 1 ? 'Mono' : `${metadata.channels} ch`}</span>
            </div>
            {#if metadata.bitrate > 0}
              <div class="meta-item">
                <span class="meta-label">Bitrate</span>
                <span class="meta-value">{metadata.bitrate} kbps</span>
              </div>
            {/if}
          </div>
        </div>

        <!-- File Info Section -->
        <div class="section">
          <h4>File Info</h4>
          <div class="metadata-grid">
            <div class="meta-item full-width">
              <span class="meta-label">Path</span>
              <span class="meta-value mono path">{metadata.path}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">Size</span>
              <span class="meta-value">{formatSize(metadata.size)}</span>
            </div>
            <div class="meta-item">
              <span class="meta-label">Cover Art</span>
              <span class="meta-value">
                {#if metadata.hasCover}
                  Yes ({metadata.coverMime}, {formatSize(metadata.coverSize || 0)})
                {:else}
                  No
                {/if}
              </span>
            </div>
          </div>
        </div>

        <!-- Quality Badge -->
        <div class="quality-section">
          {#if metadata.bitDepth >= 24 || metadata.sampleRate > 44100}
            <span class="quality-badge hi-res">Hi-Res</span>
          {:else if metadata.bitDepth === 16 && metadata.sampleRate === 44100}
            <span class="quality-badge cd">CD Quality</span>
          {:else}
            <span class="quality-badge lossless">Lossless</span>
          {/if}
          <span class="quality-details">
            {metadata.sampleRate / 1000} kHz / {metadata.bitDepth} bit
          </span>
        </div>

        <!-- Lyrics Section -->
        {#if metadata.hasLyrics}
          <div class="section lyrics-section">
            <button class="lyrics-toggle" onclick={() => showLyrics = !showLyrics}>
              <h4>Lyrics</h4>
              <div class="lyrics-badges">
                {#if metadata.syncedLyrics}
                  <span class="lyrics-badge synced">Synced</span>
                {/if}
                {#if metadata.lyrics}
                  <span class="lyrics-badge plain">Plain</span>
                {/if}
              </div>
              <svg class="chevron" class:open={showLyrics} width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="6 9 12 15 18 9"/>
              </svg>
            </button>
            {#if showLyrics}
              <div class="lyrics-content">
                <pre>{metadata.syncedLyrics || metadata.lyrics}</pre>
              </div>
            {/if}
          </div>
        {/if}
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
    max-width: 560px;
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
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid #222;
  }

  .modal-header h2 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
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

  .close-btn:hover {
    background: #222;
    color: #fff;
  }

  .modal-body {
    padding: 20px;
    overflow-y: auto;
  }

  .loading-state,
  .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: #555;
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

  .error-state svg {
    color: #ef4444;
    margin-bottom: 12px;
  }

  .error-state p {
    color: #888;
    margin: 0;
  }

  .info-header {
    display: flex;
    gap: 16px;
    margin-bottom: 24px;
    padding-bottom: 20px;
    border-bottom: 1px solid #1a1a1a;
  }

  .cover-art {
    width: 100px;
    height: 100px;
    border-radius: 10px;
    object-fit: cover;
    flex-shrink: 0;
  }

  .cover-placeholder {
    width: 100px;
    height: 100px;
    border-radius: 10px;
    background: #1a1a1a;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #444;
    flex-shrink: 0;
  }

  .basic-info {
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-width: 0;
  }

  .basic-info h3 {
    margin: 0 0 4px 0;
    font-size: 18px;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .basic-info .artist {
    margin: 0 0 2px 0;
    color: #888;
    font-size: 14px;
  }

  .basic-info .album {
    margin: 0;
    color: #666;
    font-size: 13px;
    font-style: italic;
  }

  .section {
    margin-bottom: 20px;
  }

  .section h4 {
    margin: 0 0 12px 0;
    font-size: 12px;
    font-weight: 600;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .metadata-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .meta-item {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .meta-item.full-width {
    grid-column: 1 / -1;
  }

  .meta-label {
    font-size: 11px;
    color: #555;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .meta-value {
    font-size: 14px;
    color: #ddd;
  }

  .meta-value.mono {
    font-family: 'JetBrains Mono', monospace;
    font-size: 13px;
  }

  .meta-value.path {
    word-break: break-all;
    color: #888;
    font-size: 12px;
  }

  .quality-section {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px;
    background: #0a0a0a;
    border-radius: 10px;
    margin-top: 8px;
  }

  .quality-badge {
    padding: 6px 12px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .quality-badge.hi-res {
    background: rgba(244, 114, 182, 0.15);
    color: #f472b6;
  }

  .quality-badge.cd {
    background: rgba(59, 130, 246, 0.15);
    color: #3b82f6;
  }

  .quality-badge.lossless {
    background: rgba(34, 197, 94, 0.15);
    color: #22c55e;
  }

  .quality-details {
    font-size: 14px;
    color: #888;
    font-family: 'JetBrains Mono', monospace;
  }

  .lyrics-section {
    margin-top: 16px;
    border: 1px solid #1a1a1a;
    border-radius: 10px;
    overflow: hidden;
  }

  .lyrics-toggle {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 14px 16px;
    background: #0a0a0a;
    border: none;
    cursor: pointer;
    text-align: left;
  }

  .lyrics-toggle:hover {
    background: #111;
  }

  .lyrics-toggle h4 {
    margin: 0;
    font-size: 12px;
    font-weight: 600;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .lyrics-badges {
    display: flex;
    gap: 6px;
    margin-left: auto;
  }

  .lyrics-badge {
    padding: 3px 8px;
    border-radius: 4px;
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
  }

  .lyrics-badge.synced {
    background: rgba(244, 114, 182, 0.15);
    color: #f472b6;
  }

  .lyrics-badge.plain {
    background: rgba(59, 130, 246, 0.15);
    color: #3b82f6;
  }

  .chevron {
    color: #555;
    transition: transform 0.2s;
  }

  .chevron.open {
    transform: rotate(180deg);
  }

  .lyrics-content {
    max-height: 200px;
    overflow-y: auto;
    padding: 16px;
    background: #0a0a0a;
    border-top: 1px solid #1a1a1a;
  }

  .lyrics-content pre {
    margin: 0;
    font-size: 13px;
    line-height: 1.6;
    color: #888;
    white-space: pre-wrap;
    font-family: inherit;
  }
</style>
