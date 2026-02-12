<script lang="ts">
  import { queueItems, queueStats, queueStore, downloadFolder, queuePaused } from '../stores/queue';
  import { QueueSingleDownload, RetryAllFailed, CancelDownload, PauseDownloads, ResumeDownloads } from '../../wailsjs/go/main/App.js';

  let queue = $derived($queueStore);
  let folder = $derived($downloadFolder);

  function getStatusClass(status: string) {
    switch (status) {
      case 'pending':
      case 'queued':
        return 'status-pending';
      case 'downloading':
        return 'status-downloading';
      case 'completed':
        return 'status-completed';
      case 'error':
        return 'status-error';
      case 'cancelled':
        return 'status-cancelled';
      default:
        return '';
    }
  }

  function clearCompleted() {
    queueStore.clearCompleted();
  }

  function clearFailed() {
    queueStore.clearFailed();
  }

  function clearAll() {
    queueStore.clearAll();
  }

  async function retryAllFailedDownloads() {
    try {
      const count = await RetryAllFailed();
      console.log(`Retrying ${count} failed downloads`);
    } catch (error) {
      console.error('Retry all failed error:', error);
    }
  }

  async function cancelDownload(trackId: number) {
    try {
      await CancelDownload(trackId);
      queueStore.updateItem(trackId, { status: 'cancelled' });
    } catch (error) {
      console.error('Cancel error:', error);
    }
  }

  async function retryFailed(trackId: number) {
    const item = queue.get(trackId);
    if (!item || !folder) return;

    try {
      queueStore.updateItem(trackId, { status: 'pending', error: undefined });
      await QueueSingleDownload(trackId, folder, item.title, item.artist);
    } catch (error) {
      console.error('Retry error:', error);
    }
  }

  function removeItem(trackId: number) {
    queueStore.removeItem(trackId);
  }

  async function togglePause() {
    try {
      if ($queuePaused) {
        await ResumeDownloads();
        queuePaused.set(false);
      } else {
        await PauseDownloads();
        queuePaused.set(true);
      }
    } catch (error) {
      console.error('Toggle pause error:', error);
    }
  }
</script>

