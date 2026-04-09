<script lang="ts">
  import type { ComponentType } from 'svelte';
  let { icon: Icon, title, subtitle }: { icon: ComponentType; title: string; subtitle: string } = $props();
</script>

<div class="empty-state">
  <div class="empty-icon"><Icon size={48} /></div>
  <h3 class="empty-title">{title}</h3>
  <p class="empty-subtitle">{subtitle}</p>
</div>

<style>
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 60vh;
    text-align: center;
    padding: 2rem;
  }

  .empty-icon {
    color: var(--color-text-tertiary);
    margin-bottom: 1rem;
  }

  .empty-title {
    margin: 0 0 0.5rem;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .empty-subtitle {
    margin: 0;
    font-size: 0.875rem;
    color: var(--color-text-secondary);
    max-width: 320px;
  }
</style>
