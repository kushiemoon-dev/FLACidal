<script lang="ts">
  import { queueItems } from '../stores/queue';

  let collapsed = $state(false);

  const jobList = $derived($queueItems.filter(item =>
    item.status === 'queued' || item.status === 'downloading' || item.status === 'pending'
  ));

  function statusLabel(status: string): string {
    switch (status) {
      case 'queued':
      case 'pending': return 'Queued';
      case 'downloading': return 'Downloading…';
      default: return status;
    }
  }

  function statusColor(status: string): string {
    switch (status) {
      case 'downloading': return 'var(--color-accent)';
      default: return 'var(--color-text-secondary)';
    }
  }
</script>

<div class="queue-panel" class:collapsed>
  <button class="panel-header" onclick={() => (collapsed = !collapsed)}>
    <span class="panel-title">
      Downloads
      {#if jobList.length > 0}
        <span class="badge">{jobList.length}</span>
      {/if}
    </span>
    <span class="chevron" class:rotated={!collapsed}>▲</span>
  </button>

  {#if !collapsed}
    <div class="panel-body">
      {#if jobList.length === 0}
        <p class="empty">No downloads in progress</p>
      {:else}
        <ul class="job-list">
          {#each jobList as job (job.trackId)}
            <li class="job-item">
              <div class="job-meta">
                <span class="job-title">{job.title}</span>
                <span class="job-artist">{job.artist}</span>
              </div>
              <div class="job-status" style="color: {statusColor(job.status)}">
                {statusLabel(job.status)}
              </div>
            </li>
          {/each}
        </ul>
      {/if}
    </div>
  {/if}
</div>

<style>
  .queue-panel {
    position: fixed;
    bottom: 0;
    right: 16px;
    width: 320px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    border-bottom: none;
    border-radius: 8px 8px 0 0;
    box-shadow: 0 -4px 24px rgba(0, 0, 0, 0.4);
    z-index: 200;
    font-size: 13px;
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 10px 14px;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--color-text-primary);
    font-size: 13px;
    font-weight: 600;
  }

  .panel-header:hover {
    background: var(--color-bg-hover);
  }

  .panel-title {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .badge {
    background: var(--color-accent);
    color: #fff;
    border-radius: 10px;
    padding: 1px 7px;
    font-size: 11px;
    font-weight: 700;
  }

  .chevron {
    font-size: 10px;
    color: var(--color-text-tertiary);
    transition: transform 0.2s;
  }

  .chevron.rotated {
    transform: rotate(180deg);
  }

  .panel-body {
    max-height: 260px;
    overflow-y: auto;
    border-top: 1px solid var(--color-border);
  }

  .empty {
    padding: 16px 14px;
    margin: 0;
    color: var(--color-text-tertiary);
    text-align: center;
  }

  .job-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .job-item {
    padding: 10px 14px;
    border-bottom: 1px solid var(--color-border-subtle);
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 4px 8px;
    align-items: center;
  }

  .job-item:last-child {
    border-bottom: none;
  }

  .job-meta {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }

  .job-title {
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--color-text-primary);
  }

  .job-artist {
    color: var(--color-text-secondary);
    font-size: 11px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .job-status {
    font-size: 11px;
    font-weight: 500;
    white-space: nowrap;
  }

  .progress-bar-track {
    grid-column: 1 / -1;
    height: 3px;
    background: var(--color-bg-hover);
    border-radius: 2px;
    overflow: hidden;
  }

  .progress-bar-fill {
    height: 100%;
    background: var(--color-accent);
    border-radius: 2px;
    transition: width 0.3s ease;
  }
</style>
