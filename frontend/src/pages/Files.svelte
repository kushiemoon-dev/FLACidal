<script lang="ts">
  import { onMount } from 'svelte';
  import { downloadFolder } from '../stores/queue';
  import { ListDownloadedFiles, DeleteFile, OpenDownloadFolder, IsConverterAvailable, FetchAndEmbedLyricsMultiple } from '../../wailsjs/go/main/App.js';
  import MetadataModal from '../components/MetadataModal.svelte';
  import RenameModal from '../components/RenameModal.svelte';
  import ConvertModal from '../components/ConvertModal.svelte';
  import AnalysisModal from '../components/AnalysisModal.svelte';

  interface DownloadedFile {
    path: string;
    name: string;
    size: number;
    modTime: string;
    title: string;
    artist: string;
    album: string;
  }

  let files: DownloadedFile[] = [];
  let isLoading = true;
  let sortBy: 'name' | 'date' | 'size' = 'date';
  let sortOrder: 'asc' | 'desc' = 'desc';
  let metadataFilePath: string | null = null;
  let selectedFiles: Set<string> = new Set();
  let showRenameModal = false;
  let showConvertModal = false;
  let showAnalysisModal = false;
  let converterAvailable = false;
  let isFetchingLyrics = false;
  let lyricsResults: { success: number; failed: number } | null = null;

  $: allSelected = files.length > 0 && selectedFiles.size === files.length;
  $: someSelected = selectedFiles.size > 0;

  function toggleSelectAll() {
    if (allSelected) {
      selectedFiles = new Set();
    } else {
      selectedFiles = new Set(files.map(f => f.path));
    }
  }

  function toggleSelect(path: string) {
    if (selectedFiles.has(path)) {
      selectedFiles.delete(path);
      selectedFiles = selectedFiles; // Trigger reactivity
    } else {
      selectedFiles.add(path);
      selectedFiles = selectedFiles; // Trigger reactivity
    }
  }

  function clearSelection() {
    selectedFiles = new Set();
  }

  function handleRenameComplete() {
    selectedFiles = new Set();
    loadFiles();
  }

  function handleConvertComplete() {
    selectedFiles = new Set();
    loadFiles();
  }

  async function handleFetchLyrics() {
    if (selectedFiles.size === 0 || isFetchingLyrics) return;

    isFetchingLyrics = true;
    lyricsResults = null;

    try {
      const filePaths = Array.from(selectedFiles);
      const results = await FetchAndEmbedLyricsMultiple(filePaths);

      let success = 0;
      let failed = 0;
      for (const r of results) {
        if (r.success) {
          success++;
        } else {
          failed++;
        }
      }

      lyricsResults = { success, failed };

      // Auto-clear message after 5 seconds
      setTimeout(() => {
        lyricsResults = null;
      }, 5000);
    } catch (error) {
      console.error('Error fetching lyrics:', error);
    } finally {
      isFetchingLyrics = false;
    }
  }

  onMount(async () => {
    await loadFiles();
    converterAvailable = await IsConverterAvailable();
  });

  async function loadFiles() {
    isLoading = true;
    try {
      const result = await ListDownloadedFiles();
      files = result || [];
    } catch (error) {
      console.error('Error loading files:', error);
      files = [];
    } finally {
      isLoading = false;
    }
  }

  function formatSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric'
    });
  }

  async function openInFileManager(path: string) {
    try {
      // Open the containing folder
      const folder = path.substring(0, path.lastIndexOf('/'));
      await OpenDownloadFolder(folder);
    } catch (error) {
      console.error('Error opening file manager:', error);
    }
  }

  async function deleteFileHandler(path: string) {
    if (!confirm('Are you sure you want to delete this file?')) return;

    try {
      await DeleteFile(path);
      await loadFiles();
    } catch (error) {
      console.error('Error deleting file:', error);
    }
  }

  async function openFolder() {
    if ($downloadFolder) {
      try {
        await OpenDownloadFolder($downloadFolder);
      } catch (error) {
        console.error('Error opening folder:', error);
      }
    }
  }

  function sortFiles(by: 'name' | 'date' | 'size') {
    if (sortBy === by) {
      sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
    } else {
      sortBy = by;
      sortOrder = 'desc';
    }
  }

  $: sortedFiles = [...files].sort((a, b) => {
    let comparison = 0;
    switch (sortBy) {
      case 'name':
        comparison = a.name.localeCompare(b.name);
        break;
      case 'date':
        comparison = new Date(a.modTime).getTime() - new Date(b.modTime).getTime();
        break;
      case 'size':
        comparison = a.size - b.size;
        break;
    }
    return sortOrder === 'asc' ? comparison : -comparison;
  });
</script>

