<script lang="ts">
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime.js';

  let { isOpen = $bindable(false), repoUrl }: { isOpen: boolean; repoUrl: string } = $props();

  let acknowledged = $state(false);

  function close() {
    isOpen = false;
    acknowledged = false;
  }

  function openIssues() {
    BrowserOpenURL(repoUrl);
    close();
  }

  function handleOverlayClick(e: MouseEvent) {
    if (e.target === e.currentTarget) close();
  }
</script>

{#if isOpen}
<div class="modal-overlay" onclick={handleOverlayClick}>
  <div class="modal-card">
    <h2 class="modal-title">Before Opening GitHub Issues</h2>

    <div class="warning-box">
      <strong>Important:</strong> Search existing issues first and use the issue template when opening a new report or request.
    </div>

    <label class="checkbox-row">
      <input type="checkbox" bind:checked={acknowledged} />
      <span>I understand that I should use the issue template and avoid duplicate issues.</span>
    </label>

    <div class="modal-actions">
      <button class="btn-cancel" onclick={close}>Cancel</button>
      <button class="btn-open" disabled={!acknowledged} onclick={openIssues}>Open Issues</button>
    </div>
  </div>
</div>
{/if}

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal-card {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 1.5rem;
    max-width: 440px;
    width: 90%;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
  }

  .modal-title {
    margin: 0 0 1rem;
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .warning-box {
    background: var(--color-bg-tertiary);
    border: 1px solid var(--color-warning);
    border-radius: 8px;
    padding: 0.75rem 1rem;
    font-size: 0.875rem;
    color: var(--color-text-secondary);
    margin-bottom: 1rem;
    line-height: 1.5;
  }

  .warning-box strong {
    color: var(--color-warning);
  }

  .checkbox-row {
    display: flex;
    align-items: flex-start;
    gap: 0.5rem;
    font-size: 0.8125rem;
    color: var(--color-text-secondary);
    cursor: pointer;
    margin-bottom: 1.25rem;
  }

  .checkbox-row input[type="checkbox"] {
    margin-top: 2px;
    accent-color: var(--color-accent);
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
  }

  .btn-cancel, .btn-open {
    padding: 0.5rem 1rem;
    border-radius: 8px;
    font-size: 0.8125rem;
    font-weight: 500;
    cursor: pointer;
    border: none;
    transition: background 0.15s, opacity 0.15s;
  }

  .btn-cancel {
    background: var(--color-bg-tertiary);
    color: var(--color-text-secondary);
  }

  .btn-cancel:hover {
    background: var(--color-bg-hover);
  }

  .btn-open {
    background: var(--color-accent);
    color: #000;
  }

  .btn-open:hover:not(:disabled) {
    background: var(--color-accent-hover);
  }

  .btn-open:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
</style>
