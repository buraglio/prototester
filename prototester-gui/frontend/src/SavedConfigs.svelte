<script>
  import { GetSavedConfigs, SaveConfig, DeleteSavedConfig } from '../wailsjs/go/main/App.js';
  import { onMount } from 'svelte';

  export let currentConfig = {};
  export let onLoadConfig = (config) => {};

  let configs = [];
  let loading = true;
  let saveName = '';
  let showSaveDialog = false;

  onMount(async () => {
    await loadConfigs();
  });

  async function loadConfigs() {
    loading = true;
    try {
      configs = await GetSavedConfigs();
    } catch (e) {
      console.error('Failed to load configs:', e);
    } finally {
      loading = false;
    }
  }

  async function saveCurrentConfig() {
    if (!saveName.trim()) {
      alert('Please enter a name for this configuration');
      return;
    }

    try {
      await SaveConfig(saveName, currentConfig);
      saveName = '';
      showSaveDialog = false;
      await loadConfigs();
    } catch (e) {
      alert(`Failed to save: ${e}`);
    }
  }

  async function deleteConfig(id) {
    if (confirm('Delete this saved configuration?')) {
      await DeleteSavedConfig(id);
      await loadConfigs();
    }
  }

  function loadConfig(config) {
    onLoadConfig(config.config);
  }
</script>

<div class="configs-container">
  <div class="configs-header">
    <h2>💾 Saved Configurations</h2>
    <button class="btn-small btn-primary" on:click={() => showSaveDialog = !showSaveDialog}>
      ➕ Save Current
    </button>
  </div>

  {#if showSaveDialog}
    <div class="save-dialog">
      <input
        type="text"
        bind:value={saveName}
        placeholder="Configuration name..."
        on:keypress={(e) => e.key === 'Enter' && saveCurrentConfig()}
      />
      <div class="dialog-actions">
        <button class="btn-small btn-primary" on:click={saveCurrentConfig}>Save</button>
        <button class="btn-small" on:click={() => showSaveDialog = false}>Cancel</button>
      </div>
    </div>
  {/if}

  {#if loading}
    <div class="loading-small">Loading configurations...</div>
  {:else if configs.length === 0}
    <div class="empty-state">
      <p>No saved configurations yet.</p>
      <p class="hint">Click "Save Current" to save your test configuration for later use.</p>
    </div>
  {:else}
    <div class="configs-list">
      {#each configs as config (config.id)}
        <div class="config-item">
          <div class="config-header">
            <strong>{config.name}</strong>
            <span class="config-date">{new Date(config.createdAt).toLocaleDateString()}</span>
          </div>
          <div class="config-details">
            <span class="protocol-tag">{config.config.protocol.toUpperCase()}</span>
            <span class="detail-text">
              {#if config.config.hostname}
                {config.config.hostname}
              {:else}
                {config.config.target4}
              {/if}
              : Port {config.config.port}
            </span>
          </div>
          <div class="config-actions">
            <button class="btn-mini btn-primary" on:click={() => loadConfig(config)}>
              ↻ Load
            </button>
            <button class="btn-mini btn-danger" on:click={() => deleteConfig(config.id)}>
              ✕
            </button>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .configs-container {
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  .configs-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 15px;
    padding-bottom: 10px;
    border-bottom: 2px solid #667eea;
  }

  .configs-header h2 {
    margin: 0;
    font-size: 1.5rem;
    color: #333;
  }

  .btn-small {
    padding: 6px 12px;
    font-size: 0.85rem;
    background: #6c757d;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .btn-small:hover {
    background: #5a6268;
  }

  .btn-primary {
    background: #667eea;
  }

  .btn-primary:hover {
    background: #5568d3;
  }

  .save-dialog {
    background: #f8f9fa;
    border-radius: 6px;
    padding: 15px;
    margin-bottom: 15px;
    border: 2px solid #667eea;
  }

  .save-dialog input {
    width: 100%;
    padding: 8px;
    border: 2px solid #e0e0e0;
    border-radius: 4px;
    font-size: 0.95rem;
    margin-bottom: 10px;
    box-sizing: border-box;
  }

  .save-dialog input:focus {
    outline: none;
    border-color: #667eea;
  }

  .dialog-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }

  .loading-small {
    text-align: center;
    padding: 40px;
    color: #999;
  }

  .empty-state {
    text-align: center;
    padding: 40px 20px;
    color: #999;
  }

  .hint {
    font-size: 0.9rem;
    margin-top: 10px;
  }

  .configs-list {
    display: flex;
    flex-direction: column;
    gap: 10px;
    overflow-y: auto;
  }

  .config-item {
    background: #f8f9fa;
    border-radius: 6px;
    padding: 12px;
    border: 2px solid transparent;
    transition: border-color 0.2s;
  }

  .config-item:hover {
    border-color: #667eea;
  }

  .config-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .config-header strong {
    color: #333;
    font-size: 1rem;
  }

  .config-date {
    font-size: 0.75rem;
    color: #888;
  }

  .config-details {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 10px;
    font-size: 0.85rem;
  }

  .protocol-tag {
    background: #667eea;
    color: white;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 600;
  }

  .detail-text {
    color: #666;
  }

  .config-actions {
    display: flex;
    gap: 6px;
  }

  .btn-mini {
    padding: 4px 8px;
    font-size: 0.75rem;
    background: #6c757d;
    color: white;
    border: none;
    border-radius: 3px;
    cursor: pointer;
  }

  .btn-mini.btn-primary {
    background: #667eea;
  }

  .btn-mini.btn-primary:hover {
    background: #5568d3;
  }

  .btn-danger {
    background: #dc3545;
  }

  .btn-danger:hover {
    background: #c82333;
  }
</style>
