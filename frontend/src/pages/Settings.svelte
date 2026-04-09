<script lang="ts">
  import { onMount } from 'svelte';
  import { downloadFolder } from '../stores/queue';
  import { themeStore, type ThemeMode, accentColor, accentPresets, applyAccentColor, fontPresets, applyFontFamily } from '../stores/theme';
  import { updateAudioSettings, testSound } from '../stores/audio';
  import { toastStore } from '../stores/toast';
  import TabBar from '../components/TabBar.svelte';
  import { FolderOpen } from 'lucide-svelte';
  import {
    GetConfig,
    SaveConfig,
    SelectDownloadFolder,
    SetDownloadFolder,
    GetDownloadOptions,
    SetDownloadOptions,
    ResetToDefaults,
    CheckAPIStatus,
    CheckForUpdate,
    OpenConfigFolder,
    GetFFmpegInfo,
    InstallFFmpeg,
  } from '../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';

  let config = $state({
    downloadFolder: '',
    concurrentDownloads: 4,
    embedCover: true,
    saveCoverFile: true,
    saveFolderCover: true,
    fileNameFormat: '{artist} - {title}',
    theme: 'system' as ThemeMode,
    accentColor: '#f472b6',
    soundEffects: false,
    soundVolume: 70,
    embedLyrics: false,
    preferSyncedLyrics: true,
    saveLyricsFile: false,
    autoAnalyze: false,
    tidalEnabled: true,
    qobuzEnabled: false,
    qobuzAppId: '',
    qobuzAppSecret: '',
    qobuzAuthToken: '',
    preferredSource: 'tidal',
    generateM3u8: false,
    skipUnavailableTracks: false,
    firstArtistOnly: false,
    autoQualityFallback: false,
    sourceOrder: ['tidal', 'qobuz'] as string[],
    qualityOrder: ['HI_RES', 'LOSSLESS', 'HIGH'] as string[],
    proxyUrl: '',
    skipExisting: true,
    artistSeparator: '; ',
    playlistSubfolder: true,
    folderTemplate: '',
    countryCode: 'US',
    fontFamily: '',
    downloadQuality: 'LOSSLESS',
  });
  let activeTab = $state('general');
  let apiStatuses: any[] = $state([]);
  let checkingAPI = $state(false);
  let updateInfo: any = $state(null);
  let checkingUpdate = $state(false);
  let ffmpegInfo: any = $state(null);
  let installingFFmpeg = $state(false);
  let ffmpegProgress: { stage: string; percent: number } = $state({ stage: '', percent: 0 });
  let folderTemplatePreset = $state('{artist}/{album}');

  const settingsTabs = [
    { id: 'general', label: 'General' },
    { id: 'file-management', label: 'File Management' },
    { id: 'status', label: 'Status' },
  ];

  const folderPresets = [
    '{artist}/{album}',
    '{albumartist}/{album}',
    '{artist}/{year} - {album}',
    '{year}/{artist}/{album}',
  ];

  function handleFolderTemplateChange(e: Event) {
    const value = (e.target as HTMLSelectElement).value;
    folderTemplatePreset = value;
    if (value === '') {
      config.folderTemplate = '';
    } else if (value !== 'custom') {
      config.folderTemplate = value;
    }
  }

  function syncFolderTemplatePreset(template: string) {
    if (!template) {
      folderTemplatePreset = '';
    } else if (folderPresets.includes(template)) {
      folderTemplatePreset = template;
    } else {
      folderTemplatePreset = 'custom';
    }
  }

  const namingPresets = [
    { name: 'Simple', template: '{artist} - {title}' },
    { name: 'Title - Artist', template: '{title} - {artist}' },
    { name: 'Numbered', template: '{track}. {title}' },
    { name: 'Numbered with Artist', template: '{track}. {artist} - {title}' },
    { name: 'Album Organized', template: '{track} - {title}' },
    { name: 'Full (Artist/Album/Track)', template: '{artist}/{album}/{track} - {title}' },
    { name: 'Full with Year', template: '{albumartist}/{album} ({year})/{track} - {title}' },
    { name: 'Multi-disc', template: '{albumartist}/{album}/{discnumber}-{track} - {title}' },
    { name: 'Date Organized', template: '{albumartist}/{album} [{date}]/{track}. {title}' },
    { name: 'ISRC', template: '{isrc} - {title}' },
  ];

  const artistSeparators = [
    { label: 'Semicolon (;)', value: '; ' },
    { label: 'Comma (,)', value: ', ' },
    { label: 'Slash (/)', value: ' / ' },
    { label: 'Ampersand (&)', value: ' & ' },
    { label: 'feat.', value: ' feat. ' },
  ];

  const countries = [
    { code: 'US', name: 'United States' },
    { code: 'GB', name: 'United Kingdom' },
    { code: 'DE', name: 'Germany' },
    { code: 'FR', name: 'France' },
    { code: 'JP', name: 'Japan' },
    { code: 'BR', name: 'Brazil' },
    { code: 'AU', name: 'Australia' },
    { code: 'CA', name: 'Canada' },
    { code: 'SE', name: 'Sweden' },
    { code: 'NO', name: 'Norway' },
    { code: 'DK', name: 'Denmark' },
    { code: 'NL', name: 'Netherlands' },
    { code: 'ES', name: 'Spain' },
    { code: 'IT', name: 'Italy' },
    { code: 'PL', name: 'Poland' },
    { code: 'KR', name: 'South Korea' },
    { code: 'MX', name: 'Mexico' },
    { code: 'AR', name: 'Argentina' },
  ];

  async function checkAPI() {
    checkingAPI = true;
    try {
      apiStatuses = await CheckAPIStatus();
    } catch (e) {
      console.error('API check failed:', e);
    } finally {
      checkingAPI = false;
    }
  }

  async function checkUpdate() {
    checkingUpdate = true;
    try {
      updateInfo = await CheckForUpdate();
    } catch (e) {
      console.error('Update check failed:', e);
    } finally {
      checkingUpdate = false;
    }
  }

  async function openConfig() {
    try {
      await OpenConfigFolder();
    } catch (e) {
      console.error('Failed to open config folder:', e);
    }
  }

  async function installFFmpegHandler() {
    installingFFmpeg = true;
    ffmpegProgress = { stage: 'downloading', percent: 0 };
    try {
      await InstallFFmpeg();
    } catch (e) {
      installingFFmpeg = false;
      ffmpegProgress = { stage: 'error', percent: 0 };
      console.error('FFmpeg install failed:', e);
    }
  }

  let isSaving = $state(false);
  let showResetConfirm = $state(false);
  let isResetting = $state(false);

  $effect(() => {
    updateAudioSettings(config.soundEffects, config.soundVolume);
  });

  function handleThemeChange(event: Event) {
    const select = event.target as HTMLSelectElement;
    const newTheme = select.value as ThemeMode;
    config.theme = newTheme;
    themeStore.setTheme(newTheme);
  }

  function handleAccentColorChange(color: string) {
    config.accentColor = color;
    accentColor.set(color);
    applyAccentColor(color);
  }

  function handleFontChange(event: Event) {
    const select = event.target as HTMLSelectElement;
    const value = select.value;
    config.fontFamily = value;
    if (value) {
      applyFontFamily(value);
    }
  }

  onMount(() => {
    loadConfig();
    GetFFmpegInfo().then(info => { ffmpegInfo = info; });
    EventsOn('ffmpeg-install-progress', (progress: any) => {
      ffmpegProgress = { stage: progress.Stage || progress.stage, percent: progress.Percent || progress.percent };
      if (ffmpegProgress.stage === 'complete') {
        installingFFmpeg = false;
        GetFFmpegInfo().then(info => { ffmpegInfo = info; });
      }
    });
    return () => EventsOff('ffmpeg-install-progress');
  });

  async function loadConfig() {
    try {
      const result = await GetConfig();
      if (result) {
        config.downloadFolder = result.downloadFolder || '';
        config.concurrentDownloads = result.concurrentDownloads || 4;
        config.embedCover = result.embedCover !== false;
        config.fileNameFormat = result.fileNameFormat || '{artist} - {title}';
        config.theme = (result.theme as ThemeMode) || 'system';
        config.accentColor = result.accentColor || '#f472b6';
        config.soundEffects = result.soundEffects || false;
        config.soundVolume = result.soundVolume || 70;
        config.embedLyrics = result.embedLyrics || false;
        config.preferSyncedLyrics = result.preferSyncedLyrics !== false;
        config.saveLyricsFile = result.saveLyricsFile || false;
        config.tidalEnabled = result.tidalEnabled !== false;
        config.qobuzEnabled = result.qobuzEnabled || false;
        config.qobuzAppId = result.qobuzAppId || '';
        config.qobuzAppSecret = result.qobuzAppSecret || '';
        config.qobuzAuthToken = result.qobuzAuthToken || '';
        config.preferredSource = result.preferredSource || 'tidal';
        config.generateM3u8 = result.generateM3u8 || false;
        config.skipUnavailableTracks = result.skipUnavailableTracks || false;
        config.autoQualityFallback = result.autoQualityFallback || false;
        config.firstArtistOnly = result.firstArtistOnly || false;
        config.sourceOrder = result.sourceOrder?.length ? result.sourceOrder : ['tidal', 'qobuz'];
        config.qualityOrder = result.qualityOrder?.length ? result.qualityOrder : ['HI_RES', 'LOSSLESS', 'HIGH'];
        config.proxyUrl = result.proxyUrl || '';
        config.skipExisting = result.skipExisting !== false;
        config.artistSeparator = result.artistSeparator || '; ';
        config.playlistSubfolder = result.playlistSubfolder !== false;
        config.folderTemplate = result.folderTemplate || '';
        syncFolderTemplatePreset(config.folderTemplate);
        config.countryCode = result.countryCode || 'US';
        config.fontFamily = result.fontFamily || '';
        config.downloadQuality = result.downloadQuality || 'LOSSLESS';
        downloadFolder.set(config.downloadFolder);
      }

      // Also get download options
      const opts = await GetDownloadOptions();
      if (opts) {
        config.embedCover = opts.embedCover !== false;
        config.saveCoverFile = opts.saveCoverFile !== false;
        config.saveFolderCover = opts.saveFolderCover !== false;
        config.fileNameFormat = opts.fileNameFormat || '{artist} - {title}';
        config.autoAnalyze = opts.autoAnalyze || false;
      }
    } catch (error) {
      console.error('Error loading config:', error);
    }
  }

  function moveItem(arr: string[], index: number, direction: -1 | 1): string[] {
    const target = index + direction;
    if (target < 0 || target >= arr.length) return arr;
    const next = [...arr];
    [next[index], next[target]] = [next[target], next[index]];
    return next;
  }

  async function selectFolder() {
    try {
      const folder = await SelectDownloadFolder();
      if (folder) {
        config.downloadFolder = folder;
        downloadFolder.set(folder);
        await SetDownloadFolder(folder);
      }
    } catch (error) {
      console.error('Error selecting folder:', error);
    }
  }

  async function saveConfig() {
    isSaving = true;

    try {
      // Save full config including theme, accent color, and sound settings
      const fullConfig = await GetConfig();
      await SaveConfig({
        ...fullConfig,
        theme: config.theme,
        accentColor: config.accentColor,
        downloadFolder: config.downloadFolder,
        concurrentDownloads: config.concurrentDownloads,
        embedCover: config.embedCover,
        saveCoverFile: config.saveCoverFile,
        saveFolderCover: config.saveFolderCover,
        fileNameFormat: config.fileNameFormat,
        soundEffects: config.soundEffects,
        soundVolume: config.soundVolume,
        embedLyrics: config.embedLyrics,
        preferSyncedLyrics: config.preferSyncedLyrics,
        saveLyricsFile: config.saveLyricsFile,
        autoAnalyze: config.autoAnalyze,
        tidalEnabled: config.tidalEnabled,
        qobuzEnabled: config.qobuzEnabled,
        qobuzAppId: config.qobuzAppId,
        qobuzAppSecret: config.qobuzAppSecret,
        qobuzAuthToken: config.qobuzAuthToken,
        preferredSource: config.preferredSource,
        generateM3u8: config.generateM3u8,
        skipUnavailableTracks: config.skipUnavailableTracks,
        autoQualityFallback: config.autoQualityFallback,
        firstArtistOnly: config.firstArtistOnly,
        sourceOrder: config.sourceOrder,
        qualityOrder: config.qualityOrder,
        proxyUrl: config.proxyUrl || '',
        skipExisting: config.skipExisting,
        artistSeparator: config.artistSeparator,
        playlistSubfolder: config.playlistSubfolder,
        folderTemplate: config.folderTemplate,
        countryCode: config.countryCode,
        fontFamily: config.fontFamily,
        downloadQuality: config.downloadQuality,
      });

      // Save download options
      await SetDownloadOptions(
        'LOSSLESS',
        config.fileNameFormat,
        false,
        config.embedCover,
        config.saveCoverFile,
        config.autoAnalyze
      );
      toastStore.show('Settings saved!');
    } catch (error) {
      console.error('Error saving config:', error);
      toastStore.show('Error saving settings', 'error');
    } finally {
      isSaving = false;
    }
  }

  async function handleReset() {
    isResetting = true;
    try {
      const result = await ResetToDefaults();
      if (result) {
        config.concurrentDownloads = result.concurrentDownloads || 4;
        config.embedCover = result.embedCover !== false;
        config.saveCoverFile = result.saveCoverFile !== false;
        config.saveFolderCover = result.saveFolderCover !== false;
        config.fileNameFormat = result.fileNameFormat || '{artist} - {title}';
        config.theme = (result.theme as ThemeMode) || 'system';
        config.accentColor = result.accentColor || '#f472b6';
        config.soundEffects = result.soundEffects || false;
        config.soundVolume = result.soundVolume || 70;
        config.embedLyrics = result.embedLyrics || false;
        config.preferSyncedLyrics = result.preferSyncedLyrics !== false;
        config.saveLyricsFile = result.saveLyricsFile || false;
        config.autoAnalyze = result.autoAnalyze || false;
        config.tidalEnabled = result.tidalEnabled !== false;
        config.qobuzEnabled = result.qobuzEnabled || false;
        config.preferredSource = result.preferredSource || 'tidal';
        config.generateM3u8 = result.generateM3u8 || false;
        config.skipUnavailableTracks = result.skipUnavailableTracks || false;
        config.autoQualityFallback = result.autoQualityFallback || false;
        config.firstArtistOnly = result.firstArtistOnly || false;
        config.sourceOrder = result.sourceOrder?.length ? result.sourceOrder : ['tidal', 'qobuz'];
        config.qualityOrder = result.qualityOrder?.length ? result.qualityOrder : ['HI_RES', 'LOSSLESS', 'HIGH'];
        config.skipExisting = result.skipExisting !== false;
        config.artistSeparator = result.artistSeparator || '; ';
        config.playlistSubfolder = result.playlistSubfolder !== false;
        config.folderTemplate = result.folderTemplate || '';
        syncFolderTemplatePreset(config.folderTemplate);
        config.countryCode = result.countryCode || 'US';
        config.fontFamily = result.fontFamily || '';
        config.downloadQuality = result.downloadQuality || 'LOSSLESS';
        // Note: download folder and Qobuz credentials are preserved
        themeStore.setTheme(config.theme);
        handleAccentColorChange(config.accentColor);
        toastStore.show('Settings reset to defaults!');
      }
    } catch (error) {
      console.error('Error resetting:', error);
      toastStore.show('Error resetting settings', 'error');
    } finally {
      isResetting = false;
      showResetConfirm = false;
    }
  }