<div class="files-page">
  <div class="files-header">
    <div class="header-left">
      <h1>Downloaded Files</h1>
      <p class="folder-path">{$downloadFolder || 'No folder selected'}</p>
    </div>
    <div class="header-actions">
      {#if someSelected}
        <span class="selection-count">{selectedFiles.size} selected</span>
        <button class="action-btn" on:click={clearSelection}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
          Clear
        </button>
        <button class="action-btn" on:click={() => showAnalysisModal = true} title="Analyze quality">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
          </svg>
          Analyze
        </button>
        <button
          class="action-btn"
          on:click={handleFetchLyrics}
          disabled={isFetchingLyrics}
          title="Fetch and embed lyrics"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M9 18V5l12-2v13"/>
            <circle cx="6" cy="18" r="3"/>
            <circle cx="18" cy="16" r="3"/>
          </svg>
          {#if isFetchingLyrics}
            Fetching...
          {:else}
            Get Lyrics
          {/if}
        </button>
        {#if converterAvailable}
          <button class="action-btn" on:click={() => showConvertModal = true} title="Convert to lossy format">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="17 8 12 3 7 8"/>
              <line x1="12" y1="3" x2="12" y2="15"/>
            </svg>
            Convert
          </button>
        {/if}
        <button class="action-btn primary" on:click={() => showRenameModal = true}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
            <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
          </svg>
          Rename
        </button>
      {:else}
        <button class="action-btn" on:click={loadFiles}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 2v6h-6"/>
            <path d="M3 12a9 9 0 0 1 15-6.7L21 8"/>
            <path d="M3 22v-6h6"/>
            <path d="M21 12a9 9 0 0 1-15 6.7L3 16"/>
          </svg>
          Refresh
        </button>
        <button class="action-btn primary" on:click={openFolder}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
          Open Folder
        </button>
      {/if}
    </div>
  </div>

  {#if lyricsResults}
    <div class="lyrics-notification" class:has-failures={lyricsResults.failed > 0}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M9 18V5l12-2v13"/>
        <circle cx="6" cy="18" r="3"/>
        <circle cx="18" cy="16" r="3"/>
      </svg>
      <span>
        {#if lyricsResults.success > 0}
          Lyrics embedded: {lyricsResults.success} file{lyricsResults.success !== 1 ? 's' : ''}
        {/if}
        {#if lyricsResults.failed > 0}
          {#if lyricsResults.success > 0}, {/if}
          Not found: {lyricsResults.failed}
        {/if}
      </span>
    </div>
  {/if}

  {#if isLoading}
    <div class="loading-state">
      <div class="loader"></div>
      <p>Loading files...</p>
    </div>
  {:else if files.length === 0}
    <div class="empty-state">
      <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
      </svg>
      <p>No downloaded files</p>
      <span class="hint">Downloaded FLAC files will appear here</span>
    </div>
  {:else}
    <div class="files-table">
      <div class="table-header">
        <label class="th checkbox-col">
          <input type="checkbox" checked={allSelected} on:change={toggleSelectAll} />
          <span class="custom-checkbox" class:checked={allSelected} class:indeterminate={someSelected && !allSelected}>
            {#if allSelected}
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
            {:else if someSelected}
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                <line x1="5" y1="12" x2="19" y2="12"/>
              </svg>
            {/if}
          </span>
        </label>
        <button class="th sortable" on:click={() => sortFiles('name')}>
          Name
          {#if sortBy === 'name'}
            <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
              {#if sortOrder === 'asc'}
                <path d="M7 14l5-5 5 5z"/>
              {:else}
                <path d="M7 10l5 5 5-5z"/>
              {/if}
            </svg>
          {/if}
        </button>
        <span class="th">Artist</span>
        <span class="th">Album</span>
        <button class="th sortable" on:click={() => sortFiles('size')}>
          Size
          {#if sortBy === 'size'}
            <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
              {#if sortOrder === 'asc'}
                <path d="M7 14l5-5 5 5z"/>
              {:else}
                <path d="M7 10l5 5 5-5z"/>
              {/if}
            </svg>
          {/if}
        </button>
        <button class="th sortable" on:click={() => sortFiles('date')}>
          Date
          {#if sortBy === 'date'}
            <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
              {#if sortOrder === 'asc'}
                <path d="M7 14l5-5 5 5z"/>
              {:else}
                <path d="M7 10l5 5 5-5z"/>
              {/if}
            </svg>
          {/if}
        </button>
        <span class="th">Actions</span>
      </div>

      <div class="table-body">
        {#each sortedFiles as file}
          <div class="table-row" class:selected={selectedFiles.has(file.path)}>
            <label class="cell checkbox-col">
              <input type="checkbox" checked={selectedFiles.has(file.path)} on:change={() => toggleSelect(file.path)} />
              <span class="custom-checkbox" class:checked={selectedFiles.has(file.path)}>
                {#if selectedFiles.has(file.path)}
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
                    <polyline points="20 6 9 17 4 12"/>
                  </svg>
                {/if}
              </span>
            </label>
            <div class="cell name-cell">
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="#f472b6" stroke-width="2">
                <path d="M9 18V5l12-2v13"/>
                <circle cx="6" cy="18" r="3"/>
                <circle cx="18" cy="16" r="3"/>
              </svg>
              <div class="file-info">
                <span class="file-name">{file.title || file.name}</span>
                <span class="file-path">{file.name}</span>
              </div>
            </div>
            <span class="cell">{file.artist || '--'}</span>
            <span class="cell">{file.album || '--'}</span>
            <span class="cell size">{formatSize(file.size)}</span>
            <span class="cell date">{formatDate(file.modTime)}</span>
            <div class="cell actions">
              <button
                class="file-btn info"
                on:click={() => metadataFilePath = file.path}
                title="View metadata"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="12" cy="12" r="10"/>
                  <line x1="12" y1="16" x2="12" y2="12"/>
                  <line x1="12" y1="8" x2="12.01" y2="8"/>
                </svg>
              </button>
              <button
                class="file-btn"
                on:click={() => openInFileManager(file.path)}
                title="Show in folder"
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                </svg>
              </button>
              <button
                class="file-btn danger"
                on:click={() => deleteFileHandler(file.path)}
                title="Delete"
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

    <div class="files-footer">
      <span class="total-count">{files.length} files</span>
      <span class="total-size">{formatSize(files.reduce((acc, f) => acc + f.size, 0))} total</span>
    </div>
  {/if}
</div>

{#if metadataFilePath}
  <MetadataModal filePath={metadataFilePath} onClose={() => metadataFilePath = null} />
{/if}

{#if showRenameModal}
  <RenameModal
    files={Array.from(selectedFiles)}
    onClose={() => showRenameModal = false}
    onComplete={handleRenameComplete}
  />
{/if}

{#if showConvertModal}
  <ConvertModal
    files={Array.from(selectedFiles)}
    onClose={() => showConvertModal = false}
    onComplete={handleConvertComplete}
  />
{/if}

{#if showAnalysisModal}
  <AnalysisModal
    files={Array.from(selectedFiles)}
    onClose={() => showAnalysisModal = false}
  />
{/if}

<style>
  .files-page {
    padding: 32px;
    max-width: 1200px;
  }

  .files-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 32px;
    gap: 24px;
  }

  .header-left h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0 0 8px 0;
  }

  .folder-path {
    color: #666;
    font-size: 14px;
    margin: 0;
    font-family: monospace;
  }

  .header-actions {
    display: flex;
    gap: 12px;
  }

  .lyrics-notification {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 16px;
    background: rgba(34, 197, 94, 0.1);
    border: 1px solid rgba(34, 197, 94, 0.2);
    border-radius: 8px;
    color: #22c55e;
    font-size: 13px;
    margin-bottom: 20px;
    animation: slideIn 0.2s ease;
  }

  .lyrics-notification.has-failures {
    background: rgba(234, 179, 8, 0.1);
    border-color: rgba(234, 179, 8, 0.2);
    color: #eab308;
  }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: translateY(-10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
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

  .action-btn.primary {
    background: rgba(244, 114, 182, 0.15);
    border-color: rgba(244, 114, 182, 0.3);
    color: #f472b6;
  }

  .action-btn.primary:hover {
    background: rgba(244, 114, 182, 0.25);
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

  .files-table {
    background: #111;
    border: 1px solid #1a1a1a;
    border-radius: 12px;
    overflow: hidden;
  }

  .table-header {
    display: grid;
    grid-template-columns: 36px 1fr 150px 150px 80px 100px 110px;
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
    background: transparent;
    border: none;
    padding: 0;
  }

  .th.sortable {
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 4px;
    color: #666;
  }

  .th.sortable:hover {
    color: #888;
  }

  .table-body {
    max-height: calc(100vh - 350px);
    overflow-y: auto;
  }

  .table-row {
    display: grid;
    grid-template-columns: 36px 1fr 150px 150px 80px 100px 110px;
    gap: 16px;
    padding: 12px 16px;
    align-items: center;
    border-bottom: 1px solid #1a1a1a;
    transition: background 0.2s;
  }

  .table-row.selected {
    background: rgba(244, 114, 182, 0.05);
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
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .name-cell {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .file-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .file-name {
    font-weight: 500;
    color: #fff;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .file-path {
    font-size: 12px;
    color: #555;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .cell.size {
    font-family: monospace;
    font-size: 13px;
  }

  .cell.date {
    color: #666;
  }

  .cell.actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }

  .file-btn {
    width: 28px;
    height: 28px;
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

  .file-btn:hover {
    background: #1a1a1a;
    color: #fff;
  }

  .file-btn.danger:hover {
    border-color: #ef4444;
    color: #ef4444;
  }

  .file-btn.info:hover {
    border-color: #3b82f6;
    color: #3b82f6;
  }

  .files-footer {
    display: flex;
    gap: 24px;
    padding: 16px 0;
    color: #555;
    font-size: 14px;
  }

  .total-size {
    color: #888;
  }

  .selection-count {
    font-size: 14px;
    color: #f472b6;
    font-weight: 500;
  }

  .checkbox-col {
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
  }

  .checkbox-col input {
    display: none;
  }

  .custom-checkbox {
    width: 18px;
    height: 18px;
    border: 2px solid #444;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
  }

  .custom-checkbox:hover {
    border-color: #666;
  }

  .custom-checkbox.checked {
    background: #f472b6;
    border-color: #f472b6;
    color: #000;
  }

  .custom-checkbox.indeterminate {
    background: #f472b6;
    border-color: #f472b6;
    color: #000;
  }
</style>
