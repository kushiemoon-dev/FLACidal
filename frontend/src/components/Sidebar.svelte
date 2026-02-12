<script lang="ts">
  let { activePage = 'home', onNavigate = (page: string) => {}, queueCount = 0 }: {
    activePage?: string;
    onNavigate?: (page: string) => void;
    queueCount?: number;
  } = $props();

  const navItems = [
    { id: 'home', label: 'Home', icon: 'home' },
    { id: 'search', label: 'Search', icon: 'search' },
    { id: 'queue', label: 'Queue', icon: 'download' },
    { id: 'files', label: 'Files', icon: 'folder' },
    { id: 'history', label: 'History', icon: 'clock' },
  ];

  const bottomItems = [
    { id: 'terminal', label: 'Terminal', icon: 'terminal' },
    { id: 'settings', label: 'Settings', icon: 'settings' },
    { id: 'about', label: 'About', icon: 'info' },
  ];
</script>

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

  <!-- Navigation -->
  <nav class="nav-main">
    {#each navItems as item}
      <button
        class="nav-item"
        class:active={activePage === item.id}
        onclick={() => onNavigate(item.id)}
        title={item.label}
      >
        {#if item.icon === 'home'}
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
            <polyline points="9 22 9 12 15 12 15 22"/>
          </svg>
        {:else if item.icon === 'search'}
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8"/>
            <path d="m21 21-4.35-4.35"/>
          </svg>
        {:else if item.icon === 'download'}
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="7 10 12 15 17 10"/>
            <line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          {#if queueCount > 0}
            <span class="badge">{queueCount}</span>
          {/if}
        {:else if item.icon === 'folder'}
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
          </svg>
        {:else if item.icon === 'clock'}
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <polyline points="12 6 12 12 16 14"/>
          </svg>
        {/if}
      </button>
    {/each}
  </nav>

  <!-- Bottom nav -->
  <nav class="nav-bottom">
    {#each bottomItems as item}
      <button
        class="nav-item"
        class:active={activePage === item.id}
        onclick={() => onNavigate(item.id)}
        title={item.label}
      >
        {#if item.icon === 'terminal'}
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="4 17 10 11 4 5"/>
            <line x1="12" y1="19" x2="20" y2="19"/>
          </svg>
        {:else if item.icon === 'settings'}
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"/>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
          </svg>
        {:else if item.icon === 'info'}
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="16" x2="12" y2="12"/>
            <line x1="12" y1="8" x2="12.01" y2="8"/>
          </svg>
        {/if}
      </button>
    {/each}
  </nav>
</aside>

<style>
  .sidebar {
    display: flex;
    flex-direction: column;
    align-items: center;
    width: 64px;
    min-height: 100vh;
    padding: 16px 0;
    background: var(--color-bg-primary);
    border-right: 1px solid var(--color-border);
  }

  .logo {
    margin-bottom: 24px;
    animation: logoEntrance 0.5s ease-out both;
  }

  @keyframes logoEntrance {
    from {
      opacity: 0;
      transform: scale(0.8) rotate(-10deg);
    }
    to {
      opacity: 1;
      transform: scale(1) rotate(0);
    }
  }

  .logo-icon {
    width: 40px;
    height: 40px;
    border-radius: 12px;
    background: linear-gradient(135deg, #f472b6, #a855f7);
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 0 20px rgba(244, 114, 182, 0.3);
    transition: transform 0.2s ease, box-shadow 0.2s ease;
  }

  .logo-icon:hover {
    transform: scale(1.05);
    box-shadow: 0 0 30px rgba(244, 114, 182, 0.5);
  }

  .waveform {
    display: flex;
    align-items: center;
    gap: 2px;
    height: 18px;
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
    gap: 8px;
    flex: 1;
  }

  .nav-main .nav-item {
    animation: slideIn 0.3s ease-out both;
  }

  .nav-main .nav-item:nth-child(1) { animation-delay: 0.05s; }
  .nav-main .nav-item:nth-child(2) { animation-delay: 0.1s; }
  .nav-main .nav-item:nth-child(3) { animation-delay: 0.15s; }
  .nav-main .nav-item:nth-child(4) { animation-delay: 0.2s; }
  .nav-main .nav-item:nth-child(5) { animation-delay: 0.25s; }

  .nav-bottom {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-top: auto;
  }

  .nav-bottom .nav-item {
    animation: slideIn 0.3s ease-out both;
  }

  .nav-bottom .nav-item:nth-child(1) { animation-delay: 0.3s; }
  .nav-bottom .nav-item:nth-child(2) { animation-delay: 0.35s; }
  .nav-bottom .nav-item:nth-child(3) { animation-delay: 0.4s; }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: translateX(-8px);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }

  .nav-item {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 44px;
    height: 44px;
    border: none;
    border-radius: 12px;
    background: transparent;
    color: var(--color-text-tertiary);
    cursor: pointer;
    transition: all 0.2s;
  }

  .nav-item:hover {
    background: var(--color-bg-tertiary);
    color: var(--color-text-secondary);
    transform: scale(1.05);
  }

  .nav-item.active {
    background: var(--color-accent-subtle);
    color: var(--color-accent);
  }

  .nav-item.active::before {
    content: '';
    position: absolute;
    left: -10px;
    width: 3px;
    height: 20px;
    background: var(--color-accent);
    border-radius: 0 3px 3px 0;
    animation: activeIndicator 0.2s ease-out;
  }

  @keyframes activeIndicator {
    from {
      height: 0;
      opacity: 0;
    }
    to {
      height: 20px;
      opacity: 1;
    }
  }

  .badge {
    position: absolute;
    top: 6px;
    right: 6px;
    min-width: 16px;
    height: 16px;
    padding: 0 4px;
    font-size: 10px;
    font-weight: 600;
    line-height: 16px;
    text-align: center;
    color: #000;
    background: #f472b6;
    border-radius: 8px;
  }
</style>
