<script lang="ts">
  import { onMount } from 'svelte';
  import { downloadFolder } from '../stores/queue';
  import { themeStore, type ThemeMode, accentColor, accentPresets, applyAccentColor } from '../stores/theme';
  import { updateAudioSettings, testSound } from '../stores/audio';
  import {
    GetConfig,
    SaveConfig,
    SelectDownloadFolder,
    SetDownloadFolder,
    GetDownloadOptions,
    SetDownloadOptions,
    ResetToDefaults
  } from '../../wailsjs/go/main/App.js';

  let config = {
    downloadFolder: '',
    concurrentDownloads: 4,
    embedCover: true,
    fileNameFormat: '{artist} - {title}',
    theme: 'system' as ThemeMode,
    accentColor: '#f472b6',
    soundEffects: false,
    soundVolume: 70,
    embedLyrics: false,
    preferSyncedLyrics: true,
    tidalEnabled: true,
    qobuzEnabled: false,
    qobuzAppId: '',
    qobuzAppSecret: '',
    qobuzAuthToken: '',
    preferredSource: 'tidal'
  };
  let isSaving = false;
  let saveMessage = '';
  let showResetConfirm = false;
  let isResetting = false;

  // Update audio settings reactively
  $: updateAudioSettings(config.soundEffects, config.soundVolume);

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

  onMount(async () => {
    await loadConfig();
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
        config.tidalEnabled = result.tidalEnabled !== false;
        config.qobuzEnabled = result.qobuzEnabled || false;
        config.qobuzAppId = result.qobuzAppId || '';
        config.qobuzAppSecret = result.qobuzAppSecret || '';
        config.qobuzAuthToken = result.qobuzAuthToken || '';
        config.preferredSource = result.preferredSource || 'tidal';
        downloadFolder.set(config.downloadFolder);
      }

      // Also get download options
      const opts = await GetDownloadOptions();
      if (opts) {
        config.embedCover = opts.embedCover !== false;
        config.fileNameFormat = opts.fileNameFormat || '{artist} - {title}';
      }
    } catch (error) {
      console.error('Error loading config:', error);
    }
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
    saveMessage = '';

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
        fileNameFormat: config.fileNameFormat,
        soundEffects: config.soundEffects,
        soundVolume: config.soundVolume,
        embedLyrics: config.embedLyrics,
        preferSyncedLyrics: config.preferSyncedLyrics,
        tidalEnabled: config.tidalEnabled,
        qobuzEnabled: config.qobuzEnabled,
        qobuzAppId: config.qobuzAppId,
        qobuzAppSecret: config.qobuzAppSecret,
        qobuzAuthToken: config.qobuzAuthToken,
        preferredSource: config.preferredSource
      });

      // Save download options
      await SetDownloadOptions(
        'LOSSLESS',
        config.fileNameFormat,
        false,
        config.embedCover
      );
      saveMessage = 'Settings saved!';
      setTimeout(() => saveMessage = '', 3000);
    } catch (error) {
      console.error('Error saving config:', error);
      saveMessage = 'Error saving settings';
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
        config.fileNameFormat = result.fileNameFormat || '{artist} - {title}';
        config.theme = (result.theme as ThemeMode) || 'system';
        config.accentColor = result.accentColor || '#f472b6';
        config.soundEffects = result.soundEffects || false;
        config.soundVolume = result.soundVolume || 70;
        config.embedLyrics = result.embedLyrics || false;
        config.preferSyncedLyrics = result.preferSyncedLyrics !== false;
        config.tidalEnabled = result.tidalEnabled !== false;
        config.qobuzEnabled = result.qobuzEnabled || false;
        config.preferredSource = result.preferredSource || 'tidal';
        // Note: download folder and Qobuz credentials are preserved
        themeStore.setTheme(config.theme);
        handleAccentColorChange(config.accentColor);
        saveMessage = 'Settings reset to defaults!';
        setTimeout(() => saveMessage = '', 3000);
      }
    } catch (error) {
      console.error('Error resetting:', error);
      saveMessage = 'Error resetting settings';
    } finally {
      isResetting = false;
      showResetConfirm = false;
    }
  }
