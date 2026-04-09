<script lang="ts">
  import { onMount } from 'svelte';
  import { GetDownloadHistoryFiltered, DeleteHistoryRecord, ClearDownloadHistory, RefetchFromHistory } from '../../wailsjs/go/main/App.js';
  import TabBar from '../components/TabBar.svelte';
  import { Clock, Search, Trash2, RefreshCw, ExternalLink, ArrowUpDown, X } from 'lucide-svelte';

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

  interface RecentFetch {
    url: string;
    title: string;
    creator: string;
    coverUrl: string;
    type: string;
    source: string;
    timestamp: number;
  }

  let records: DownloadRecord[] = $state([]);
  let total = $state(0);
  let isLoading = $state(true);
  let searchQuery = $state('');
  let contentTypeFilter = $state('');
  let currentPage = $state(1);
  let sortBy = $state('default');
  const pageSize = 20;

  let activeTab = $state('downloads');
  const tabs = [
    { id: 'downloads', label: 'Downloads' },
    { id: 'fetches', label: 'Fetches' },
  ];

  // Fetches tab state
  let fetches: RecentFetch[] = $state([]);
  let fetchSearchQuery = $state('');

  let { onRefetch = (content: any) => {}, onNavigateHome = (url: string) => {} }: { onRefetch?: (content: any) => void; onNavigateHome?: (url: string) => void } = $props();

  onMount(async () => {
    await loadHistory();
    loadFetches();
  });

  function loadFetches() {
    if (typeof localStorage === 'undefined') {
      fetches = [];
      return;
    }
    try {
      fetches = JSON.parse(localStorage.getItem('flacidal-recent-fetches') || '[]');
    } catch {
      fetches = [];
    }
  }

  let filteredFetches = $derived(
    fetchSearchQuery.trim()
      ? fetches.filter(f =>
          f.title.toLowerCase().includes(fetchSearchQuery.trim().toLowerCase()) ||
          f.url.toLowerCase().includes(fetchSearchQuery.trim().toLowerCase()) ||
          f.creator.toLowerCase().includes(fetchSearchQuery.trim().toLowerCase())
        )
      : fetches
  );

  function deleteFetch(url: string) {
    fetches = fetches.filter(f => f.url !== url);
    localStorage.setItem('flacidal-recent-fetches', JSON.stringify(fetches));
  }

  function refetchUrl(url: string) {
    onNavigateHome(url);
  }

  function formatFetchDate(timestamp: number): string {
    const date = new Date(timestamp);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

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

      if (sortBy && sortBy !== 'default') {
        filter.sortBy = sortBy;
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

  function handleSortChange() {
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

  let totalPages = $derived(Math.ceil(total / pageSize));
</script>

<div class="history-page">
  <div class="history-header">
    <h1>History</h1>
  </div>

  <TabBar {tabs} bind:activeTab />

  {#if activeTab === 'downloads'}
    <div class="toolbar">
      <div class="toolbar-left">
        <div class="search-box">
          <Search size={16} />
          <input
            type="text"
            placeholder="Search by name..."
            bind:value={searchQuery}
            onkeydown={(e) => e.key === 'Enter' && handleSearch()}
          />
          {#if searchQuery}
            <button class="clear-search" onclick={() => { searchQuery = ''; handleSearch(); }} aria-label="Clear search">
              <X size={14} />
            </button>
          {/if}
        </div>

        <select bind:value={contentTypeFilter} onchange={handleFilterChange}>
          <option value="">All Types</option>
          <option value="playlist">Playlists</option>
          <option value="album">Albums</option>
          <option value="track">Tracks</option>
        </select>

        <div class="sort-dropdown">
          <ArrowUpDown size={14} />
          <select bind:value={sortBy} onchange={handleSortChange}>
            <option value="default">Default</option>
            <option value="date">Date</option>
            <option value="name">Name</option>
            <option value="failed">Failed Downloads</option>
          </select>
        </div>
      </div>

      <div class="toolbar-right">
        <button class="icon-btn" onclick={loadHistory} title="Refresh">
          <RefreshCw size={16} />
        </button>
        {#if records.length > 0}
          <button class="icon-btn danger" onclick={handleClearAll} title="Clear all history">
            <Trash2 size={16} />
          </button>
        {/if}
      </div>
    </div>

    <p class="record-count">{total} records</p>

    {#if isLoading}
      <div class="loading-state">
        <div class="loader"></div>
        <p>Loading history...</p>
      </div>
    {:else if records.length === 0}
      <div class="empty-state">
        <Clock size={48} strokeWidth={1} />
        <p>No download history</p>
        <span class="hint">Your downloaded tracks will appear here.</span>
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
                  onclick={() => handleRefetch(record)}
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
                  onclick={() => handleDelete(record)}
                  title="Delete from history"
                >
                  <Trash2 size={16} />
                </button>
              </div>
            </div>
          {/each}
        </div>
      </div>

      {#if totalPages > 1}
        <div class="pagination">
          <button class="page-btn" onclick={prevPage} disabled={currentPage === 1}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="15 18 9 12 15 6"/>
            </svg>
            Previous
          </button>
          <span class="page-info">Page {currentPage} of {totalPages}</span>
          <button class="page-btn" onclick={nextPage} disabled={currentPage >= totalPages}>
            Next
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="9 18 15 12 9 6"/>
            </svg>
          </button>
        </div>
      {/if}
    {/if}

  {:else if activeTab === 'fetches'}
    <div class="toolbar">
      <div class="toolbar-left">
        <div class="search-box">
          <Search size={16} />
          <input
            type="text"
            placeholder="Search fetches..."
            bind:value={fetchSearchQuery}
          />
          {#if fetchSearchQuery}
            <button class="clear-search" onclick={() => { fetchSearchQuery = ''; }} aria-label="Clear search">
              <X size={14} />
            </button>
          {/if}
        </div>
      </div>
      <div class="toolbar-right">
        <button class="icon-btn" onclick={loadFetches} title="Refresh">
          <RefreshCw size={16} />
        </button>
      </div>
    </div>

    {#if filteredFetches.length === 0}
      <div class="empty-state">
        <ExternalLink size={48} strokeWidth={1} />
        <p>No fetch history</p>
        <span class="hint">URLs you fetch will appear here.</span>
      </div>
    {:else}
      <div class="fetches-list">
        {#each filteredFetches as fetch}
          <div class="fetch-card">
            {#if fetch.coverUrl}
              <img class="fetch-cover" src={fetch.coverUrl} alt="" />
            {:else}
              <div class="fetch-cover-placeholder">
                <ExternalLink size={20} />
              </div>
            {/if}
            <div class="fetch-info">
              <span class="fetch-title">{fetch.title}</span>
              <span class="fetch-meta">
                {#if fetch.creator}{fetch.creator} &middot; {/if}
                <span class="fetch-type">{fetch.type}</span>
                &middot; {formatFetchDate(fetch.timestamp)}
              </span>
              <span class="fetch-url">{fetch.url}</span>
            </div>
            <div class="fetch-actions">
              <button
                class="action-icon-btn primary"
                onclick={() => refetchUrl(fetch.url)}
                title="Re-fetch"
              >
                <RefreshCw size={16} />
              </button>
              <button
                class="action-icon-btn danger"
                onclick={() => deleteFetch(fetch.url)}
                title="Remove from history"
              >
                <Trash2 size={16} />
              </button>
            </div>
          </div>
        {/each}
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
    margin-bottom: 8px;
  }

  .history-header h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0;
  }

  .record-count {
    color: var(--color-text-tertiary);
    font-size: 13px;
    margin: 0 0 16px 0;
  }

  /* Toolbar */
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
  }

  .toolbar-left {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
  }

  .toolbar-right {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .icon-btn {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
    color: var(--color-text-secondary);
    cursor: pointer;
    transition: all 0.2s;
  }

  .icon-btn:hover {
    background: var(--color-bg-tertiary);
    border-color: var(--color-bg-hover);
    color: var(--color-text-primary);
  }

  .icon-btn.danger {
    border-color: rgba(239, 68, 68, 0.3);
    color: #ef4444;
  }

  .icon-btn.danger:hover {
    background: rgba(239, 68, 68, 0.1);
    border-color: #ef4444;
  }

  .search-box {
    flex: 1;
    max-width: 400px;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 16px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
  }

  .search-box :global(svg) {
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  .search-box input {
    flex: 1;
    background: transparent;
    border: none;
    color: var(--color-text-primary);
    font-size: 14px;
    outline: none;
  }

  .search-box input::placeholder {
    color: var(--color-text-muted);
  }

  .clear-search {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    background: var(--color-border-subtle);
    border: none;
    border-radius: 4px;
    color: var(--color-text-tertiary);
    cursor: pointer;
    transition: all 0.2s;
  }

  .clear-search:hover {
    background: var(--color-bg-hover);
    color: var(--color-text-primary);
  }

  .sort-dropdown {
    display: flex;
    align-items: center;
    gap: 6px;
    color: var(--color-text-muted);
  }

  .sort-dropdown select {
    padding: 8px 32px 8px 8px;
    background: var(--color-bg-secondary);
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23888' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 10px center;
    appearance: none;
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
    color: var(--color-text-secondary);
    font-size: 13px;
    cursor: pointer;
    outline: none;
  }

  .sort-dropdown select:hover {
    border-color: var(--color-bg-hover);
  }

  .toolbar-left select {
    padding: 8px 32px 8px 12px;
    background: var(--color-bg-secondary);
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23888' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 10px center;
    appearance: none;
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
    color: var(--color-text-secondary);
    font-size: 13px;
    cursor: pointer;
    outline: none;
  }

  .toolbar-left select:hover {
    border-color: var(--color-bg-hover);
  }

  /* Loading & Empty states */
  .loading-state,
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 60vh;
    color: var(--color-text-muted);
    text-align: center;
  }

  .empty-state :global(svg) {
    color: var(--color-text-tertiary);
    margin-bottom: 16px;
    opacity: 0.6;
  }

  .empty-state p {
    margin: 0;
    font-size: 16px;
    color: var(--color-text-muted);
  }

  .empty-state .hint {
    margin-top: 8px;
    font-size: 14px;
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

  /* Downloads table */
  .history-table {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    overflow: hidden;
  }

  .table-header {
    display: grid;
    grid-template-columns: 1fr 100px 140px 160px 100px;
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
    color: var(--color-text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .content-id {
    font-size: 12px;
    color: var(--color-text-muted);
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
    color: var(--color-text-muted);
  }

  .tracks-total {
    color: var(--color-text-secondary);
  }

  .tracks-failed {
    color: #ef4444;
    font-size: 11px;
    margin-left: 4px;
  }

  .cell.date {
    color: var(--color-text-tertiary);
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
    border: 1px solid var(--color-border-subtle);
    border-radius: 6px;
    color: var(--color-text-tertiary);
    cursor: pointer;
    transition: all 0.2s;
  }

  .action-icon-btn:hover {
    background: var(--color-bg-tertiary);
  }

  .action-icon-btn.primary:hover {
    border-color: var(--color-accent);
    color: var(--color-accent);
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
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 6px;
    color: var(--color-text-secondary);
    font-size: 13px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .page-btn:hover:not(:disabled) {
    background: var(--color-bg-tertiary);
    border-color: var(--color-bg-hover);
    color: var(--color-text-primary);
  }

  .page-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .page-info {
    font-size: 14px;
    color: var(--color-text-tertiary);
  }

  /* Fetches tab */
  .fetches-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .fetch-card {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 12px 16px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 10px;
    transition: background 0.2s;
  }

  .fetch-card:hover {
    background: var(--color-bg-tertiary);
  }

  .fetch-cover {
    width: 48px;
    height: 48px;
    border-radius: 6px;
    object-fit: cover;
    flex-shrink: 0;
  }

  .fetch-cover-placeholder {
    width: 48px;
    height: 48px;
    border-radius: 6px;
    background: var(--color-bg-tertiary);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  .fetch-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .fetch-title {
    font-weight: 500;
    font-size: 14px;
    color: var(--color-text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .fetch-meta {
    font-size: 12px;
    color: var(--color-text-tertiary);
  }

  .fetch-type {
    text-transform: capitalize;
  }

  .fetch-url {
    font-size: 11px;
    color: var(--color-text-muted);
    font-family: monospace;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .fetch-actions {
    display: flex;
    gap: 8px;
    flex-shrink: 0;
  }
</style>
