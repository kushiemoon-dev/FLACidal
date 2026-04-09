<script lang="ts">
  import { onMount } from 'svelte';
  import { GetAppVersion } from '../../wailsjs/go/main/App.js';
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime.js';
  import { Heart, ExternalLink, LayoutGrid, Star, GitFork } from 'lucide-svelte';
  import kofiLogo from '../assets/logos/kofi-logo.png';
  import flacidalLogo from '../assets/logos/flacidal.png';
  import youflacLogo from '../assets/logos/youflac.png';
  import opendropLogo from '../assets/logos/opendrop-vj.png';

  let version = $state('...');
  let activeTab = $state('projects');

  interface Project {
    name: string;
    repo: string;
    description: string;
    logo: string;
    color: string;
    tag: string;
    url: string;
    stars: number;
    forks: number;
    updatedAt: string;
  }

  let projects = $state<Project[]>([
    {
      name: 'FLACidal Mobile',
      repo: 'kushiemoon-dev/FLACidal-Mobile',
      description: 'FLACidal on the go — download lossless FLAC from your phone',
      logo: flacidalLogo,
      color: '#f472b6',
      tag: 'flutter',
      url: 'https://github.com/kushiemoon-dev/FLACidal-Mobile',
      stars: 0, forks: 0, updatedAt: '',
    },
    {
      name: 'YouFLAC',
      repo: 'kushiemoon-dev/YouFLAC',
      description: 'YouTube video + lossless FLAC audio — create high-quality music videos',
      logo: youflacLogo,
      color: '#ef4444',
      tag: 'go',
      url: 'https://github.com/kushiemoon-dev/YouFLAC',
      stars: 0, forks: 0, updatedAt: '',
    },
    {
      name: 'OpenDrop VJ',
      repo: 'kushiemoon-dev/OpenDrop-VJ',
      description: 'Open-source multi-deck audio visualizer with MilkDrop presets and MIDI control',
      logo: opendropLogo,
      color: '#38bdf8',
      tag: 'rust',
      url: 'https://github.com/kushiemoon-dev/OpenDrop-VJ',
      stars: 0, forks: 0, updatedAt: '',
    },
  ]);

  const kofiUrl = 'https://ko-fi.com/kushiemoon';

  function timeAgo(dateStr: string): string {
    if (!dateStr) return '';
    const diff = Date.now() - new Date(dateStr).getTime();
    const days = Math.floor(diff / 86400000);
    if (days === 0) return 'today';
    if (days === 1) return '1 day ago';
    if (days < 30) return `${days} days ago`;
    const months = Math.floor(days / 30);
    return months === 1 ? '1 month ago' : `${months} months ago`;
  }

  onMount(async () => {
    try {
      version = await GetAppVersion();
    } catch {
      version = '0.0.0';
    }

    // Fetch GitHub stats for each project
    for (let i = 0; i < projects.length; i++) {
      try {
        const resp = await fetch(`https://api.github.com/repos/${projects[i].repo}`);
        if (resp.ok) {
          const data = await resp.json();
          projects[i] = {
            ...projects[i],
            stars: data.stargazers_count || 0,
            forks: data.forks_count || 0,
            updatedAt: data.updated_at || '',
          };
        }
      } catch {
        // silently fail — stats stay at 0
      }
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
          <div class="card-top">
            <img class="project-logo" src={project.logo} alt={project.name} />
            {#if project.updatedAt}
              <span class="updated-ago">{timeAgo(project.updatedAt)}</span>
            {/if}
          </div>
          <h3 class="project-name">{project.name}</h3>
          <span class="project-tag" style="background: {project.color}20; color: {project.color}; border-color: {project.color}40">
            {project.tag}
          </span>
          <p class="project-desc">{project.description}</p>
          <div class="project-footer">
            <div class="project-stats">
              <span class="stat"><Star size={12} /> {project.stars}</span>
              <span class="stat"><GitFork size={12} /> {project.forks}</span>
            </div>
            <span class="project-link">
              <ExternalLink size={12} />
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
    max-width: 900px;
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

  /* Projects grid — compact 3 columns */
  .projects-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
    gap: 14px;
  }

  .project-card {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 8px;
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

  .card-top {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
  }

  .project-logo {
    width: 40px;
    height: 40px;
    border-radius: 10px;
    object-fit: cover;
  }

  .updated-ago {
    font-size: 10px;
    color: var(--color-text-tertiary);
    white-space: nowrap;
  }

  .project-name {
    font-size: 14px;
    font-weight: 600;
    margin: 0;
    color: var(--color-text-primary);
  }

  .project-tag {
    display: inline-block;
    width: fit-content;
    font-size: 10px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 4px;
    border: 1px solid;
    text-transform: lowercase;
  }

  .project-desc {
    font-size: 11px;
    color: var(--color-text-secondary);
    margin: 0;
    line-height: 1.5;
    flex: 1;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .project-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 4px;
    padding-top: 8px;
    border-top: 1px solid var(--color-border);
  }

  .project-stats {
    display: flex;
    gap: 10px;
  }

  .stat {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    font-size: 11px;
    color: var(--color-text-tertiary);
  }

  .project-link {
    color: var(--color-text-tertiary);
    transition: color 0.2s;
  }

  .project-card:hover .project-link {
    color: var(--color-accent);
  }

  /* Support card — centered */
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
