<script lang="ts">
  import { Upload } from 'lucide-svelte';

  let {
    supportedFormats = 'FLAC',
    onFilesSelected,
    onFolderSelected,
  }: {
    supportedFormats?: string;
    onFilesSelected?: () => void;
    onFolderSelected?: () => void;
  } = $props();

  let isDragOver = $state(false);
  let dragCounter = $state(0);

  function handleDragEnter(e: DragEvent) {
    if (e.dataTransfer?.types.includes('Files')) {
      dragCounter++;
      isDragOver = true;
    }
  }

  function handleDragLeave(e: DragEvent) {
    if (e.dataTransfer?.types.includes('Files')) {
      dragCounter--;
      if (dragCounter === 0) isDragOver = false;
    }
  }

  function handleDragOver(e: DragEvent) {
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
  ondragenter={handleDragEnter}
  ondragleave={handleDragLeave}
  ondragover={handleDragOver}
  ondrop={handleDrop}
  role="region"
  aria-label="File drop zone"
>
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
