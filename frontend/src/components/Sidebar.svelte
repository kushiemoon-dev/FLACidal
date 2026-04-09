<script lang="ts">
  import {
    Home, Search, Download, FolderOpen, Clock,
    Terminal, Settings, Info, LayoutGrid,
    AudioWaveform, SlidersHorizontal, FileAudio, FolderCog, Bug
  } from 'lucide-svelte';

  let { activePage = 'home', onNavigate = (page: string) => {}, queueCount = 0 }: {
    activePage?: string;
    onNavigate?: (page: string) => void;
    queueCount?: number;
  } = $props();

  let showToolsFlyout = $state(false);

  const navItems = [
    { id: 'home',    label: 'Home',    Icon: Home },
    { id: 'search',  label: 'Search',  Icon: Search },
    { id: 'queue',   label: 'Queue',   Icon: Download },
    { id: 'files',   label: 'Files',   Icon: FolderOpen },
    { id: 'history', label: 'History', Icon: Clock },
  ];

  const toolItems = [
    { id: 'tool-analyzer',    label: 'Audio Quality Analyzer', Icon: AudioWaveform },
    { id: 'tool-resampler',   label: 'Audio Resampler',        Icon: SlidersHorizontal },
    { id: 'tool-converter',   label: 'Audio Converter',        Icon: FileAudio },
    { id: 'tool-filemanager', label: 'File Manager',           Icon: FolderCog },
  ];

  const bottomItems = [
    { id: 'settings', label: 'Settings', Icon: Settings },
    { id: 'terminal', label: 'Terminal', Icon: Terminal },
    { id: 'about',    label: 'About',    Icon: Info },
  ];

  function toggleToolsFlyout() {
    showToolsFlyout = !showToolsFlyout;
  }

  function navigateTool(id: string) {
    showToolsFlyout = false;
    onNavigate(id);
  }

  function handleWindowClick(event: MouseEvent) {
    const sidebar = (event.target as Element).closest('.sidebar');
    if (!sidebar) {
      showToolsFlyout = false;
    }
  }

  function handleBugReport() {
    window.open('https://github.com/flacidal/flacidal/issues', '_blank');
  }
</script>

<svelte:window onclick={handleWindowClick} />

<aside class="sidebar">
  <!-- Logo -->
  <div class="logo">
    <div class="logo-icon">
      <div class="waveform">
        <span class="bar"></span>
        <span class="bar"></span>
        <span class="bar"></span>
        <span class="bar"></span>
      </div>
    </div>
  </div>

  <!-- Main navigation -->
  <nav class="nav-main">
    {#each navItems as item}
      <button
        class="nav-item"
        class:active={activePage === item.id}
        onclick={() => onNavigate(item.id)}
        title={item.label}
      >
        <item.Icon size={20} />
        {#if item.id === 'queue' && queueCount > 0}
          <span class="badge">{queueCount}</span>
        {/if}
      </button>
    {/each}
  </nav>

  <!-- Bottom navigation -->
  <nav class="nav-bottom">
    <!-- Tools flyout button -->
    <div class="flyout-wrapper">
      <button
        class="nav-item"
        class:active={showToolsFlyout || ['tool-analyzer','tool-resampler','tool-converter','tool-filemanager'].includes(activePage)}
        onclick={toggleToolsFlyout}
        title="Tools"
      >
        <LayoutGrid size={20} />
      </button>

      {#if showToolsFlyout}
        <div class="flyout">
          <div class="flyout-title">Tools</div>
          {#each toolItems as tool}
            <button
              class="flyout-item"
              class:active={activePage === tool.id}
              onclick={() => navigateTool(tool.id)}
            >
              <tool.Icon size={16} />
              <span>{tool.label}</span>
            </button>
          {/each}
        </div>
      {/if}
    </div>

    {#each bottomItems as item}
      <button
        class="nav-item"
        class:active={activePage === item.id}
        onclick={() => onNavigate(item.id)}
        title={item.label}
      >
        <item.Icon size={20} />
      </button>
    {/each}

    <!-- Bug report -->
    <button
      class="nav-item"
      onclick={handleBugReport}
      title="Report a Bug"
    >
      <Bug size={20} />
    </button>
  </nav>
</aside>

<style>
  .sidebar {
    position: fixed;
    top: 0;
    left: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 56px;
    height: 100vh;
    padding: 12px 0;
    background: var(--color-bg-primary);
    border-right: 1px solid var(--color-border);
    z-index: 100;
  }

  .logo {
    margin-bottom: 20px;
  }

  .logo-icon {
    width: 36px;
    height: 36px;
    border-radius: 10px;
    background: var(--color-accent, #f472b6);
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 0 16px var(--color-accent-subtle, rgba(244, 114, 182, 0.3));
    transition: transform 0.2s ease, box-shadow 0.2s ease;
    cursor: default;
  }

  .logo-icon:hover {
    transform: scale(1.06);
    box-shadow: 0 0 24px rgba(244, 114, 182, 0.5);
  }

  .waveform {
    display: flex;
    align-items: center;
    gap: 2px;
    height: 16px;
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

  .nav-main {
    display: flex;
    flex-direction: column;
    gap: 4px;
    flex: 1;
  }

  .nav-bottom {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-top: auto;
  }

  .nav-item {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    border: none;
    border-radius: 10px;
    background: transparent;
    color: var(--color-text-tertiary);
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
  }

  .nav-item:hover {
    background: var(--color-bg-secondary);
    color: var(--color-text-secondary);
  }

  .nav-item.active {
    background: var(--color-accent-subtle);
    color: var(--color-accent);
  }

  /* Left indicator bar for active state */
  .nav-item.active::before {
    content: '';
    position: absolute;
    left: -8px;
    top: 50%;
    transform: translateY(-50%);
    width: 3px;
    height: 20px;
    background: var(--color-accent);
    border-radius: 0 3px 3px 0;
  }

  .badge {
    position: absolute;
    top: 5px;
    right: 5px;
    min-width: 15px;
    height: 15px;
    padding: 0 3px;
    font-size: 9px;
    font-weight: 700;
    line-height: 15px;
    text-align: center;
    color: #000;
    background: var(--color-accent);
    border-radius: 8px;
  }

  /* Flyout */
  .flyout-wrapper {
    position: relative;
  }

  .flyout {
    position: absolute;
    left: calc(100% + 10px);
    bottom: 0;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 10px;
    padding: 8px;
    min-width: 200px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    z-index: 200;
  }

  .flyout-title {
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--color-text-tertiary);
    padding: 4px 8px 8px;
  }

  .flyout-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 8px 10px;
    border: none;
    border-radius: 7px;
    background: transparent;
    color: var(--color-text-secondary);
    cursor: pointer;
    font-size: 13px;
    font-weight: 500;
    text-align: left;
    transition: background 0.12s, color 0.12s;
  }

  .flyout-item:hover {
    background: var(--color-bg-void);
    color: var(--color-text-primary);
  }

  .flyout-item.active {
    color: var(--color-accent);
    background: var(--color-accent-subtle);
  }
</style>
