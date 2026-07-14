<script lang="ts">
  import { Upload, AlertTriangle } from 'lucide-svelte';
  import { isWailsRuntime } from '../lib/api';

  let {
    supportedFormats = 'FLAC',
    onFilesSelected,
    onFolderSelected,
  }: {
    supportedFormats?: string;
    onFilesSelected?: () => void;
    onFolderSelected?: () => void;
  } = $props();

  // Neither drag-and-drop nor the native file/folder picker can produce a
  // real result in a plain browser (see lib/runtime.ts's onNativeFileDrop
  // and lib/api.ts's OpenFLACFilesDialog/SelectDownloadFolder/
  // SelectFolderForConversion docs) — show that honestly instead of
  // rendering controls that silently do nothing when clicked.
  const nativeFileAccessAvailable = isWailsRuntime();

  let isDragOver = $state(false);
  let dragCounter = $state(0);

  function handleDragEnter(e: DragEvent) {
    if (!nativeFileAccessAvailable) return;
    if (e.dataTransfer?.types.includes('Files')) {
      dragCounter++;
      isDragOver = true;
    }
  }

  function handleDragLeave(e: DragEvent) {
    if (!nativeFileAccessAvailable) return;
    if (e.dataTransfer?.types.includes('Files')) {
      dragCounter--;
      if (dragCounter === 0) isDragOver = false;
    }
  }

  function handleDragOver(e: DragEvent) {
    if (!nativeFileAccessAvailable) return;
    if (e.dataTransfer?.types.includes('Files')) {
      e.preventDefault();
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    dragCounter = 0;
    isDragOver = false;
  }
</script>

<div
  class="drop-zone"
  class:drag-over={isDragOver}
  class:unavailable={!nativeFileAccessAvailable}
  ondragenter={handleDragEnter}
  ondragleave={handleDragLeave}
  ondragover={handleDragOver}
  ondrop={handleDrop}
  role="region"
  aria-label="File drop zone"
>
  {#if nativeFileAccessAvailable}
    <Upload size={48} strokeWidth={1.5} color="var(--color-text-tertiary)" />
    <p class="drop-text">Drag and drop audio files here, or click the button below to select</p>
    <div class="drop-actions">
      {#if onFilesSelected}
        <button class="btn btn-primary" onclick={onFilesSelected}>Select Files</button>
      {/if}
      {#if onFolderSelected}
        <button class="btn btn-outline" onclick={onFolderSelected}>Select Folder</button>
      {/if}
    </div>
    <p class="supported-formats">Supported formats: {supportedFormats}</p>
  {:else}
    <AlertTriangle size={40} strokeWidth={1.5} color="var(--color-warning, #f59e0b)" />
    <p class="drop-text">File selection isn't available in browser mode</p>
    <p class="supported-formats">Drag-and-drop and the file picker need the FLACidal desktop app — a browser page can't hand this server real filesystem paths.</p>
  {/if}
</div>

<style>
  .drop-zone {
    border: 2px dashed var(--color-border);
    border-radius: 16px;
    padding: 48px 32px;
    text-align: center;
    transition: all 0.2s;
    min-height: 300px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 16px;
  }

  .drop-zone.drag-over {
    border-color: var(--color-accent);
    background: rgba(244, 114, 182, 0.05);
  }

  .drop-zone.unavailable {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .drop-text {
    margin: 0;
    font-size: 16px;
    color: var(--color-text-secondary);
    max-width: 360px;
  }

  .drop-actions {
    display: flex;
    gap: 12px;
    margin-top: 8px;
  }

  .btn {
    padding: 10px 20px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-primary {
    background: var(--color-accent);
    border: 1px solid var(--color-accent);
    color: #000;
  }

  .btn-primary:hover {
    opacity: 0.9;
  }

  .btn-outline {
    background: transparent;
    border: 1px solid var(--color-border);
    color: var(--color-text-secondary);
  }

  .btn-outline:hover {
    border-color: var(--color-text-tertiary);
    color: var(--color-text-primary);
  }

  .supported-formats {
    margin: 0;
    font-size: 13px;
    color: var(--color-text-muted);
  }
</style>
