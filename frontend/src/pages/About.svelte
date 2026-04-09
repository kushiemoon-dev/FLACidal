<script lang="ts">
  import { onMount } from 'svelte';
  import { GetAppVersion } from '../../wailsjs/go/main/App.js';
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime.js';
  import TabBar from '../components/TabBar.svelte';
  import { Heart, ExternalLink, Copy, Check, LayoutGrid } from 'lucide-svelte';

  let version = $state('...');
  let activeTab = $state('projects');
  let copied = $state(false);

  const tabs = [
    { id: 'projects', label: 'Other Projects' },
    { id: 'support', label: 'Support Me' },
  ];

  const projects = [
    {
      name: 'FLACidal',
      description: 'Get Tidal & Qobuz tracks in true lossless FLAC',
      icon: '🎵',
      url: 'https://github.com/kushiemoon-dev/flacidal',
      stars: 0,
      version: 'v3.3.0',
    },
    {
      name: 'Project Alpha',
      description: 'A placeholder for your next awesome project',
      icon: '🚀',
      url: 'https://github.com/kushiemoon-dev',
      stars: 0,
      version: 'v0.1.0',
    },
    {
      name: 'Project Beta',
      description: 'Another cool tool in the works',
      icon: '🛠️',
      url: 'https://github.com/kushiemoon-dev',
      stars: 0,
      version: 'v0.1.0',
    },
  ];

  const cryptoAddress = 'TXxxxxxxxxxxxxxxxxxxxxxxxxxxxx';

  onMount(async () => {
    try {
      version = await GetAppVersion();
    } catch {
      version = '0.0.0';
    }
  });

  function openURL(url: string) {
    BrowserOpenURL(url);
  }

  async function copyAddress() {
    try {
      await navigator.clipboard.writeText(cryptoAddress);
      copied = true;
      setTimeout(() => { copied = false; }, 2000);
    } catch {
      // silently fail
    }
  }
</script>

<div class="about-page">
  <div class="about-header">
    <h1>About</h1>
    <div class="version-badge">v{version}</div>
  </div>

  <TabBar {tabs} bind:activeTab />

  {#if activeTab === 'projects'}
    <div class="projects-grid">
      {#each projects as project (project.name)}
        <div class="project-card">
          <div class="project-top">
            <span class="project-icon">{project.icon}</span>
            <span class="project-version">{project.version}</span>
          </div>
          <h3 class="project-name">{project.name}</h3>
          <p class="project-desc">{project.description}</p>
          <div class="project-footer">
            <span class="project-stars">⭐ {project.stars}</span>
            <button class="icon-link-btn" onclick={() => openURL(project.url)} title="Open on GitHub">
              <ExternalLink size={14} />
              GitHub
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  {#if activeTab === 'support'}
    <div class="support-grid">
      <div class="support-card">
        <div class="support-card-header">
          <Heart size={20} class="heart-icon" />
          <h2>Ko-fi</h2>
        </div>
        <p class="support-desc">
          Enjoying the project? You can support ongoing development by buying me a coffee.
        </p>
        <button class="kofi-btn" onclick={() => openURL('https://ko-fi.com/')}>
          <Heart size={16} />
          Support me on Ko-fi
        </button>
      </div>

      <div class="support-card">
        <div class="support-card-header">
          <h2>USDT (TRC20)</h2>
        </div>
        <p class="support-desc">
          Crypto donations are also accepted.
        </p>
        <div class="crypto-row">
          <input
            class="crypto-input"
            type="text"
            readonly
            value={cryptoAddress}
          />
          <button class="copy-btn" onclick={copyAddress} title="Copy address">
            {#if copied}
              <Check size={16} />
            {:else}
              <Copy size={16} />
            {/if}
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .about-page {
    padding: 32px;
    max-width: 800px;
    margin: 0 auto;
  }

  .about-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 24px;
  }

  .about-header h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0;
  }

  .version-badge {
    padding: 4px 12px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 20px;
    font-size: 12px;
    color: var(--color-text-secondary);
  }

  /* Projects grid */
  .projects-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 16px;
  }

  .project-card {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 8px;
    transition: transform 0.2s ease, border-color 0.2s ease;
    cursor: default;
  }

  .project-card:hover {
    transform: translateY(-2px);
    border-color: var(--color-accent);
  }

  .project-top {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .project-icon {
    font-size: 24px;
  }

  .project-version {
    font-size: 11px;
    color: var(--color-text-tertiary);
    background: var(--color-bg-tertiary, rgba(255,255,255,0.05));
    border: 1px solid var(--color-border);
    border-radius: 4px;
    padding: 2px 6px;
  }

  .project-name {
    font-size: 15px;
    font-weight: 600;
    margin: 0;
    color: var(--color-text-primary);
  }

  .project-desc {
    font-size: 13px;
    color: var(--color-text-secondary);
    margin: 0;
    line-height: 1.5;
    flex: 1;
  }

  .project-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 4px;
  }

  .project-stars {
    font-size: 12px;
    color: var(--color-text-tertiary);
  }

  .icon-link-btn {
    display: flex;
    align-items: center;
    gap: 5px;
    padding: 5px 10px;
    background: none;
    border: 1px solid var(--color-border);
    border-radius: 6px;
    font-size: 12px;
    color: var(--color-text-secondary);
    cursor: pointer;
    transition: color 0.2s, border-color 0.2s;
  }

  .icon-link-btn:hover {
    color: var(--color-text-primary);
    border-color: var(--color-accent);
  }

  /* Support grid */
  .support-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 20px;
  }

  @media (max-width: 560px) {
    .support-grid {
      grid-template-columns: 1fr;
    }
  }

  .support-card {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
    padding: 28px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .support-card-header {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .support-card-header h2 {
    font-size: 18px;
    font-weight: 600;
    margin: 0;
    color: var(--color-text-primary);
  }

  :global(.heart-icon) {
    color: #ff5e5b;
  }

  .support-desc {
    font-size: 13px;
    color: var(--color-text-secondary);
    line-height: 1.6;
    margin: 0;
  }

  .kofi-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 12px 20px;
    background: #29ABE0;
    color: #fff;
    border: none;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.2s;
    margin-top: auto;
  }

  .kofi-btn:hover {
    opacity: 0.85;
  }

  .crypto-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .crypto-input {
    flex: 1;
    background: var(--color-bg-tertiary, rgba(255,255,255,0.05));
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 8px 12px;
    font-size: 12px;
    color: var(--color-text-secondary);
    font-family: monospace;
    min-width: 0;
  }

  .copy-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 8px;
    background: var(--color-bg-tertiary, rgba(255,255,255,0.05));
    border: 1px solid var(--color-border);
    border-radius: 8px;
    color: var(--color-text-secondary);
    cursor: pointer;
    transition: color 0.2s, border-color 0.2s;
    flex-shrink: 0;
  }

  .copy-btn:hover {
    color: var(--color-accent);
    border-color: var(--color-accent);
  }
</style>
