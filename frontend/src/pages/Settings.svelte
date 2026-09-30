<script lang="ts">
  import { onMount } from 'svelte';
  import { downloadFolder } from '../stores/queue';
  import { themeStore, type ThemeMode, accentColor, accentPresets, applyAccentColor, fontPresets, applyFontFamily } from '../stores/theme';
  import { updateAudioSettings, testSound } from '../stores/audio';
  import { toastStore } from '../stores/toast';
  import TabBar from '../components/TabBar.svelte';
  import { t, setLanguage, type MessageKey } from '../lib/i18n';
  import { FolderOpen } from 'lucide-svelte';
  import {
    GetConfig,
    SelectDownloadFolder,
    GetFFmpegInfo,
    GetAppVersion,
    SaveConfig,
    SetDownloadFolder,
    GetDownloadOptions,
    SetDownloadOptions,
    ResetToDefaults,
    CheckAPIStatus,
    CheckForUpdate,
    OpenConfigFolder,
    GetLogFilePath,
    isWailsRuntime,
    ExportConfig,
    ImportConfig,
    InstallFFmpeg,
    SetSourceOrder,
    GetSldlStatus,
    GetSourceHealth,
    InstallSldl,
    TestSoulseekConnection,
  } from '../lib/api';
  import { EventsOn, EventsOff } from '../lib/websocket';

  let config = $state({
    downloadFolder: '',
    concurrentDownloads: 4,
    embedCover: true,
    saveCoverFile: true,
    saveFolderCover: true,
    fileNameFormat: '{artist} - {title}',
    theme: 'system' as ThemeMode,
    language: '',
    accentColor: '#f472b6',
    dateFormat: 'full',
    soundEffects: false,
    soundVolume: 70,
    embedLyrics: false,
    preferSyncedLyrics: true,
    saveLyricsFile: false,
    autoAnalyze: false,
    embedGenre: true,
    useSingleGenre: false,
    tidalEnabled: true,
    qobuzEnabled: true,
    soulseekEnabled: false,
    soulseekUsername: '',
    soulseekPassword: '',
    jellyfinEnabled: false,
    jellyfinUrl: '',
    jellyfinApiKey: '',
    preferredSource: 'tidal',
    generateM3u8: false,
    skipUnavailableTracks: false,
    firstArtistOnly: false,
    autoQualityFallback: false,
    autoStopOnCooldown: false,
    sourceOrder: ['soulseek', 'tidal', 'qobuz', 'amazon', 'bandcamp'] as string[],
    qualityOrder: ['HI_RES', 'LOSSLESS', 'HIGH'] as string[],
    proxyUrl: '',
    tidalCustomEndpoint: '',
    qobuzCustomEndpoint: '',
    tidalPriorityEndpoints: [] as string[],
    qobuzPriorityEndpoints: [] as string[],
    amazonPriorityEndpoints: [] as string[],
    skipExisting: true,
    externalLibraryPaths: [] as string[],
    artistSeparator: '; ',
    playlistSubfolder: true,
    folderTemplate: '',
    countryCode: 'US',
    fontFamily: '',
    downloadQuality: 'LOSSLESS',
    enableReplayGain: false,
  });
  let activeTab = $state('general');
  let logFilePath = $state('');
  let apiStatuses: any[] = $state([]);
  let checkingAPI = $state(false);
  let appVersion = $state('');
  let updateInfo: any = $state(null);
  let checkingUpdate = $state(false);
  let ffmpegInfo: any = $state(null);
  let sldlStatus: any = $state(null);
  let soulseekLoginResult: { success: boolean; message: string } | null = $state(null);
  let testingLogin = $state(false);
  let installingFFmpeg = $state(false);
  let ffmpegProgress: { stage: string; percent: number } = $state({ stage: '', percent: 0 });
  let sourceHealth: any[] = $state([]);
  let checkingSourceHealth = $state(false);
  let installingSldl = $state(false);
  let sldlInstallProgress = $state({ stage: '', percent: 0 });
  let folderTemplatePreset = $state('{artist}/{album}');
  let sourceOrder = $state<string[]>([]);
  let dragIndex = $state<number | null>(null);
  let tidalPriorityText = $state('');
  let qobuzPriorityText = $state('');
  let amazonPriorityText = $state('');
  let externalLibraryPathsText = $state('');

  const sourceLabels: Record<string, string> = {
    tidal: 'Tidal',
    qobuz: 'Qobuz',
    amazon: 'Amazon Music',
    bandcamp: 'Bandcamp',
    soulseek: 'Soulseek',
  };

  function onDragStart(e: DragEvent, index: number) {
    dragIndex = index;
    (e.dataTransfer as DataTransfer).effectAllowed = 'move';
  }

  function onDragOver(e: DragEvent, index: number) {
    e.preventDefault();
    (e.dataTransfer as DataTransfer).dropEffect = 'move';
  }

  function onDrop(e: DragEvent, index: number) {
    e.preventDefault();
    if (dragIndex === null || dragIndex === index) return;
    const newOrder = [...sourceOrder];
    const [moved] = newOrder.splice(dragIndex, 1);
    newOrder.splice(index, 0, moved);
    sourceOrder = newOrder;
    dragIndex = null;
    SetSourceOrder(newOrder).catch(() => {
      toastStore.show($t('settings.toast.sourceOrderFailed'), 'error');
    });
  }

  const settingsTabs = $derived([
    { id: 'general', label: $t('settings.tabs.general') },
    { id: 'file-management', label: $t('settings.tabs.fileManagement') },
    { id: 'status', label: $t('settings.tabs.status') },
  ]);

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

  // Filename-only templates, use the "Folder Structure" control below to organize
  // into subfolders. A template here must never contain '/': it names one file,
  // and the download engine strips slashes from it (SanitizeFileName).
  const namingPresets = [
    { labelKey: 'settings.naming.simple' as MessageKey, template: '{artist} - {title}' },
    { labelKey: 'settings.naming.titleArtist' as MessageKey, template: '{title} - {artist}' },
    { labelKey: 'settings.naming.numbered' as MessageKey, template: '{track}. {title}' },
    { labelKey: 'settings.naming.numberedArtist' as MessageKey, template: '{track}. {artist} - {title}' },
    { labelKey: 'settings.naming.albumOrganized' as MessageKey, template: '{track} - {title}' },
    { labelKey: 'settings.naming.multiDisc' as MessageKey, template: '{discnumber}-{track} - {title}' },
    { labelKey: 'settings.naming.isrc' as MessageKey, template: '{isrc} - {title}' },
  ];

  const artistSeparators = [
    { labelKey: 'settings.sep.semicolon' as MessageKey, value: '; ' },
    { labelKey: 'settings.sep.comma' as MessageKey, value: ', ' },
    { labelKey: 'settings.sep.slash' as MessageKey, value: ' / ' },
    { labelKey: 'settings.sep.ampersand' as MessageKey, value: ' & ' },
    { labelKey: 'settings.sep.feat' as MessageKey, value: ' feat. ' },
  ];

  const countries = [
    { code: 'US', labelKey: 'settings.country.us' as MessageKey },
    { code: 'GB', labelKey: 'settings.country.gb' as MessageKey },
    { code: 'DE', labelKey: 'settings.country.de' as MessageKey },
    { code: 'FR', labelKey: 'settings.country.fr' as MessageKey },
    { code: 'JP', labelKey: 'settings.country.jp' as MessageKey },
    { code: 'BR', labelKey: 'settings.country.br' as MessageKey },
    { code: 'AU', labelKey: 'settings.country.au' as MessageKey },
    { code: 'CA', labelKey: 'settings.country.ca' as MessageKey },
    { code: 'SE', labelKey: 'settings.country.se' as MessageKey },
    { code: 'NO', labelKey: 'settings.country.no' as MessageKey },
    { code: 'DK', labelKey: 'settings.country.dk' as MessageKey },
    { code: 'NL', labelKey: 'settings.country.nl' as MessageKey },
    { code: 'ES', labelKey: 'settings.country.es' as MessageKey },
    { code: 'IT', labelKey: 'settings.country.it' as MessageKey },
    { code: 'PL', labelKey: 'settings.country.pl' as MessageKey },
    { code: 'KR', labelKey: 'settings.country.kr' as MessageKey },
    { code: 'MX', labelKey: 'settings.country.mx' as MessageKey },
    { code: 'AR', labelKey: 'settings.country.ar' as MessageKey },
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

  async function exportConfig() {
    try {
      await ExportConfig();
      toastStore.show($t('settings.toast.configExported'));
    } catch (e) {
      console.error('Failed to export config:', e);
      toastStore.show($t('settings.toast.configExportError'), 'error');
    }
  }

  async function importConfig() {
    try {
      const result = await ImportConfig();
      if (result) {
        await loadConfig();
        toastStore.show($t('settings.toast.configImported'));
      }
    } catch (e) {
      console.error('Failed to import config:', e);
      toastStore.show($t('settings.toast.configImportError'), 'error');
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

  async function checkSourceHealth() {
    checkingSourceHealth = true;
    try {
      sourceHealth = await GetSourceHealth();
    } catch (e) {
      console.error('Source health check failed:', e);
    } finally {
      checkingSourceHealth = false;
    }
  }

  async function installSldlHandler() {
    installingSldl = true;
    sldlInstallProgress = { stage: 'downloading', percent: 0 };
    try {
      await InstallSldl();
    } catch (e) {
      installingSldl = false;
      sldlInstallProgress = { stage: 'error', percent: 0 };
      console.error('sldl install failed:', e);
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

  function handleLanguageChange(event: Event) {
    const value = (event.target as HTMLSelectElement).value;
    config.language = value;
    setLanguage(value);
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
    GetAppVersion().then(v => { appVersion = v; });
    GetFFmpegInfo().then(info => { ffmpegInfo = info; });
    GetSldlStatus().then(s => { sldlStatus = s; });
    EventsOn('ffmpeg-install-progress', (progress: any) => {
      ffmpegProgress = { stage: progress.Stage || progress.stage, percent: progress.Percent || progress.percent };
      if (ffmpegProgress.stage === 'complete') {
        installingFFmpeg = false;
        GetFFmpegInfo().then(info => { ffmpegInfo = info; });
      }
    });
    EventsOn('sldl-install-progress', (progress: any) => {
      sldlInstallProgress = { stage: progress.Stage || progress.stage, percent: progress.Percent || progress.percent };
      if (sldlInstallProgress.stage === 'complete') {
        installingSldl = false;
        GetSldlStatus().then(s => { sldlStatus = s; });
      }
    });
    return () => {
      EventsOff('ffmpeg-install-progress');
      EventsOff('sldl-install-progress');
    };
  });

  async function loadConfig() {
    try {
      const result = await GetConfig();
      if (result) {
        config.downloadFolder = result.downloadFolder || '';
        config.dateFormat = result.dateFormat === 'year' ? 'year' : 'full';
        config.concurrentDownloads = result.concurrentDownloads || 4;
        config.embedCover = result.embedCover !== false;
        config.fileNameFormat = result.fileNameFormat || '{artist} - {title}';
        config.theme = (result.theme as ThemeMode) || 'system';
        config.language = result.language ?? '';
        setLanguage(config.language);
        config.accentColor = result.accentColor || '#f472b6';
        config.soundEffects = result.soundEffects || false;
        config.soundVolume = result.soundVolume || 70;
        config.embedLyrics = result.embedLyrics || false;
        config.preferSyncedLyrics = result.preferSyncedLyrics !== false;
        config.saveLyricsFile = result.saveLyricsFile || false;
        config.tidalEnabled = result.tidalEnabled !== false;
        config.qobuzEnabled = true;
        config.preferredSource = result.preferredSource || 'tidal';
        config.generateM3u8 = result.generateM3u8 || false;
        config.skipUnavailableTracks = result.skipUnavailableTracks || false;
        config.autoQualityFallback = result.autoQualityFallback || false;
        config.autoStopOnCooldown = result.autoStopOnCooldown || false;
        config.firstArtistOnly = result.firstArtistOnly || false;
        config.sourceOrder = result.sourceOrder?.length ? result.sourceOrder : ['soulseek', 'tidal', 'qobuz', 'amazon', 'bandcamp'];
        sourceOrder = config.sourceOrder.length > 0
          ? config.sourceOrder
          : ['tidal', 'qobuz', 'amazon', 'bandcamp', 'soulseek'];
        config.qualityOrder = result.qualityOrder?.length ? result.qualityOrder : ['HI_RES', 'LOSSLESS', 'HIGH'];
        config.proxyUrl = result.proxyUrl || '';
        config.tidalCustomEndpoint = result.tidalCustomEndpoint || '';
        config.qobuzCustomEndpoint = result.qobuzCustomEndpoint || '';
        config.tidalPriorityEndpoints = result.tidalPriorityEndpoints || [];
        config.qobuzPriorityEndpoints = result.qobuzPriorityEndpoints || [];
        config.amazonPriorityEndpoints = result.amazonPriorityEndpoints || [];
        tidalPriorityText = config.tidalPriorityEndpoints.join('\n');
        qobuzPriorityText = config.qobuzPriorityEndpoints.join('\n');
        amazonPriorityText = config.amazonPriorityEndpoints.join('\n');
        config.skipExisting = result.skipExisting !== false;
        config.externalLibraryPaths = result.externalLibraryPaths || [];
        externalLibraryPathsText = config.externalLibraryPaths.join('\n');
        config.artistSeparator = result.artistSeparator || '; ';
        config.playlistSubfolder = result.playlistSubfolder !== false;
        config.folderTemplate = result.folderTemplate || '';
        syncFolderTemplatePreset(config.folderTemplate);
        config.countryCode = result.countryCode || 'US';
        config.fontFamily = result.fontFamily || '';
        config.downloadQuality = result.downloadQuality || 'LOSSLESS';
        config.enableReplayGain = result.enableReplayGain || false;
        config.soulseekEnabled = result.soulseekEnabled || false;
        config.soulseekUsername = result.soulseekUsername || '';
        config.soulseekPassword = result.soulseekPassword || '';
        config.jellyfinEnabled = result.jellyfinEnabled || false;
        config.jellyfinUrl = result.jellyfinUrl || '';
        config.jellyfinApiKey = result.jellyfinApiKey || '';
        downloadFolder.set(config.downloadFolder);
      }

      try {
        logFilePath = await GetLogFilePath();
      } catch (e) {
        console.error('Failed to get log file path:', e);
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

  async function testSoulseekLogin() {
    testingLogin = true;
    soulseekLoginResult = null;
    try {
      const result = await TestSoulseekConnection(config.soulseekUsername, config.soulseekPassword);
      soulseekLoginResult = { success: result.success, message: result.message };
    } catch (e) {
      soulseekLoginResult = { success: false, message: $t('settings.soulseek.testError') };
    } finally {
      testingLogin = false;
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
        language: config.language,
        accentColor: config.accentColor,
        dateFormat: config.dateFormat,
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
        soulseekEnabled: config.soulseekEnabled,
        soulseekUsername: config.soulseekUsername || '',
        soulseekPassword: config.soulseekPassword || '',
        jellyfinEnabled: config.jellyfinEnabled,
        jellyfinUrl: config.jellyfinUrl || '',
        jellyfinApiKey: config.jellyfinApiKey || '',
        preferredSource: config.preferredSource,
        generateM3u8: config.generateM3u8,
        skipUnavailableTracks: config.skipUnavailableTracks,
        autoQualityFallback: config.autoQualityFallback,
        autoStopOnCooldown: config.autoStopOnCooldown,
        firstArtistOnly: config.firstArtistOnly,
        sourceOrder: config.sourceOrder,
        qualityOrder: config.qualityOrder,
        proxyUrl: config.proxyUrl || '',
        tidalCustomEndpoint: config.tidalCustomEndpoint || '',
        qobuzCustomEndpoint: config.qobuzCustomEndpoint || '',
        tidalPriorityEndpoints: config.tidalPriorityEndpoints,
        qobuzPriorityEndpoints: config.qobuzPriorityEndpoints,
        amazonPriorityEndpoints: config.amazonPriorityEndpoints,
        skipExisting: config.skipExisting,
        externalLibraryPaths: config.externalLibraryPaths,
        artistSeparator: config.artistSeparator,
        playlistSubfolder: config.playlistSubfolder,
        folderTemplate: config.folderTemplate,
        countryCode: config.countryCode,
        fontFamily: config.fontFamily,
        downloadQuality: config.downloadQuality,
        enableReplayGain: config.enableReplayGain,
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
      toastStore.show($t('settings.toast.saved'));
    } catch (error) {
      console.error('Error saving config:', error);
      toastStore.show($t('settings.toast.saveError'), 'error');
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
        config.language = result.language ?? '';
        setLanguage(config.language);
        config.accentColor = result.accentColor || '#f472b6';
        config.dateFormat = result.dateFormat === 'year' ? 'year' : 'full';
        config.soundEffects = result.soundEffects || false;
        config.soundVolume = result.soundVolume || 70;
        config.embedLyrics = result.embedLyrics || false;
        config.preferSyncedLyrics = result.preferSyncedLyrics !== false;
        config.saveLyricsFile = result.saveLyricsFile || false;
        config.autoAnalyze = result.autoAnalyze || false;
        config.tidalEnabled = result.tidalEnabled !== false;
        config.qobuzEnabled = result.qobuzEnabled || false;
        config.soulseekEnabled = result.soulseekEnabled || false;
        config.soulseekUsername = result.soulseekUsername || '';
        config.soulseekPassword = result.soulseekPassword || '';
        config.jellyfinEnabled = result.jellyfinEnabled || false;
        config.jellyfinUrl = result.jellyfinUrl || '';
        config.jellyfinApiKey = result.jellyfinApiKey || '';
        config.preferredSource = result.preferredSource || 'tidal';
        config.generateM3u8 = result.generateM3u8 || false;
        config.skipUnavailableTracks = result.skipUnavailableTracks || false;
        config.autoQualityFallback = result.autoQualityFallback || false;
        config.autoStopOnCooldown = result.autoStopOnCooldown || false;
        config.firstArtistOnly = result.firstArtistOnly || false;
        config.sourceOrder = result.sourceOrder?.length ? result.sourceOrder : ['soulseek', 'tidal', 'qobuz', 'amazon', 'bandcamp'];
        sourceOrder = config.sourceOrder.length > 0
          ? config.sourceOrder
          : ['tidal', 'qobuz', 'amazon', 'bandcamp', 'soulseek'];
        config.qualityOrder = result.qualityOrder?.length ? result.qualityOrder : ['HI_RES', 'LOSSLESS', 'HIGH'];
        config.skipExisting = result.skipExisting !== false;
        config.externalLibraryPaths = result.externalLibraryPaths || [];
        externalLibraryPathsText = config.externalLibraryPaths.join('\n');
        config.artistSeparator = result.artistSeparator || '; ';
        config.playlistSubfolder = result.playlistSubfolder !== false;
        config.folderTemplate = result.folderTemplate || '';
        syncFolderTemplatePreset(config.folderTemplate);
        config.countryCode = result.countryCode || 'US';
        config.fontFamily = result.fontFamily || '';
        config.downloadQuality = result.downloadQuality || 'LOSSLESS';
        config.enableReplayGain = result.enableReplayGain || false;
        // Note: download folder and Qobuz credentials are preserved
        themeStore.setTheme(config.theme);
        handleAccentColorChange(config.accentColor);
        toastStore.show($t('settings.toast.reset'));
      }
    } catch (error) {
      console.error('Error resetting:', error);
      toastStore.show($t('settings.toast.resetError'), 'error');
    } finally {
      isResetting = false;
      showResetConfirm = false;
    }
  }
</script>

<div class="settings-page">
  <div class="settings-header">
    <h1>{$t('settings.title')}</h1>
    <div class="header-actions">
      <button class="btn-secondary" onclick={openConfig}>
        <FolderOpen size={16} />
        {$t('settings.header.openConfigFolder')}
      </button>
      <button class="btn-secondary" onclick={exportConfig}>
        {$t('settings.header.exportConfig')}
      </button>
      <button class="btn-secondary" onclick={importConfig}>
        {$t('settings.header.importConfig')}
      </button>
      <button class="btn-secondary" onclick={() => showResetConfirm = true} disabled={isResetting}>
        {$t('settings.header.reset')}
      </button>
      <button class="btn-accent" onclick={saveConfig} disabled={isSaving}>
        {#if isSaving}
          <div class="spinner"></div>
          {$t('settings.header.saving')}
        {:else}
          {$t('settings.header.save')}
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
        <div class="group-title">{$t('settings.group.downloadsAppearance')}</div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="download-folder">{$t('settings.download.path')}</label>
            <span class="setting-desc">{$t('settings.download.pathDesc')}</span>
          </div>
          <div class="setting-control folder-control">
            <input
              type="text"
              id="download-folder"
              bind:value={config.downloadFolder}
              readonly
              placeholder={$t('settings.download.selectFolder')}
              class="setting-input folder-input"
            />
            <button class="browse-btn" onclick={selectFolder}>
              <FolderOpen size={16} />
              {$t('settings.download.browse')}
            </button>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="theme">{$t('settings.theme.mode')}</label>
            <span class="setting-desc">{$t('settings.theme.modeDesc')}</span>
          </div>
          <div class="setting-control">
            <select id="theme" value={config.theme} onchange={handleThemeChange} class="setting-select">
              <option value="dark">{$t('settings.theme.dark')}</option>
              <option value="light">{$t('settings.theme.light')}</option>
              <option value="system">{$t('settings.theme.system')}</option>
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="language">{$t('settings.language.label')}</label>
            <span class="setting-desc">{$t('settings.language.description')}</span>
          </div>
          <div class="setting-control">
            <select id="language" value={config.language} onchange={handleLanguageChange} class="setting-select">
              <option value="">{$t('settings.language.system')}</option>
              <option value="en">English</option>
              <option value="fr">Français</option>
              <option value="de">Deutsch</option>
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">{$t('settings.accent.label')}</span>
          </div>
          <div class="accent-swatches" role="radiogroup" aria-label={$t('settings.accent.groupLabel')}>
            {#each accentPresets as preset}
              <button
                class="swatch"
                class:active={config.accentColor === preset.color}
                style="background-color: {preset.color}"
                title={$t(preset.labelKey)}
                aria-label={$t(preset.labelKey)}
                onclick={() => handleAccentColorChange(preset.color)}
              ></button>
            {/each}
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="font-family">{$t('settings.font.label')}</label>
          </div>
          <div class="setting-control">
            <select id="font-family" value={config.fontFamily} onchange={handleFontChange} class="setting-select">
              <option value="">{$t('settings.font.default')}</option>
              {#each fontPresets as font}
                <option value={font.value}>{font.name}</option>
              {/each}
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="sound-effects">{$t('settings.sound.label')}</label>
            <span class="setting-desc">{$t('settings.sound.desc')}</span>
          </div>
          <div class="setting-control sound-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.soundEffects} />
              <span class="toggle-slider"></span>
            </label>
            <button class="test-sound-btn" onclick={testSound} title={$t('settings.sound.test')}>
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
              <label for="sound-volume">{$t('settings.sound.volume')}</label>
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

        <div class="group-title" style="margin-top:1.5rem">{$t('settings.soulseek.group')}</div>

        <div class="soulseek-info-box">
          <p>{$t('settings.soulseek.info1a')} <code>sldl</code> {$t('settings.soulseek.info1b')}</p>
          <p>{$t('settings.soulseek.info2a')} <strong>Nicotine+</strong>{$t('settings.soulseek.info2b')}</p>
          <p>{$t('settings.soulseek.info3a')} <a href="https://www.slsknet.org/news/node/1" target="_blank" rel="noopener">{$t('settings.soulseek.info3link')}</a> {$t('settings.soulseek.info3b')}</p>
          {#if sldlStatus}
            {#if sldlStatus.installed}
              <p class="sldl-status sldl-ok">✓ {$t('settings.soulseek.installed', { version: sldlStatus.version })}</p>
            {:else}
              <p class="sldl-status sldl-missing">✗ {$t('settings.soulseek.notFoundAt')} <code>{sldlStatus.path}</code></p>
              {#if installingSldl}
                <div class="ffmpeg-progress" style="margin-top:0.5rem">
                  <div class="ffmpeg-progress-bar">
                    <div class="ffmpeg-progress-fill" style="width: {sldlInstallProgress.percent}%"></div>
                  </div>
                  <span class="ffmpeg-progress-text">
                    {sldlInstallProgress.stage === 'downloading' ? $t('settings.progress.downloading', { percent: Math.round(sldlInstallProgress.percent) }) : ''}
                    {sldlInstallProgress.stage === 'extracting' ? $t('settings.progress.extracting') : ''}
                    {sldlInstallProgress.stage === 'complete' ? $t('settings.progress.done') : ''}
                    {sldlInstallProgress.stage === 'error' ? $t('settings.progress.failed') : ''}
                  </span>
                </div>
              {:else}
                <button class="btn-accent" style="margin-top:0.5rem" onclick={installSldlHandler}>{$t('settings.soulseek.install')}</button>
              {/if}
            {/if}
          {/if}
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">{$t('settings.soulseek.enable')}</span>
            <span class="setting-desc">{$t('settings.soulseek.enableDesc')}</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.soulseekEnabled} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        {#if config.soulseekEnabled}
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">{$t('settings.soulseek.username')}</span>
            <span class="setting-desc">{$t('settings.soulseek.usernameDesc')}</span>
          </div>
          <div class="setting-control wide">
            <input
              type="text"
              class="setting-input"
              bind:value={config.soulseekUsername}
              placeholder={$t('settings.soulseek.usernamePlaceholder')}
            />
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">{$t('settings.soulseek.password')}</span>
            <span class="setting-desc">{$t('settings.soulseek.passwordDesc')}</span>
          </div>
          <div class="setting-control wide">
            <input
              type="password"
              class="setting-input"
              bind:value={config.soulseekPassword}
              placeholder="••••••••"
            />
          </div>
        </div>

        <div class="soulseek-login-row">
          <button
            class="btn-soulseek-login"
            onclick={testSoulseekLogin}
            disabled={testingLogin || !config.soulseekUsername || !config.soulseekPassword}
          >
            {testingLogin ? $t('settings.soulseek.connecting') : $t('settings.soulseek.login')}
          </button>
          {#if soulseekLoginResult}
            <span class="soulseek-login-result" class:ok={soulseekLoginResult.success} class:fail={!soulseekLoginResult.success}>
              {soulseekLoginResult.success ? '✓' : '✗'} {soulseekLoginResult.message}
              {#if soulseekLoginResult.success}<span class="save-hint">{$t('settings.soulseek.saveHint')}</span>{/if}
            </span>
          {/if}
        </div>
        <p class="firewall-hint">{$t('settings.soulseek.firewallA')} <code>sldl</code> {$t('settings.soulseek.firewallB')}</p>
        {/if}

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">{$t('settings.jellyfin.enable')}</span>
            <span class="setting-desc">{$t('settings.jellyfin.enableDesc')}</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.jellyfinEnabled} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        {#if config.jellyfinEnabled}
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">{$t('settings.jellyfin.url')}</span>
            <span class="setting-desc">{$t('settings.jellyfin.urlDesc')}</span>
          </div>
          <div class="setting-control wide">
            <input
              type="text"
              class="setting-input"
              bind:value={config.jellyfinUrl}
              placeholder={'http://localhost:8096'}
            />
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">{$t('settings.jellyfin.apiKey')}</span>
            <span class="setting-desc">{$t('settings.jellyfin.apiKeyDesc')}</span>
          </div>
          <div class="setting-control wide">
            <input
              type="password"
              class="setting-input"
              bind:value={config.jellyfinApiKey}
              placeholder="••••••••"
            />
          </div>
        </div>
        {/if}
      </div>

      <!-- Right Column -->
      <div class="settings-column">
        <div class="group-title">{$t('settings.group.sourcesQuality')}</div>

        <div class="settings-section">
          <p class="settings-hint">{$t('settings.sources.dragHint')}</p>
          <div class="source-priority-list">
            {#each sourceOrder as source, i}
              <div
                class="source-priority-item"
                draggable="true"
                ondragstart={(e) => onDragStart(e, i)}
                ondragover={(e) => onDragOver(e, i)}
                ondrop={(e) => onDrop(e, i)}
                ondragend={() => { dragIndex = null; }}
              >
                <span class="drag-handle">⠿</span>
                <span class="source-priority-name">{sourceLabels[source] ?? source}</span>
                <span class="priority-badge">{i + 1}</span>
              </div>
            {/each}
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="preferred-source">{$t('settings.sources.mode')}</label>
            <span class="setting-desc">{$t('settings.sources.modeDesc')}</span>
          </div>
          <div class="setting-control">
            <select id="preferred-source" bind:value={config.preferredSource} class="setting-select">
              <option value="tidal">Tidal</option>
              <option value="qobuz">Qobuz</option>
              <option value="auto">{$t('settings.sources.auto')}</option>
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="download-quality">{$t('settings.quality.label')}</label>
            <span class="setting-desc">{$t('settings.quality.desc')}</span>
          </div>
          <div class="setting-control">
            <select id="download-quality" bind:value={config.downloadQuality} class="setting-select">
              <option value="HI_RES">{$t('settings.quality.hiRes')}</option>
              <option value="LOSSLESS">{$t('settings.quality.lossless')}</option>
              <option value="HIGH">{$t('settings.quality.high')}</option>
              <option value="ATMOS">{$t('settings.quality.atmos')}</option>
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>{$t('settings.quality.fallback')}</label>
            <span class="setting-desc">{$t('settings.quality.fallbackDesc')}</span>
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
            <label>ReplayGain</label>
            <span class="setting-desc">{$t('settings.replayGain.desc')}</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.enableReplayGain} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="country-code">{$t('settings.region.label')}</label>
            <span class="setting-desc">{$t('settings.region.desc')}</span>
          </div>
          <div class="setting-control">
            <select id="country-code" bind:value={config.countryCode} class="setting-select">
              {#each countries as c}
                <option value={c.code}>{$t(c.labelKey)} ({c.code})</option>
              {/each}
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>Tidal</label>
            <span class="setting-desc">{$t('settings.tidal.enableDesc')}</span>
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
            <label>{$t('settings.skipExisting.label')}</label>
            <span class="setting-desc">{$t('settings.skipExisting.desc')}</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.skipExisting} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item setting-item-stack">
          <div class="setting-info">
            <span class="setting-label">{$t('settings.externalLibs.label')}</span>
            <span class="setting-desc">{$t('settings.externalLibs.desc')}</span>
          </div>
          <div class="setting-control wide">
            <textarea
              class="setting-input endpoint-list"
              value={externalLibraryPathsText}
              oninput={(e) => {
                externalLibraryPathsText = (e.target as HTMLTextAreaElement).value;
                config.externalLibraryPaths = externalLibraryPathsText.split('\n').map(s => s.trim()).filter(Boolean);
              }}
              placeholder={"/mnt/music/navidrome\n/srv/jellyfin/library"}
              rows={3}
              spellcheck={false}
            ></textarea>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>{$t('settings.skipUnavailable.label')}</label>
            <span class="setting-desc">{$t('settings.skipUnavailable.desc')}</span>
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
            <label>{$t('settings.autoStop.label')}</label>
            <span class="setting-desc">{$t('settings.autoStop.desc')}</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.autoStopOnCooldown} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="concurrent">{$t('settings.concurrent.label')}</label>
            <span class="setting-desc">{$t('settings.concurrent.desc')}</span>
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
            <span class="setting-label">{$t('settings.proxy.label')}</span>
            <span class="setting-desc">{$t('settings.proxy.desc')}</span>
          </div>
          <div class="setting-control wide">
            <input
              type="text"
              class="setting-input"
              bind:value={config.proxyUrl}
              placeholder={$t('settings.proxy.placeholder')}
            />
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">{$t('settings.logFile.label')}</span>
            <span class="setting-desc log-path">{logFilePath || $t('settings.logFile.unavailable')}</span>
          </div>
          {#if isWailsRuntime()}
            <div class="setting-control">
              <button class="btn-secondary" onclick={openConfig}>{$t('settings.logFile.open')}</button>
            </div>
          {/if}
        </div>

        <div class="setting-item setting-item-stack">
          <div class="setting-info">
            <span class="setting-label">{$t('settings.selfHosted.label')}</span>
            <span class="setting-desc">
              {$t('settings.selfHosted.descA')}
              <a href="https://github.com/kushiemoon-dev/FLACidal#self-hosted--private-endpoints" target="_blank" rel="noopener">{$t('settings.selfHosted.howTo')}</a>
            </span>
          </div>
        </div>

        <details class="advanced-instances">
          <summary>{$t('settings.selfHosted.advanced')}</summary>

          <div class="setting-item setting-item-stack">
            <div class="setting-info">
              <span class="setting-label">
                Tidal HiFi
                {#if config.tidalPriorityEndpoints.length > 0}
                  <span class="status-badge ok status-badge-sm">{$t('settings.selfHosted.configured', { n: config.tidalPriorityEndpoints.length })}</span>
                {/if}
              </span>
            </div>
            <div class="setting-control wide">
              <textarea
                class="setting-input endpoint-list"
                value={tidalPriorityText}
                oninput={(e) => {
                  tidalPriorityText = (e.target as HTMLTextAreaElement).value;
                  config.tidalPriorityEndpoints = tidalPriorityText.split('\n').map(s => s.trim()).filter(Boolean);
                }}
                placeholder={"https://your-hifi-api-1.com\nhttps://your-hifi-api-2.com"}
                rows={3}
                spellcheck={false}
              ></textarea>
            </div>
          </div>

          <div class="setting-item setting-item-stack">
            <div class="setting-info">
              <span class="setting-label">
                Qobuz
                {#if config.qobuzPriorityEndpoints.length > 0}
                  <span class="status-badge ok status-badge-sm">{$t('settings.selfHosted.configured', { n: config.qobuzPriorityEndpoints.length })}</span>
                {/if}
              </span>
            </div>
            <div class="setting-control wide">
              <textarea
                class="setting-input endpoint-list"
                value={qobuzPriorityText}
                oninput={(e) => {
                  qobuzPriorityText = (e.target as HTMLTextAreaElement).value;
                  config.qobuzPriorityEndpoints = qobuzPriorityText.split('\n').map(s => s.trim()).filter(Boolean);
                }}
                placeholder={"https://your-qobuz-proxy-1.com\nhttps://your-qobuz-proxy-2.com"}
                rows={3}
                spellcheck={false}
              ></textarea>
            </div>
          </div>

          <div class="setting-item setting-item-stack">
            <div class="setting-info">
              <span class="setting-label">
                Amazon
                {#if config.amazonPriorityEndpoints.length > 0}
                  <span class="status-badge ok status-badge-sm">{$t('settings.selfHosted.configured', { n: config.amazonPriorityEndpoints.length })}</span>
                {/if}
              </span>
            </div>
            <div class="setting-control wide">
              <textarea
                class="setting-input endpoint-list"
                value={amazonPriorityText}
                oninput={(e) => {
                  amazonPriorityText = (e.target as HTMLTextAreaElement).value;
                  config.amazonPriorityEndpoints = amazonPriorityText.split('\n').map(s => s.trim()).filter(Boolean);
                }}
                placeholder={"https://your-amazon-proxy-1.com\nhttps://your-amazon-proxy-2.com"}
                rows={3}
                spellcheck={false}
              ></textarea>
            </div>
          </div>
        </details>

      </div>
    </div>

    <!-- ==================== FILE MANAGEMENT TAB ==================== -->
    {:else if activeTab === 'file-management'}
    <div class="settings-grid">
      <!-- Left Column -->
      <div class="settings-column">
        <div class="group-title">{$t('settings.group.fileNaming')}</div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="naming-preset">{$t('settings.naming.preset')}</label>
            <span class="setting-desc">{$t('settings.naming.presetDesc')}</span>
          </div>
          <div class="setting-control">
            <select
              id="naming-preset"
              class="setting-select"
              onchange={(e) => { const v = (e.target as HTMLSelectElement).value; if (v) config.fileNameFormat = v; }}
            >
              <option value="">{$t('settings.naming.custom')}</option>
              {#each namingPresets as preset}
                <option value={preset.template} selected={config.fileNameFormat === preset.template}>{$t(preset.labelKey)}: {preset.template}</option>
              {/each}
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="file-naming">{$t('settings.naming.template')}</label>
            <span class="setting-desc">{$t('settings.naming.templateVars')} {'{artist}'}, {'{albumartist}'}, {'{title}'}, {'{album}'}, {'{track}'}, {'{discnumber}'}, {'{year}'}, {'{isrc}'}</span>
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
            <label for="date-format">{$t('settings.naming.dateFormat')}</label>
            <span class="setting-desc">{$t('settings.naming.dateFormatDesc')}</span>
          </div>
          <div class="setting-control">
            <select id="date-format" bind:value={config.dateFormat} class="setting-select">
              <option value="full">{$t('settings.naming.dateFormatFull')}</option>
              <option value="year">{$t('settings.naming.dateFormatYear')}</option>
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="artist-separator">{$t('settings.naming.artistSeparator')}</label>
            <span class="setting-desc">{$t('settings.naming.artistSeparatorDesc')}</span>
          </div>
          <div class="setting-control">
            <select id="artist-separator" bind:value={config.artistSeparator} class="setting-select">
              {#each artistSeparators as sep}
                <option value={sep.value}>{$t(sep.labelKey)}</option>
              {/each}
            </select>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="folder-template">{$t('settings.folder.label')}</label>
            <span class="setting-desc">{$t('settings.folder.desc')}</span>
          </div>
          <div class="setting-control">
            <select
              id="folder-template"
              class="setting-select"
              value={folderTemplatePreset}
              onchange={handleFolderTemplateChange}
            >
              <option value="">{$t('settings.folder.none')}</option>
              <option value={'{artist}/{album}'}>{$t('settings.folder.artistAlbum')}</option>
              <option value={'{albumartist}/{album}'}>{$t('settings.folder.albumArtistAlbum')}</option>
              <option value={'{artist}/{year} - {album}'}>{$t('settings.folder.artistYearAlbum')}</option>
              <option value={'{year}/{artist}/{album}'}>{$t('settings.folder.yearArtistAlbum')}</option>
              <option value="custom">{$t('settings.folder.customOption')}</option>
            </select>
          </div>
        </div>
        {#if folderTemplatePreset === 'custom'}
          <div class="setting-item">
            <div class="setting-info">
              <label for="folder-template-custom">{$t('settings.folder.customLabel')}</label>
              <span class="setting-desc">{$t('settings.naming.templateVars')} {'{artist}'}, {'{albumartist}'}, {'{album}'}, {'{year}'}, {'{label}'}</span>
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
            <label>{$t('settings.playlistSubfolder.label')}</label>
            <span class="setting-desc">{$t('settings.playlistSubfolder.desc')}</span>
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
            <label>{$t('settings.m3u8.label')}</label>
            <span class="setting-desc">{$t('settings.m3u8.desc')}</span>
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
        <div class="group-title">{$t('settings.group.metadataTags')}</div>

        <div class="setting-item">
          <div class="setting-info">
            <label>{$t('settings.lyrics.embed')}</label>
            <span class="setting-desc">{$t('settings.lyrics.embedDesc')}</span>
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
              <label>{$t('settings.lyrics.synced')}</label>
              <span class="setting-desc">{$t('settings.lyrics.syncedDesc')}</span>
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
              <label>{$t('settings.lyrics.saveFile')}</label>
              <span class="setting-desc">{$t('settings.lyrics.saveFileDesc')}</span>
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
            <label>{$t('settings.cover.embed')}</label>
            <span class="setting-desc">{$t('settings.cover.embedDesc')}</span>
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
            <label>{$t('settings.cover.saveFile')}</label>
            <span class="setting-desc">{$t('settings.cover.saveFileDesc')}</span>
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
            <label>{$t('settings.cover.saveFolder')}</label>
            <span class="setting-desc">{$t('settings.cover.saveFolderDesc')}</span>
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
            <label>{$t('settings.firstArtist.label')}</label>
            <span class="setting-desc">{$t('settings.firstArtist.desc')}</span>
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
            <label>{$t('settings.genre.embed')}</label>
            <span class="setting-desc">{$t('settings.genre.embedDesc')}</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.embedGenre} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>{$t('settings.genre.single')}</label>
            <span class="setting-desc">{$t('settings.genre.singleDesc')}</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.useSingleGenre} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label>{$t('settings.autoAnalyze.label')}</label>
            <span class="setting-desc">{$t('settings.autoAnalyze.desc')}</span>
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
      <div class="group-title">{$t('settings.health.group')}</div>
      <div class="api-status-header">
        <button class="btn-secondary" onclick={checkSourceHealth} disabled={checkingSourceHealth}>
          {checkingSourceHealth ? $t('settings.health.checking') : $t('settings.health.check')}
        </button>
      </div>
      {#if sourceHealth.length > 0}
        <div class="api-status-list">
          {#each sourceHealth as src}
            <!-- status only counts live/probation endpoints, while tier1.healthy counts
                 anything not dead, so a merely blacklisted self-hosted endpoint would
                 otherwise render as "dead" right next to "Self-host: healthy". A working
                 self-hosted endpoint means the source is degraded, never fully dead. -->
            {@const displayStatus = src.tier1?.healthy && src.status === 'dead' ? 'degraded' : src.status}
            <div class="api-status-item" class:api-status-item-expanded={src.endpoints?.length > 0}>
              <div class="api-status-row">
                <span class="api-name">{src.displayName}</span>
                <div style="display:flex;flex-direction:column;align-items:flex-end;gap:2px">
                  <span class="status-badge"
                    class:ok={displayStatus === 'online'}
                    class:error={displayStatus === 'dead'}
                    class:slow={displayStatus === 'degraded' || displayStatus === 'untested'}
                    title={displayStatus !== src.status ? $t('settings.health.overrideTip') : null}>
                    {displayStatus}{src.latencyMs > 0 ? ` (${src.latencyMs}ms)` : ''}
                  </span>
                  {#if src.tier1}
                    <span class="status-badge status-badge-sm" class:ok={src.tier1.healthy} class:error={!src.tier1.healthy}>
                      {src.tier1.healthy ? $t('settings.health.selfHostHealthy') : $t('settings.health.selfHostUnhealthy')}
                    </span>
                  {/if}
                  {#if src.reason}
                    <span style="font-size:0.7rem;color:var(--color-text-tertiary);max-width:260px;text-align:right">{src.reason}</span>
                  {/if}
                </div>
              </div>
              {#if src.endpoints?.length > 0}
                <div class="endpoint-health-list">
                  {#each src.endpoints as ep}
                    <div class="endpoint-health-row">
                      <span class="endpoint-health-url" title={ep.url}>{ep.url.replace(/^https?:\/\//, '')}</span>
                      <div style="display:flex;align-items:center;gap:6px;flex-shrink:0">
                        {#if ep.latencyMs > 0}
                          <span class="endpoint-latency">{ep.latencyMs}ms</span>
                        {/if}
                        <span class="status-badge status-badge-sm"
                          class:ok={ep.state === 'live'}
                          class:error={ep.state === 'dead'}
                          class:slow={ep.state === 'blacklisted' || ep.state === 'probation'}>
                          {ep.state}{ep.state === 'probation' ? ` #${ep.revivals}` : ''}
                        </span>
                      </div>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </section>

    <section class="settings-section">
      <div class="group-title">{$t('settings.api.group')}</div>
      <div class="api-status-header">
        <button class="btn-secondary" onclick={checkAPI} disabled={checkingAPI}>
          {checkingAPI ? $t('settings.health.checking') : $t('settings.api.check')}
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
          <span class="setting-label">{$t('settings.ffmpeg.status')}</span>
          <span class="setting-desc">{$t('settings.ffmpeg.statusDesc')}</span>
        </div>
        <div class="setting-control">
          {#if ffmpegInfo?.available}
            <span class="status-badge ok">{$t('settings.ffmpeg.installed')}</span>
          {:else}
            <span class="status-badge error">{$t('settings.ffmpeg.notFound')}</span>
          {/if}
        </div>
      </div>

      {#if ffmpegInfo?.available}
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">{$t('settings.ffmpeg.version')}</span>
            <span class="setting-desc ffmpeg-version">{ffmpegInfo.version || $t('settings.ffmpeg.unknown')}</span>
          </div>
        </div>
      {:else}
        <div class="setting-item">
          <div class="setting-info">
            <span class="setting-label">{$t('settings.ffmpeg.autoInstall')}</span>
            <span class="setting-desc">{$t('settings.ffmpeg.autoInstallDesc')}</span>
          </div>
          <div class="setting-control">
            {#if installingFFmpeg}
              <div class="ffmpeg-progress">
                <div class="ffmpeg-progress-bar">
                  <div class="ffmpeg-progress-fill" style="width: {ffmpegProgress.percent}%"></div>
                </div>
                <span class="ffmpeg-progress-text">
                  {ffmpegProgress.stage === 'downloading' ? $t('settings.progress.downloading', { percent: Math.round(ffmpegProgress.percent) }) : ''}
                  {ffmpegProgress.stage === 'extracting' ? $t('settings.progress.extracting') : ''}
                  {ffmpegProgress.stage === 'complete' ? $t('settings.progress.done') : ''}
                  {ffmpegProgress.stage === 'error' ? $t('settings.progress.failed') : ''}
                </span>
              </div>
            {:else}
              <button class="btn-accent" onclick={installFFmpegHandler}>{$t('settings.ffmpeg.install')}</button>
            {/if}
          </div>
        </div>
      {/if}
    </section>

    <!-- About & Updates -->
    <section class="settings-section about">
      <div class="group-title">{$t('settings.about.group')}</div>
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
            <span class="version">{$t('settings.about.version', { version: appVersion || '...' })}</span>
          </div>
        </div>
        <p class="app-desc">{$t('settings.about.desc')}</p>
        <div class="update-check">
          <button class="btn-secondary" onclick={checkUpdate} disabled={checkingUpdate}>
            {checkingUpdate ? $t('settings.health.checking') : $t('settings.about.checkUpdates')}
          </button>
          {#if updateInfo}
            {#if updateInfo.hasUpdate}
              <span class="update-available">{$t('settings.about.updateAvailable', { version: updateInfo.version })} <a href={updateInfo.releaseUrl} target="_blank" rel="noopener">{$t('settings.about.viewRelease')}</a></span>
            {:else}
              <span class="update-current">{$t('settings.about.upToDate')}</span>
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
      <h3>{$t('settings.reset.title')}</h3>
      <p>{$t('settings.reset.body')}</p>
      <div class="modal-actions">
        <button class="modal-btn cancel" onclick={() => showResetConfirm = false}>
          {$t('settings.reset.cancel')}
        </button>
        <button class="modal-btn confirm" onclick={handleReset} disabled={isResetting}>
          {#if isResetting}
            {$t('settings.reset.resetting')}
          {:else}
            {$t('settings.reset.confirm')}
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .log-path {
    user-select: text;
    word-break: break-all;
  }

  .settings-page {
    padding: 32px;
    max-width: 960px;
  }

  .source-priority-list { display: flex; flex-direction: column; gap: 0.4rem; max-width: 360px; margin-bottom: 1rem; }
  .source-priority-item { display: flex; align-items: center; gap: 0.75rem; padding: 0.5rem 0.75rem; border-radius: 6px; background: rgba(255,255,255,0.06); cursor: grab; user-select: none; transition: background 0.1s; }
  .source-priority-item:hover { background: rgba(255,255,255,0.1); }
  .source-priority-item:active { cursor: grabbing; }
  .drag-handle { font-size: 1.1rem; opacity: 0.5; }
  .source-priority-name { flex: 1; font-size: 0.9rem; }
  .priority-badge { font-size: 0.75rem; opacity: 0.5; font-weight: 600; min-width: 1.5rem; text-align: right; }
  .settings-hint { font-size: 0.8rem; opacity: 0.6; margin-bottom: 0.75rem; }
  .firewall-hint { font-size: 0.8rem; opacity: 0.6; margin-top: 0.5rem; }

  .soulseek-info-box {
    background: color-mix(in srgb, var(--color-accent, #6366f1) 8%, transparent);
    border: 1px solid color-mix(in srgb, var(--color-accent, #6366f1) 30%, transparent);
    border-radius: 8px;
    padding: 0.75rem 1rem;
    margin-bottom: 0.75rem;
    font-size: 0.82rem;
    line-height: 1.5;
  }
  .soulseek-info-box p { margin: 0 0 0.35rem; opacity: 0.85; }
  .soulseek-info-box p:last-child { margin-bottom: 0; }
  .soulseek-info-box code { opacity: 0.9; font-size: 0.8rem; }
  .soulseek-info-box a { color: var(--color-accent, #6366f1); text-decoration: underline; }
  .sldl-status { font-weight: 600; margin-top: 0.4rem !important; }
  .sldl-ok { color: #4ade80; }
  .sldl-missing { color: var(--color-warning, #f59e0b); }

  .soulseek-login-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-top: 0.5rem;
    padding: 0 0 0.25rem;
  }
  .btn-soulseek-login {
    padding: 0.4rem 1rem;
    border-radius: 6px;
    border: 1px solid var(--color-accent, #6366f1);
    background: transparent;
    color: var(--color-accent, #6366f1);
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
  }
  .btn-soulseek-login:hover:not(:disabled) {
    background: var(--color-accent, #6366f1);
    color: #fff;
  }
  .btn-soulseek-login:disabled { opacity: 0.4; cursor: not-allowed; }
  .soulseek-login-result { font-size: 0.82rem; font-weight: 600; }
  .soulseek-login-result.ok { color: #4ade80; }
  .soulseek-login-result.fail { color: var(--color-warning, #f59e0b); }
  .save-hint { font-weight: 400; opacity: 0.75; }

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

  .setting-item-stack {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .setting-item-stack .setting-control.wide {
    max-width: 100%;
    margin-left: 0;
    width: 100%;
  }

  .endpoint-list {
    resize: vertical;
    min-height: 72px;
    font-family: 'JetBrains Mono', monospace;
    font-size: 12px;
    line-height: 1.5;
  }

  .setting-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex: 1;
    min-width: 0;
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
    flex-direction: column;
    padding: 10px 14px;
    background: var(--color-bg-primary, #000);
    border: 1px solid var(--color-border-subtle, #222);
    border-radius: 8px;
    gap: 0;
  }

  .api-status-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .api-name {
    font-size: 14px;
    font-weight: 500;
  }

  .endpoint-health-list {
    margin-top: 8px;
    padding-top: 8px;
    border-top: 1px solid var(--color-border, #1a1a1a);
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .endpoint-health-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 8px;
  }

  .endpoint-health-url {
    font-family: 'JetBrains Mono', monospace;
    font-size: 11px;
    color: var(--color-text-tertiary, #666);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }

  .endpoint-latency {
    font-size: 11px;
    color: var(--color-text-tertiary, #666);
  }

  .status-badge-sm {
    font-size: 11px;
    padding: 2px 7px;
  }

  /* Advanced: self-hosted instances (collapsible) */
  .advanced-instances {
    margin-top: 8px;
  }

  .advanced-instances > summary {
    cursor: pointer;
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-tertiary, #525252);
    padding: 8px 0;
    list-style: none;
  }

  .advanced-instances > summary::-webkit-details-marker {
    display: none;
  }

  .advanced-instances > summary::before {
    content: '▸';
    display: inline-block;
    margin-right: 6px;
    transition: transform 0.15s;
  }

  .advanced-instances[open] > summary::before {
    transform: rotate(90deg);
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
