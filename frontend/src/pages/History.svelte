<script lang="ts">
  import { onMount } from 'svelte';
  import { GetDownloadHistoryFiltered, DeleteHistoryRecord, ClearDownloadHistory, RefetchFromHistory } from '../../wailsjs/go/main/App.js';

  interface DownloadRecord {
    id: number;
    tidalContentId: string;
    tidalContentName: string;
    contentType: string;
    lastDownloadAt: string;
    tracksTotal: number;
    tracksDownloaded: number;
    tracksFailed: number;
    createdAt: string;
  }

  let records: DownloadRecord[] = [];
  let total = 0;
  let isLoading = true;
  let searchQuery = '';
  let contentTypeFilter = '';
  let currentPage = 1;
  const pageSize = 20;

  // Export function for triggering refetch from parent
  export let onRefetch: (content: any) => void = () => {};

  onMount(async () => {
    await loadHistory();
  });

  async function loadHistory() {
    isLoading = true;
    try {
      const filter: Record<string, any> = {
        limit: pageSize,
        offset: (currentPage - 1) * pageSize
      };

      if (contentTypeFilter) {
        filter.contentType = contentTypeFilter;
      }

      if (searchQuery.trim()) {
        filter.search = searchQuery.trim();
      }

      const result = await GetDownloadHistoryFiltered(filter);
      records = result.records || [];
      total = result.total || 0;
    } catch (error) {
      console.error('Error loading history:', error);
      records = [];
      total = 0;
    } finally {
      isLoading = false;
    }
  }

  function formatDate(dateStr: string): string {
    if (!dateStr) return '--';
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  function getContentTypeLabel(type: string): string {
    switch (type) {
      case 'playlist': return 'Playlist';
      case 'album': return 'Album';
      case 'track': return 'Track';
      default: return type;
    }
  }

  async function handleRefetch(record: DownloadRecord) {
    try {
      const content = await RefetchFromHistory(record.tidalContentId);
      if (content && onRefetch) {
        onRefetch(content);
      }
    } catch (error) {
      console.error('Error refetching:', error);
    }
  }

  async function handleDelete(record: DownloadRecord) {
    if (!confirm(`Are you sure you want to delete "${record.tidalContentName}" from history?`)) return;

    try {
      await DeleteHistoryRecord(record.id);
      await loadHistory();
    } catch (error) {
      console.error('Error deleting record:', error);
    }
  }

  async function handleClearAll() {
    if (!confirm('Are you sure you want to clear all download history? This cannot be undone.')) return;

    try {
      await ClearDownloadHistory();
      await loadHistory();
    } catch (error) {
      console.error('Error clearing history:', error);
    }
  }

  function handleSearch() {
    currentPage = 1;
    loadHistory();
  }

  function handleFilterChange() {
    currentPage = 1;
    loadHistory();
  }

  function nextPage() {
    if (currentPage * pageSize < total) {
      currentPage++;
      loadHistory();
    }
  }

  function prevPage() {
    if (currentPage > 1) {
      currentPage--;
      loadHistory();
    }
  }

  $: totalPages = Math.ceil(total / pageSize);
</script>

<div class="history-page">
  <div class="history-header">
    <div class="header-left">
      <h1>Download History</h1>
      <p class="record-count">{total} records</p>
    </div>
    <div class="header-actions">
      <button class="action-btn" on:click={loadHistory}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 2v6h-6"/>
          <path d="M3 12a9 9 0 0 1 15-6.7L21 8"/>
          <path d="M3 22v-6h6"/>
          <path d="M21 12a9 9 0 0 1-15 6.7L3 16"/>
        </svg>
        Refresh
      </button>
      {#if records.length > 0}
        <button class="action-btn danger" on:click={handleClearAll}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 6h18"/>
            <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/>
            <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
          </svg>
          Clear All
        </button>
      {/if}
    </div>
  </div>

  <div class="filters">
    <div class="search-box">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="8"/>
        <path d="m21 21-4.35-4.35"/>
      </svg>
      <input
        type="text"
        placeholder="Search by name..."
        bind:value={searchQuery}
        on:keydown={(e) => e.key === 'Enter' && handleSearch()}
      />
      {#if searchQuery}
        <button class="clear-search" on:click={() => { searchQuery = ''; handleSearch(); }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      {/if}
    </div>

    <select bind:value={contentTypeFilter} on:change={handleFilterChange}>
      <option value="">All Types</option>
      <option value="playlist">Playlists</option>
      <option value="album">Albums</option>
      <option value="track">Tracks</option>
    </select>
  </div>

  {#if isLoading}
    <div class="loading-state">
      <div class="loader"></div>
      <p>Loading history...</p>
    </div>
  {:else if records.length === 0}
    <div class="empty-state">
      <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
        <circle cx="12" cy="12" r="10"/>
        <polyline points="12 6 12 12 16 14"/>
      </svg>
      <p>No download history</p>
      <span class="hint">Downloaded playlists and albums will appear here</span>
    </div>
  {:else}
    <div class="history-table">
      <div class="table-header">
        <span class="th">Name</span>
        <span class="th">Type</span>
        <span class="th">Tracks</span>
        <span class="th">Last Download</span>
        <span class="th">Actions</span>
      </div>

      <div class="table-body">
        {#each records as record}
          <div class="table-row">
            <div class="cell name-cell">
              <div class="content-icon" class:playlist={record.contentType === 'playlist'} class:album={record.contentType === 'album'} class:track={record.contentType === 'track'}>
                {#if record.contentType === 'playlist'}
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="8" y1="6" x2="21" y2="6"/>
                    <line x1="8" y1="12" x2="21" y2="12"/>
                    <line x1="8" y1="18" x2="21" y2="18"/>
                    <line x1="3" y1="6" x2="3.01" y2="6"/>
                    <line x1="3" y1="12" x2="3.01" y2="12"/>
                    <line x1="3" y1="18" x2="3.01" y2="18"/>
                  </svg>
                {:else if record.contentType === 'album'}
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"/>
                    <circle cx="12" cy="12" r="3"/>
                  </svg>
                {:else}
                  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M9 18V5l12-2v13"/>
                    <circle cx="6" cy="18" r="3"/>
                    <circle cx="18" cy="16" r="3"/>
                  </svg>
                {/if}
              </div>
              <div class="content-info">
                <span class="content-name">{record.tidalContentName || record.tidalContentId}</span>
                <span class="content-id">{record.tidalContentId}</span>
              </div>
            </div>
            <span class="cell type-badge {record.contentType}">{getContentTypeLabel(record.contentType)}</span>
            <div class="cell tracks-cell">
              <span class="tracks-downloaded">{record.tracksDownloaded}</span>
              <span class="tracks-separator">/</span>
              <span class="tracks-total">{record.tracksTotal}</span>
              {#if record.tracksFailed > 0}
                <span class="tracks-failed">({record.tracksFailed} failed)</span>
              {/if}
            </div>
            <span class="cell date">{formatDate(record.lastDownloadAt)}</span>
            <div class="cell actions">
              <button
                class="action-icon-btn primary"
                on:click={() => handleRefetch(record)}
                title="Re-download"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="7 10 12 15 17 10"/>
                  <line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
              </button>
              <button
                class="action-icon-btn danger"
                on:click={() => handleDelete(record)}
                title="Delete from history"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M3 6h18"/>
                  <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/>
                  <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
                </svg>
              </button>
            </div>
          </div>
        {/each}
      </div>
    </div>

    {#if totalPages > 1}
      <div class="pagination">
        <button class="page-btn" on:click={prevPage} disabled={currentPage === 1}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15 18 9 12 15 6"/>
          </svg>
          Previous
        </button>
        <span class="page-info">Page {currentPage} of {totalPages}</span>
        <button class="page-btn" on:click={nextPage} disabled={currentPage >= totalPages}>
          Next
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
        </button>
      </div>
    {/if}
  {/if}
</div>

<style>
  .history-page {
    padding: 32px;
    max-width: 1200px;
  }

  .history-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 24px;
    gap: 24px;
  }

  .header-left h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0 0 8px 0;
  }

  .record-count {
    color: #666;
    font-size: 14px;
    margin: 0;
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

  .action-btn:hover {
    background: #1a1a1a;
    border-color: #333;
    color: #fff;
  }

  .action-btn.danger {
    border-color: rgba(239, 68, 68, 0.3);
    color: #ef4444;
  }

  .action-btn.danger:hover {
    background: rgba(239, 68, 68, 0.1);
    border-color: #ef4444;
  }

  .filters {
    display: flex;
    gap: 12px;
    margin-bottom: 24px;
  }

  .search-box {
    flex: 1;
    max-width: 400px;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 16px;
    background: #111;
    border: 1px solid #222;
    border-radius: 8px;
  }

  .search-box svg {
    color: #555;
    flex-shrink: 0;
  }

  .search-box input {
    flex: 1;
    background: transparent;
    border: none;
    color: #fff;
    font-size: 14px;
    outline: none;
  }

  .search-box input::placeholder {
    color: #555;
  }

  .clear-search {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    background: #222;
    border: none;
    border-radius: 4px;
    color: #666;
    cursor: pointer;
    transition: all 0.2s;
  }

  .clear-search:hover {
    background: #333;
    color: #fff;
  }

  .filters select {
    padding: 10px 16px;
    background: #111;
    border: 1px solid #222;
    border-radius: 8px;
    color: #888;
    font-size: 14px;
    cursor: pointer;
    outline: none;
  }

  .filters select:hover {
    border-color: #333;
  }

  .loading-state,
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

  .history-table {
    background: #111;
    border: 1px solid #1a1a1a;
    border-radius: 12px;
    overflow: hidden;
  }

  .table-header {
    display: grid;
    grid-template-columns: 1fr 100px 140px 160px 100px;
    gap: 16px;
    padding: 12px 16px;
    background: #0a0a0a;
    border-bottom: 1px solid #1a1a1a;
  }

  .th {
    font-size: 12px;
    font-weight: 600;
    color: #666;
    text-transform: uppercase;
    text-align: left;
  }

  .table-body {
    max-height: calc(100vh - 350px);
    overflow-y: auto;
  }

  .table-row {
    display: grid;
    grid-template-columns: 1fr 100px 140px 160px 100px;
    gap: 16px;
    padding: 12px 16px;
    align-items: center;
    border-bottom: 1px solid #1a1a1a;
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
    color: #888;
  }

  .name-cell {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;
  }

  .content-icon {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 8px;
    flex-shrink: 0;
  }

  .content-icon.playlist {
    background: rgba(168, 85, 247, 0.15);
    color: #a855f7;
  }

  .content-icon.album {
    background: rgba(59, 130, 246, 0.15);
    color: #3b82f6;
  }

  .content-icon.track {
    background: rgba(244, 114, 182, 0.15);
    color: #f472b6;
  }

  .content-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .content-name {
    font-weight: 500;
    color: #fff;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .content-id {
    font-size: 12px;
    color: #555;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    font-family: monospace;
  }

  .type-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 4px 10px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 500;
    width: fit-content;
  }

  .type-badge.playlist {
    background: rgba(168, 85, 247, 0.15);
    color: #a855f7;
  }

  .type-badge.album {
    background: rgba(59, 130, 246, 0.15);
    color: #3b82f6;
  }

  .type-badge.track {
    background: rgba(244, 114, 182, 0.15);
    color: #f472b6;
  }

  .tracks-cell {
    display: flex;
    align-items: center;
    gap: 4px;
    font-family: monospace;
    font-size: 13px;
  }

  .tracks-downloaded {
    color: #22c55e;
  }

  .tracks-separator {
    color: #444;
  }

  .tracks-total {
    color: #888;
  }

  .tracks-failed {
    color: #ef4444;
    font-size: 11px;
    margin-left: 4px;
  }

  .cell.date {
    color: #666;
  }

  .cell.actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }

  .action-icon-btn {
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

  .action-icon-btn:hover {
    background: #1a1a1a;
  }

  .action-icon-btn.primary:hover {
    border-color: #f472b6;
    color: #f472b6;
  }

  .action-icon-btn.danger:hover {
    border-color: #ef4444;
    color: #ef4444;
  }

  .pagination {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 16px;
    padding: 16px 0;
  }

  .page-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 14px;
    background: #111;
    border: 1px solid #222;
    border-radius: 6px;
    color: #888;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .page-btn:hover:not(:disabled) {
    background: #1a1a1a;
    border-color: #333;
    color: #fff;
  }

  .page-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .page-info {
    font-size: 14px;
    color: #666;
  }
</style>
