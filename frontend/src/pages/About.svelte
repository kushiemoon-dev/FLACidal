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

  const cryptoAddress = 'YOUR_USDT_TRC20_ADDRESS';
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
        <!-- Ko-fi side -->
        <div class="support-col kofi-col">
          <div class="kofi-logo-wrapper">
            <img
              src="https://storage.ko-fi.com/cdn/brandasset/v2/kofi_s_logo_nolabel.png"
              alt="Ko-fi"
              class="kofi-logo-img"
            />
            <span class="kofi-brand-text">Ko-fi</span>
          </div>
          <h3 class="support-subtitle">Support via Ko-fi</h3>
          <p class="support-desc">
            Enjoying the project? You can support ongoing development by buying me a coffee.
          </p>
          <button class="kofi-btn" onclick={() => openURL(kofiUrl)}>
            <Heart size={16} />
            Support me on Ko-fi
          </button>
        </div>

        <!-- Crypto side -->
        <div class="support-col crypto-col">
          <h3 class="crypto-title">USDT (TRC20)</h3>
          <p class="support-desc">
            Crypto donations are also accepted. Scan the QR code or copy the address.
          </p>
          <div class="crypto-address-row">
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

  /* Tab badges like SpotiFLAC */
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

  /* Support - single card, 2 columns */
  .support-card {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
    overflow: hidden;
  }

  .support-inner {
    display: grid;
    grid-template-columns: 1fr 1fr;
  }

  .support-col {
    padding: 32px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .kofi-col {
    align-items: center;
    text-align: center;
    border-right: 1px solid var(--color-border);
  }

  .crypto-col {
    display: flex;
    flex-direction: column;
    justify-content: center;
    gap: 12px;
  }

  /* Ko-fi logo */
  .kofi-logo-wrapper {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 12px;
    margin-bottom: 8px;
  }

  .kofi-logo-img {
    width: 64px;
    height: 64px;
    object-fit: contain;
    animation: kofi-wiggle 2s ease-in-out infinite;
  }

  @keyframes kofi-wiggle {
    0%, 100% { transform: rotate(0deg) scale(1); }
    10% { transform: rotate(-8deg) scale(1.1); }
    20% { transform: rotate(8deg) scale(1.1); }
    30% { transform: rotate(-4deg) scale(1.05); }
    40% { transform: rotate(4deg) scale(1.05); }
    50% { transform: rotate(0deg) scale(1); }
  }

  .kofi-brand-text {
    font-size: 42px;
    font-weight: 800;
    color: var(--color-text-primary);
    letter-spacing: -1px;
  }

  .support-subtitle {
    font-size: 16px;
    font-weight: 600;
    margin: 0;
    color: var(--color-text-primary);
  }

  .crypto-title {
    font-size: 18px;
    font-weight: 600;
    margin: 0;
    color: var(--color-text-primary);
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
    padding: 12px 24px;
    background: #29ABE0;
    color: #fff;
    border: none;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 600;
    font-family: var(--font-family);
    cursor: pointer;
    transition: opacity 0.2s, transform 0.2s;
    margin-top: auto;
    width: 100%;
  }

  .kofi-btn:hover {
    opacity: 0.9;
    transform: translateY(-1px);
  }

  .crypto-address-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .crypto-input {
    flex: 1;
    background: var(--color-bg-tertiary, rgba(255,255,255,0.05));
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 10px 12px;
    font-size: 11px;
    color: var(--color-text-secondary);
    font-family: 'JetBrains Mono', monospace;
    min-width: 0;
  }

  .copy-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 10px;
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
