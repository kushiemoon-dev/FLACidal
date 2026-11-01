<script lang="ts">
  import { onMount } from 'svelte';
  import { t, type MessageKey } from '../lib/i18n';
  import { toastStore } from '../stores/toast';
  import { toUiError } from '../lib/errors';
  import ConfirmDialog from './ConfirmDialog.svelte';
  import ExtensionDetailDialog from './ExtensionDetailDialog.svelte';
  import {
    ListExtensions,
    InstallExtension,
    InstallExtensionFromRegistry,
    UninstallExtension,
    SetExtensionEnabled,
    SetExtensionAuth,
    GetRegistrySources,
    AddRegistrySource,
    RemoveRegistrySource,
    BrowseRegistry,
    type ExtensionInfo,
    type RegistryEntry,
  } from '../lib/api';

  // Called after anything that can change the download chain, so the source
  // order list can reload.
  let { onChanged = () => {} }: { onChanged?: () => void } = $props();

  const DEFAULT_REGISTRY = 'https://github.com/kushiemoon-dev/flacidal-extensions';
  const CATEGORY_KEYS: Record<string, MessageKey> = {
    download: 'settings.ext.category.download',
    metadata: 'settings.ext.category.metadata',
    lyrics: 'settings.ext.category.lyrics',
    utility: 'settings.ext.category.utility',
  };
  // Core only acts on these two; the others are listed and can be enabled.
  const ACTIVE_CATEGORIES = ['download', 'metadata'];

  let extensions: ExtensionInfo[] = $state([]);
  let registries: string[] = $state([]);
  let browsed: Record<string, RegistryEntry[]> = $state({});
  let installUrl = $state('');
  let registryUrl = $state('');
  let authDraft: Record<string, string> = $state({});
  let busy = $state(false);

  type Pending =
    | { kind: 'install'; url: string }
    | { kind: 'registry'; repo: string; entry: RegistryEntry }
    | { kind: 'uninstall'; ext: ExtensionInfo };
  let pending: Pending | null = $state(null);
  let activating: ExtensionInfo | null = $state(null);

  function categoryText(ext: ExtensionInfo): string {
    const key = CATEGORY_KEYS[ext.category ?? ''] ?? 'settings.ext.category.other';
    return $t(key);
  }

  async function refresh() {
    try {
      [extensions, registries] = await Promise.all([ListExtensions(), GetRegistrySources()]);
    } catch (e) {
      toastStore.showMsg(toUiError(e), 'error');
    }
  }

  // Runs one action, reports its outcome and reloads. Always releases the busy flag.
  async function run(action: () => Promise<unknown>, success: MessageKey) {
    busy = true;
    try {
      await action();
      toastStore.showKey(success);
    } catch (e) {
      toastStore.showMsg(toUiError(e), 'error');
    } finally {
      busy = false;
      await refresh();
      onChanged();
    }
  }

  function askInstall() {
    const url = installUrl.trim();
    if (url) pending = { kind: 'install', url };
  }

  function confirmPending() {
    const p = pending;
    pending = null;
    if (!p) return;
    if (p.kind === 'install') {
      run(async () => {
        await InstallExtension(p.url);
        installUrl = '';
      }, 'settings.ext.toast.installed');
    } else if (p.kind === 'registry') {
      run(() => InstallExtensionFromRegistry(p.repo, p.entry.id), 'settings.ext.toast.installed');
    } else {
      run(() => UninstallExtension(p.ext.manifest.id), 'settings.ext.toast.uninstalled');
    }
  }

  function onToggle(e: MouseEvent, ext: ExtensionInfo) {
    if (!ext.enabled) {
      // Turning on goes through the review dialog first.
      e.preventDefault();
      activating = ext;
      return;
    }
    run(() => SetExtensionEnabled(ext.manifest.id, false), 'settings.ext.toast.disabled');
  }

  function confirmActivation() {
    const ext = activating;
    activating = null;
    if (ext) run(() => SetExtensionEnabled(ext.manifest.id, true), 'settings.ext.toast.enabled');
  }

  function saveAuth(ext: ExtensionInfo) {
    const data: Record<string, string> = {};
    for (const f of ext.manifest.authFields ?? []) {
      const v = authDraft[`${ext.manifest.id}:${f.key}`];
      if (v) data[f.key] = v;
    }
    run(async () => {
      await SetExtensionAuth(ext.manifest.id, data);
      for (const f of ext.manifest.authFields ?? []) delete authDraft[`${ext.manifest.id}:${f.key}`];
    }, 'settings.ext.toast.authSaved');
  }

  function addRegistry() {
    const url = registryUrl.trim();
    if (!url) return;
    run(async () => {
      await AddRegistrySource(url);
      registryUrl = '';
    }, 'settings.ext.toast.registryAdded');
  }

  function browse(repo: string) {
    busy = true;
    BrowseRegistry(repo)
      .then((entries) => { browsed = { ...browsed, [repo]: entries ?? [] }; })
      .catch((e) => toastStore.showMsg(toUiError(e), 'error'))
      .finally(() => { busy = false; });
  }

  const pendingTitle = $derived.by(() => {
    if (!pending) return '';
    return pending.kind === 'uninstall'
      ? $t('settings.ext.uninstallTitle', { name: pending.ext.manifest.name })
      : $t('settings.ext.installTitle');
  });
  const pendingMessage = $derived.by(() => {
    if (!pending) return '';
    if (pending.kind === 'uninstall') return $t('settings.ext.uninstallMessage');
    return $t('settings.ext.installMessage', { url: pending.kind === 'install' ? pending.url : pending.entry.downloadURL });
  });

  onMount(refresh);
