<script lang="ts">
  import { onMount } from 'svelte';
  import { ListDownloadedFiles, PreviewRename, RenameFiles, SelectDownloadFolder, GetDownloadFolder } from '../../lib/api';
  import { formatBytes } from '../../lib/format';
  import TabBar from '../../components/TabBar.svelte';
  import { toastStore } from '../../stores/toast';
  import { FolderOpen, RefreshCw, Eye, Pencil } from 'lucide-svelte';
  import { t, tm, type UiMsg } from '../../lib/i18n';

  interface FileEntry {
    path: string;
    name: string;
    size: number;
    selected: boolean;
  }

  let currentFolder = $state('');
  let files: FileEntry[] = $state([]);
  let loading = $state(false);
  let renaming = $state(false);
  let previewing = $state(false);
  let previewResult = $state<UiMsg>('');
  let activeTab = $state('tracks');
  let selectAll = $state(false);

  const renameTemplates = [
    '{title} - {artist}',
    '{artist} - {title}',
    '{track} {title}',
    '{track} {artist} - {title}',
    '{artist}/{album}/{track} {title}',
  ];
  let selectedTemplate = $state('{title} - {artist}');

  let tabs = $derived([
    { id: 'tracks', label: $t('tools.files.tabTracks', { count: files.length }) },
    { id: 'lyrics', label: $t('tools.files.tabLyrics', { count: 0 }) },
    { id: 'covers', label: $t('tools.files.tabCovers', { count: 0 }) },
  ]);

  function toggleSelectAll() {
    selectAll = !selectAll;
    files = files.map(f => ({ ...f, selected: selectAll }));
  }

  function toggleFile(index: number) {
    files = files.map((f, i) => i === index ? { ...f, selected: !f.selected } : f);
    selectAll = files.every(f => f.selected);
  }

  function getSelectedFiles(): string[] {
    return files.filter(f => f.selected).map(f => f.path);
  }

  function getFileName(path: string): string {
    return path.split('/').pop()?.split('\\').pop() || path;
  }

  onMount(async () => {
    try {
      const folder = await GetDownloadFolder();
      if (folder) {
        currentFolder = folder;
        await loadFiles();
      }
    } catch (err) {
      toastStore.showMsg((err as Error)?.message || { key: 'tools.files.loadFolderFailed' }, 'error');
    }
  });

  async function browseFolder() {
    try {
      const folder = await SelectDownloadFolder();
      if (folder) {
        currentFolder = folder;
        await loadFiles();
      }
    } catch (err) {
      toastStore.showMsg((err as Error)?.message || { key: 'tools.files.browseFailed' }, 'error');
    }
  }

  async function loadFiles() {
    if (!currentFolder) return;
    loading = true;
    try {
      const result = await ListDownloadedFiles();
      if (Array.isArray(result)) {
        files = result.map((f) => ({
          path: f.path,
          name: f.name || getFileName(f.path),
          size: f.size || 0,
          selected: false,
        }));
      } else {
        files = [];
      }
      selectAll = false;
    } catch {
      files = [];
    } finally {
      loading = false;
    }
  }

  async function previewRename() {
    const selected = getSelectedFiles();
    if (selected.length === 0) return;
    previewing = true;
    try {
      const result = await PreviewRename(selected, selectedTemplate);
      if (typeof result === 'string') {
        previewResult = result;
      } else if (Array.isArray(result) && result.length > 0) {
        previewResult = result.map((r) => r.newName).join('\n');
      } else {
        previewResult = { key: 'tools.files.noPreview' };
      }
    } catch (err) {
      previewResult = (err as Error)?.message || { key: 'tools.files.previewFailed' };
    } finally {
      previewing = false;
    }
  }

  async function applyRename() {
    const selected = getSelectedFiles();
    if (selected.length === 0) return;
    renaming = true;
    try {
      await RenameFiles(selected, selectedTemplate);
      await loadFiles();
      previewResult = '';
    } catch (err) {
      toastStore.showMsg((err as Error)?.message || { key: 'tools.files.renameFailed' }, 'error');
    }
    renaming = false;
  }
</script>

