<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    FetchTidalContent,
    ValidateTidalURL,
    SelectDownloadFolder,
    GetDownloadFolder,
    SetDownloadFolder,
    OpenDownloadFolder,
    QueueDownloads,
    QueueQobuzDownloads,
    QueueSingleDownload,
    QueueArtistAlbum,
    DownloadArtistAssets,
    GetAppVersion,
    DetectSourceFromURL,
    FetchContentFromURL,
  } from '../../wailsjs/go/main/App.js';
  import { queueStore, queueStats, downloadFolder, currentContent, type TidalTrack } from '../stores/queue';
  import ContextMenu from '../components/ContextMenu.svelte';

  // Accept initial content from history refetch
  let { initialContent = null, onContentCleared = () => {} }: { initialContent?: any; onContentCleared?: () => void } = $props();

  let tidalUrl = $state('');
  let urlInputEl: HTMLInputElement | null = $state(null);
  let loading = $state(false);
  let error = $state('');
  let version = $state('');

  // Source detection
  let detectedSource: { source: string; displayName: string; contentType: string; available: boolean } | null = $state(null);
  let detectingSource = $state(false);

  let content = $derived($currentContent);
  let stats = $derived($queueStats);
  let folder = $derived($downloadFolder);

  // Audio preview state
  let previewAudio: HTMLAudioElement | null = $state(null);
  let previewingTrackId: number | null = $state(null);
  let previewPlaying = $state(false);

  // Animated placeholder (typewriter effect)
  const exampleUrls = [
    'https://tidal.com/browse/album/123456789',
    'https://tidal.com/browse/track/987654321',
    'https://tidal.com/browse/playlist/abc-def-123',
    'https://tidal.com/browse/artist/12345',
    'https://tidal.com/browse/mix/abcdef123',
    'https://www.qobuz.com/us-en/album/discovery-daft-punk/0060075335265',
  ];
  let placeholderText = $state('|');
  let typewriterTimeout: ReturnType<typeof setTimeout> | undefined;

  function startTypewriter() {
    let urlIndex = 0;
    let charIndex = 0;
    let deleting = false;

    function tick() {
      const currentUrl = exampleUrls[urlIndex];

      if (!deleting) {
        // Typing
        charIndex++;
        placeholderText = currentUrl.slice(0, charIndex) + '|';
        if (charIndex >= currentUrl.length) {
          // Fully typed — pause then delete
          typewriterTimeout = setTimeout(() => { deleting = true; tick(); }, 2000);
          return;
        }
        typewriterTimeout = setTimeout(tick, 45);
      } else {
        // Deleting
        charIndex--;
        placeholderText = currentUrl.slice(0, charIndex) + '|';
        if (charIndex <= 0) {
          // Fully deleted — pause then move to next URL
          deleting = false;
          urlIndex = (urlIndex + 1) % exampleUrls.length;
          typewriterTimeout = setTimeout(tick, 400);
          return;
        }
        typewriterTimeout = setTimeout(tick, 25);
      }
    }

    tick();
  }

  // Context menu state
  let contextMenu: { x: number; y: number; track: TidalTrack } | null = $state(null);

  function showContextMenu(e: MouseEvent, track: TidalTrack) {
    e.preventDefault();
    contextMenu = { x: e.clientX, y: e.clientY, track };
  }

  function getContextMenuItems(track: TidalTrack) {
    return [
      { label: 'Download Track', icon: '\u2B07', action: () => downloadSingleTrack(track) },
      { label: 'Copy Track URL', icon: '\uD83D\uDD17', action: () => navigator.clipboard.writeText(track.tidalUrl) },
      { label: 'Copy ISRC', icon: '\uD83C\uDFAB', action: () => navigator.clipboard.writeText(track.isrc), disabled: !track.isrc },
      { label: '', action: () => {}, divider: true },
      { label: 'Open on Tidal', icon: '\uD83C\uDF10', action: () => {
        if ((window as any).__wails?.BrowserOpenURL) {
          (window as any).__wails.BrowserOpenURL(track.tidalUrl);
        } else {
          window.open(track.tidalUrl, '_blank');
        }
      }},
    ];
  }

  // Pagination state
  const tracksPerPage = 500;
  let currentPage = $state(1);

  // Reset page when content changes
  $effect(() => {
    if (content) currentPage = 1;
  });

  function togglePreview(track: TidalTrack) {
    if (!previewAudio || !track.previewUrl) return;

    if (previewingTrackId === track.id) {
      // Same track — toggle play/pause
      if (previewPlaying) {
        previewAudio.pause();
        previewPlaying = false;
      } else {
        previewAudio.play();
        previewPlaying = true;
      }
    } else {
      // New track — swap source and play
      previewAudio.pause();
      previewAudio.src = track.previewUrl;
      previewingTrackId = track.id;
      previewPlaying = true;
      previewAudio.play();
    }
  }

  function stopPreview() {
    if (previewAudio) {
      previewAudio.pause();
      previewAudio.src = '';
    }
    previewingTrackId = null;
    previewPlaying = false;
  }

  // Detect source when URL changes
  $effect(() => {
    if (tidalUrl.trim()) {
      detectSource(tidalUrl);
    } else {
      detectedSource = null;
    }
  });

  async function detectSource(url: string) {
    detectingSource = true;
    try {
      const result = await DetectSourceFromURL(url);
      if (result.source) {
        detectedSource = {
          source: result.source,
          displayName: result.displayName,
          contentType: result.contentType,
          available: result.available
        };
      } else {
        detectedSource = null;
      }
    } catch {
      detectedSource = null;
    }
    detectingSource = false;
  }

  // Handle initial content from history
  $effect(() => {
    if (initialContent) {
      currentContent.set({
        type: initialContent.type,
        id: initialContent.id,
        title: initialContent.title,
        creator: initialContent.creator,
        coverUrl: initialContent.coverUrl,
        tracks: initialContent.tracks || []
      });
      queueStore.reset();
      onContentCleared();
    }
  });

  onMount(async () => {
    // Load version
    try {
      version = await GetAppVersion();
    } catch (e) {
      version = '';
    }

    const savedFolder = await GetDownloadFolder();
    if (savedFolder) {
      downloadFolder.set(savedFolder);
    }

    startTypewriter();
  });

  onDestroy(() => {
    if (typewriterTimeout) clearTimeout(typewriterTimeout);
  });

  async function fetchContent() {
    if (!tidalUrl.trim()) return;

    loading = true;
    error = '';
    stopPreview();
    currentContent.set(null);
    queueStore.reset();

    try {
      // Use multi-source fetch if a source is detected, otherwise fall back to Tidal validation
      if (detectedSource) {
        if (!detectedSource.available) {
          throw new Error(`${detectedSource.displayName} is not available. Check your settings.`);
        }
        const result = await FetchContentFromURL(tidalUrl);
        currentContent.set({
          type: result.type,
          id: result.id,
          title: result.title,
          creator: result.creator,
          coverUrl: result.coverUrl,
          tracks: result.tracks || [],
          source: result.source,
          artistId: result.artistId
        });
      } else {
        // Fallback to Tidal-only validation
        const validation = await ValidateTidalURL(tidalUrl);
        if (!validation.valid) {
          throw new Error(validation.error);
        }
        const result = await FetchTidalContent(tidalUrl);
        currentContent.set({
          type: result.type,
          id: result.id,
          title: result.title,
          creator: result.creator,
          coverUrl: result.coverUrl,
          tracks: result.tracks || [],
          source: 'tidal',
          artistId: result.artistId
        });
      }
    } catch (e: any) {
      error = e.message || (typeof e === 'string' ? e : 'Failed to fetch content');
    }

    loading = false;
  }

  async function selectFolder() {
    try {
      const selected = await SelectDownloadFolder();
      if (selected) {
        downloadFolder.set(selected);
        await SetDownloadFolder(selected);
      }
    } catch (e: any) {
      error = e.message || 'Failed to select folder';
    }
  }

  async function openFolder() {
    if ($downloadFolder) {
      await OpenDownloadFolder($downloadFolder);
    }
  }

  async function downloadSingleTrack(track: TidalTrack) {
    if (!$downloadFolder) {
      error = 'Please select a download folder first';
      return;
    }

    queueStore.addItem({
      trackId: track.id,
      title: track.title,
      artist: track.artists,
      status: 'queued'
    });

    try {
      await QueueSingleDownload(track.id, $downloadFolder, track.title, track.artists);
    } catch (e: any) {
      queueStore.updateItem(track.id, { status: 'error', error: e.message });
    }
  }

  async function downloadAllTracks() {
    if (!$downloadFolder) {
      error = 'Please select a download folder first';
      return;
    }
    if (!content?.tracks?.length) return;

    // Add all tracks to queue
    const tracksToDownload = content.tracks.filter(track => {
      const existing = trackStatuses[track.id];
      return !existing || existing.status === 'error';
    });

    tracksToDownload.forEach(track => {
      queueStore.addItem({
        trackId: track.id,
        title: track.title,
        artist: track.artists,
        status: 'queued'
      });
    });

    try {
      if (content.source === 'qobuz') {
        await QueueQobuzDownloads(tracksToDownload, $downloadFolder, content.title);
      } else {
        await QueueDownloads(tracksToDownload, $downloadFolder, content.title, content.id ?? '', content.type);
      }
    } catch (e: any) {
      error = e.message || 'Failed to queue downloads';
    }
  }

  // Reactive: convert Map to object for proper Svelte reactivity
  let trackStatuses = $derived(Object.fromEntries($queueStore));

  function formatDuration(seconds: number): string {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }

  function formatFileSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  }

  function getContentTypeLabel(type: string): string {
    switch (type) {
      case 'playlist': return 'Playlist';
      case 'album': return 'Album';
      case 'track': return 'Track';
      case 'artist': return 'Artist';
      default: return 'Content';
    }
  }

  // Artist discography state
  let albumTypeFilter = $state('all');

  // Track sort state
  let sortByPopularity = $state(false);
  let displayedTracks = $derived(
    sortByPopularity
      ? [...(content?.tracks || [])].sort((a, b) => (b.popularity || 0) - (a.popularity || 0))
      : (content?.tracks || [])
  );

  let totalPages = $derived(Math.max(1, Math.ceil(displayedTracks.length / tracksPerPage)));
  let paginatedTracks = $derived(
    displayedTracks.slice((currentPage - 1) * tracksPerPage, currentPage * tracksPerPage)
  );

  let filteredAlbums = $derived(() => {
    const albums = (content as any)?.albums || [];
    if (albumTypeFilter === 'all') return albums;
    if (albumTypeFilter === 'albums') return albums.filter((a: any) => a.albumType?.toUpperCase() === 'ALBUM');
    if (albumTypeFilter === 'epssingles') return albums.filter((a: any) => ['EP', 'SINGLE'].includes(a.albumType?.toUpperCase()));
    if (albumTypeFilter === 'compilations') return albums.filter((a: any) => a.albumType?.toUpperCase() === 'COMPILATION');
    return albums;
  });

  function getAlbumTypeLabel(type: string): string {
    switch (type?.toUpperCase()) {
      case 'ALBUM': return 'Album';
      case 'EP': return 'EP';
      case 'SINGLE': return 'Single';
      case 'COMPILATION': return 'Compilation';
      default: return 'Album';
    }
  }

  async function downloadArtistAlbum(albumId: number) {
    if (!$downloadFolder) { error = 'Please select a download folder first'; return; }
    try {
      await QueueArtistAlbum(String(albumId), content?.title || '', $downloadFolder);
    } catch (e: any) {
      error = e.message || 'Failed to queue album';
    }
  }

  async function downloadAllFilteredAlbums() {
    if (!$downloadFolder) { error = 'Please select a download folder first'; return; }
    const albums = filteredAlbums();
    for (const album of albums) {
      await downloadArtistAlbum(album.id);
    }
  }

  let downloadingAssets = $state(false);
  let assetsResult = $state('');

  async function downloadArtistAssetsHandler() {
    if (!$downloadFolder) { error = 'Please select a download folder first'; return; }
    const artistId = String(content?.artistId || '');
    if (!artistId) { error = 'No artist ID available'; return; }
    downloadingAssets = true;
    assetsResult = '';
    try {
      const count = await DownloadArtistAssets(artistId, content?.title || '', $downloadFolder);
      assetsResult = `${count} image${count !== 1 ? 's' : ''} saved`;
    } catch (e: any) {
      error = e.message || 'Failed to download artist assets';
    }
    downloadingAssets = false;
  }
