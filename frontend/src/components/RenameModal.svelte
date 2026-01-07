<script lang="ts">
  import { onMount } from 'svelte';
  import { GetRenameTemplates, PreviewRename, RenameFiles } from '../../wailsjs/go/main/App.js';

  export let files: string[];
  export let onClose: () => void;
  export let onComplete: () => void;

  interface Template {
    name: string;
    template: string;
  }

  interface Preview {
    oldPath: string;
    oldName: string;
    newName: string;
    newPath: string;
    hasError: boolean;
    error?: string;
  }

  let templates: Template[] = [];
  let selectedTemplate = '';
  let customTemplate = '';
  let useCustom = false;
  let previews: Preview[] = [];
  let isLoading = true;
  let isRenaming = false;
  let error = '';

  $: activeTemplate = useCustom ? customTemplate : selectedTemplate;
  $: canRename = previews.length > 0 && !previews.some(p => p.hasError) && activeTemplate;

  onMount(async () => {
    await loadTemplates();
  });

  async function loadTemplates() {
    isLoading = true;
    try {
      templates = (await GetRenameTemplates()) as unknown as Template[];
      if (templates.length > 0) {
        selectedTemplate = templates[0].template;
        await updatePreview();
      }
    } catch (e: any) {
      error = e.message || 'Failed to load templates';
    } finally {
      isLoading = false;
    }
  }

  async function updatePreview() {
    if (!activeTemplate) {
      previews = [];
      return;
    }

    try {
      previews = await PreviewRename(files, activeTemplate);
    } catch (e: any) {
      error = e.message || 'Failed to generate preview';
    }
  }

  async function handleRename() {
    if (!canRename || isRenaming) return;

    isRenaming = true;
    error = '';

    try {
      const results = await RenameFiles(files, activeTemplate);
      const failed = results.filter(r => !r.success);

      if (failed.length > 0) {
        error = `${failed.length} file(s) failed to rename`;
      }

      onComplete();
      onClose();
    } catch (e: any) {
      error = e.message || 'Rename failed';
    } finally {
      isRenaming = false;
    }
  }

  function handleTemplateChange() {
    updatePreview();
  }

  function handleCustomTemplateInput() {
    updatePreview();
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      onClose();
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onClose();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="modal-backdrop" on:click={handleBackdropClick} on:keydown={handleKeydown} role="dialog" aria-modal="true">
  <div class="modal-content">
    <div class="modal-header">
      <h2>Rename Files</h2>
      <span class="file-count">{files.length} file{files.length !== 1 ? 's' : ''} selected</span>
      <button class="close-btn" on:click={onClose}>
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
        </svg>
      </button>
    </div>

    {#if isLoading}
      <div class="loading-state">
        <div class="loader"></div>
        <p>Loading...</p>
      </div>
    {:else}
      <div class="modal-body">
        <!-- Template Selection -->
        <div class="template-section">
          <span class="section-label">Rename Template</span>

          <div class="template-options">
            {#each templates as tmpl}
              <label class="template-option" class:selected={!useCustom && selectedTemplate === tmpl.template}>
                <input
                  type="radio"
                  name="template"
                  value={tmpl.template}
                  bind:group={selectedTemplate}
                  on:change={() => { useCustom = false; handleTemplateChange(); }}
                />
                <span class="template-name">{tmpl.name}</span>
                <span class="template-pattern">{tmpl.template}</span>
              </label>
            {/each}

            <label class="template-option custom" class:selected={useCustom}>
              <input
                type="radio"
                name="template"
                value="custom"
                checked={useCustom}
                on:change={() => { useCustom = true; handleTemplateChange(); }}
              />
              <span class="template-name">Custom</span>
              {#if useCustom}
                <input
                  type="text"
                  class="custom-input"
                  placeholder="e.g. {'{'}artist{'}'} - {'{'}title{'}'}"
                  bind:value={customTemplate}
                  on:input={handleCustomTemplateInput}
                />
              {/if}
            </label>
          </div>

          <div class="template-vars">
            <span class="vars-label">Available:</span>
            <code>{`{title}`}</code>
            <code>{`{artist}`}</code>
            <code>{`{album}`}</code>
            <code>{`{tracknumber}`}</code>
            <code>{`{date}`}</code>
            <code>{`{genre}`}</code>
          </div>
        </div>

        <!-- Preview Section -->
        <div class="preview-section">
          <span class="section-label">Preview</span>

          <div class="preview-list">
            {#each previews as preview}
              <div class="preview-item" class:error={preview.hasError}>
                <div class="preview-old">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M9 18V5l12-2v13"/>
                    <circle cx="6" cy="18" r="3"/>
                    <circle cx="18" cy="16" r="3"/>
                  </svg>
                  <span>{preview.oldName}</span>
                </div>
                <svg class="arrow" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M5 12h14"/>
                  <path d="m12 5 7 7-7 7"/>
                </svg>
                <div class="preview-new" class:error={preview.hasError}>
                  <span>{preview.newName}</span>
                  {#if preview.hasError}
                    <span class="error-text">{preview.error}</span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        </div>

        {#if error}
          <div class="error-banner">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="10"/>
              <line x1="12" y1="8" x2="12" y2="12"/>
              <line x1="12" y1="16" x2="12.01" y2="16"/>
            </svg>
            <span>{error}</span>
          </div>
        {/if}
      </div>

      <div class="modal-footer">
        <button class="btn-secondary" on:click={onClose}>Cancel</button>
        <button class="btn-primary" on:click={handleRename} disabled={!canRename || isRenaming}>
          {#if isRenaming}
            <span class="spinner"></span>
            Renaming...
          {:else}
            Rename {files.length} File{files.length !== 1 ? 's' : ''}
          {/if}
        </button>
      </div>
    {/if}
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.8);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    animation: fadeIn 0.2s ease;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .modal-content {
    background: #111;
    border: 1px solid #222;
    border-radius: 16px;
    width: 90%;
    max-width: 600px;
    max-height: 85vh;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    animation: slideIn 0.2s ease;
  }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: scale(0.95) translateY(10px);
    }
    to {
      opacity: 1;
      transform: scale(1) translateY(0);
    }
  }

  .modal-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px 20px;
    border-bottom: 1px solid #222;
  }

  .modal-header h2 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
  }

  .file-count {
    font-size: 13px;
    color: #666;
    margin-left: auto;
    margin-right: 12px;
  }

  .close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    background: transparent;
    border: none;
    border-radius: 8px;
    color: #666;
    cursor: pointer;
    transition: all 0.2s;
  }

  .close-btn:hover {
    background: #222;
    color: #fff;
  }

  .modal-body {
    padding: 20px;
    overflow-y: auto;
    flex: 1;
  }

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: #555;
  }

  .loader {
    width: 36px;
    height: 36px;
    border: 3px solid #222;
    border-top-color: #f472b6;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin-bottom: 16px;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .section-label {
    display: block;
    font-size: 12px;
    font-weight: 600;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 12px;
  }

  .template-section {
    margin-bottom: 24px;
  }

  .template-options {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .template-option {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 14px;
    background: #0a0a0a;
    border: 1px solid #1a1a1a;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .template-option:hover {
    background: #111;
    border-color: #333;
  }

  .template-option.selected {
    background: rgba(244, 114, 182, 0.1);
    border-color: rgba(244, 114, 182, 0.3);
  }

  .template-option input[type="radio"] {
    display: none;
  }

  .template-name {
    font-size: 14px;
    color: #fff;
    font-weight: 500;
  }

  .template-pattern {
    font-size: 12px;
    color: #666;
    font-family: 'JetBrains Mono', monospace;
    margin-left: auto;
  }

  .template-option.custom {
    flex-wrap: wrap;
  }

  .custom-input {
    width: 100%;
    margin-top: 10px;
    padding: 10px 12px;
    background: #111;
    border: 1px solid #333;
    border-radius: 6px;
    color: #fff;
    font-size: 14px;
    font-family: 'JetBrains Mono', monospace;
    outline: none;
  }

  .custom-input:focus {
    border-color: #f472b6;
  }

  .template-vars {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    margin-top: 12px;
    padding: 10px 12px;
    background: #0a0a0a;
    border-radius: 6px;
  }

  .vars-label {
    font-size: 12px;
    color: #555;
  }

  .template-vars code {
    font-size: 11px;
    padding: 2px 6px;
    background: #1a1a1a;
    border-radius: 4px;
    color: #888;
    font-family: 'JetBrains Mono', monospace;
  }

  .preview-section {
    margin-bottom: 16px;
  }

  .preview-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-height: 200px;
    overflow-y: auto;
    padding: 4px;
  }

  .preview-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px;
    background: #0a0a0a;
    border: 1px solid #1a1a1a;
    border-radius: 8px;
  }

  .preview-item.error {
    border-color: rgba(239, 68, 68, 0.3);
    background: rgba(239, 68, 68, 0.05);
  }

  .preview-old {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
    color: #666;
  }

  .preview-old span {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: 13px;
  }

  .arrow {
    color: #444;
    flex-shrink: 0;
  }

  .preview-new {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .preview-new span {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: 13px;
    color: #22c55e;
  }

  .preview-new.error span:first-child {
    color: #ef4444;
  }

  .error-text {
    font-size: 11px;
    color: #ef4444 !important;
  }

  .error-banner {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px 14px;
    background: rgba(239, 68, 68, 0.1);
    border: 1px solid rgba(239, 68, 68, 0.2);
    border-radius: 8px;
    color: #ef4444;
    font-size: 13px;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    padding: 16px 20px;
    border-top: 1px solid #222;
  }

  .btn-secondary {
    padding: 10px 20px;
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 8px;
    color: #888;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-secondary:hover {
    background: #222;
    color: #fff;
  }

  .btn-primary {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 10px 20px;
    background: #f472b6;
    border: none;
    border-radius: 8px;
    color: #000;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-primary:hover:not(:disabled) {
    background: #ec4899;
  }

  .btn-primary:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .spinner {
    width: 14px;
    height: 14px;
    border: 2px solid transparent;
    border-top-color: currentColor;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
</style>
