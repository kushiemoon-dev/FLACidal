<script lang="ts">
  import { onMount } from 'svelte';
  import { GetAppVersion } from '../../wailsjs/go/main/App.js';
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime.js';
  import { Heart, ExternalLink, LayoutGrid } from 'lucide-svelte';
  import kofiLogo from '../assets/logos/kofi-logo.png';

  let version = $state('...');
  let activeTab = $state('projects');

  const projects = [
    {
      name: 'FLACidal Mobile',
      description: 'FLACidal on the go — download lossless FLAC from your phone',
      logo: 'https://raw.githubusercontent.com/kushiemoon-dev/FLACidal-Mobile/main/assets/icon.png',
      url: 'https://github.com/kushiemoon-dev/FLACidal-Mobile',
    },
    {
      name: 'YouFLAC',
      description: 'YouTube video + lossless FLAC audio — create high-quality music videos',
      logo: 'https://raw.githubusercontent.com/kushiemoon-dev/YouFLAC/main/build/appicon.png',
      url: 'https://github.com/kushiemoon-dev/YouFLAC',
    },
  ];

  const kofiUrl = 'https://ko-fi.com/kushiemoon';

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
</script>

<div class="about-page">
  <div class="about-header">
    <h1>About</h1>
  </div>

  <div class="tab-row">
    <button
      class="tab-badge"
      class:active={activeTab === 'projects'}
      style="--badge-color: #22c55e"
      onclick={() => activeTab = 'projects'}
    >
      <LayoutGrid size={14} />
      Other Projects
    </button>
    <button
      class="tab-badge"
      class:active={activeTab === 'support'}
      style="--badge-color: #ef4444"
      onclick={() => activeTab = 'support'}
    >
      <Heart size={14} />
      Support Me
    </button>
  </div>

  {#if activeTab === 'projects'}
    <div class="projects-grid">
      {#each projects as project (project.name)}
        <button class="project-card" onclick={() => openURL(project.url)}>
          <img class="project-logo" src={project.logo} alt={project.name} />
          <h3 class="project-name">{project.name}</h3>
          <p class="project-desc">{project.description}</p>
          <div class="project-footer">
            <span class="project-link">
              <ExternalLink size={12} />
              GitHub
            </span>
          </div>
        </button>
      {/each}
    </div>
  {/if}

  {#if activeTab === 'support'}
    <div class="support-card">
      <div class="support-inner">
        <div class="kofi-logo-area">
          <img src={kofiLogo} alt="Ko-fi" class="kofi-logo-img" />
        </div>

        <div class="kofi-content">
          <h3 class="support-subtitle">Support via Ko-fi</h3>
          <p class="support-desc">
            Enjoying the project? You can support ongoing development by buying me a coffee.
          </p>
          <button class="kofi-btn" onclick={() => openURL(kofiUrl)}>
            <Heart size={16} />
            Support me on Ko-fi
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
    margin-bottom: 20px;
  }

  .about-header h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0;
  }

  /* Tab badges */
  .tab-row {
    display: flex;
    gap: 10px;
    margin-bottom: 24px;
  }

  .tab-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    border-radius: 8px;
    border: 1px solid var(--color-border);
    background: var(--color-bg-secondary);
    color: var(--color-text-secondary);
    font-size: 13px;
    font-weight: 500;
    font-family: var(--font-family);
    cursor: pointer;
    transition: all 0.2s;
  }

  .tab-badge:hover {
    border-color: var(--badge-color);
    color: var(--color-text-primary);
  }

  .tab-badge.active {
    background: var(--badge-color);
    border-color: var(--badge-color);
    color: #fff;
    font-weight: 600;
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
    border-radius: 14px;
    padding: 20px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    transition: transform 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
    cursor: pointer;
    text-align: left;
    font-family: inherit;
    color: inherit;
  }

  .project-card:hover {
    transform: translateY(-3px);
    border-color: var(--color-accent);
    box-shadow: 0 6px 20px rgba(0, 0, 0, 0.3);
  }

  .project-logo {
    width: 48px;
    height: 48px;
    border-radius: 12px;
    object-fit: cover;
  }

  .project-name {
    font-size: 15px;
    font-weight: 600;
    margin: 0;
    color: var(--color-text-primary);
  }

  .project-desc {
    font-size: 12px;
    color: var(--color-text-secondary);
    margin: 0;
    line-height: 1.5;
    flex: 1;
  }

  .project-footer {
    margin-top: 4px;
  }

  .project-link {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    color: var(--color-text-tertiary);
  }

  /* Support card — single centered */
  .support-card {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
    overflow: hidden;
    max-width: 500px;
    margin: 0 auto;
  }

  .support-inner {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 40px 32px;
    gap: 20px;
  }

  /* Ko-fi SVG logo animated */
  .kofi-logo-area {
    display: flex;
    justify-content: center;
  }

  .kofi-logo-img {
    height: 100px;
    object-fit: contain;
    animation: kofi-bounce 2s ease-in-out infinite;
    filter: drop-shadow(0 4px 16px rgba(255, 94, 91, 0.2));
  }

  @keyframes kofi-bounce {
    0%, 100% { transform: translateY(0) scale(1); }
    25% { transform: translateY(-6px) scale(1.03); }
    50% { transform: translateY(0) scale(1); }
    75% { transform: translateY(-3px) scale(1.01); }
  }

  .kofi-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
  }

  .support-subtitle {
    font-size: 16px;
    font-weight: 600;
    margin: 0;
    color: var(--color-text-primary);
  }

  .support-desc {
    font-size: 13px;
    color: var(--color-text-secondary);
    line-height: 1.6;
    margin: 0;
    max-width: 360px;
  }

  .kofi-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    padding: 12px 32px;
    background: #29ABE0;
    color: #fff;
    border: none;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 600;
    font-family: var(--font-family);
    cursor: pointer;
    transition: opacity 0.2s, transform 0.2s;
    margin-top: 4px;
  }

  .kofi-btn:hover {
    opacity: 0.9;
    transform: translateY(-1px);
  }
</style>
