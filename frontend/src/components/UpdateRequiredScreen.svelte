<script lang="ts">
  import type { UpdateStatus } from '../lib/api';

  let { status, onUpdate }: { status: UpdateStatus; onUpdate: () => void } = $props();

  let downloading = $state(false);
  let error = $state<string | null>(null);

  async function handleUpdate() {
    downloading = true;
    error = null;
    try {
      await onUpdate();
      // A successful update relaunches the app (see DownloadAndInstallUpdate,
      // internal/app/app_update.go), so control normally never returns here.
    } catch (e) {
      downloading = false;
      error = e instanceof Error ? e.message : String(e);
    }
  }

  function quit() {
    (window as any).runtime?.Quit?.();
  }
</script>

<div class="update-required">
  <div class="update-card">
    <h1 class="update-title">Update Required</h1>

    <p class="update-body">
      This version of FLACidal ({status.currentVersion}) is {status.versionsBehind} versions
      behind the latest release ({status.latestVersion}). Update now to keep using the app.
    </p>

    {#if error}
      <div class="error-box">{error}</div>
    {/if}

    <div class="update-actions">
      <button class="btn-quit" onclick={quit} disabled={downloading}>Quit</button>
      <button class="btn-update" onclick={handleUpdate} disabled={downloading}>
        {downloading ? 'Downloading update...' : 'Update Now'}
      </button>
    </div>
  </div>
</div>

<style>
  .update-required {
    position: fixed;
    inset: 0;
    background: var(--color-bg-primary);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
  }

  .update-card {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 2rem;
    max-width: 480px;
    width: 90%;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
    text-align: center;
  }

  .update-title {
    margin: 0 0 1rem;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .update-body {
    margin: 0 0 1.5rem;
    font-size: 0.9375rem;
    color: var(--color-text-secondary);
    line-height: 1.5;
  }

  .error-box {
    background: var(--color-bg-tertiary);
    border: 1px solid var(--color-danger, #e5484d);
    border-radius: 8px;
    padding: 0.75rem 1rem;
    font-size: 0.8125rem;
    color: var(--color-text-secondary);
    margin-bottom: 1.25rem;
    line-height: 1.5;
    text-align: left;
  }

  .update-actions {
    display: flex;
    justify-content: center;
    gap: 0.75rem;
  }

  .btn-quit, .btn-update {
    padding: 0.625rem 1.25rem;
    border-radius: 8px;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    border: none;
    transition: background 0.15s, opacity 0.15s;
  }

  .btn-quit {
    background: var(--color-bg-tertiary);
    color: var(--color-text-secondary);
  }

  .btn-quit:hover:not(:disabled) {
    background: var(--color-bg-hover);
  }

  .btn-update {
    background: var(--color-accent);
    color: #000;
  }

  .btn-update:hover:not(:disabled) {
    background: var(--color-accent-hover);
  }

  .btn-quit:disabled, .btn-update:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
</style>
