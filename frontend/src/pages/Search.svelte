<script lang="ts">
  import { queueStore, downloadFolder, type TidalTrack } from '../stores/queue';
  import { SearchTidal, SearchTidalAlbums, SearchTidalArtists, QueueSingleDownload, QueueArtistAlbum } from '../../wailsjs/go/main/App.js';
  import { formatNumber } from '../lib/format';

  type SearchType = 'tracks' | 'albums' | 'artists';

  interface SearchAlbum {
    id: number;
    title: string;
    artist: string;
    releaseDate: string;
    trackCount: number;
    coverUrl: string;
    albumType?: string;
  }

  interface SearchArtist {
    id: number;
    name: string;
    pictureUrl?: string;
  }

  let searchQuery = $state('');
  let searchType: SearchType = $state('tracks');
  let searchResults: TidalTrack[] = $state([]);
  let albumResults: SearchAlbum[] = $state([]);
  let artistResults: SearchArtist[] = $state([]);
  let isSearching = $state(false);
  let hasSearched = $state(false);
  let filterText = $state('');
  let filterTimeout: ReturnType<typeof setTimeout> | undefined;
  let debouncedFilter = $state('');
  let downloadingAlbums = $state(new Set<number>());

  function onFilterInput(e: Event) {
    const value = (e.target as HTMLInputElement).value;
    filterText = value;
    clearTimeout(filterTimeout);
    filterTimeout = setTimeout(() => {
      debouncedFilter = value;
    }, 300);
  }

  function clearFilter() {
    filterText = '';
    debouncedFilter = '';
    clearTimeout(filterTimeout);
  }

  const filteredResults = $derived(
    debouncedFilter.trim() === ''
      ? searchResults
      : searchResults.filter(track => {
          const q = debouncedFilter.toLowerCase();
          return (
            track.title.toLowerCase().includes(q) ||
            track.artists.toLowerCase().includes(q) ||
            track.album.toLowerCase().includes(q)
          );
        })
  );

  const filteredAlbums = $derived(
    debouncedFilter.trim() === ''
      ? albumResults
      : albumResults.filter(album => {
          const q = debouncedFilter.toLowerCase();
          return (
            album.title.toLowerCase().includes(q) ||
            album.artist.toLowerCase().includes(q)
          );
        })
  );

  const filteredArtists = $derived(
    debouncedFilter.trim() === ''
      ? artistResults
      : artistResults.filter(artist => {
          const q = debouncedFilter.toLowerCase();
          return artist.name.toLowerCase().includes(q);
        })
  );

  function switchTab(tab: SearchType) {
    searchType = tab;
    clearFilter();
  }

  async function handleSearch() {
    if (!searchQuery.trim()) return;

    isSearching = true;
    hasSearched = true;
    searchResults = [];
    albumResults = [];
    artistResults = [];
    clearFilter();

    try {
      if (searchType === 'tracks') {
        const results = await SearchTidal(searchQuery);
        searchResults = results || [];
      } else if (searchType === 'albums') {
        const results = await SearchTidalAlbums(searchQuery);
        albumResults = results || [];
      } else if (searchType === 'artists') {
        const results = await SearchTidalArtists(searchQuery);
        artistResults = results || [];
      }
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

  function formatYear(releaseDate: string): string {
    if (!releaseDate) return '';
    return releaseDate.substring(0, 4);
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
    for (const track of filteredResults) {
      await downloadTrack(track);
    }
  }

  async function downloadAlbum(album: SearchAlbum) {
    if (!$downloadFolder) {
      console.error('No download folder set');
      return;
    }

    downloadingAlbums = new Set([...downloadingAlbums, album.id]);

    try {
      await QueueArtistAlbum(String(album.id), album.artist, $downloadFolder);
    } catch (error) {
      console.error('Album download error:', error);
    } finally {
      const next = new Set(downloadingAlbums);
      next.delete(album.id);
      downloadingAlbums = next;
    }
  }

  function currentResultCount(): number {
    if (searchType === 'tracks') return searchResults.length;
    if (searchType === 'albums') return albumResults.length;
    return artistResults.length;
  }

  function currentFilteredCount(): number {
    if (searchType === 'tracks') return filteredResults.length;
    if (searchType === 'albums') return filteredAlbums.length;
    return filteredArtists.length;
  }
</script>

<div class="search-page">
  <div class="search-header">
    <h1>Search Tidal</h1>
    <p class="subtitle">Find tracks, albums, and artists</p>
  </div>

  <div class="search-tabs">
    <button class="tab" class:active={searchType === 'tracks'} onclick={() => switchTab('tracks')}>Tracks</button>
    <button class="tab" class:active={searchType === 'albums'} onclick={() => switchTab('albums')}>Albums</button>
    <button class="tab" class:active={searchType === 'artists'} onclick={() => switchTab('artists')}>Artists</button>
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
        onkeypress={handleKeyPress}
        placeholder="Search for {searchType}..."
        class="search-input"
      />
      <button
        class="search-btn"
        onclick={handleSearch}
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
  {:else if hasSearched && currentResultCount() === 0}
    <div class="empty-state">
      <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
        <circle cx="11" cy="11" r="8"/>
        <path d="m21 21-4.35-4.35"/>
      </svg>
      <p>No results found for "{searchQuery}"</p>
    </div>
  {:else if currentResultCount() > 0}
    <div class="results-header">
      <span class="results-count">{formatNumber(currentFilteredCount())} of {formatNumber(currentResultCount())} results</span>
      {#if searchType === 'tracks'}
        <button class="download-all-btn" onclick={downloadAll}>
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="7 10 12 15 17 10"/>
            <line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          Download All
        </button>
      {/if}
    </div>

    <div class="filter-input-wrapper">
      <svg class="filter-icon" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polygon points="22 3 2 3 10 12.46 10 19 14 21 14 12.46 22 3"/>
      </svg>
      <input
        type="text"
        value={filterText}
        oninput={onFilterInput}
        placeholder={searchType === 'artists' ? 'Filter by name...' : 'Filter by artist, album, or title...'}
        class="filter-input"
      />
      {#if filterText}
        <button class="filter-clear-btn" onclick={clearFilter} title="Clear filter">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      {/if}
    </div>

    {#if searchType === 'tracks'}
      <div class="results-list">
        {#each filteredResults as track, i}
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
              onclick={() => downloadTrack(track)}
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
    {:else if searchType === 'albums'}
      <div class="album-grid">
        {#each filteredAlbums as album}
          <div class="album-card">
            {#if album.coverUrl}
              <img src={album.coverUrl} alt={album.title} class="album-cover" />
            {:else}
              <div class="album-cover-placeholder">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <circle cx="12" cy="12" r="10"/>
                  <circle cx="12" cy="12" r="3"/>
                </svg>
              </div>
            {/if}
            <div class="album-card-info">
              <span class="album-card-title" title={album.title}>{album.title}</span>
              <span class="album-card-artist">{album.artist}</span>
              <div class="album-card-meta">
                {#if album.releaseDate}
                  <span>{formatYear(album.releaseDate)}</span>
                {/if}
                {#if album.albumType}
                  <span class="album-type-badge">{album.albumType}</span>
                {/if}
                <span>{album.trackCount} tracks</span>
              </div>
            </div>
            <button
              class="album-download-btn"
              onclick={() => downloadAlbum(album)}
              disabled={downloadingAlbums.has(album.id)}
              title="Download Album"
            >
              {#if downloadingAlbums.has(album.id)}
                <div class="spinner-small"></div>
              {:else}
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="7 10 12 15 17 10"/>
                  <line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
                Download
              {/if}
            </button>
          </div>
        {/each}
      </div>
    {:else if searchType === 'artists'}
      <div class="artist-list">
        {#each filteredArtists as artist}
          <div class="artist-row">
            {#if artist.pictureUrl}
              <img src={artist.pictureUrl} alt={artist.name} class="artist-picture" />
            {:else}
              <div class="artist-picture-placeholder">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
                  <circle cx="12" cy="7" r="4"/>
                </svg>
              </div>
            {/if}
            <div class="artist-info">
              <span class="artist-name">{artist.name}</span>
            </div>
            <a
              class="artist-link-btn"
              href="https://tidal.com/browse/artist/{artist.id}"
              target="_blank"
              rel="noopener noreferrer"
              title="Open on Tidal"
            >
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/>
                <polyline points="15 3 21 3 21 9"/>
                <line x1="10" y1="14" x2="21" y2="3"/>
              </svg>
              Open on Tidal
            </a>
          </div>
        {/each}
      </div>
    {/if}
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
    margin-bottom: 24px;
  }

  .search-header h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0 0 8px 0;
  }

  .subtitle {
    color: var(--color-text-tertiary);
    margin: 0;
  }

  .search-tabs {
    display: flex;
    gap: 4px;
    margin-bottom: 20px;
    background: var(--color-bg-secondary);
    border-radius: 10px;
    padding: 4px;
    width: fit-content;
  }

  .tab {
    padding: 8px 20px;
    background: transparent;
    border: none;
    border-radius: 8px;
    color: var(--color-text-secondary);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .tab:hover {
    color: var(--color-text-primary);
    background: rgba(255, 255, 255, 0.05);
  }

  .tab.active {
    background: rgba(244, 114, 182, 0.15);
    color: #f472b6;
  }

  .search-box {
    margin-bottom: 32px;
  }

  .search-input-wrapper {
    display: flex;
    align-items: center;
    gap: 12px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 12px;
    padding: 8px 16px;
    transition: border-color 0.2s;
  }

  .search-input-wrapper:focus-within {
    border-color: var(--color-accent);
  }

  .search-icon {
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  .search-input {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    color: var(--color-text-primary);
    font-size: 16px;
    padding: 8px 0;
  }

  .search-input::placeholder {
    color: var(--color-text-muted);
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

  .spinner-small {
    width: 14px;
    height: 14px;
    border: 2px solid transparent;
    border-top-color: currentColor;
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
    color: var(--color-text-muted);
    text-align: center;
  }

  .empty-state.initial {
    color: var(--color-text-muted);
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
    color: var(--color-text-muted);
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

  .results-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 16px;
    padding-bottom: 16px;
    border-bottom: 1px solid var(--color-border);
  }

  .results-count {
    color: var(--color-text-secondary);
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

  .filter-input-wrapper {
    display: flex;
    align-items: center;
    gap: 8px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
    padding: 6px 12px;
    margin-bottom: 16px;
    transition: border-color 0.2s;
  }

  .filter-input-wrapper:focus-within {
    border-color: var(--color-accent);
  }

  .filter-icon {
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  .filter-input {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    color: var(--color-text-primary);
    font-size: 14px;
    padding: 4px 0;
  }

  .filter-input::placeholder {
    color: var(--color-text-muted);
  }

  .filter-clear-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    background: rgba(255, 255, 255, 0.08);
    border: none;
    border-radius: 4px;
    color: var(--color-text-muted);
    cursor: pointer;
    transition: all 0.15s;
    flex-shrink: 0;
  }

  .filter-clear-btn:hover {
    background: rgba(255, 255, 255, 0.15);
    color: var(--color-text-primary);
  }

  /* Track results */
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
    color: var(--color-text-muted);
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
    color: var(--color-text-tertiary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .track-album {
    color: var(--color-text-tertiary);
    font-size: 13px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .track-duration {
    color: var(--color-text-muted);
    font-size: 13px;
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  .track-download-btn {
    width: 36px;
    height: 36px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: 1px solid var(--color-bg-hover);
    border-radius: 8px;
    color: var(--color-text-secondary);
    cursor: pointer;
    transition: all 0.2s;
  }

  .track-download-btn:hover {
    background: rgba(244, 114, 182, 0.15);
    border-color: #f472b6;
    color: #f472b6;
  }

  /* Album results */
  .album-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 16px;
  }

  .album-card {
    display: flex;
    flex-direction: column;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 12px;
    overflow: hidden;
    transition: border-color 0.2s;
  }

  .album-card:hover {
    border-color: var(--color-border);
  }

  .album-cover {
    width: 100%;
    aspect-ratio: 1;
    object-fit: cover;
  }

  .album-cover-placeholder {
    width: 100%;
    aspect-ratio: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.03);
    color: var(--color-text-muted);
  }

  .album-card-info {
    padding: 12px 16px 8px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  .album-card-title {
    font-weight: 600;
    font-size: 15px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .album-card-artist {
    font-size: 13px;
    color: var(--color-text-tertiary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .album-card-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: var(--color-text-muted);
    margin-top: 2px;
  }

  .album-type-badge {
    padding: 1px 6px;
    background: rgba(168, 85, 247, 0.15);
    border-radius: 4px;
    color: #a855f7;
    font-size: 11px;
    font-weight: 500;
    text-transform: uppercase;
  }

  .album-download-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin: 8px 16px 16px;
    padding: 10px;
    background: rgba(244, 114, 182, 0.1);
    border: 1px solid rgba(244, 114, 182, 0.2);
    border-radius: 8px;
    color: #f472b6;
    font-weight: 500;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .album-download-btn:hover:not(:disabled) {
    background: rgba(244, 114, 182, 0.2);
  }

  .album-download-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Artist results */
  .artist-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .artist-row {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 12px;
    border-radius: 8px;
    transition: background 0.2s;
  }

  .artist-row:hover {
    background: rgba(255, 255, 255, 0.03);
  }

  .artist-picture {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    object-fit: cover;
    flex-shrink: 0;
  }

  .artist-picture-placeholder {
    width: 56px;
    height: 56px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.05);
    color: var(--color-text-muted);
    flex-shrink: 0;
  }

  .artist-info {
    flex: 1;
    min-width: 0;
  }

  .artist-name {
    font-weight: 600;
    font-size: 16px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .artist-link-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: rgba(168, 85, 247, 0.1);
    border: 1px solid rgba(168, 85, 247, 0.2);
    border-radius: 8px;
    color: #a855f7;
    font-weight: 500;
    font-size: 13px;
    text-decoration: none;
    cursor: pointer;
    transition: all 0.2s;
    flex-shrink: 0;
  }

  .artist-link-btn:hover {
    background: rgba(168, 85, 247, 0.2);
  }
</style>
