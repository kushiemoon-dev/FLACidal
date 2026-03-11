<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  let {
    title,
    message,
    confirmText = 'Confirm',
    cancelText = 'Cancel',
    variant = 'default',
    onConfirm,
    onCancel,
  }: {
    title: string;
    message: string;
    confirmText?: string;
    cancelText?: string;
    variant?: 'default' | 'danger';
    onConfirm: () => void;
    onCancel: () => void;
  } = $props();

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onCancel();
    }
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      onCancel();
    }
  }

  onMount(() => {
    document.addEventListener('keydown', handleKeydown);
  });

  onDestroy(() => {
    document.removeEventListener('keydown', handleKeydown);
  });
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="backdrop" onclick={handleBackdropClick}>
  <div class="dialog" class:danger={variant === 'danger'}>
    <h2 class="dialog-title">{title}</h2>
    <p class="dialog-message">{message}</p>
    <div class="dialog-actions">
      <button class="btn btn-cancel" onclick={onCancel}>{cancelText}</button>
      <button class="btn btn-confirm" class:btn-danger={variant === 'danger'} onclick={onConfirm}>{confirmText}</button>
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
    animation: fadeIn 0.15s ease;
  }

  .dialog {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 24px;
    max-width: 420px;
    width: 90%;
    animation: scaleIn 0.15s ease;
  }

  .dialog-title {
    margin: 0 0 12px 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .dialog-message {
    margin: 0 0 24px 0;
    font-size: 14px;
    color: var(--color-text-secondary);
    line-height: 1.5;
  }

  .dialog-actions {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  .btn {
    padding: 10px 20px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-cancel {
    background: var(--color-bg-tertiary, rgba(255, 255, 255, 0.05));
    border: 1px solid var(--color-border-subtle);
    color: var(--color-text-secondary);
  }

  .btn-cancel:hover {
    background: var(--color-bg-hover, rgba(255, 255, 255, 0.08));
    color: var(--color-text-primary);
  }

  .btn-confirm {
    background: rgba(244, 114, 182, 0.15);
    border: 1px solid rgba(244, 114, 182, 0.3);
    color: #f472b6;
  }

  .btn-confirm:hover {
    background: rgba(244, 114, 182, 0.25);
  }

  .btn-confirm.btn-danger {
    background: rgba(239, 68, 68, 0.15);
    border: 1px solid rgba(239, 68, 68, 0.3);
    color: #ef4444;
  }

  .btn-confirm.btn-danger:hover {
    background: rgba(239, 68, 68, 0.25);
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  @keyframes scaleIn {
    from {
      opacity: 0;
      transform: scale(0.95);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }
</style>