</script>

<!-- Shared audio element for track previews -->
<audio bind:this={previewAudio} onended={() => { previewPlaying = false; previewingTrackId = null; }}></audio>

<div class="home-page">
  <!-- Header -->
  <header class="page-header">
    <div class="title-row">
      <h1>FLACidal</h1>
      {#if version}
        <span class="version-badge">v{version}</span>
      {/if}
    </div>
    <p class="subtitle">Download lossless FLAC from Tidal & Qobuz</p>
  </header>

  <!-- URL Input -->
  <div class="input-section">
    <div class="url-input-wrapper">
      <div class="input-with-badge" class:has-url={!!tidalUrl}>
        <input
          type="text"
          bind:value={tidalUrl}
          bind:this={urlInputEl}
          placeholder={placeholderText || 'Paste a Tidal or Qobuz URL...'}
          onkeydown={(e) => e.key === 'Enter' && fetchContent()}
          class="url-input"
        />
        {#if tidalUrl}
          <button
            class="clear-input-btn"
            onclick={() => { tidalUrl = ''; urlInputEl?.focus(); }}
            aria-label="Clear URL"
            tabindex="-1"
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        {/if}
        {#if detectedSource}
          <div class="source-badge" class:tidal={detectedSource.source === 'tidal'} class:qobuz={detectedSource.source === 'qobuz'} class:unavailable={!detectedSource.available}>
            {#if detectedSource.source === 'tidal'}
              <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12.012 3.992L8.008 7.996 4.004 3.992 0 7.996l4.004 4.004L0 16.004l4.004 4.004 4.004-4.004 4.004 4.004 4.004-4.004-4.004-4.004 4.004-4.004-4.004-4.004z"/>
              </svg>
            {:else if detectedSource.source === 'qobuz'}
              <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor">
                <circle cx="12" cy="12" r="10"/>
              </svg>
            {/if}
            <span>{detectedSource.displayName}</span>
            {#if !detectedSource.available}
              <span class="unavailable-text">(unavailable)</span>
            {/if}
          </div>
        {:else if detectingSource}
          <div class="source-badge detecting">
            <span class="spinner small"></span>
          </div>
        {/if}
      </div>
      <button class="btn-primary" onclick={fetchContent} disabled={loading || (detectedSource && !detectedSource.available)}>
        {#if loading}
          <span class="spinner"></span>
        {:else}
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"/>
            <path d="m21 21-4.35-4.35"/>
          </svg>
          Fetch
        {/if}
      </button>
    </div>
  </div>

  <!-- Error -->
  {#if error}
    <div class="error-banner">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="10"/>
        <line x1="12" y1="8" x2="12" y2="12"/>
        <line x1="12" y1="16" x2="12.01" y2="16"/>
      </svg>
      <span>{error}</span>
      <button class="btn-icon" onclick={() => error = ''} aria-label="Dismiss error">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>
  {/if}

  <!-- Content Display -->
  {#if content}
    <div class="content-card">
      <!-- Content Header -->
      <div class="content-header">
        {#if content.coverUrl}
          <div class="cover-wrapper">
            <img src={content.coverUrl} alt="Cover" class="cover-art" />
          </div>
        {/if}
        <div class="content-info">
          <div class="badges-row">
            <span class="badge" class:badge-ep={content.albumType === 'EP'} class:badge-single={content.albumType === 'SINGLE'} class:badge-compilation={content.albumType === 'COMPILATION'}>
              {content.type === 'album' && content.albumType ? getAlbumTypeLabel(content.albumType) : getContentTypeLabel(content.type)}
            </span>
            {#if content.source}
              <span class="badge source-tag" class:tidal={content.source === 'tidal'} class:qobuz={content.source === 'qobuz'}>
                {content.source === 'tidal' ? 'Tidal' : content.source === 'qobuz' ? 'Qobuz' : content.source}
              </span>
            {/if}
          </div>
          <h2>{content.title}</h2>
          <p class="creator">{content.creator}</p>
          {#if content.type === 'artist'}
            <p class="track-count">{(content as any).albums?.length || 0} albums</p>
          {:else}
            <p class="track-count">{content.tracks?.length || 0} tracks</p>
          {/if}
        </div>
        <div class="folder-section">
          <button class="btn-secondary" onclick={selectFolder}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            </svg>
            {folder ? folder.split('/').pop() : 'Select Folder'}
          </button>
          {#if folder}
            <button class="btn-ghost" onclick={openFolder}>Open</button>
          {/if}
        </div>
      </div>

      {#if content.type === 'artist'}
        <!-- Artist Album Type Filter -->
        <div class="album-filter-bar">
          {#each [['all','All'],['albums','Albums'],['epssingles','EPs & Singles'],['compilations','Compilations']] as [val, label]}
            <button
              class="filter-btn"
              class:active={albumTypeFilter === val}
              onclick={() => albumTypeFilter = val}
            >{label}</button>
          {/each}
        </div>

        <!-- Album List -->
        <div class="albums-container">
          {#each filteredAlbums() as album}
            <div class="album-row">
              {#if album.coverUrl}
                <img src={album.coverUrl} alt="Cover" class="album-thumb" />
              {:else}
                <div class="album-thumb placeholder"></div>
              {/if}
              <div class="album-details">
                <span class="album-title">{album.title}</span>
                <span class="album-meta">
                  {album.artist}
                  {#if album.releaseDate}· {album.releaseDate.slice(0,4)}{/if}
                  · {album.trackCount} tracks
                </span>
              </div>
              {#if album.albumType}
                <span class="album-type-badge">{getAlbumTypeLabel(album.albumType)}</span>
              {/if}
              <button class="btn-icon download" onclick={() => downloadArtistAlbum(album.id)} disabled={!folder} aria-label="Download album">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                  <polyline points="7 10 12 15 17 10"/>
                  <line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
              </button>
            </div>
          {/each}
        </div>

        <!-- Download All Albums + Artist Assets -->
        <div class="download-section">
          <button class="btn-primary btn-large" onclick={downloadAllFilteredAlbums} disabled={!folder}>
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
              <polyline points="7 10 12 15 17 10"/>
              <line x1="12" y1="15" x2="12" y2="3"/>
            </svg>
            Download All {albumTypeFilter === 'all' ? '' : getContentTypeLabel(albumTypeFilter)} Albums
          </button>
          <button class="btn-secondary" onclick={downloadArtistAssetsHandler} disabled={!folder || downloadingAssets}>
            {#if downloadingAssets}
              <span class="spinner small"></span>
              Downloading...
            {:else}
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="3" y="3" width="18" height="18" rx="2"/>
                <circle cx="8.5" cy="8.5" r="1.5"/>
                <polyline points="21 15 16 10 5 21"/>
              </svg>
              Download Artist Images
            {/if}
          </button>
          {#if assetsResult}
            <span class="assets-result">{assetsResult}</span>
          {/if}
        </div>

      {:else}
        <!-- Sort Controls -->
        <div class="sort-bar">
          <button
            class="sort-btn"
            class:active={sortByPopularity}
            onclick={() => sortByPopularity = !sortByPopularity}
            title={sortByPopularity ? 'Switch to track order' : 'Sort by popularity'}
          >
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polyline points="22 7 13 7 13 2"/>
              <polyline points="2 17 11 17 11 22"/>
              <path d="M22 7L11 18"/>
              <path d="M2 17L13 6"/>
            </svg>
            {sortByPopularity ? 'By Popularity' : 'Track Order'}
          </button>
        </div>

        <!-- Track List -->
        <div class="tracks-container">
          {#each paginatedTracks as track, i}
            {@const status = trackStatuses[track.id]}
            <div class="track-row" class:completed={status?.status === 'completed'} class:downloading={status?.status === 'downloading'} class:unavailable-track={track.available === false} oncontextmenu={(e) => showContextMenu(e, track)}>
              <span class="track-num">{String((currentPage - 1) * tracksPerPage + i + 1).padStart(2, '0')}</span>
              <div class="track-details">
                <div class="title-row">
                  <span class="track-title">{track.title}</span>
                  {#if track.explicit}
                    <span class="explicit-badge" title="Explicit content">E</span>
                  {/if}
                </div>
                <span class="track-artist">{track.artists}</span>
                {#if track.available === false}
                  <span class="unavailable-label" title="Not available for streaming in your region">Unavailable</span>
                {/if}
              </div>
              <span class="track-duration">{formatDuration(track.duration)}</span>
              {#if track.popularity > 0}
                <span class="popularity-badge" title="Popularity: {track.popularity}/100">
                  <svg width="10" height="10" viewBox="0 0 24 24" fill="currentColor" stroke="none">
                    <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 14.5v-9l6 4.5-6 4.5z"/>
                  </svg>
                  {track.popularity}
                </span>
              {/if}
              {#if track.previewUrl}
                <button
                  class="btn-icon preview-btn"
                  class:active={previewingTrackId === track.id}
                  onclick={() => togglePreview(track)}
                  title={previewingTrackId === track.id && previewPlaying ? 'Pause preview' : 'Play 30s preview'}
                  aria-label={previewingTrackId === track.id && previewPlaying ? 'Pause preview' : 'Play preview'}
                >
                  {#if previewingTrackId === track.id && previewPlaying}
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="6" y="4" width="4" height="16"/>
                      <rect x="14" y="4" width="4" height="16"/>
                    </svg>
                  {:else}
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polygon points="5 3 19 12 5 21 5 3"/>
                    </svg>
                  {/if}
                </button>
              {/if}
              <div class="track-status">
                {#if status?.status === 'completed'}
                  <span class="status-badge success">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="20 6 9 17 4 12"/>
                    </svg>
                    {formatFileSize(status.result?.fileSize || 0)}
                  </span>
                {:else if status?.status === 'downloading'}
                  <span class="status-badge downloading">
                    <span class="spinner small"></span>
                    Downloading
                  </span>
                {:else if status?.status === 'queued'}
                  <span class="status-badge queued">Queued</span>
                {:else if status?.status === 'error'}
                  <span class="status-badge error">Failed</span>
                  <button class="btn-icon" onclick={() => downloadSingleTrack(track)} aria-label="Retry download">
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <polyline points="23 4 23 10 17 10"/>
                      <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"/>
                    </svg>
                  </button>
                {:else}
                  <button class="btn-icon download" onclick={() => downloadSingleTrack(track)} disabled={!folder} aria-label="Download track">
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                      <polyline points="7 10 12 15 17 10"/>
                      <line x1="12" y1="15" x2="12" y2="3"/>
                    </svg>
                  </button>
                {/if}
              </div>
            </div>
          {/each}
        </div>

        {#if displayedTracks.length > tracksPerPage}
          <div class="pagination-controls">
            <button
              class="btn-secondary pagination-btn"
              onclick={() => currentPage = Math.max(1, currentPage - 1)}
              disabled={currentPage <= 1}
            >Previous</button>
            <span class="pagination-info">Page {currentPage} of {totalPages}</span>
            <button
              class="btn-secondary pagination-btn"
              onclick={() => currentPage = Math.min(totalPages, currentPage + 1)}
              disabled={currentPage >= totalPages}
            >Next</button>
          </div>
        {/if}

        <!-- Download Section -->
        <div class="download-section">
          {#if stats.total > 0}
            <div class="progress-container">
              <div class="progress-info">
                <span>{stats.completed}/{stats.total} tracks</span>
                {#if stats.failed > 0}
                  <span class="error">{stats.failed} failed</span>
                {/if}
              </div>
              <div class="progress-bar">
                <div class="progress-fill" class:complete={stats.completed === stats.total} style="width: {(stats.completed / stats.total) * 100}%"></div>
              </div>
            </div>
          {/if}

          <button class="btn-primary btn-large" onclick={downloadAllTracks} disabled={stats.downloading > 0 || !folder}>
            {#if stats.downloading > 0}
              <span class="spinner"></span>
              Downloading...
            {:else if stats.completed === content.tracks?.length}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
              All Downloaded
            {:else}
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                <polyline points="7 10 12 15 17 10"/>
                <line x1="12" y1="15" x2="12" y2="3"/>
              </svg>
              Download All FLAC
            {/if}
          </button>
        </div>
      {/if}
    </div>
  {:else if !loading}
    <!-- Empty State -->
    <div class="empty-state">
      <div class="empty-icon">
        <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M9 18V5l12-2v13"/>
          <circle cx="6" cy="18" r="3"/>
          <circle cx="18" cy="16" r="3"/>
        </svg>
      </div>
      <h3>Ready to Download</h3>
      <p>Paste a Tidal URL above to get started</p>
    </div>
  {/if}
</div>

{#if contextMenu}
  <ContextMenu
    items={getContextMenuItems(contextMenu.track)}
    x={contextMenu.x}
    y={contextMenu.y}
    onClose={() => contextMenu = null}
  />
{/if}

<style>
  .home-page {
    padding: 24px;
    max-width: 900px;
    margin: 0 auto;
  }

  .page-header {
    margin-bottom: 32px;
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .page-header h1 {
    margin: 0;
    font-size: 2rem;
    font-weight: 600;
  }

  .version-badge {
    display: inline-flex;
    align-items: center;
    padding: 4px 10px;
    font-size: 12px;
    font-weight: 600;
    border-radius: 6px;
    background: linear-gradient(135deg, rgba(244, 114, 182, 0.2), rgba(168, 85, 247, 0.2));
    color: #f472b6;
    border: 1px solid rgba(244, 114, 182, 0.3);
  }

  .subtitle {
    margin: 4px 0 0 0;
    color: var(--color-text-tertiary, #666);
    font-size: 0.95rem;
  }

  .input-section {
    margin-bottom: 24px;
  }

  .url-input-wrapper {
    display: flex;
    gap: 12px;
  }

  .input-with-badge {
    flex: 1;
    position: relative;
  }

  .url-input {
    width: 100%;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 14px 140px 14px 18px;
    color: var(--color-text-primary);
    font-family: inherit;
    font-size: 0.95rem;
    transition: all 0.2s;
  }

  .url-input:focus {
    outline: none;
    border-color: var(--color-accent);
    box-shadow: 0 0 0 3px var(--color-accent-subtle);
  }

  .url-input::placeholder {
    color: var(--color-text-muted);
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.88rem;
  }

  .clear-input-btn {
    position: absolute;
    right: 12px;
    top: 50%;
    transform: translateY(-50%);
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    border-radius: 6px;
    color: var(--color-text-muted);
    cursor: pointer;
    transition: all 0.2s;
  }

  .clear-input-btn:hover {
    background: var(--color-bg-tertiary);
    color: var(--color-text-secondary);
  }

  .source-badge {
    position: absolute;
    right: 12px;
    top: 50%;
    transform: translateY(-50%);
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    background: rgba(136, 136, 136, 0.15);
    color: #888;
    transition: right 0.15s;
  }

  .input-with-badge.has-url .source-badge {
    right: 48px;
  }

  .source-badge.tidal {
    background: rgba(0, 255, 255, 0.1);
    color: #00d4d4;
  }

  .source-badge.qobuz {
    background: rgba(0, 119, 182, 0.15);
    color: #4da6d9;
  }

  .source-badge.unavailable {
    opacity: 0.6;
  }

  .source-badge .unavailable-text {
    font-size: 11px;
    color: #f87171;
  }

  .source-badge.detecting {
    padding: 6px 12px;
  }

  .badges-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .badge.source-tag {
    background: rgba(136, 136, 136, 0.15);
    color: #888;
  }

  .badge.source-tag.tidal {
    background: rgba(0, 255, 255, 0.1);
    color: #00d4d4;
  }

  .badge.source-tag.qobuz {
    background: rgba(0, 119, 182, 0.15);
    color: #4da6d9;
  }

  .btn-primary {
    background: #f472b6;
    color: #000;
    font-weight: 500;
    padding: 12px 24px;
    border: none;
    border-radius: 10px;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    gap: 8px;
    font-family: inherit;
    font-size: 0.95rem;
  }

  .btn-primary:hover:not(:disabled) {
    background: #ec4899;
    transform: translateY(-1px);
    box-shadow: 0 4px 16px rgba(244, 114, 182, 0.4);
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary.btn-large {
    padding: 16px 32px;
    font-size: 1rem;
    border-radius: 12px;
  }

  .btn-secondary {
    background: var(--color-bg-tertiary);
    color: var(--color-text-primary);
    font-weight: 500;
    padding: 10px 16px;
    border: 1px solid var(--color-bg-hover);
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    gap: 8px;
    font-family: inherit;
    font-size: 0.85rem;
  }

  .btn-secondary:hover {
    background: var(--color-bg-elevated);
    border-color: var(--color-bg-hover);
  }

  .btn-ghost {
    background: transparent;
    color: var(--color-text-secondary);
    font-weight: 500;
    padding: 6px 12px;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s;
    font-family: inherit;
    font-size: 0.8rem;
  }

  .btn-ghost:hover {
    background: var(--color-bg-tertiary);
    color: var(--color-text-primary);
  }

  .btn-icon {
    background: transparent;
    border: none;
    color: var(--color-text-secondary);
    padding: 8px;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .btn-icon:hover:not(:disabled) {
    background: var(--color-bg-tertiary);
    color: var(--color-text-primary);
  }

  .btn-icon.download:hover:not(:disabled) {
    color: var(--color-accent);
    background: var(--color-accent-subtle);
  }

  .btn-icon.preview-btn:hover:not(:disabled) {
    color: #3b82f6;
    background: rgba(59, 130, 246, 0.1);
  }

  .btn-icon.preview-btn.active {
    color: #3b82f6;
  }

  .btn-icon:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .spinner {
    width: 18px;
    height: 18px;
    border: 2px solid transparent;
    border-top-color: currentColor;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  .spinner.small {
    width: 12px;
    height: 12px;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .error-banner {
    display: flex;
    align-items: center;
    gap: 12px;
    background: rgba(248, 113, 113, 0.1);
    border: 1px solid rgba(248, 113, 113, 0.2);
    border-radius: 12px;
    padding: 12px 16px;
    margin-bottom: 24px;
    color: #f87171;
  }

  .error-banner span {
    flex: 1;
    font-size: 0.9rem;
  }

  .content-card {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
    overflow: hidden;
  }

  .content-header {
    display: flex;
    gap: 20px;
    padding: 24px;
    background: linear-gradient(135deg, var(--color-bg-tertiary) 0%, var(--color-bg-secondary) 100%);
  }

  .cover-wrapper {
    flex-shrink: 0;
  }

  .cover-art {
    width: 140px;
    height: 140px;
    border-radius: 12px;
    object-fit: cover;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  }

  .content-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-width: 0;
  }

  .badge {
    display: inline-flex;
    padding: 4px 10px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border-radius: 6px;
    width: fit-content;
    background: rgba(244, 114, 182, 0.15);
    color: #f472b6;
  }

  .badge.badge-ep {
    background: rgba(59, 130, 246, 0.15);
    color: #3b82f6;
  }

  .badge.badge-single {
    background: rgba(168, 85, 247, 0.15);
    color: #a855f7;
  }

  .badge.badge-compilation {
    background: rgba(245, 158, 11, 0.15);
    color: #f59e0b;
  }

  .content-info h2 {
    margin: 0 0 6px 0;
    font-size: 1.5rem;
    font-weight: 600;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .creator {
    margin: 0 0 8px 0;
    color: var(--color-text-secondary);
    font-size: 0.95rem;
  }

  .track-count {
    margin: 0;
    color: var(--color-accent);
    font-weight: 500;
    font-size: 0.9rem;
  }

  .folder-section {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 8px;
    margin-left: auto;
  }

  .tracks-container {
    max-height: 400px;
    overflow-y: auto;
  }

  .track-row {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 12px 24px;
    border-bottom: 1px solid var(--color-border);
    transition: all 0.2s;
  }

  .track-row:hover {
    background: var(--color-bg-tertiary);
  }

  .track-row.completed {
    background: rgba(74, 222, 128, 0.05);
  }

  .track-row.downloading {
    background: rgba(244, 114, 182, 0.05);
  }

  .track-row.unavailable-track {
    opacity: 0.5;
  }

  .unavailable-label {
    font-size: 0.7rem;
    color: var(--color-text-muted);
    background: rgba(255,255,255,0.05);
    border: 1px solid rgba(255,255,255,0.1);
    padding: 1px 5px;
    border-radius: 3px;
    margin-top: 2px;
    display: inline-block;
  }

  .track-num {
    width: 28px;
    color: var(--color-text-muted);
    font-size: 0.85rem;
    font-family: 'JetBrains Mono', monospace;
    font-variant-numeric: tabular-nums;
    text-align: right;
  }

  .track-details {
    flex: 1;
    min-width: 0;
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }

  .track-title {
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
  }

  .explicit-badge {
    flex-shrink: 0;
    font-size: 9px;
    font-weight: 700;
    padding: 1px 5px;
    border-radius: 3px;
    background: rgba(239, 68, 68, 0.15);
    color: #ef4444;
    border: 1px solid rgba(239, 68, 68, 0.3);
    letter-spacing: 0.5px;
  }

  .track-artist {
    display: block;
    color: var(--color-text-tertiary);
    font-size: 0.85rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .track-duration {
    color: var(--color-text-muted);
    font-size: 0.85rem;
    font-family: 'JetBrains Mono', monospace;
    font-variant-numeric: tabular-nums;
    width: 45px;
    text-align: right;
  }

  .popularity-badge {
    display: flex;
    align-items: center;
    gap: 3px;
    font-size: 0.75rem;
    color: var(--color-text-muted);
    font-variant-numeric: tabular-nums;
    width: 36px;
    justify-content: flex-end;
    flex-shrink: 0;
  }

  .sort-bar {
    padding: 8px 0;
    display: flex;
    align-items: center;
    gap: 8px;
    border-bottom: 1px solid var(--color-border);
    margin-bottom: 4px;
  }

  .sort-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 5px 12px;
    border-radius: 6px;
    border: 1px solid var(--color-border);
    background: transparent;
    color: var(--color-text-tertiary);
    font-size: 0.8rem;
    cursor: pointer;
    transition: all 0.15s;
  }

  .sort-btn:hover {
    border-color: var(--color-accent);
    color: var(--color-text-primary);
  }

  .sort-btn.active {
    border-color: var(--color-accent);
    background: var(--color-accent-subtle);
    color: var(--color-accent);
  }

  .assets-result {
    font-size: 0.8rem;
    color: #4ade80;
    align-self: center;
  }

  .track-status {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 100px;
    justify-content: flex-end;
  }

  .status-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    border-radius: 6px;
    font-size: 0.75rem;
    font-weight: 500;
  }

  .status-badge.success {
    background: rgba(74, 222, 128, 0.15);
    color: #4ade80;
  }

  .status-badge.downloading {
    background: rgba(244, 114, 182, 0.15);
    color: #f472b6;
  }

  .status-badge.queued {
    background: rgba(136, 136, 136, 0.15);
    color: #888;
  }

  .status-badge.error {
    background: rgba(248, 113, 113, 0.15);
    color: #f87171;
  }

  .download-section {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    padding: 24px;
    border-top: 1px solid var(--color-border);
  }

  .progress-container {
    width: 100%;
    max-width: 400px;
  }

  .progress-info {
    display: flex;
    justify-content: space-between;
    margin-bottom: 8px;
    font-size: 0.85rem;
    color: var(--color-text-secondary);
  }

  .progress-info .error {
    color: var(--color-error);
  }

  .progress-bar {
    height: 6px;
    background: var(--color-bg-tertiary);
    border-radius: 3px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #f472b6, #ec4899);
    border-radius: 3px;
    transition: width 0.3s ease;
  }

  .progress-fill.complete {
    background: linear-gradient(90deg, #4ade80, #22c55e);
  }

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 80px 24px;
    text-align: center;
  }

  .empty-icon {
    width: 80px;
    height: 80px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 20px;
    margin-bottom: 20px;
    color: var(--color-text-muted);
  }

  .empty-state h3 {
    margin: 0 0 8px 0;
    font-size: 1.2rem;
    font-weight: 600;
  }

  .empty-state p {
    margin: 0;
    color: var(--color-text-tertiary);
    font-size: 0.95rem;
  }

  /* Artist discography */
  .album-filter-bar {
    display: flex;
    gap: 8px;
    padding: 12px 0 4px 0;
    border-bottom: 1px solid var(--color-border);
    margin-bottom: 4px;
  }

  .filter-btn {
    background: none;
    border: 1px solid var(--color-border);
    color: var(--color-text-secondary);
    border-radius: 16px;
    padding: 4px 14px;
    font-size: 0.85rem;
    cursor: pointer;
    transition: all 0.15s;
    font-family: inherit;
  }

  .filter-btn:hover {
    border-color: var(--color-accent);
    color: var(--color-accent);
  }

  .filter-btn.active {
    background: var(--color-accent);
    border-color: var(--color-accent);
    color: #fff;
  }

  .albums-container {
    flex: 1;
    overflow-y: auto;
  }

  .album-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 4px;
    border-bottom: 1px solid var(--color-border);
    transition: background 0.15s;
  }

  .album-row:hover {
    background: var(--color-bg-hover, rgba(255,255,255,0.04));
  }

  .album-thumb {
    width: 48px;
    height: 48px;
    border-radius: 4px;
    object-fit: cover;
    flex-shrink: 0;
  }

  .album-thumb.placeholder {
    background: var(--color-bg-secondary);
  }

  .album-details {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .album-title {
    font-size: 0.95rem;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .album-meta {
    font-size: 0.8rem;
    color: var(--color-text-muted);
  }

  .album-type-badge {
    font-size: 0.75rem;
    padding: 2px 8px;
    border-radius: 10px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    color: var(--color-text-secondary);
    flex-shrink: 0;
  }

  .pagination-controls {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 16px;
    padding: 12px 24px;
    border-top: 1px solid var(--color-border);
  }

  .pagination-btn {
    padding: 8px 16px;
    font-size: 0.85rem;
  }

  .pagination-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .pagination-info {
    font-size: 0.85rem;
    color: var(--color-text-secondary);
    font-variant-numeric: tabular-nums;
  }
</style>
