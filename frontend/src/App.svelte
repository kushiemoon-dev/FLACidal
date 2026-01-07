<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';
  import Sidebar from './components/Sidebar.svelte';
  import Home from './pages/Home.svelte';
  import Search from './pages/Search.svelte';
  import Queue from './pages/Queue.svelte';
  import Files from './pages/Files.svelte';
  import History from './pages/History.svelte';
  import Settings from './pages/Settings.svelte';
  import Terminal from './pages/Terminal.svelte';
  import About from './pages/About.svelte';
  import { queueStore, queueStats, downloadFolder, queuePaused } from './stores/queue';
  import { themeStore, initializeAccentColor } from './stores/theme';
  import { initializeAudioSettings, playSound } from './stores/audio';
  import { GetDownloadFolder, GetConfig, IsQueuePaused } from '../wailsjs/go/main/App.js';

  let activePage = 'home';
  let unsubscribeProgress: () => void;
  let unsubscribePaused: () => void;
  let refetchedContent: any = null;

  function handleNavigate(page: string) {
    activePage = page;
  }

  function handleHistoryRefetch(content: any) {
    refetchedContent = content;
    activePage = 'home';
  }

  onMount(async () => {
    // Load config and initialize theme + accent color
    try {
      const config = await GetConfig();
      if (config?.theme) {
        themeStore.initialize(config.theme as 'dark' | 'light' | 'system');
      } else {
        themeStore.initialize('system');
      }
      // Initialize accent color
      initializeAccentColor(config?.accentColor || '#f472b6');
      // Initialize audio settings
      initializeAudioSettings(config?.soundEffects || false, config?.soundVolume || 70);
    } catch {
      themeStore.initialize('system');
      initializeAccentColor('#f472b6');
      initializeAudioSettings(false, 70);
    }

    // Load download folder
    const folder = await GetDownloadFolder();
    if (folder) {
      downloadFolder.set(folder);
    }

    // Load initial paused state
    try {
      const paused = await IsQueuePaused();
      queuePaused.set(paused);
    } catch {
      queuePaused.set(false);
    }

    // Listen for queue paused state changes
    unsubscribePaused = EventsOn('queue-paused', (paused: boolean) => {
      queuePaused.set(paused);
    });

    // Listen for download progress events and update queue store
    unsubscribeProgress = EventsOn('download-progress', (data: any) => {
      const { trackId, status, result } = data;

      if (status === 'queued') {
        queueStore.updateItem(trackId, { status: 'queued' });
      } else if (status === 'downloading') {
        queueStore.updateItem(trackId, { status: 'downloading' });
      } else if (status === 'completed' && result) {
        queueStore.updateItem(trackId, {
          status: 'completed',
          result: {
            filePath: result.filePath,
            fileSize: result.fileSize
          }
        });
        // Play complete sound
        playSound('complete');
      } else if (status === 'error') {
        queueStore.updateItem(trackId, {
          status: 'error',
          error: result?.error || 'Download failed'
        });
        // Play error sound
        playSound('error');
      } else if (status === 'cancelled') {
        queueStore.updateItem(trackId, { status: 'cancelled' });
      }
    });
  });

  onDestroy(() => {
    if (unsubscribeProgress) {
      unsubscribeProgress();
    }
    if (unsubscribePaused) {
      unsubscribePaused();
    }
  });
</script>

<main class="app-layout">
  <Sidebar
    {activePage}
    onNavigate={handleNavigate}
    queueCount={$queueStats.pending + $queueStats.downloading}
  />

  <div class="main-content">
    {#if activePage === 'home'}
      <Home initialContent={refetchedContent} on:contentCleared={() => refetchedContent = null} />
    {:else if activePage === 'search'}
      <Search />
    {:else if activePage === 'queue'}
      <Queue />
    {:else if activePage === 'files'}
      <Files />
    {:else if activePage === 'history'}
      <History onRefetch={handleHistoryRefetch} />
    {:else if activePage === 'settings'}
      <Settings />
    {:else if activePage === 'terminal'}
      <Terminal />
    {:else if activePage === 'about'}
      <About />
    {/if}
  </div>
</main>

<style>
  @import url('https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap');

  /* Dark theme (default) */
  :global(:root), :global([data-theme="dark"]) {
    --color-bg-void: #050505;
    --color-bg-primary: #0a0a0a;
    --color-bg-secondary: #111111;
    --color-bg-tertiary: #1a1a1a;
    --color-bg-elevated: #222222;
    --color-bg-hover: #2a2a2a;
    --color-text-primary: #fafafa;
    --color-text-secondary: #a1a1a1;
    --color-text-tertiary: #666666;
    --color-text-muted: #444444;
    --color-accent: #f472b6;
    --color-accent-hover: #f9a8d4;
    --color-accent-subtle: rgba(244, 114, 182, 0.15);
    --color-border: #1a1a1a;
    --color-border-subtle: #222222;
    --color-success: #10b981;
    --color-warning: #f59e0b;
    --color-error: #ef4444;
    --color-info: #3b82f6;
  }

  /* Light theme */
  :global([data-theme="light"]) {
    --color-bg-void: #f5f5f5;
    --color-bg-primary: #ffffff;
    --color-bg-secondary: #fafafa;
    --color-bg-tertiary: #f0f0f0;
    --color-bg-elevated: #ffffff;
    --color-bg-hover: #e5e5e5;
    --color-text-primary: #171717;
    --color-text-secondary: #525252;
    --color-text-tertiary: #737373;
    --color-text-muted: #a3a3a3;
    --color-accent: #db2777;
    --color-accent-hover: #be185d;
    --color-accent-subtle: rgba(219, 39, 119, 0.1);
    --color-border: #e5e5e5;
    --color-border-subtle: #f0f0f0;
    --color-success: #059669;
    --color-warning: #d97706;
    --color-error: #dc2626;
    --color-info: #2563eb;
  }

  :global(*) {
    box-sizing: border-box;
  }

  :global(body) {
    margin: 0;
    padding: 0;
    font-family: 'Outfit', system-ui, -apple-system, sans-serif;
    background: var(--color-bg-void);
    color: var(--color-text-primary);
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    transition: background-color 0.2s ease, color 0.2s ease;
  }

  :global(::selection) {
    background: var(--color-accent-subtle);
    color: var(--color-accent);
  }

  :global(::-webkit-scrollbar) {
    width: 6px;
    height: 6px;
  }

  :global(::-webkit-scrollbar-track) {
    background: transparent;
  }

  :global(::-webkit-scrollbar-thumb) {
    background: var(--color-bg-hover);
    border-radius: 3px;
  }

  :global(::-webkit-scrollbar-thumb:hover) {
    background: var(--color-bg-elevated);
  }

  .app-layout {
    display: flex;
    min-height: 100vh;
    background: var(--color-bg-primary);
    transition: background-color 0.2s ease;
  }

  .main-content {
    flex: 1;
    overflow-y: auto;
    max-height: 100vh;
  }
</style>
