<script lang="ts">
  import { queueStore, downloadFolder, type TidalTrack } from '../stores/queue';
  import { SearchTidal, QueueSingleDownload } from '../../wailsjs/go/main/App.js';

  let searchQuery = '';
  let searchResults: TidalTrack[] = [];
  let isSearching = false;
  let hasSearched = false;

  async function handleSearch() {
    if (!searchQuery.trim()) return;

    isSearching = true;
    hasSearched = true;
    searchResults = [];

    try {
      const results = await SearchTidal(searchQuery);
      searchResults = results || [];
    } catch (error) {
      console.error('Search error:', error);
    } finally {
      isSearching = false;
    }
  }

  function handleKeyPress(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleSearch();
    }
  }

  function formatDuration(seconds: number): string {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }

  async function downloadTrack(track: TidalTrack) {
    if (!$downloadFolder) {
      console.error('No download folder set');
      return;
    }

    queueStore.addItem({
      trackId: track.id,
      title: track.title,
      artist: track.artist,
      status: 'pending'
    });

    try {
      await QueueSingleDownload(track.id, $downloadFolder, track.title, track.artist);
    } catch (error) {
      console.error('Download error:', error);
      queueStore.updateItem(track.id, {
        status: 'error',
        error: String(error)
      });
    }
  }

  async function downloadAll() {
    for (const track of searchResults) {
      await downloadTrack(track);
    }
  }
</script>

<div class="search-page">
  <div class="search-header">
    <h1>Search Tidal</h1>
    <p class="subtitle">Find tracks, albums, and playlists</p>
  </div>

  <div class="search-box">
    <div class="search-input-wrapper">
      <svg class="search-icon" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="11" cy="11" r="8"/>
        <path d="m21 21-4.35-4.35"/>
      </svg>
      <input
        type="text"
        bind:value={searchQuery}
        on:keypress={handleKeyPress}
        placeholder="Search for tracks, albums, artists..."
        class="search-input"
      />
      <button
        class="search-btn"
        on:click={handleSearch}
        disabled={isSearching || !searchQuery.trim()}
      >
        {#if isSearching}
          <div class="spinner"></div>
        {:else}
          Search
        {/if}
      </button>
    </div>
  </div>

  {#if isSearching}
    <div class="loading-state">
      <div class="loader"></div>
      <p>Searching Tidal...</p>
    </div>
  {:else if hasSearched && searchResults.length === 0}
    <div class="empty-state">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="11" cy="11" r="8"/>
        <path d="m21 21-4.35-4.35"/>
      </svg>
      <p>No results found for "{searchQuery}"</p>
    </div>
  {:else if searchResults.length > 0}
    <div class="results-header">
      <span class="results-count">{searchResults.length} results</span>
      <button class="download-all-btn" on:click={downloadAll}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
          <polyline points="7 10 12 15 17 10"/>
          <line x1="12" y1="15" x2="12" y2="3"/>
        </svg>
        Download All
      </button>
    </div>

    <div class="results-list">
      {#each searchResults as track, i}
        <div class="track-row">
          <span class="track-num">{i + 1}</span>
          <img
            src={track.coverUrl}
            alt={track.album}
            class="track-cover"
          />
          <div class="track-info">
            <span class="track-title">{track.title}</span>
            <span class="track-artist">{track.artists}</span>
          </div>
          <span class="track-album">{track.album}</span>
          <span class="track-duration">{formatDuration(track.duration)}</span>
          <button
            class="track-download-btn"
            on:click={() => downloadTrack(track)}
            title="Download"
          >
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/>
              <line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
          </button>
        </div>
      {/each}
    </div>
  {:else}
    <div class="empty-state initial">
      <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
        <circle cx="11" cy="11" r="8"/>
        <path d="m21 21-4.35-4.35"/>
      </svg>
      <p>Search for music on Tidal</p>
      <span class="hint">Enter a track name, artist, or album</span>
    </div>
  {/if}
</div>

<style>
  .search-page {
    padding: 32px;
    max-width: 1200px;
  }

  .search-header {
    margin-bottom: 32px;
  }

  .search-header h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0 0 8px 0;
  }

  .subtitle {
    color: #666;
    margin: 0;
  }

  .search-box {
    margin-bottom: 32px;
  }

  .search-input-wrapper {
    display: flex;
    align-items: center;
    gap: 12px;
    background: #111;
    border: 1px solid #222;
    border-radius: 12px;
    padding: 8px 16px;
    transition: border-color 0.2s;
  }

  .search-input-wrapper:focus-within {
    border-color: #f472b6;
  }

  .search-icon {
    color: #555;
    flex-shrink: 0;
  }

  .search-input {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    color: #fff;
    font-size: 16px;
    padding: 8px 0;
  }

  .search-input::placeholder {
    color: #555;
  }

  .search-btn {
    padding: 10px 24px;
    background: linear-gradient(135deg, #f472b6, #a855f7);
    border: none;
    border-radius: 8px;
    color: #000;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.2s;
    min-width: 100px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .search-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .search-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .spinner {
    width: 18px;
    height: 18px;
    border: 2px solid transparent;
    border-top-color: #000;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .loading-state,
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 80px 20px;
    color: #555;
    text-align: center;
  }

  .empty-state.initial {
    color: #444;
  }

  .empty-state svg {
    margin-bottom: 16px;
    opacity: 0.5;
  }

  .empty-state p {
    margin: 0;
    font-size: 16px;
  }

  .empty-state .hint {
    margin-top: 8px;
    font-size: 14px;
    color: #444;
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

  .results-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
    padding-bottom: 16px;
    border-bottom: 1px solid #1a1a1a;
  }

  .results-count {
    color: #888;
    font-size: 14px;
  }

  .download-all-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: rgba(244, 114, 182, 0.15);
    border: 1px solid rgba(244, 114, 182, 0.3);
    border-radius: 8px;
    color: #f472b6;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .download-all-btn:hover {
    background: rgba(244, 114, 182, 0.25);
  }

  .results-list {
    display: flex;
    flex-direction: column;
  }

  .track-row {
    display: grid;
    grid-template-columns: 40px 48px 1fr 200px 60px 44px;
    align-items: center;
    gap: 16px;
    padding: 12px;
    border-radius: 8px;
    transition: background 0.2s;
  }

  .track-row:hover {
    background: rgba(255, 255, 255, 0.03);
  }

  .track-num {
    color: #555;
    font-size: 14px;
    text-align: center;
  }

  .track-cover {
    width: 48px;
    height: 48px;
    border-radius: 4px;
    object-fit: cover;
  }

  .track-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  .track-title {
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .track-artist {
    font-size: 13px;
    color: #888;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .track-album {
    color: #666;
    font-size: 13px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .track-duration {
    color: #555;
    font-size: 13px;
    text-align: right;
  }

  .track-download-btn {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 1px solid #333;
    border-radius: 8px;
    color: #888;
    cursor: pointer;
    transition: all 0.2s;
  }

  .track-download-btn:hover {
    background: rgba(244, 114, 182, 0.15);
    border-color: #f472b6;
    color: #f472b6;
  }
</style>