</script>

<div class="settings-page">
  <div class="settings-header">
    <h1>Settings</h1>
    <p class="subtitle">Configure FLACidal preferences</p>
  </div>

  <div class="settings-sections">
    <!-- Appearance Settings -->
    <section class="settings-section">
      <h2>Appearance</h2>

      <div class="setting-item">
        <div class="setting-info">
          <label for="theme">Theme</label>
          <span class="setting-desc">Choose your preferred color scheme</span>
        </div>
        <div class="setting-control">
          <select id="theme" value={config.theme} on:change={handleThemeChange} class="select-input">
            <option value="system">System</option>
            <option value="dark">Dark</option>
            <option value="light">Light</option>
          </select>
        </div>
      </div>

      <div class="setting-item">
        <div class="setting-info">
          <span class="setting-label">Accent Color</span>
          <span class="setting-desc">Customize the app's primary color</span>
        </div>
        <div class="setting-control accent-colors" role="radiogroup" aria-label="Accent color selection">
          {#each accentPresets as preset}
            <button
              class="color-swatch"
              class:active={config.accentColor === preset.color}
              style="background-color: {preset.color}"
              title={preset.name}
              on:click={() => handleAccentColorChange(preset.color)}
            ></button>
          {/each}
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
          <button class="test-sound-btn" on:click={testSound} title="Test sound">
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
            <span class="setting-desc">Sound effects volume ({config.soundVolume}%)</span>
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
    </section>

    <!-- Download Settings -->
    <section class="settings-section">
      <h2>Downloads</h2>

      <div class="setting-item">
        <div class="setting-info">
          <label for="download-folder">Download Folder</label>
          <span class="setting-desc">Where your FLAC files will be saved</span>
        </div>
        <div class="setting-control folder-control">
          <input
            type="text"
            id="download-folder"
            bind:value={config.downloadFolder}
            readonly
            placeholder="Select a folder..."
            class="folder-input"
          />
          <button class="browse-btn" on:click={selectFolder}>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
            </svg>
            Browse
          </button>
        </div>
      </div>

      <div class="setting-item">
        <div class="setting-info">
          <label for="concurrent">Concurrent Downloads</label>
          <span class="setting-desc">Number of simultaneous downloads</span>
        </div>
        <div class="setting-control">
          <select id="concurrent" bind:value={config.concurrentDownloads} class="select-input">
            <option value={1}>1</option>
            <option value={2}>2</option>
            <option value={3}>3</option>
            <option value={4}>4</option>
            <option value={6}>6</option>
            <option value={8}>8</option>
          </select>
        </div>
      </div>
    </section>

    <!-- Metadata Settings -->
    <section class="settings-section">
      <h2>Metadata</h2>

      <div class="setting-item">
        <div class="setting-info">
          <label for="embed-cover">Embed Cover Art</label>
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
          <label for="file-naming">File Naming</label>
          <span class="setting-desc">Template for downloaded file names</span>
        </div>
        <div class="setting-control">
          <select id="file-naming" bind:value={config.fileNameFormat} class="select-input">
            <option value={'{artist} - {title}'}>{'{artist} - {title}'}</option>
            <option value={'{title} - {artist}'}>{'{title} - {artist}'}</option>
            <option value={'{track}. {title}'}>{'{track}. {title}'}</option>
            <option value={'{track}. {artist} - {title}'}>{'{track}. {artist} - {title}'}</option>
          </select>
        </div>
      </div>
    </section>

    <!-- Lyrics Settings -->
    <section class="settings-section">
      <h2>Lyrics</h2>

      <div class="setting-item">
        <div class="setting-info">
          <label for="embed-lyrics">Auto-fetch Lyrics</label>
          <span class="setting-desc">Automatically fetch and embed lyrics during download</span>
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
            <label for="prefer-synced">Prefer Synced Lyrics</label>
            <span class="setting-desc">Prioritize time-synced (LRC) lyrics when available</span>
          </div>
          <div class="setting-control">
            <label class="toggle">
              <input type="checkbox" bind:checked={config.preferSyncedLyrics} />
              <span class="toggle-slider"></span>
            </label>
          </div>
        </div>
      {/if}

      <div class="lyrics-info">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="16" x2="12" y2="12"/>
          <line x1="12" y1="8" x2="12.01" y2="8"/>
        </svg>
        <span>Lyrics are fetched from LRCLIB. You can also manually fetch lyrics from the Files page.</span>
      </div>
    </section>

    <!-- Sources Settings -->
    <section class="settings-section">
      <h2>Music Sources</h2>

      <div class="setting-item">
        <div class="setting-info">
          <label for="tidal-enabled">Tidal</label>
          <span class="setting-desc">Download from Tidal (no account required)</span>
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
          <label for="qobuz-enabled">Qobuz</label>
          <span class="setting-desc">Download from Qobuz (requires credentials)</span>
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
            <span class="setting-desc">Your Qobuz application ID</span>
          </div>
          <div class="setting-control">
            <input
              type="text"
              id="qobuz-app-id"
              bind:value={config.qobuzAppId}
              placeholder="Enter App ID..."
              class="text-input"
            />
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="qobuz-app-secret">Qobuz App Secret</label>
            <span class="setting-desc">Your Qobuz application secret</span>
          </div>
          <div class="setting-control">
            <input
              type="password"
              id="qobuz-app-secret"
              bind:value={config.qobuzAppSecret}
              placeholder="Enter App Secret..."
              class="text-input"
            />
          </div>
        </div>

        <div class="setting-item">
          <div class="setting-info">
            <label for="qobuz-auth-token">Qobuz Auth Token</label>
            <span class="setting-desc">Your Qobuz user authentication token</span>
          </div>
          <div class="setting-control">
            <input
              type="password"
              id="qobuz-auth-token"
              bind:value={config.qobuzAuthToken}
              placeholder="Enter Auth Token..."
              class="text-input"
            />
          </div>
        </div>
      {/if}

      <div class="setting-item">
        <div class="setting-info">
          <label for="preferred-source">Preferred Source</label>
          <span class="setting-desc">Default source when both are available</span>
        </div>
        <div class="setting-control">
          <select id="preferred-source" bind:value={config.preferredSource} class="select-input">
            <option value="tidal">Tidal</option>
            <option value="qobuz">Qobuz</option>
          </select>
        </div>
      </div>

      <div class="source-info">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="16" x2="12" y2="12"/>
          <line x1="12" y1="8" x2="12.01" y2="8"/>
        </svg>
        <span>The source is automatically detected from the URL you paste. Tidal works without login.</span>
      </div>
    </section>

    <!-- About Section -->
    <section class="settings-section about">
      <h2>About</h2>
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
            <span class="version">Version 1.0.0</span>
          </div>
        </div>
        <p class="app-desc">High-quality FLAC downloader for Tidal. Download your favorite music in lossless quality.</p>
      </div>
    </section>
  </div>

  <div class="settings-footer">
    {#if saveMessage}
      <span class="save-message" class:error={saveMessage.includes('Error')}>{saveMessage}</span>
    {/if}
    <button
      class="reset-btn"
      on:click={() => showResetConfirm = true}
      disabled={isResetting}
    >
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
        <path d="M3 3v5h5"/>
      </svg>
      Reset to Defaults
    </button>
    <button
      class="save-btn"
      on:click={saveConfig}
      disabled={isSaving}
    >
      {#if isSaving}
        <div class="spinner"></div>
        Saving...
      {:else}
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/>
          <polyline points="17 21 17 13 7 13 7 21"/>
          <polyline points="7 3 7 8 15 8"/>
        </svg>
        Save Settings
      {/if}
    </button>
  </div>
</div>

<!-- Reset Confirmation Modal -->
{#if showResetConfirm}
  <div class="modal-overlay" on:click={() => showResetConfirm = false} on:keydown={(e) => e.key === 'Escape' && (showResetConfirm = false)} role="dialog" aria-modal="true">
    <div class="modal" on:click|stopPropagation on:keydown|stopPropagation role="document">
      <div class="modal-icon">
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
          <path d="M3 3v5h5"/>
        </svg>
      </div>
      <h3>Reset to Defaults?</h3>
      <p>This will reset all settings to their default values. Your download folder will be preserved.</p>
      <div class="modal-actions">
        <button class="modal-btn cancel" on:click={() => showResetConfirm = false}>
          Cancel
        </button>
        <button class="modal-btn confirm" on:click={handleReset} disabled={isResetting}>
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
    max-width: 700px;
  }

  .settings-header {
    margin-bottom: 32px;
  }

  .settings-header h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0 0 8px 0;
  }

  .subtitle {
    color: var(--color-text-tertiary);
    margin: 0;
  }

  .settings-sections {
    display: flex;
    flex-direction: column;
    gap: 32px;
  }

  .settings-section {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
    padding: 24px;
  }

  .settings-section h2 {
    font-size: 16px;
    font-weight: 600;
    margin: 0 0 20px 0;
    color: var(--color-text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .setting-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 0;
    border-bottom: 1px solid var(--color-border);
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
    color: var(--color-text-primary);
  }

  .setting-desc {
    font-size: 13px;
    color: var(--color-text-tertiary);
  }

  .setting-control {
    flex-shrink: 0;
  }

  .folder-control {
    display: flex;
    gap: 8px;
    flex: 1;
    max-width: 350px;
    margin-left: 24px;
  }

  .folder-input {
    flex: 1;
    padding: 10px 14px;
    background: var(--color-bg-primary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
    color: var(--color-text-secondary);
    font-size: 13px;
    font-family: monospace;
  }

  .browse-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    background: var(--color-bg-tertiary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
    color: var(--color-text-primary);
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s;
    white-space: nowrap;
  }

  .browse-btn:hover {
    background: var(--color-bg-hover);
  }

  .select-input {
    padding: 10px 14px;
    background: var(--color-bg-primary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
    color: var(--color-text-primary);
    font-size: 14px;
    min-width: 180px;
    cursor: pointer;
    -webkit-appearance: none;
    -moz-appearance: none;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23666' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right 12px center;
    padding-right: 36px;
  }

  .select-input option {
    background: var(--color-bg-secondary);
    color: var(--color-text-primary);
    padding: 8px;
  }

  /* Fix for WebKit in light theme */
  :global([data-theme="light"]) .select-input {
    background-color: #ffffff;
    color: #171717;
    border-color: #d4d4d4;
  }

  :global([data-theme="light"]) .select-input option {
    background-color: #fafafa;
    color: #171717;
  }

  :global([data-theme="light"]) .folder-input {
    background-color: #ffffff;
    color: #525252;
    border-color: #d4d4d4;
  }

  .select-input:focus {
    outline: none;
    border-color: var(--color-accent);
  }

  /* Accent Color Swatches */
  .accent-colors {
    display: flex;
    gap: 8px;
  }

  .color-swatch {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    border: 2px solid transparent;
    cursor: pointer;
    transition: all 0.2s;
    position: relative;
  }

  .color-swatch:hover {
    transform: scale(1.1);
  }

  .color-swatch.active {
    border-color: var(--color-text-primary);
    box-shadow: 0 0 0 2px var(--color-bg-primary);
  }

  .color-swatch.active::after {
    content: '';
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 12px;
    height: 12px;
    background: white;
    border-radius: 50%;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
  }

  /* Volume Slider */
  .volume-control {
    width: 150px;
  }

  .volume-slider {
    width: 100%;
    height: 6px;
    -webkit-appearance: none;
    appearance: none;
    background: var(--color-bg-hover);
    border-radius: 3px;
    outline: none;
    cursor: pointer;
  }

  .volume-slider::-webkit-slider-thumb {
    -webkit-appearance: none;
    appearance: none;
    width: 16px;
    height: 16px;
    background: var(--color-accent);
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
    background: var(--color-accent);
    border-radius: 50%;
    cursor: pointer;
    border: none;
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
    background: var(--color-bg-tertiary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 6px;
    color: var(--color-text-secondary);
    cursor: pointer;
    transition: all 0.2s;
  }

  .test-sound-btn:hover {
    background: var(--color-bg-hover);
    color: var(--color-accent);
    border-color: var(--color-accent);
  }

  /* Toggle Switch */
  .toggle {
    position: relative;
    display: inline-block;
    width: 48px;
    height: 26px;
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
    background: var(--color-bg-hover);
    transition: 0.3s;
    border-radius: 26px;
  }

  .toggle-slider:before {
    position: absolute;
    content: "";
    height: 20px;
    width: 20px;
    left: 3px;
    bottom: 3px;
    background: var(--color-text-tertiary);
    transition: 0.3s;
    border-radius: 50%;
  }

  .toggle input:checked + .toggle-slider {
    background: linear-gradient(135deg, #f472b6, #a855f7);
  }

  .toggle input:checked + .toggle-slider:before {
    transform: translateX(22px);
    background: #fff;
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
    color: var(--color-text-tertiary);
  }

  .app-desc {
    margin: 0;
    color: var(--color-text-secondary);
    font-size: 14px;
    line-height: 1.5;
  }

  /* Footer */
  .settings-footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 16px;
    margin-top: 32px;
    padding-top: 24px;
    border-top: 1px solid var(--color-border);
  }

  .save-message {
    font-size: 14px;
    color: var(--color-success);
  }

  .save-message.error {
    color: var(--color-error);
  }

  .save-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 24px;
    background: linear-gradient(135deg, #f472b6, #a855f7);
    border: none;
    border-radius: 10px;
    color: #000;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.2s;
  }

  .save-btn:hover:not(:disabled) {
    opacity: 0.9;
  }

  .save-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

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

  /* Reset Button */
  .reset-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 24px;
    background: transparent;
    border: 1px solid var(--color-border);
    border-radius: 10px;
    color: var(--color-text-secondary);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .reset-btn:hover:not(:disabled) {
    border-color: var(--color-warning);
    color: var(--color-warning);
    background: rgba(245, 158, 11, 0.1);
  }

  .reset-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
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
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
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
    color: var(--color-warning);
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
    color: var(--color-text-secondary);
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
  }

  .modal-btn.cancel {
    background: var(--color-bg-tertiary);
    border: 1px solid var(--color-border);
    color: var(--color-text-secondary);
  }

  .modal-btn.cancel:hover {
    background: var(--color-bg-hover);
    color: var(--color-text-primary);
  }

  .modal-btn.confirm {
    background: var(--color-warning);
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

  .lyrics-info {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 12px 14px;
    background: rgba(59, 130, 246, 0.08);
    border: 1px solid rgba(59, 130, 246, 0.15);
    border-radius: 8px;
    font-size: 13px;
    color: #888;
    margin-top: 8px;
  }

  .lyrics-info svg {
    color: #3b82f6;
    flex-shrink: 0;
    margin-top: 1px;
  }

  .source-info {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 12px 14px;
    background: rgba(34, 197, 94, 0.08);
    border: 1px solid rgba(34, 197, 94, 0.15);
    border-radius: 8px;
    font-size: 13px;
    color: #888;
    margin-top: 8px;
  }

  .source-info svg {
    color: #22c55e;
    flex-shrink: 0;
    margin-top: 1px;
  }

  .text-input {
    padding: 10px 14px;
    background: #0a0a0a;
    border: 1px solid #222;
    border-radius: 8px;
    color: #fff;
    font-size: 14px;
    width: 250px;
    outline: none;
    transition: border-color 0.2s;
  }

  .text-input:focus {
    border-color: var(--accent-color, #f472b6);
  }

  .text-input::placeholder {
    color: #555;
  }
</style>