</script>

<div class="settings-page">
  <div class="settings-header">
    <h1>Settings</h1>
    <div class="header-actions">
      <button class="btn-secondary" onclick={openConfig}>
        <FolderOpen size={16} />
        Open Config Folder
      </button>
      <button class="btn-secondary" onclick={() => showResetConfirm = true} disabled={isResetting}>
        Reset to Default
      </button>
      <button class="btn-accent" onclick={saveConfig} disabled={isSaving}>
        {#if isSaving}
          <div class="spinner"></div>
          Saving...
        {:else}
          Save Changes
        {/if}
      </button>
    </div>
  </div>

  <TabBar tabs={settingsTabs} bind:activeTab />

  <div class="settings-content">
    <!-- ==================== GENERAL TAB ==================== -->
    {#if activeTab === 'general'}
    <div class="settings-grid">
      <!-- Left Column -->
      <div class="settings-column">
        <div class="group-title">Downloads &amp; Appearance</div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="download-folder">Download Path</label>
            <span class="setting-desc">Where your FLAC files will be saved</span>
          </div>
          <div class="setting-control folder-control">
            <input
              type="text"
              id="download-folder"
              bind:value={config.downloadFolder}
              readonly
              placeholder="Select a folder..."
              class="setting-input folder-input"
            />
            <button class="browse-btn" onclick={selectFolder}>
              <FolderOpen size={16} />
              Browse
            </button>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="theme">Mode</label>
            <span class="setting-desc">Color scheme</span>
          </div>
          <div class="setting-control">
            <select id="theme" value={config.theme} onchange={handleThemeChange} class="setting-select">
              <option value="dark">Dark</option>
              <option value="light">Light</option>
              <option value="system">System</option>
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Accent Color</span>
          </div>
          <div class="accent-swatches" role="radiogroup" aria-label="Accent color selection">
            {#each accentPresets as preset}
              <button
                class="swatch"
                class:active={config.accentColor === preset.color}
                style="background-color: {preset.color}"
                title={preset.name}
                onclick={() => handleAccentColorChange(preset.color)}
              ></button>
            {/each}
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="font-family">Font</label>
          </div>
          <div class="setting-control">
            <select id="font-family" value={config.fontFamily} onchange={handleFontChange} class="setting-select">
              <option value="">System Default</option>
              {#each fontPresets as font}
                <option value={font.value}>{font.name}</option>
              {/each}
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="sound-effects">Sound Effects</label>
            <span class="setting-desc">Play sounds on download events</span>
          </div>
          <div class="setting-control sound-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.soundEffects} />
              <span class="toggle-slider"></span>
            </label>
            <button class="test-sound-btn" onclick={testSound} title="Test sound">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/>
                <path d="M15.54 8.46a5 5 0 0 1 0 7.07"/>
                <path d="M19.07 4.93a10 10 0 0 1 0 14.14"/>
              </svg>
            </button>
          </div>
        </div>

        {#if config.soundEffects}
          <div class="setting-item">
            <div class="setting-info">
              <label for="sound-volume">Volume</label>
              <span class="setting-desc">{config.soundVolume}%</span>
            </div>
            <div class="setting-control volume-control">
              <input
                type="range"
                id="sound-volume"
                min="0"
                max="100"
                bind:value={config.soundVolume}
                class="volume-slider"
              />
            </div>
          </div>
        {/if}
      </div>

      <!-- Right Column -->
      <div class="settings-column">
        <div class="group-title">Sources &amp; Quality</div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="preferred-source">Source Mode</label>
            <span class="setting-desc">Primary download source</span>
          </div>
          <div class="setting-control">
            <select id="preferred-source" bind:value={config.preferredSource} class="setting-select">
              <option value="tidal">Tidal</option>
              <option value="qobuz">Qobuz</option>
              <option value="auto">Auto</option>
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="download-quality">Quality</label>
            <span class="setting-desc">Preferred audio quality tier</span>
          </div>
          <div class="setting-control">
            <select id="download-quality" bind:value={config.downloadQuality} class="setting-select">
              <option value="HI_RES">Hi-Res (24-bit/48kHz+)</option>
              <option value="LOSSLESS">Lossless (16-bit/44.1kHz)</option>
              <option value="HIGH">High (320kbps)</option>
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>Allow Quality Fallback</label>
            <span class="setting-desc">Retry with lower quality when unavailable</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.autoQualityFallback} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="country-code">Region</label>
            <span class="setting-desc">Country code for API (affects availability)</span>
          </div>
          <div class="setting-control">
            <select id="country-code" bind:value={config.countryCode} class="setting-select">
              {#each countries as c}
                <option value={c.code}>{c.name} ({c.code})</option>
              {/each}
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>Tidal</label>
            <span class="setting-desc">Enable Tidal source</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.tidalEnabled} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>Qobuz</label>
            <span class="setting-desc">Enable Qobuz source (requires credentials)</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.qobuzEnabled} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        {#if config.qobuzEnabled}
          <div class="setting-item">
            <div class="setting-info">
              <label for="qobuz-app-id">Qobuz App ID</label>
            </div>
            <div class="setting-control">
              <input type="text" id="qobuz-app-id" bind:value={config.qobuzAppId} placeholder="Enter App ID..." class="setting-input" />
            </div>
          </div>
          <div class="setting-item">
            <div class="setting-info">
              <label for="qobuz-app-secret">Qobuz App Secret</label>
            </div>
            <div class="setting-control">
              <input type="password" id="qobuz-app-secret" bind:value={config.qobuzAppSecret} placeholder="Enter App Secret..." class="setting-input" />
            </div>
          </div>
          <div class="setting-item">
            <div class="setting-info">
              <label for="qobuz-auth-token">Qobuz Auth Token</label>
            </div>
            <div class="setting-control">
              <input type="password" id="qobuz-auth-token" bind:value={config.qobuzAuthToken} placeholder="Enter Auth Token..." class="setting-input" />
            </div>
          </div>
        {/if}

        <div class="setting-item">
          <div class="setting-info">
            <label>Skip Existing Files</label>
            <span class="setting-desc">Skip files already on disk (matched by ISRC)</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.skipExisting} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>Skip Unavailable Tracks</label>
            <span class="setting-desc">Skip tracks not available for streaming</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.skipUnavailableTracks} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="concurrent">Concurrent Downloads</label>
            <span class="setting-desc">Simultaneous downloads</span>
          </div>
          <div class="setting-control">
            <select id="concurrent" bind:value={config.concurrentDownloads} class="setting-select">
              <option value={1}>1</option>
              <option value={2}>2</option>
              <option value={3}>3</option>
              <option value={4}>4</option>
              <option value={6}>6</option>
              <option value={8}>8</option>
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">HTTP / SOCKS5 Proxy</span>
            <span class="setting-desc">Route requests through a proxy</span>
          </div>
          <div class="setting-control wide">
            <input
              type="text"
              class="setting-input"
              bind:value={config.proxyUrl}
              placeholder="e.g. socks5://127.0.0.1:1080"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- ==================== FILE MANAGEMENT TAB ==================== -->
    {:else if activeTab === 'file-management'}
    <div class="settings-grid">
      <!-- Left Column -->
      <div class="settings-column">
        <div class="group-title">File Naming</div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="naming-preset">Naming Preset</label>
            <span class="setting-desc">Quick-select a naming template</span>
          </div>
          <div class="setting-control">
            <select
              id="naming-preset"
              class="setting-select"
              onchange={(e) => { const v = (e.target as HTMLSelectElement).value; if (v) config.fileNameFormat = v; }}
            >
              <option value="">Custom...</option>
              {#each namingPresets as preset}
                <option value={preset.template} selected={config.fileNameFormat === preset.template}>{preset.name}: {preset.template}</option>
              {/each}
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="file-naming">Template</label>
            <span class="setting-desc">Variables: {'{artist}'}, {'{title}'}, {'{album}'}, {'{track}'}, {'{year}'}, {'{isrc}'}</span>
          </div>
          <div class="setting-control wide">
            <input
              type="text"
              id="file-naming"
              bind:value={config.fileNameFormat}
              class="setting-input"
              placeholder="{'{artist}'} - {'{title}'}"
            />
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="artist-separator">Artist Separator</label>
            <span class="setting-desc">How multiple artists are joined</span>
          </div>
          <div class="setting-control">
            <select id="artist-separator" bind:value={config.artistSeparator} class="setting-select">
              {#each artistSeparators as sep}
                <option value={sep.value}>{sep.label}</option>
              {/each}
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="folder-template">Folder Structure</label>
            <span class="setting-desc">Organize downloads into subfolders</span>
          </div>
          <div class="setting-control">
            <select
              id="folder-template"
              class="setting-select"
              value={folderTemplatePreset}
              onchange={handleFolderTemplateChange}
            >
              <option value="">No organization</option>
              <option value={'{artist}/{album}'}>Artist / Album</option>
              <option value={'{albumartist}/{album}'}>Album Artist / Album</option>
              <option value={'{artist}/{year} - {album}'}>Artist / Year - Album</option>
              <option value={'{year}/{artist}/{album}'}>Year / Artist / Album</option>
              <option value="custom">Custom template...</option>
            </select>
          </div>
        </div>
        {#if folderTemplatePreset === 'custom'}
          <div class="setting-item">
            <div class="setting-info">
              <label for="folder-template-custom">Custom Template</label>
              <span class="setting-desc">Variables: {'{artist}'}, {'{albumartist}'}, {'{album}'}, {'{year}'}, {'{label}'}</span>
            </div>
            <div class="setting-control wide">
              <input
                type="text"
                id="folder-template-custom"
                bind:value={config.folderTemplate}
                class="setting-input"
                placeholder="{'{artist}'}/{'{album}'}"
              />
            </div>
          </div>
        {/if}

        <div class="setting-item">
          <div class="setting-info">
            <label>Playlist Subfolder</label>
            <span class="setting-desc">Create subfolder for playlist downloads</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.playlistSubfolder} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>Generate M3U8 Playlist</label>
            <span class="setting-desc">Create .m3u8 after batch downloads</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.generateM3u8} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>
      </div>

      <!-- Right Column -->
      <div class="settings-column">
        <div class="group-title">Metadata &amp; Tags</div>

        <div class="setting-item">
          <div class="setting-info">
            <label>Embed Lyrics</label>
            <span class="setting-desc">Fetch and embed lyrics during download</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.embedLyrics} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        {#if config.embedLyrics}
          <div class="setting-item">
            <div class="setting-info">
              <label>Prefer Synced Lyrics</label>
              <span class="setting-desc">Prioritize time-synced (LRC) lyrics</span>
            </div>
            <div class="setting-control">
              <label class="toggle">
                <input type="checkbox" bind:checked={config.preferSyncedLyrics} />
                <span class="toggle-slider"></span>
              </label>
            </div>
          </div>

          <div class="setting-item">
            <div class="setting-info">
              <label>Save Lyrics File</label>
              <span class="setting-desc">Save .lrc or .txt alongside FLAC</span>
            </div>
            <div class="setting-control">
              <label class="toggle">
                <input type="checkbox" bind:checked={config.saveLyricsFile} />
                <span class="toggle-slider"></span>
              </label>
            </div>
          </div>
        {/if}

        <div class="setting-item">
          <div class="setting-info">
            <label>Embed Cover Art</label>
            <span class="setting-desc">Include album artwork in FLAC files</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.embedCover} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>Save Cover as File</label>
            <span class="setting-desc">Save album artwork as .jpg next to each track</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.saveCoverFile} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>Save Folder Cover</label>
            <span class="setting-desc">Save folder.jpg in album directories (Plex, Jellyfin)</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.saveFolderCover} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>First Artist Only</label>
            <span class="setting-desc">Use only primary artist in tags and filenames</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.firstArtistOnly} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>Auto-analyze Downloads</label>
            <span class="setting-desc">Detect upscaled files after download</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.autoAnalyze} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>
      </div>
    </div>

    <!-- ==================== STATUS TAB ==================== -->
    {:else if activeTab === 'status'}

    <section class="settings-section">
      <div class="group-title">API Status</div>
      <div class="api-status-header">
        <button class="btn-secondary" onclick={checkAPI} disabled={checkingAPI}>
          {checkingAPI ? 'Checking...' : 'Check Status'}
        </button>
      </div>
      {#if apiStatuses.length > 0}
        <div class="api-status-list">
          {#each apiStatuses as ep}
            <div class="api-status-item">
              <span class="api-name">{ep.name}</span>
              <span class="status-badge" class:ok={ep.status === 'online'} class:error={ep.status === 'offline'} class:slow={ep.status === 'slow'}>
                {ep.status} ({ep.latencyMs}ms)
              </span>
            </div>
          {/each}
        </div>
      {/if}
    </section>

    <section class="settings-section">
      <div class="group-title">FFmpeg</div>
      <div class="setting-item">
        <div class="setting-info">
          <span class="setting-label">FFmpeg Status</span>
          <span class="setting-desc">Required for audio conversion (FLAC to MP3, AAC, etc.)</span>
        </div>
        <div class="setting-control">
          {#if ffmpegInfo?.available}
            <span class="status-badge ok">Installed</span>
          {:else}
            <span class="status-badge error">Not Found</span>
          {/if}
        </div>
      </div>

      {#if ffmpegInfo?.available}
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Version</span>
            <span class="setting-desc ffmpeg-version">{ffmpegInfo.version || 'Unknown'}</span>
          </div>
        </div>
      {:else}
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">Auto Install</span>
            <span class="setting-desc">Download a static FFmpeg build to ~/.flacidal/bin/</span>
          </div>
          <div class="setting-control">
            {#if installingFFmpeg}
              <div class="ffmpeg-progress">
                <div class="ffmpeg-progress-bar">
                  <div class="ffmpeg-progress-fill" style="width: {ffmpegProgress.percent}%"></div>
                </div>
                <span class="ffmpeg-progress-text">
                  {ffmpegProgress.stage === 'downloading' ? `Downloading... ${Math.round(ffmpegProgress.percent)}%` : ''}
                  {ffmpegProgress.stage === 'extracting' ? 'Extracting...' : ''}
                  {ffmpegProgress.stage === 'complete' ? 'Done!' : ''}
                  {ffmpegProgress.stage === 'error' ? 'Failed' : ''}
                </span>
              </div>
            {:else}
              <button class="btn-accent" onclick={installFFmpegHandler}>Install FFmpeg</button>
            {/if}
          </div>
        </div>
      {/if}
    </section>

    <!-- About & Updates -->
    <section class="settings-section about">
      <div class="group-title">About</div>
      <div class="about-content">
        <div class="app-info">
          <div class="app-logo">
            <div class="waveform">
              <span class="bar"></span>
              <span class="bar"></span>
              <span class="bar"></span>
              <span class="bar"></span>
            </div>
          </div>
          <div class="app-details">
            <h3>FLACidal</h3>
            <span class="version">Version 3.3.0</span>
          </div>
        </div>
        <p class="app-desc">High-quality FLAC downloader for Tidal. Download your favorite music in lossless quality.</p>
        <div class="update-check">
          <button class="btn-secondary" onclick={checkUpdate} disabled={checkingUpdate}>
            {checkingUpdate ? 'Checking...' : 'Check for Updates'}
          </button>
          {#if updateInfo}
            {#if updateInfo.hasUpdate}
              <span class="update-available">Update available: v{updateInfo.version} - <a href={updateInfo.releaseUrl} target="_blank" rel="noopener">View Release</a></span>
            {:else}
              <span class="update-current">You're up to date!</span>
            {/if}
          {/if}
        </div>
      </div>
    </section>

    {/if}
  </div>
</div>

<!-- Reset Confirmation Modal -->
{#if showResetConfirm}
  <div class="modal-overlay" onclick={() => showResetConfirm = false} onkeydown={(e) => e.key === 'Escape' && (showResetConfirm = false)} role="dialog" aria-modal="true" tabindex="-1">
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div class="modal" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()} role="document">
      <div class="modal-icon">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
          <path d="M3 3v5h5"/>
        </svg>
      </div>
      <h3>Reset to Defaults?</h3>
      <p>This will reset all settings to their default values. Your download folder will be preserved.</p>
      <div class="modal-actions">
        <button class="modal-btn cancel" onclick={() => showResetConfirm = false}>
          Cancel
        </button>
        <button class="modal-btn confirm" onclick={handleReset} disabled={isResetting}>
          {#if isResetting}
            Resetting...
          {:else}
            Reset Settings
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .settings-page {
    padding: 32px;
    max-width: 960px;
  }

  /* Header */
  .settings-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
  }

  .settings-header h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0;
  }

  .header-actions {
    display: flex;
    gap: 10px;
    align-items: center;
  }

  /* Buttons */
  .btn-accent {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 20px;
    background: var(--color-accent, #f472b6);
    border: none;
    border-radius: 8px;
    color: #000;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.2s;
    font-family: var(--font-family, 'Plus Jakarta Sans', sans-serif);
  }

  .btn-accent:hover:not(:disabled) {
    opacity: 0.9;
  }

  .btn-accent:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .btn-secondary {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    background: transparent;
    border: 1px solid var(--color-border, #1a1a1a);
    border-radius: 8px;
    color: var(--color-text-secondary, #a1a1a1);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
    font-family: var(--font-family, 'Plus Jakarta Sans', sans-serif);
  }

  .btn-secondary:hover:not(:disabled) {
    background: var(--color-bg-hover, #1a1a1a);
    color: var(--color-text-primary, #fafafa);
    border-color: var(--color-text-tertiary, #525252);
  }

  .btn-secondary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  /* Grid Layout */
  .settings-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 32px;
  }

  .settings-column {
    display: flex;
    flex-direction: column;
  }

  .group-title {
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-tertiary, #525252);
    text-transform: uppercase;
    letter-spacing: 0.8px;
    margin-bottom: 16px;
    padding-bottom: 8px;
    border-bottom: 1px solid var(--color-border, #1a1a1a);
  }

  /* Settings Content */
  .settings-content {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .settings-section {
    background: var(--color-bg-secondary, #0a0a0a);
    border: 1px solid var(--color-border, #1a1a1a);
    border-radius: 16px;
    padding: 24px;
  }

  .settings-section .group-title {
    margin-top: 0;
  }

  /* Setting Items */
  .setting-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 0;
    border-bottom: 1px solid var(--color-border, #1a1a1a);
  }

  .setting-item:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }

  .setting-item:first-of-type {
    padding-top: 0;
  }

  .setting-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .setting-info label,
  .setting-info .setting-label {
    font-weight: 500;
    color: var(--color-text-primary, #fafafa);
    font-size: 14px;
  }

  .setting-desc {
    font-size: 13px;
    color: var(--color-text-tertiary, #525252);
  }

  .setting-control {
    flex-shrink: 0;
  }

  .setting-control.wide {
    flex: 1;
    max-width: 280px;
    margin-left: 24px;
  }

  .setting-control.wide .setting-input {
    width: 100%;
  }

  /* Inputs */
  .setting-select {
    padding: 10px 14px;
    background: var(--color-bg-primary, #000);
    border: 1px solid var(--color-border-subtle, #222);
    border-radius: 8px;
    color: var(--color-text-primary, #fafafa);
    font-size: 14px;
    min-width: 170px;
    cursor: pointer;
    font-family: var(--font-family, 'Plus Jakarta Sans', sans-serif);
    -webkit-appearance: none;
    -moz-appearance: none;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23666' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 12px center;
    padding-right: 36px;
  }

  .setting-select option {
    background: var(--color-bg-secondary, #0a0a0a);
    color: var(--color-text-primary, #fafafa);
    padding: 8px;
  }

  .setting-select:focus {
    outline: none;
    border-color: var(--color-accent, #f472b6);
  }

  .setting-input {
    padding: 10px 14px;
    background: var(--color-bg-primary, #000);
    border: 1px solid var(--color-border-subtle, #222);
    border-radius: 8px;
    color: var(--color-text-primary, #fafafa);
    font-size: 14px;
    width: 220px;
    outline: none;
    transition: border-color 0.2s;
    font-family: var(--font-family, 'Plus Jakarta Sans', sans-serif);
  }

  .setting-input:focus {
    border-color: var(--color-accent, #f472b6);
  }

  .setting-input::placeholder {
    color: #555;
  }

  /* Fix for WebKit in light theme */
  :global([data-theme="light"]) .setting-select {
    background-color: #ffffff;
    color: #171717;
    border-color: #d4d4d4;
  }

  :global([data-theme="light"]) .setting-select option {
    background-color: #fafafa;
    color: #171717;
  }

  :global([data-theme="light"]) .folder-input {
    background-color: #ffffff;
    color: #525252;
    border-color: #d4d4d4;
  }

  /* Folder Control */
  .folder-control {
    display: flex;
    gap: 8px;
    flex: 1;
    max-width: 320px;
    margin-left: 24px;
  }

  .folder-input {
    flex: 1;
    font-family: monospace;
    font-size: 13px;
  }

  .browse-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    background: var(--color-bg-tertiary, #171717);
    border: 1px solid var(--color-border-subtle, #222);
    border-radius: 8px;
    color: var(--color-text-primary, #fafafa);
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s;
    white-space: nowrap;
    font-family: var(--font-family, 'Plus Jakarta Sans', sans-serif);
  }

  .browse-btn:hover {
    background: var(--color-bg-hover, #1a1a1a);
  }

  /* Accent Swatches */
  .accent-swatches {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .swatch {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
    transition: all 0.2s;
    position: relative;
    padding: 0;
  }

  .swatch:hover {
    transform: scale(1.15);
  }

  .swatch.active {
    border-color: #fff;
    box-shadow: 0 0 0 2px var(--color-bg-primary, #000);
  }

  .swatch.active::after {
    content: '';
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 10px;
    height: 10px;
    background: white;
    border-radius: 50%;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
  }

  /* Toggle Switch */
  .toggle {
    position: relative;
    display: inline-block;
    width: 44px;
    height: 24px;
  }

  .toggle input {
    opacity: 0;
    width: 0;
    height: 0;
  }

  .toggle-slider {
    position: absolute;
    cursor: pointer;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: var(--color-bg-hover, #1a1a1a);
    transition: 0.3s;
    border-radius: 24px;
  }

  .toggle-slider:before {
    position: absolute;
    content: "";
    height: 18px;
    width: 18px;
    left: 3px;
    bottom: 3px;
    background: var(--color-text-tertiary, #525252);
    transition: 0.3s;
    border-radius: 50%;
  }

  .toggle input:checked + .toggle-slider {
    background: var(--color-accent, #f472b6);
  }

  .toggle input:checked + .toggle-slider:before {
    transform: translateX(20px);
    background: #fff;
  }

  /* Sound Control */
  .sound-control {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .test-sound-btn {
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-bg-tertiary, #171717);
    border: 1px solid var(--color-border-subtle, #222);
    border-radius: 6px;
    color: var(--color-text-secondary, #a1a1a1);
    cursor: pointer;
    transition: all 0.2s;
  }

  .test-sound-btn:hover {
    background: var(--color-bg-hover, #1a1a1a);
    color: var(--color-accent, #f472b6);
    border-color: var(--color-accent, #f472b6);
  }

  /* Volume */
  .volume-control {
    width: 150px;
  }

  .volume-slider {
    width: 100%;
    height: 6px;
    -webkit-appearance: none;
    appearance: none;
    background: var(--color-bg-hover, #1a1a1a);
    border-radius: 3px;
    outline: none;
    cursor: pointer;
  }

  .volume-slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: 16px;
    height: 16px;
    background: var(--color-accent, #f472b6);
    border-radius: 50%;
    cursor: pointer;
    transition: transform 0.2s;
  }

  .volume-slider::-webkit-slider-thumb:hover {
    transform: scale(1.2);
  }

  .volume-slider::-moz-range-thumb {
    width: 16px;
    height: 16px;
    background: var(--color-accent, #f472b6);
    border-radius: 50%;
    cursor: pointer;
    border: none;
  }

  /* Status Badge */
  .status-badge {
    font-size: 13px;
    font-weight: 600;
    padding: 4px 12px;
    border-radius: 12px;
  }

  .status-badge.ok {
    color: #10b981;
    background: rgba(16, 185, 129, 0.1);
  }

  .status-badge.error {
    color: #ef4444;
    background: rgba(239, 68, 68, 0.1);
  }

  .status-badge.slow {
    color: #f59e0b;
    background: rgba(245, 158, 11, 0.1);
  }

  /* API Status */
  .api-status-header {
    margin-bottom: 16px;
  }

  .api-status-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .api-status-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 14px;
    background: var(--color-bg-primary, #000);
    border: 1px solid var(--color-border-subtle, #222);
    border-radius: 8px;
  }

  .api-name {
    font-size: 14px;
    font-weight: 500;
  }

  /* FFmpeg */
  .ffmpeg-version {
    font-family: monospace;
    font-size: 12px !important;
  }

  .ffmpeg-progress {
    display: flex;
    flex-direction: column;
    gap: 6px;
    min-width: 200px;
  }

  .ffmpeg-progress-bar {
    height: 8px;
    background: var(--color-bg-tertiary, #171717);
    border-radius: 4px;
    overflow: hidden;
  }

  .ffmpeg-progress-fill {
    height: 100%;
    background: linear-gradient(90deg, var(--color-accent, #f472b6), #a855f7);
    border-radius: 4px;
    transition: width 0.3s ease;
  }

  .ffmpeg-progress-text {
    font-size: 12px;
    color: var(--color-text-tertiary, #525252);
  }

  /* About Section */
  .settings-section.about {
    background: linear-gradient(135deg, rgba(244, 114, 182, 0.05), rgba(168, 85, 247, 0.05));
    border-color: rgba(244, 114, 182, 0.2);
  }

  .about-content {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .app-info {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .app-logo {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    background: linear-gradient(135deg, #f472b6, #a855f7);
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .waveform {
    display: flex;
    align-items: center;
    gap: 3px;
    height: 20px;
  }

  .waveform .bar {
    width: 3px;
    background: #000;
    border-radius: 2px;
    animation: waveform 0.8s ease-in-out infinite;
  }

  .waveform .bar:nth-child(1) { height: 40%; animation-delay: 0s; }
  .waveform .bar:nth-child(2) { height: 70%; animation-delay: 0.1s; }
  .waveform .bar:nth-child(3) { height: 100%; animation-delay: 0.2s; }
  .waveform .bar:nth-child(4) { height: 60%; animation-delay: 0.3s; }

  @keyframes waveform {
    0%, 100% { transform: scaleY(0.3); }
    50% { transform: scaleY(1); }
  }

  .app-details h3 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
  }

  .version {
    font-size: 13px;
    color: var(--color-text-tertiary, #525252);
  }

  .app-desc {
    margin: 0;
    color: var(--color-text-secondary, #a1a1a1);
    font-size: 14px;
    line-height: 1.5;
  }

  .update-check {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 8px;
  }

  .update-available {
    color: var(--color-accent, #f472b6);
    font-size: 14px;
  }

  .update-available a {
    color: var(--color-accent, #f472b6);
    text-decoration: underline;
  }

  .update-current {
    color: #10b981;
    font-size: 14px;
  }

  /* Spinner */
  .spinner {
    width: 16px;
    height: 16px;
    border: 2px solid transparent;
    border-top-color: #000;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  /* Modal */
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    animation: fadeIn 0.2s ease-out;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .modal {
    background: var(--color-bg-secondary, #0a0a0a);
    border: 1px solid var(--color-border, #1a1a1a);
    border-radius: 16px;
    padding: 32px;
    max-width: 400px;
    width: 90%;
    text-align: center;
    animation: slideUp 0.3s ease-out;
  }

  @keyframes slideUp {
    from {
      opacity: 0;
      transform: translateY(20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .modal-icon {
    width: 64px;
    height: 64px;
    border-radius: 50%;
    background: rgba(245, 158, 11, 0.15);
    color: var(--color-warning, #f59e0b);
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0 auto 20px;
  }

  .modal h3 {
    margin: 0 0 12px 0;
    font-size: 20px;
    font-weight: 600;
  }

  .modal p {
    margin: 0 0 24px 0;
    color: var(--color-text-secondary, #a1a1a1);
    font-size: 14px;
    line-height: 1.5;
  }

  .modal-actions {
    display: flex;
    gap: 12px;
    justify-content: center;
  }

  .modal-btn {
    padding: 12px 24px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
    font-family: var(--font-family, 'Plus Jakarta Sans', sans-serif);
  }

  .modal-btn.cancel {
    background: var(--color-bg-tertiary, #171717);
    border: 1px solid var(--color-border, #1a1a1a);
    color: var(--color-text-secondary, #a1a1a1);
  }

  .modal-btn.cancel:hover {
    background: var(--color-bg-hover, #1a1a1a);
    color: var(--color-text-primary, #fafafa);
  }

  .modal-btn.confirm {
    background: var(--color-warning, #f59e0b);
    border: none;
    color: #000;
  }

  .modal-btn.confirm:hover:not(:disabled) {
    opacity: 0.9;
  }

  .modal-btn.confirm:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }
</style>
