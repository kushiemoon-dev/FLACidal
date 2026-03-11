<script lang="ts">
  import { onMount } from 'svelte';

  interface MenuItem {
    label: string;
    icon?: string;
    action: () => void;
    divider?: boolean;
    disabled?: boolean;
  }

  let { items, x, y, onClose }: { items: MenuItem[]; x: number; y: number; onClose: () => void } = $props();

  let menuEl: HTMLDivElement | null = $state(null);
  let adjustedX = $state(x);
  let adjustedY = $state(y);
  let visible = $state(false);

  onMount(() => {
    if (menuEl) {
      const rect = menuEl.getBoundingClientRect();
      const vw = window.innerWidth;
      const vh = window.innerHeight;

      adjustedX = x + rect.width > vw ? vw - rect.width - 8 : x;
      adjustedY = y + rect.height > vh ? vh - rect.height - 8 : y;
    }
    // Trigger entrance animation on next frame
    requestAnimationFrame(() => { visible = true; });

    function handleKeydown(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose();
    }
    function handleClickOutside(e: MouseEvent) {
      if (menuEl && !menuEl.contains(e.target as Node)) onClose();
    }

    document.addEventListener('keydown', handleKeydown);
    document.addEventListener('mousedown', handleClickOutside);

    return () => {
      document.removeEventListener('keydown', handleKeydown);
      document.removeEventListener('mousedown', handleClickOutside);
    };
  });

  function handleItemClick(item: MenuItem) {
    if (item.disabled) return;
    item.action();
    onClose();
  }
</script>

<div
  class="context-menu"
  class:visible
  bind:this={menuEl}
  style="left: {adjustedX}px; top: {adjustedY}px;"
  role="menu"
>
  {#each items as item}
    {#if item.divider}
      <div class="divider"></div>
    {:else}
      <button
        class="menu-item"
        class:disabled={item.disabled}
        onclick={() => handleItemClick(item)}
        role="menuitem"
        disabled={item.disabled}
      >
        {#if item.icon}
          <span class="menu-icon">{item.icon}</span>
        {/if}
        <span class="menu-label">{item.label}</span>
      </button>
    {/if}
  {/each}
</div>

<style>
  .context-menu {
    position: fixed;
    z-index: 9999;
    min-width: 180px;
    background: var(--color-bg-secondary, #1e293b);
    border: 1px solid var(--color-border, rgba(255, 255, 255, 0.1));
    border-radius: 8px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5), 0 2px 8px rgba(0, 0, 0, 0.3);
    padding: 4px;
    opacity: 0;
    transform: scale(0.95);
    transition: opacity 0.12s ease, transform 0.12s ease;
  }

  .context-menu.visible {
    opacity: 1;
    transform: scale(1);
  }

  .divider {
    height: 1px;
    background: var(--color-border, rgba(255, 255, 255, 0.1));
    margin: 4px 8px;
  }

  .menu-item {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    padding: 8px 12px;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--color-text-primary, #e2e8f0);
    font-family: inherit;
    font-size: 0.85rem;
    cursor: pointer;
    transition: background 0.1s;
    text-align: left;
  }

  .menu-item:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.05);
  }

  .menu-item.disabled {
    color: var(--color-text-muted, #555);
    cursor: not-allowed;
    opacity: 0.5;
  }

  .menu-icon {
    font-size: 0.9rem;
    width: 18px;
    text-align: center;
    flex-shrink: 0;
  }

  .menu-label {
    white-space: nowrap;
  }
</style>