<div class="page">
  <header class="page-header">
    <h1>{$t('tools.files.title')}</h1>
    <p class="page-subtitle">{$t('tools.files.subtitle')}</p>
  </header>

  <div class="folder-bar">
    <input type="text" class="input folder-input" value={currentFolder} readonly placeholder={$t('tools.files.noFolder')} />
    <button class="btn btn-accent" onclick={browseFolder}>
      <FolderOpen size={16} />
      {$t('tools.files.browse')}
    </button>
    <button class="btn btn-outline" onclick={loadFiles} disabled={loading || !currentFolder}>
      <RefreshCw size={16} />
    </button>
  </div>

  <TabBar {tabs} bind:activeTab />

  {#if activeTab === 'tracks'}
    <div class="rename-section">
      <h3 class="section-title">{$t('tools.files.renameFormat')}</h3>
      <div class="rename-controls">
        <select class="select" bind:value={selectedTemplate}>
          {#each renameTemplates as tmpl (tmpl)}
            <option value={tmpl}>{tmpl}</option>
          {/each}
        </select>
      </div>
      {#if previewResult}
        <div class="preview-box">
          <span class="preview-label">{$t('tools.files.previewLabel')}</span>
          <span class="preview-text">{$tm(previewResult)}</span>
        </div>
      {/if}
    </div>

    <div class="file-list-header">
      <div class="file-list-left">
        <label class="checkbox-label">
          <input type="checkbox" checked={selectAll} onchange={toggleSelectAll} />
          {$t('tools.files.selectAll')}
        </label>
        <span class="file-count">{$t('tools.files.fileCount', { count: files.length })}</span>
      </div>
      <div class="file-list-actions">
        <button
          class="btn btn-outline btn-sm"
          onclick={previewRename}
          disabled={previewing || getSelectedFiles().length === 0}
        >
          <Eye size={14} />
          {$t('tools.files.preview')}
        </button>
        <button
          class="btn btn-accent btn-sm"
          onclick={applyRename}
          disabled={renaming || getSelectedFiles().length === 0}
        >
          <Pencil size={14} />
          {$t('tools.files.rename')}
        </button>
      </div>
    </div>

    {#if loading}
      <div class="empty-state">{$t('tools.files.loading')}</div>
    {:else if files.length === 0}
      <div class="empty-state">{$t('tools.files.noTracks')}</div>
    {:else}
      <div class="file-list">
        {#each files as file, i (file.path)}
          <label class="file-item">
            <input type="checkbox" checked={file.selected} onchange={() => toggleFile(i)} />
            <span class="file-name">{file.name}</span>
            {#if file.size > 0}
              <span class="file-size">{formatBytes(file.size)}</span>
            {/if}
          </label>
        {/each}
      </div>
    {/if}
  {:else}
    <div class="empty-state">{activeTab === 'lyrics' ? $t('tools.files.noLyrics') : $t('tools.files.noCovers')}</div>
  {/if}
</div>

<style>
  .page {
    padding: 32px 40px;
    max-width: 900px;
  }

  .page-header {
    margin-bottom: 32px;
  }

  .page-header h1 {
    font-size: 28px;
    font-weight: 700;
    margin: 0 0 6px;
    color: var(--color-text-primary);
  }

  .page-subtitle {
    margin: 0;
    font-size: 14px;
    color: var(--color-text-tertiary);
  }

  .folder-bar {
    display: flex;
    gap: 8px;
    align-items: center;
    margin-bottom: 24px;
  }

  .folder-input {
    flex: 1;
    min-width: 0;
  }

  .rename-section {
    margin-bottom: 20px;
  }

  .section-title {
    font-size: 15px;
    font-weight: 600;
    margin: 0 0 12px;
    color: var(--color-text-primary);
  }

  .rename-controls {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .preview-box {
    margin-top: 10px;
    padding: 10px 14px;
    border-radius: 8px;
    background: var(--color-bg-secondary);
    border: 1px solid var(--color-border);
    font-size: 13px;
  }

  .preview-label {
    color: var(--color-text-tertiary);
    margin-right: 8px;
  }

  .preview-text {
    color: var(--color-text-primary);
    font-family: 'JetBrains Mono', monospace;
  }

  .file-list-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .file-list-left {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    color: var(--color-text-secondary);
    cursor: pointer;
  }

  .file-count {
    font-size: 13px;
    color: var(--color-text-tertiary);
  }

  .file-list-actions {
    display: flex;
    gap: 8px;
  }

  .file-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
    max-height: 400px;
    overflow-y: auto;
    border: 1px solid var(--color-border);
    border-radius: 10px;
    padding: 6px;
    background: var(--color-bg-secondary);
  }

  .file-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 7px 10px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    color: var(--color-text-secondary);
  }

  .file-item:hover {
    background: var(--color-bg-hover);
  }

  .file-name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--color-text-primary);
  }

  .file-size {
    font-size: 12px;
    color: var(--color-text-tertiary);
    flex-shrink: 0;
  }

  .empty-state {
    text-align: center;
    padding: 48px 24px;
    color: var(--color-text-tertiary);
    font-size: 14px;
  }

  .input, .select {
    padding: 9px 12px;
    border-radius: 8px;
    border: 1px solid var(--color-border);
    background: var(--color-bg-secondary);
    color: var(--color-text-primary);
    font-family: var(--font-family);
    font-size: 14px;
  }

  .input:focus, .select:focus {
    outline: none;
    border-color: var(--color-accent);
  }

  .btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 8px 16px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    font-family: var(--font-family);
    cursor: pointer;
    transition: all 0.15s;
    border: 1px solid transparent;
  }

  .btn-sm {
    padding: 5px 12px;
    font-size: 13px;
  }

  .btn-accent {
    background: var(--color-accent);
    color: #000;
    border-color: var(--color-accent);
  }

  .btn-accent:hover:not(:disabled) {
    opacity: 0.9;
  }

  .btn-accent:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-outline {
    background: transparent;
    border: 1px solid var(--color-border);
    color: var(--color-text-secondary);
  }

  .btn-outline:hover:not(:disabled) {
    border-color: var(--color-text-tertiary);
    color: var(--color-text-primary);
  }

  .btn-outline:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
