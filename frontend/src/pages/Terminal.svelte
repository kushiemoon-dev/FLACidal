<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { EventsOn } from '../../wailsjs/runtime/runtime.js';
  import { GetLogs, ClearLogs } from '../../wailsjs/go/main/App.js';

  interface LogEntry {
    timestamp: string;
    level: string;
    message: string;
  }

  let logs: LogEntry[] = [];
  let unsubscribe: () => void;
  let terminalContent: HTMLDivElement;

  onMount(async () => {
    // Load existing logs
    try {
      logs = await GetLogs();
    } catch (error) {
      console.error('Error loading logs:', error);
    }

    // Listen for new log events
    unsubscribe = EventsOn('log', (entry: LogEntry) => {
      logs = [...logs, entry];
      // Auto-scroll to bottom
      setTimeout(() => {
        if (terminalContent) {
          terminalContent.scrollTop = terminalContent.scrollHeight;
        }
      }, 10);
    });
  });

  onDestroy(() => {
    if (unsubscribe) {
      unsubscribe();
    }
  });

  function getLogColor(level: string): string {
    switch (level) {
      case 'error': return 'var(--color-error)';
      case 'warn': return 'var(--color-warning)';
      case 'success': return 'var(--color-success)';
      case 'info': return 'var(--color-text-tertiary)';
      default: return 'var(--color-text-secondary)';
    }
  }

  function getLevelPrefix(level: string): string {
    switch (level) {
      case 'error': return '[ERROR]';
      case 'warn': return '[WARN]';
      case 'success': return '[OK]';
      case 'info': return '[INFO]';
      default: return '[LOG]';
    }
  }

  async function handleClear() {
    await ClearLogs();
    logs = [];
  }
</script>

<div class="terminal-page">
  <div class="terminal-header">
    <div class="header-left">
      <h1>Terminal</h1>
      <p class="subtitle">Application logs and activity</p>
    </div>
    <div class="header-actions">
      <span class="log-count">{logs.length} entries</span>
      <button class="action-btn" on:click={handleClear} disabled={logs.length === 0}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M3 6h18"/>
          <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/>
          <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
        </svg>
        Clear Logs
      </button>
    </div>
  </div>

  <div class="terminal-container">
    <div class="terminal-window">
      <div class="terminal-titlebar">
        <div class="terminal-buttons">
          <span class="btn red"></span>
          <span class="btn yellow"></span>
          <span class="btn green"></span>
        </div>
        <span class="terminal-title">flacidal - logs</span>
      </div>

      <div class="terminal-content" bind:this={terminalContent}>
        {#if logs.length === 0}
          <div class="empty-terminal">
            <span class="prompt">$</span> No logs yet...
          </div>
        {:else}
          {#each logs as log}
            <div class="log-entry">
              <span class="timestamp">{log.timestamp}</span>
              <span class="level" style="color: {getLogColor(log.level)}">{getLevelPrefix(log.level)}</span>
              <span class="message" style="color: {getLogColor(log.level)}">{log.message}</span>
            </div>
          {/each}
        {/if}
        <div class="cursor-line">
          <span class="prompt">$</span>
          <span class="cursor"></span>
        </div>
      </div>
    </div>
  </div>
</div>

<style>
  .terminal-page {
    padding: 32px;
    max-width: 1000px;
    height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .terminal-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 24px;
  }

  .header-left h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0 0 8px 0;
  }

  .subtitle {
    color: var(--color-text-tertiary);
    margin: 0;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .log-count {
    font-size: 13px;
    color: var(--color-text-tertiary);
  }

  .action-btn {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 16px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    color: var(--color-text-secondary);
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .action-btn:hover:not(:disabled) {
    background: var(--color-bg-tertiary);
    color: var(--color-text-primary);
  }

  .action-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  .terminal-container {
    flex: 1;
    min-height: 0;
  }

  .terminal-window {
    height: 100%;
    max-height: calc(100vh - 180px);
    background: #0d0d0d;
    border-radius: 12px;
    overflow: hidden;
    border: 1px solid var(--color-border);
    display: flex;
    flex-direction: column;
  }

  .terminal-titlebar {
    display: flex;
    align-items: center;
    padding: 12px 16px;
    background: #1a1a1a;
    border-bottom: 1px solid #222;
  }

  .terminal-buttons {
    display: flex;
    gap: 8px;
  }

  .terminal-buttons .btn {
    width: 12px;
    height: 12px;
    border-radius: 50%;
  }

  .terminal-buttons .btn.red { background: #ff5f57; }
  .terminal-buttons .btn.yellow { background: #febc2e; }
  .terminal-buttons .btn.green { background: #28c840; }

  .terminal-title {
    flex: 1;
    text-align: center;
    font-size: 13px;
    color: #666;
    font-family: 'JetBrains Mono', monospace;
  }

  .terminal-content {
    flex: 1;
    padding: 16px;
    overflow-y: auto;
    font-family: 'JetBrains Mono', monospace;
    font-size: 13px;
    line-height: 1.6;
  }

  .empty-terminal {
    color: #444;
  }

  .log-entry {
    display: flex;
    gap: 12px;
    margin-bottom: 4px;
  }

  .timestamp {
    color: #444;
    flex-shrink: 0;
  }

  .level {
    flex-shrink: 0;
    font-weight: 500;
  }

  .message {
    word-break: break-all;
  }

  .cursor-line {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
  }

  .prompt {
    color: var(--color-accent);
    font-weight: 600;
  }

  .cursor {
    width: 8px;
    height: 16px;
    background: var(--color-accent);
    animation: blink 1s step-end infinite;
  }

  @keyframes blink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0; }
  }
</style>
