<script lang="ts">
  interface Tab {
    id: string;
    label: string;
    icon?: string;
  }

  let {
    tabs,
    activeTab = $bindable(),
  }: {
    tabs: Tab[];
    activeTab: string;
  } = $props();

  let tabRefs: Record<string, HTMLButtonElement> = {};
  let indicatorStyle = $state('');

  function selectTab(tabId: string) {
    activeTab = tabId;
    updateIndicator(tabId);
  }

  function updateIndicator(tabId: string) {
    const el = tabRefs[tabId];
    if (el) {
      indicatorStyle = `left: ${el.offsetLeft}px; width: ${el.offsetWidth}px;`;
    }
  }

  $effect(() => {
    if (activeTab) {
      requestAnimationFrame(() => updateIndicator(activeTab));
    }
  });
</script>

<div class="tab-bar">
  <div class="tab-bar-inner">
    {#each tabs as tab (tab.id)}
      <button
        bind:this={tabRefs[tab.id]}
        class="tab-item"
        class:active={activeTab === tab.id}
        onclick={() => selectTab(tab.id)}
      >
        {tab.label}
      </button>
    {/each}
    <div class="tab-indicator" style={indicatorStyle}></div>
  </div>
</div>

<style>
  .tab-bar {
    margin-bottom: 24px;
  }

  .tab-bar-inner {
    display: flex;
    gap: 0;
    position: relative;
    border-bottom: 1px solid var(--color-border, #1a1a1a);
  }

  .tab-item {
    padding: 10px 20px;
    background: none;
    border: none;
    color: var(--color-text-secondary, #a1a1a1);
    font-family: var(--font-family, 'Plus Jakarta Sans', sans-serif);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: color 0.2s ease;
    position: relative;
  }

  .tab-item:hover {
    color: var(--color-text-primary, #fafafa);
  }

  .tab-item.active {
    color: var(--color-text-primary, #fafafa);
    font-weight: 600;
  }

  .tab-indicator {
    position: absolute;
    bottom: -1px;
    height: 2px;
    background: var(--color-accent, #f472b6);
    transition: left 0.3s ease, width 0.3s ease;
    border-radius: 1px;
  }
</style>