</script>

<section class="ext" aria-labelledby="ext-title">
  <h2 id="ext-title" class="ext-heading">{$t('settings.ext.title')}</h2>
  <p class="ext-hint">{$t('settings.ext.intro')}</p>

  {#if extensions.length === 0}
    <p class="ext-empty">{$t('settings.ext.empty')}</p>
  {:else}
    <ul class="ext-list" aria-label={$t('settings.ext.listLabel')}>
      {#each extensions as ext (ext.manifest.id)}
        <li class="ext-card">
          <div class="ext-head">
            <div class="ext-name">
              <strong>{ext.manifest.name}</strong>
              <span class="ext-meta">{$t('settings.ext.version', { version: ext.manifest.version })} {$t('settings.ext.by', { author: ext.manifest.author })}</span>
            </div>
            <label class="ext-toggle">
              <input
                type="checkbox"
                aria-label={$t('settings.ext.enable', { name: ext.manifest.name })}
                checked={ext.enabled}
                disabled={busy}
                onclick={(e) => onToggle(e, ext)}
              />
              <span>{ext.enabled ? $t('settings.ext.stateOn') : $t('settings.ext.stateOff')}</span>
            </label>
          </div>
          <p class="ext-meta">{$t('settings.ext.category', { category: categoryText(ext) })}</p>
          {#if !ACTIVE_CATEGORIES.includes(ext.category ?? '')}
            <p class="ext-meta">{$t('settings.ext.noEffect')}</p>
          {/if}
          <p class="ext-meta">
            {$t('settings.ext.permissions')}: {ext.manifest.permissions?.length ? ext.manifest.permissions.join(', ') : $t('settings.ext.permissionsNone')}
          </p>
          {#if ext.hosts?.length}
            <p class="ext-meta">{$t('settings.ext.hosts')}: {ext.hosts.join(', ')}</p>
          {/if}

          {#if ext.manifest.authFields?.length}
            <fieldset class="ext-auth">
              <legend>{$t('settings.ext.authTitle')}</legend>
              {#each ext.manifest.authFields as field (field.key)}
                <label class="ext-field">
                  <span>{field.label}</span>
                  <input
                    type={field.type === 'password' ? 'password' : 'text'}
                    autocomplete="off"
                    placeholder={ext.authSet?.[field.key] ? $t('settings.ext.authAlreadySet') : ''}
                    bind:value={authDraft[`${ext.manifest.id}:${field.key}`]}
                  />
                </label>
              {/each}
              <button class="ext-btn" disabled={busy} onclick={() => saveAuth(ext)}>{$t('settings.ext.authSave')}</button>
            </fieldset>
          {/if}

          <button
            class="ext-btn ext-btn-danger"
            disabled={busy}
            aria-label={$t('settings.ext.uninstallAria', { name: ext.manifest.name })}
            onclick={() => (pending = { kind: 'uninstall', ext })}
          >{$t('settings.ext.uninstall')}</button>
        </li>
      {/each}
    </ul>
  {/if}

  <div class="ext-row">
    <label class="ext-field ext-grow">
      <span>{$t('settings.ext.installLabel')}</span>
      <input type="url" bind:value={installUrl} placeholder={$t('settings.ext.installPlaceholder')} autocomplete="off" />
    </label>
    <button class="ext-btn" disabled={busy || !installUrl.trim()} onclick={askInstall}>{$t('settings.ext.installButton')}</button>
  </div>

  <h3 class="ext-subheading">{$t('settings.ext.registriesTitle')}</h3>
  {#if registries.length === 0}
    <p class="ext-empty">{$t('settings.ext.registriesNone')}</p>
  {:else}
    <ul class="ext-list">
      {#each registries as repo (repo)}
        <li class="ext-card">
          <div class="ext-head">
            <span class="ext-url">{repo}{#if repo === DEFAULT_REGISTRY} <em>({$t('settings.ext.registryDefault')})</em>{/if}</span>
            <span class="ext-actions">
              <button class="ext-btn" disabled={busy} aria-label={$t('settings.ext.registryBrowseAria', { url: repo })} onclick={() => browse(repo)}>{$t('settings.ext.registryBrowse')}</button>
              <button
                class="ext-btn ext-btn-danger"
                disabled={busy}
                aria-label={$t('settings.ext.registryRemove', { url: repo })}
                onclick={() => run(() => RemoveRegistrySource(repo), 'settings.ext.toast.registryRemoved')}
              >{$t('settings.ext.registryRemoveButton')}</button>
            </span>
          </div>
          {#if browsed[repo]}
            {#if browsed[repo].length === 0}
              <p class="ext-empty">{$t('settings.ext.registryEmpty')}</p>
            {:else}
              <ul class="ext-list">
                {#each browsed[repo] as entry (entry.id)}
                  <li class="ext-entry">
                    <div>
                      <strong>{entry.name}</strong>
                      <span class="ext-meta">{entry.latestVersion}</span>
                      <p class="ext-meta">{entry.description}</p>
                    </div>
                    <button
                      class="ext-btn"
                      disabled={busy}
                      aria-label={$t('settings.ext.registryInstallAria', { name: entry.name })}
                      onclick={() => (pending = { kind: 'registry', repo, entry })}
                    >{$t('settings.ext.installButton')}</button>
                  </li>
                {/each}
              </ul>
            {/if}
          {/if}
        </li>
      {/each}
    </ul>
  {/if}

  <div class="ext-row">
    <label class="ext-field ext-grow">
      <span>{$t('settings.ext.registryAddLabel')}</span>
      <input type="url" bind:value={registryUrl} placeholder={$t('settings.ext.registryAddPlaceholder')} autocomplete="off" />
    </label>
    <button class="ext-btn" disabled={busy || !registryUrl.trim()} onclick={addRegistry}>{$t('settings.ext.registryAddButton')}</button>
  </div>
</section>

{#if pending}
  <ConfirmDialog
    title={pendingTitle}
    message={pendingMessage}
    confirmText={pending.kind === 'uninstall' ? $t('settings.ext.uninstall') : $t('settings.ext.installButton')}
    variant={pending.kind === 'uninstall' ? 'danger' : 'default'}
    onConfirm={confirmPending}
    onCancel={() => (pending = null)}
  />
{/if}

{#if activating}
  <ExtensionDetailDialog ext={activating} onConfirm={confirmActivation} onCancel={() => (activating = null)} />
{/if}

<style>
  .ext {
    margin-top: 1.5rem;
  }
  .ext-heading {
    margin: 0 0 0.5rem 0;
    font-size: 14px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
  .ext-subheading {
    margin: 1.25rem 0 0.5rem 0;
    font-size: 13px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
  .ext-hint,
  .ext-empty,
  .ext-meta {
    margin: 0 0 0.5rem 0;
    font-size: 13px;
    color: var(--color-text-secondary);
    line-height: 1.5;
    word-break: break-word;
  }
  .ext-list {
    list-style: none;
    margin: 0 0 1rem 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .ext-card {
    border: 1px solid var(--color-border);
    border-radius: 10px;
    padding: 12px;
    background: var(--color-bg-secondary);
  }
  .ext-entry {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
  }
  .ext-head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
    margin-bottom: 6px;
  }
  .ext-name {
    display: flex;
    flex-direction: column;
    gap: 2px;
    color: var(--color-text-primary);
  }
  .ext-url {
    font-size: 13px;
    color: var(--color-text-primary);
    word-break: break-all;
  }
  .ext-toggle {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    min-height: 44px;
    font-size: 13px;
    color: var(--color-text-primary);
  }
  .ext-actions {
    display: inline-flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .ext-row {
    display: flex;
    gap: 10px;
    align-items: flex-end;
    flex-wrap: wrap;
    margin-bottom: 0.5rem;
  }
  .ext-grow {
    flex: 1 1 220px;
  }
  .ext-field {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 13px;
    color: var(--color-text-secondary);
    margin-bottom: 8px;
  }
  .ext-field input {
    min-height: 44px;
    padding: 8px 10px;
    border-radius: 8px;
    border: 1px solid var(--color-border);
    background: var(--color-bg-primary, transparent);
    color: var(--color-text-primary);
  }
  .ext-auth {
    border: 1px solid var(--color-border-subtle);
    border-radius: 8px;
    padding: 8px 10px;
    margin: 0 0 10px 0;
    color: var(--color-text-secondary);
    font-size: 13px;
  }
  .ext-btn {
    min-height: 44px;
    padding: 8px 16px;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    background: var(--color-bg-tertiary, rgba(255, 255, 255, 0.05));
    border: 1px solid var(--color-border-subtle);
    color: var(--color-text-primary);
  }
  .ext-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .ext-btn-danger {
    color: #ef4444;
    border-color: rgba(239, 68, 68, 0.3);
  }
  :global([data-theme="light"]) .ext-btn-danger {
    color: #b91c1c;
  }
  @media (max-width: 480px) {
    .ext-head,
    .ext-entry {
      flex-direction: column;
      align-items: stretch;
    }
    .ext-btn {
      width: 100%;
    }
  }
</style>
