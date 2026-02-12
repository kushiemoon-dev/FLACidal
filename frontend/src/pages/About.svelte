<script lang="ts">
  import { onMount } from 'svelte';
  import { GetAppVersion } from '../../wailsjs/go/main/App.js';
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime.js';

  let version = '...';

  onMount(async () => {
    try {
      version = await GetAppVersion();
    } catch (error) {
      version = '0.0.0';
    }
  });

  function openGitHub() {
    BrowserOpenURL('https://github.com/kushiemoon-dev/flacidal');
  }
</script>

<div class="about-page">
  <div class="about-card main-card">
    <div class="logo-container">
      <div class="logo">
        <div class="waveform">
          <span class="bar"></span>
          <span class="bar"></span>
          <span class="bar"></span>
          <span class="bar"></span>
        </div>
      </div>
      <div class="logo-glow"></div>
    </div>

    <h1>FLACidal</h1>
    <p class="tagline">High-quality FLAC downloader for Tidal</p>

    <div class="version-badge">
      <span>Version {version}</span>
    </div>

    <div class="tech-stack">
      <span class="badge">Go</span>
      <span class="badge">Wails v2</span>
      <span class="badge">Svelte</span>
      <span class="badge">TypeScript</span>
    </div>

    <div class="links">
      <button class="link-btn github" onclick={openGitHub}>
        <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
          <path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/>
        </svg>
        GitHub
      </button>
    </div>
  </div>

  <div class="about-card credits-card">
    <h2>Powered By</h2>
    <div class="credits-list">
      <div class="credit-item">
        <span class="credit-name">Tidal API</span>
        <span class="credit-desc">Lossless music streaming</span>
      </div>
      <div class="credit-item">
        <span class="credit-name">Wails</span>
        <span class="credit-desc">Desktop application framework</span>
      </div>
      <div class="credit-item">
        <span class="credit-name">Svelte</span>
        <span class="credit-desc">Frontend framework</span>
      </div>
    </div>
  </div>

  <div class="about-card disclaimer-card">
    <h2>Disclaimer</h2>
    <p>
      FLACidal is for educational and personal use only.
      This tool is not affiliated with or endorsed by Tidal or any streaming service.
      Please respect copyright laws and support artists by purchasing their music.
    </p>
  </div>
</div>

<style>
  .about-page {
    padding: 32px;
    max-width: 600px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 24px;
  }

  .about-card {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 16px;
    padding: 32px;
  }

  .main-card {
    text-align: center;
    background: linear-gradient(135deg,
      rgba(244, 114, 182, 0.05),
      rgba(168, 85, 247, 0.05)
    );
    border-color: rgba(244, 114, 182, 0.2);
  }

  .logo-container {
    position: relative;
    display: inline-block;
    margin-bottom: 24px;
  }

  .logo {
    width: 80px;
    height: 80px;
    border-radius: 20px;
    background: linear-gradient(135deg, #f472b6, #a855f7);
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
    z-index: 1;
  }

  .logo-glow {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    width: 100px;
    height: 100px;
    background: linear-gradient(135deg, #f472b6, #a855f7);
    border-radius: 24px;
    filter: blur(30px);
    opacity: 0.4;
    z-index: 0;
  }

  .waveform {
    display: flex;
    align-items: center;
    gap: 4px;
    height: 32px;
  }

  .waveform .bar {
    width: 4px;
    background: rgba(0, 0, 0, 0.8);
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

  h1 {
    font-size: 32px;
    font-weight: 700;
    margin: 0 0 8px 0;
  }

  .tagline {
    color: var(--color-text-secondary);
    margin: 0 0 20px 0;
    font-size: 16px;
  }

  .version-badge {
    display: inline-block;
    padding: 6px 16px;
    background: var(--color-bg-tertiary);
    border-radius: 20px;
    font-size: 13px;
    color: var(--color-text-secondary);
    margin-bottom: 24px;
  }

  .tech-stack {
    display: flex;
    justify-content: center;
    gap: 8px;
    flex-wrap: wrap;
    margin-bottom: 24px;
  }

  .badge {
    padding: 6px 12px;
    background: var(--color-bg-tertiary);
    border: 1px solid var(--color-border);
    border-radius: 6px;
    font-size: 12px;
    color: var(--color-text-secondary);
  }

  .links {
    display: flex;
    justify-content: center;
    gap: 12px;
  }

  .link-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 24px;
    border: none;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .link-btn.github {
    background: var(--color-bg-tertiary);
    color: var(--color-text-primary);
    border: 1px solid var(--color-border);
  }

  .link-btn.github:hover {
    background: var(--color-bg-hover);
  }

  .credits-card h2,
  .disclaimer-card h2 {
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin: 0 0 16px 0;
  }

  .credits-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .credit-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 0;
    border-bottom: 1px solid var(--color-border);
  }

  .credit-item:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }

  .credit-name {
    font-weight: 500;
    color: var(--color-text-primary);
  }

  .credit-desc {
    font-size: 13px;
    color: var(--color-text-tertiary);
  }

  .disclaimer-card {
    background: rgba(245, 158, 11, 0.05);
    border-color: rgba(245, 158, 11, 0.2);
  }

  .disclaimer-card p {
    margin: 0;
    font-size: 13px;
    color: var(--color-text-secondary);
    line-height: 1.6;
  }
</style>
