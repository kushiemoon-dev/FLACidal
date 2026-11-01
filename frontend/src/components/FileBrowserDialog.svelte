<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { t, tm, type UiMsg } from '../lib/i18n';
  import { dialogFocus } from '../lib/dialogFocus';
  import { toUiError } from '../lib/errors';
  import { BrowseFolder, type BrowseResult } from '../lib/api';
  import type { PickScope } from '../lib/filePicker';

  const MAX_ENTRIES = 2000;

  let {
    kind,
    scope,
    title,
    onDone,
  }: {
    kind: 'files' | 'folder';
    scope: PickScope;
    title?: string;
    onDone: (paths: string[] | null) => void;
  } = $props();

  let listing = $state<BrowseResult | null>(null);
  let pathInput = $state('');
  let selected = $state<string[]>([]);
  let loading = $state(true);
  let error = $state<UiMsg>(null);
  let pathField: HTMLInputElement | undefined = $state();

  async function load(path: string) {
    loading = true;
    error = null;
    try {
      listing = await BrowseFolder(path, scope === 'tools' ? 'tools' : 'folders', kind === 'files');
      pathInput = listing.path;
      selected = [];
    } catch (e) {
      error = toUiError(e);
    } finally {
      loading = false;
      // A clicked folder button is gone after the reload; keep focus inside the dialog.
      pathField?.focus();
    }
  }

  function toggle(path: string, checked: boolean) {
    selected = checked ? [...selected, path] : selected.filter((p) => p !== path);
  }

  function confirm() {
    if (kind === 'folder') {
      if (listing) onDone([listing.path]);
    } else {
      onDone(selected);
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onDone(null);
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) onDone(null);
  }

  onMount(() => {
    document.addEventListener('keydown', handleKeydown);
    load('');
  });

  onDestroy(() => {
    document.removeEventListener('keydown', handleKeydown);
  });

  const heading = $derived(title ?? $t(kind === 'files' ? 'filePicker.title.files' : 'filePicker.title.folder'));
  const canConfirm = $derived(kind === 'folder' ? listing !== null : selected.length > 0);
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="backdrop" onclick={handleBackdropClick}>
  <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="file-browser-title" use:dialogFocus={{ trap: true }}>
    <h2 id="file-browser-title" class="dialog-title">{heading}</h2>

    <form class="path-row" onsubmit={(e) => { e.preventDefault(); load(pathInput); }}>
      <button type="button" class="btn" data-testid="up" disabled={!listing?.parent} onclick={() => listing && load(listing.parent)}>
        {$t('filePicker.up')}
      </button>
      <input type="text" class="path-input" aria-label={$t('filePicker.path')} bind:value={pathInput} bind:this={pathField} spellcheck="false" autocomplete="off" />
    </form>

    <div class="status" aria-live="polite">
      {#if loading}
        {$t('filePicker.loading')}
      {:else if error}
        <span class="error">{$tm(error)}</span>
      {:else if listing?.truncated}
        {$t('filePicker.truncated', { count: MAX_ENTRIES })}
      {/if}
    </div>

    <ul class="entries" aria-label={$t('filePicker.list')}>
      {#each listing?.entries ?? [] as entry (entry.path)}
        <li>
          {#if entry.isDir}
            <button type="button" class="entry folder" onclick={() => load(entry.path)}>{entry.name}<span aria-hidden="true">/</span></button>
          {:else}
            <label class="entry file">
              <input type="checkbox" checked={selected.includes(entry.path)} onchange={(e) => toggle(entry.path, e.currentTarget.checked)} />
              {entry.name}
            </label>
          {/if}
        </li>
      {/each}
    </ul>
    {#if !loading && !error && listing && listing.entries.length === 0}
      <p class="empty">{$t('filePicker.empty')}</p>
    {/if}

    <div class="dialog-actions">
      <button type="button" class="btn" onclick={() => onDone(null)}>{$t('common.cancel')}</button>
      <button type="button" class="btn btn-primary" disabled={!canConfirm} onclick={confirm}>
        {kind === 'folder' ? $t('filePicker.selectFolder') : $t('filePicker.select')}
      </button>
    </div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
  }

  .dialog {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border-subtle);
    border-radius: 12px;
    padding: 24px;
    max-width: 560px;
    width: 90%;
    max-height: 80vh;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .dialog-title {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .path-row {
    display: flex;
    gap: 8px;
  }

  .path-input {
    flex: 1;
    min-width: 0;
    padding: 8px 10px;
    border-radius: 8px;
    border: 1px solid var(--color-border-subtle);
    background: var(--color-bg-tertiary);
    color: var(--color-text-primary);
    font-size: 14px;
  }

  .status {
    min-height: 20px;
    font-size: 13px;
    color: var(--color-text-secondary);
  }

  .error {
    color: var(--color-error);
  }

  .entries {
    list-style: none;
    margin: 0;
    padding: 0;
    overflow-y: auto;
    flex: 1;
    min-height: 120px;
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
  }

  .entry {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 8px 12px;
    background: none;
    border: none;
    color: var(--color-text-primary);
    font-size: 14px;
    text-align: left;
    cursor: pointer;
  }

  .entry:hover {
    background: var(--color-bg-hover);
  }

  .folder {
    font-weight: 500;
  }

  .empty {
    margin: 0;
    font-size: 13px;
    color: var(--color-text-secondary);
  }

  .dialog-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .btn {
    padding: 8px 16px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    background: var(--color-bg-tertiary);
    border: 1px solid var(--color-border-subtle);
    color: var(--color-text-primary);
  }

  .btn:hover:not(:disabled) {
    background: var(--color-bg-hover);
  }

  .btn:disabled {
    cursor: not-allowed;
    opacity: 0.6;
  }

  .btn-primary {
    background: var(--color-accent-subtle);
    border-color: var(--color-accent);
  }
</style>