<div class="queue-page">
  <div class="queue-header">
    <div class="header-left">
      <h1>Download Queue</h1>
      <div class="stats">
        {#if $queuePaused}
          <span class="stat paused-indicator">
            <span class="stat-value paused">PAUSED</span>
            <span class="stat-label">Status</span>
          </span>
        {/if}
        <span class="stat">
          <span class="stat-value">{$queueStats.total}</span>
          <span class="stat-label">Total</span>
        </span>
        <span class="stat">
          <span class="stat-value downloading">{$queueStats.downloading}</span>
          <span class="stat-label">Downloading</span>
        </span>
        <span class="stat">
          <span class="stat-value pending">{$queueStats.pending}</span>
          <span class="stat-label">Pending</span>
        </span>
        <span class="stat">
          <span class="stat-value completed">{$queueStats.completed}</span>
          <span class="stat-label">Completed</span>
        </span>
        <span class="stat">
          <span class="stat-value failed">{$queueStats.failed}</span>
          <span class="stat-label">Failed</span>
        </span>
      </div>
    </div>
    <div class="header-actions">
      <button
        class="action-btn pause-btn"
        class:paused={$queuePaused}
        onclick={togglePause}
        disabled={$queueStats.pending === 0 && $queueStats.downloading === 0 && !$queuePaused}
      >
        {#if $queuePaused}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polygon points="5 3 19 12 5 21 5 3"/>
          </svg>
          Resume
        {:else}
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="6" y="4" width="4" height="16"/>
            <rect x="14" y="4" width="4" height="16"/>
          </svg>
          Pause
        {/if}
      </button>
      <button class="action-btn retry-all" onclick={retryAllFailedDownloads} disabled={$queueStats.failed === 0}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 2v6h-6"/>
          <path d="M3 12a9 9 0 0 1 15-6.7L21 8"/>
          <path d="M3 22v-6h6"/>
          <path d="M21 12a9 9 0 0 1-15 6.7L3 16"/>
        </svg>
        Retry Failed ({$queueStats.failed})
      </button>
      <button class="action-btn" onclick={clearCompleted} disabled={$queueStats.completed === 0}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <polyline points="20 6 9 17 4 12"/>
        </svg>
        Clear Completed
      </button>
      <button class="action-btn" onclick={clearFailed} disabled={$queueStats.failed === 0}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="15" y1="9" x2="9" y2="15"/>
          <line x1="9" y1="9" x2="15" y2="15"/>
        </svg>
        Clear Failed
      </button>
      <button class="action-btn danger" onclick={clearAll} disabled={$queueStats.total === 0}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 6h18"/>
          <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/>
          <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
        </svg>
        Clear All
      </button>
    </div>
  </div>

  {#if $queueItems.length === 0}
    <div class="empty-state">
      <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
        <polyline points="7 10 12 15 17 10"/>
        <line x1="12" y1="15" x2="12" y2="3"/>
      </svg>
      <p>No downloads in queue</p>
      <span class="hint">Add tracks from Home or Search to start downloading</span>
    </div>
  {:else}
    <div class="queue-list">
      {#each $queueItems as item (item.trackId)}
        <div class="queue-item {getStatusClass(item.status)}">
          <div class="item-status">
            {#if item.status === 'downloading'}
              <div class="spinner"></div>
            {:else if item.status === 'completed'}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
            {:else if item.status === 'error'}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <line x1="15" y1="9" x2="9" y2="15"/>
                <line x1="9" y1="9" x2="15" y2="15"/>
              </svg>
            {:else if item.status === 'cancelled'}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <line x1="8" y1="12" x2="16" y2="12"/>
              </svg>
            {:else}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="12" cy="12" r="10"/>
                <polyline points="12 6 12 12 16 14"/>
              </svg>
            {/if}
          </div>

          <div class="item-info">
            <span class="item-title">{item.title}</span>
            <span class="item-artist">{item.artist}</span>
            {#if item.error}
              <span class="item-error">{item.error}</span>
            {/if}
          </div>

          <div class="item-actions">
            {#if item.status === 'downloading'}
              <button
                class="item-btn cancel"
                onclick={() => cancelDownload(item.trackId)}
                title="Cancel"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="6" y="6" width="12" height="12" rx="2"/>
                </svg>
              </button>
            {/if}
            {#if item.status === 'error'}
              <button
                class="item-btn retry"
                onclick={() => retryFailed(item.trackId)}
                title="Retry"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 2v6h-6"/>
                  <path d="M3 12a9 9 0 0 1 15-6.7L21 8"/>
                  <path d="M3 22v-6h6"/>
                  <path d="M21 12a9 9 0 0 1-15 6.7L3 16"/>
                </svg>
              </button>
            {/if}
            <button
              class="item-btn remove"
              onclick={() => removeItem(item.trackId)}
              title="Remove"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="18" y1="6" x2="6" y2="18"/>
                <line x1="6" y1="6" x2="18" y2="18"/>
              </svg>
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .queue-page {
    padding: 32px;
    max-width: 1000px;
  }

  .queue-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 32px;
    gap: 24px;
  }

  .header-left h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0 0 16px 0;
  }

  .stats {
    display: flex;
    gap: 24px;
  }

  .stat {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .stat-value {
    font-size: 20px;
    font-weight: 600;
  }

  .stat-value.downloading {
    color: #3b82f6;
  }

  .stat-value.pending {
    color: #f59e0b;
  }

  .stat-value.completed {
    color: #10b981;
  }

  .stat-value.failed {
    color: #ef4444;
  }

  .stat-value.paused {
    color: #f59e0b;
    font-size: 14px;
    font-weight: 700;
    animation: pulse 1.5s ease-in-out infinite;
  }

  .paused-indicator {
    background: rgba(245, 158, 11, 0.1);
    padding: 8px 12px;
    border-radius: 8px;
    border: 1px solid rgba(245, 158, 11, 0.3);
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
  }

  .stat-label {
    font-size: 12px;
    color: #666;
    text-transform: uppercase;
  }

  .header-actions {
    display: flex;
    gap: 12px;
  }

  .action-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    background: #111;
    border: 1px solid #222;
    border-radius: 8px;
    color: #888;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .action-btn:hover:not(:disabled) {
    background: #1a1a1a;
    border-color: #333;
    color: #fff;
  }

  .action-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .action-btn.danger:hover:not(:disabled) {
    border-color: #ef4444;
    color: #ef4444;
  }

  .action-btn.retry-all:not(:disabled) {
    border-color: rgba(59, 130, 246, 0.3);
    color: #3b82f6;
  }

  .action-btn.retry-all:hover:not(:disabled) {
    border-color: #3b82f6;
    background: rgba(59, 130, 246, 0.1);
  }

  .action-btn.pause-btn:not(:disabled) {
    border-color: rgba(245, 158, 11, 0.3);
    color: #f59e0b;
  }

  .action-btn.pause-btn:hover:not(:disabled) {
    border-color: #f59e0b;
    background: rgba(245, 158, 11, 0.1);
  }

  .action-btn.pause-btn.paused:not(:disabled) {
    border-color: rgba(16, 185, 129, 0.3);
    color: #10b981;
  }

  .action-btn.pause-btn.paused:hover:not(:disabled) {
    border-color: #10b981;
    background: rgba(16, 185, 129, 0.1);
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 80px 20px;
    color: #444;
    text-align: center;
  }

  .empty-state svg {
    margin-bottom: 16px;
    opacity: 0.5;
  }

  .empty-state p {
    margin: 0;
    font-size: 16px;
    color: #555;
  }

  .empty-state .hint {
    margin-top: 8px;
    font-size: 14px;
  }

  .queue-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .queue-item {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px;
    background: #111;
    border: 1px solid #1a1a1a;
    border-radius: 12px;
    transition: all 0.2s;
  }

  .queue-item:hover {
    background: #151515;
  }

  .queue-item.status-downloading {
    border-color: rgba(59, 130, 246, 0.3);
  }

  .queue-item.status-completed {
    border-color: rgba(16, 185, 129, 0.3);
  }

  .queue-item.status-error {
    border-color: rgba(239, 68, 68, 0.3);
  }

  .queue-item.status-cancelled {
    border-color: rgba(107, 114, 128, 0.3);
  }

  .item-status {
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 10px;
    flex-shrink: 0;
  }

  .status-pending .item-status {
    background: rgba(245, 158, 11, 0.15);
    color: #f59e0b;
  }

  .status-downloading .item-status {
    background: rgba(59, 130, 246, 0.15);
    color: #3b82f6;
  }

  .status-completed .item-status {
    background: rgba(16, 185, 129, 0.15);
    color: #10b981;
  }

  .status-error .item-status {
    background: rgba(239, 68, 68, 0.15);
    color: #ef4444;
  }

  .status-cancelled .item-status {
    background: rgba(107, 114, 128, 0.15);
    color: #6b7280;
  }

  .spinner {
    width: 20px;
    height: 20px;
    border: 2px solid transparent;
    border-top-color: currentColor;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .item-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  .item-title {
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .item-artist {
    font-size: 13px;
    color: #888;
  }

  .item-error {
    font-size: 12px;
    color: #ef4444;
    margin-top: 4px;
  }

  .item-actions {
    display: flex;
    gap: 8px;
  }

  .item-btn {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 1px solid #333;
    border-radius: 6px;
    color: #666;
    cursor: pointer;
    transition: all 0.2s;
  }

  .item-btn:hover {
    background: #1a1a1a;
    color: #fff;
  }

  .item-btn.retry:hover {
    border-color: #3b82f6;
    color: #3b82f6;
  }

  .item-btn.remove:hover {
    border-color: #ef4444;
    color: #ef4444;
  }

  .item-btn.cancel:hover {
    border-color: #f59e0b;
    color: #f59e0b;
  }
</style>
