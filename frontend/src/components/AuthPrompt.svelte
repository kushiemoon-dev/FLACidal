<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { isWailsRuntime } from '../lib/api';
  import { getAuthToken, setAuthToken, clearAuthToken, onAuthRequired } from '../lib/auth';
  import { t } from '../lib/i18n';

  let visible = $state(false);
  let rejected = $state(false);
  let value = $state('');
  let unsubscribe: (() => void) | undefined;

  onMount(() => {
    // The desktop app talks to Go directly: no token, no prompt.
    if (isWailsRuntime()) return;
    unsubscribe = onAuthRequired(() => {
      if (getAuthToken()) {
        clearAuthToken();
        rejected = true;
      }
      visible = true;
    });
  });

  onDestroy(() => unsubscribe?.());

  function submit(e: Event) {
    e.preventDefault();
    const token = value.trim();
    if (!token) return;
    setAuthToken(token);
    // A reload restarts the WebSocket and the initial loads with the token.
    window.location.reload();
  }
</script>

{#if visible}
  <div class="auth-required">
    <form class="auth-card" onsubmit={submit}>
      <h1 class="auth-title">{$t('shell.auth.title')}</h1>
      <p class="auth-body">{$t('shell.auth.body')}</p>

      {#if rejected}
        <div class="error-box" data-testid="auth-rejected">{$t('shell.auth.rejected')}</div>
      {/if}

      <label class="auth-label">
        {$t('shell.auth.label')}
        <input type="password" class="auth-input" bind:value autocomplete="current-password" />
      </label>

      <button type="submit" class="btn-unlock" disabled={!value.trim()}>{$t('shell.auth.submit')}</button>
    </form>
  </div>
{/if}

<style>
  .auth-required {
    position: fixed;
    inset: 0;
    background: var(--color-bg-primary);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
  }

  .auth-card {
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 12px;
    padding: 2rem;
    max-width: 480px;
    width: 90%;
    box-shadow: 0 16px 48px rgba(0, 0, 0, 0.5);
    text-align: center;
  }

  .auth-title {
    margin: 0 0 1rem;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--color-text-primary);
  }

  .auth-body {
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

  .auth-label {
    display: block;
    margin-bottom: 1.25rem;
    font-size: 0.8125rem;
    color: var(--color-text-secondary);
    text-align: left;
  }

  .auth-input {
    display: block;
    width: 100%;
    margin-top: 0.375rem;
    padding: 0.625rem 0.75rem;
    box-sizing: border-box;
    background: var(--color-bg-tertiary);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    color: var(--color-text-primary);
    font-size: 0.875rem;
  }

  .btn-unlock {
    padding: 0.625rem 1.25rem;
    border-radius: 8px;
    font-size: 0.875rem;
    font-weight: 500;
    cursor: pointer;
    border: none;
    background: var(--color-accent);
    color: #000;
    transition: background 0.15s, opacity 0.15s;
  }

  .btn-unlock:hover:not(:disabled) {
    background: var(--color-accent-hover);
  }

  .btn-unlock:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
</style>
