<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { t } from '../lib/i18n';
  import { dialogFocus } from '../lib/dialogFocus';
  import type { ExtensionInfo } from '../lib/api';

  let {
    ext,
    onConfirm,
    onCancel,
  }: {
    ext: ExtensionInfo;
    onConfirm: () => void;
    onCancel: () => void;
  } = $props();

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onCancel();
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) onCancel();
  }

  onMount(() => document.addEventListener('keydown', handleKeydown));
  onDestroy(() => document.removeEventListener('keydown', handleKeydown));
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="backdrop" onclick={handleBackdropClick}>
  <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="ext-detail-title" use:dialogFocus>
    <h2 id="ext-detail-title" class="dialog-title">{$t('settings.ext.activateTitle', { name: ext.manifest.name })}</h2>
    <p class="dialog-message">{$t('settings.ext.activateMessage')}</p>
    <dl class="facts">
      <dt>{$t('settings.ext.author')}</dt>
      <dd>{ext.manifest.author}</dd>
      <dt>{$t('settings.ext.permissions')}</dt>
      <dd>{ext.manifest.permissions?.length ? ext.manifest.permissions.join(', ') : $t('settings.ext.permissionsNone')}</dd>
      <dt>{$t('settings.ext.hosts')}</dt>
      <dd>{ext.hosts?.length ? ext.hosts.join(', ') : $t('settings.ext.permissionsNone')}</dd>
    </dl>
    <div class="dialog-actions">
      <button class="btn btn-cancel" onclick={onCancel}>{$t('common.cancel')}</button>
      <button class="btn btn-confirm" onclick={onConfirm}>{$t('settings.ext.activateConfirm')}</button>
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
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 24px;
    max-width: 460px;
    width: 90%;
  }
  .dialog-title {
    margin: 0 0 12px 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--color-text-primary);
  }
  .dialog-message {
    margin: 0 0 16px 0;
    font-size: 14px;
    color: var(--color-text-secondary);
    line-height: 1.5;
  }
  .facts {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 6px 16px;
    margin: 0 0 24px 0;
    font-size: 14px;
    color: var(--color-text-primary);
  }
  .facts dt {
    color: var(--color-text-secondary);
  }
  .facts dd {
    margin: 0;
    word-break: break-word;
  }
  .dialog-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
  .btn {
    min-height: 44px;
    padding: 10px 20px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
  }
  .btn-cancel {
    background: var(--color-bg-tertiary, rgba(255, 255, 255, 0.05));
    border: 1px solid var(--color-border-subtle);
    color: var(--color-text-secondary);
  }
  .btn-confirm {
    background: rgba(244, 114, 182, 0.15);
    border: 1px solid rgba(244, 114, 182, 0.3);
    color: #f472b6;
  }
  :global([data-theme="light"]) .btn-confirm {
    color: #9d174d;
  }
  @media (max-width: 480px) {
    .facts {
      grid-template-columns: 1fr;
    }
  }
</style>
