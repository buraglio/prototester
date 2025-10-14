<script>
  import { GetHistory, DeleteHistoryEntry, ClearHistory, ExportHistoryToJSON } from '../wailsjs/go/main/App.js';
  import { onMount } from 'svelte';

  export let onLoadTest = (config) => {};

  let history = [];
  let selectedEntry = null;
  let loading = true;

  onMount(async () => {
    await loadHistory();
  });

  async function loadHistory() {
    loading = true;
    try {
      history = await GetHistory();
    } catch (e) {
      console.error('Failed to load history:', e);
    } finally {
      loading = false;
    }
  }

  async function deleteEntry(id) {
    if (confirm('Delete this test from history?')) {
      await DeleteHistoryEntry(id);
      await loadHistory();
      if (selectedEntry && selectedEntry.id === id) {
        selectedEntry = null;
      }
    }
  }

  async function clearAll() {
    if (confirm('Clear all test history?')) {
      await ClearHistory();
      await loadHistory();
      selectedEntry = null;
    }
  }

  async function exportHistory() {
    try {
      const path = await ExportHistoryToJSON();
      alert(`History exported to: ${path}`);
    } catch (e) {
      alert(`Failed to export: ${e}`);
    }
  }

  function selectEntry(entry) {
    selectedEntry = entry;
  }

  function loadTestConfig(entry) {
    onLoadTest(entry.request);
  }

  function formatDuration(ns) {
    if (!ns) return 'N/A';
    const ms = ns / 1000000;
    return ms.toFixed(2) + 'ms';
  }

  function formatSuccessRate(rate) {
    if (rate === undefined || rate === null) return 'N/A';
    return rate.toFixed(1) + '%';
  }
</script>

<div class="history-container">
  <div class="history-header">
    <h2>📜 Test History</h2>
    <div class="header-actions">
      <button class="btn-small" on:click={loadHistory}>🔄 Refresh</button>
      <button class="btn-small" on:click={exportHistory} disabled={history.length === 0}>💾 Export</button>
      <button class="btn-small btn-danger" on:click={clearAll} disabled={history.length === 0}>🗑️ Clear</button>
    </div>
  </div>

  {#if loading}
    <div class="loading-small">Loading history...</div>
  {:else if history.length === 0}
    <div class="empty-state">
      <p>No test history yet. Run a test to get started!</p>
    </div>
  {:else}
    <div class="history-content">
      <div class="history-list">
        {#each history as entry (entry.id)}
          <div
            class="history-item"
            class:selected={selectedEntry && selectedEntry.id === entry.id}
            on:click={() => selectEntry(entry)}
          >
            <div class="item-header">
              <span class="protocol-tag">{entry.request.protocol.toUpperCase()}</span>
              <span class="item-time">{new Date(entry.timestamp).toLocaleString()}</span>
            </div>
            <div class="item-summary">
              {#if entry.request.hostname}
                <strong>{entry.request.hostname}</strong>
              {:else}
                <strong>{entry.request.target4}</strong>
              {/if}
              {#if entry.result.ipv4_results}
                <span class="metric">Avg: {formatDuration(entry.result.ipv4_results.avg_ms)}</span>
                <span class="metric success">{formatSuccessRate(entry.result.ipv4_results.success_rate)}</span>
              {/if}
            </div>
            <div class="item-actions">
              <button class="btn-mini" on:click|stopPropagation={() => loadTestConfig(entry)}>
                ↻ Rerun
              </button>
              <button class="btn-mini btn-danger" on:click|stopPropagation={() => deleteEntry(entry.id)}>
                ✕
              </button>
            </div>
          </div>
        {/each}
      </div>

      {#if selectedEntry}
        <div class="history-detail">
          <h3>Test Details</h3>
          <div class="detail-content">
            <div class="detail-section">
              <h4>Configuration</h4>
              <div class="config-grid">
                <div><strong>Protocol:</strong> {selectedEntry.request.protocol}</div>
                <div><strong>Count:</strong> {selectedEntry.request.count}</div>
                <div><strong>Interval:</strong> {selectedEntry.request.interval}ms</div>
                <div><strong>Timeout:</strong> {selectedEntry.request.timeout}ms</div>
                {#if selectedEntry.request.hostname}
                  <div><strong>Hostname:</strong> {selectedEntry.request.hostname}</div>
                {:else}
                  <div><strong>IPv4:</strong> {selectedEntry.request.target4}</div>
                  <div><strong>IPv6:</strong> {selectedEntry.request.target6}</div>
                {/if}
              </div>
            </div>

            {#if selectedEntry.result.ipv4_results}
              <div class="detail-section">
                <h4>IPv4 Results</h4>
                <div class="stats-mini">
                  <div><strong>Success:</strong> {formatSuccessRate(selectedEntry.result.ipv4_results.success_rate)}</div>
                  <div><strong>Min:</strong> {formatDuration(selectedEntry.result.ipv4_results.min_ms)}</div>
                  <div><strong>Avg:</strong> {formatDuration(selectedEntry.result.ipv4_results.avg_ms)}</div>
                  <div><strong>Max:</strong> {formatDuration(selectedEntry.result.ipv4_results.max_ms)}</div>
                  <div><strong>Jitter:</strong> {formatDuration(selectedEntry.result.ipv4_results.jitter_ms)}</div>
                </div>
              </div>
            {/if}

            {#if selectedEntry.result.ipv6_results}
              <div class="detail-section">
                <h4>IPv6 Results</h4>
                <div class="stats-mini">
                  <div><strong>Success:</strong> {formatSuccessRate(selectedEntry.result.ipv6_results.success_rate)}</div>
                  <div><strong>Min:</strong> {formatDuration(selectedEntry.result.ipv6_results.min_ms)}</div>
                  <div><strong>Avg:</strong> {formatDuration(selectedEntry.result.ipv6_results.avg_ms)}</div>
                  <div><strong>Max:</strong> {formatDuration(selectedEntry.result.ipv6_results.max_ms)}</div>
                  <div><strong>Jitter:</strong> {formatDuration(selectedEntry.result.ipv6_results.jitter_ms)}</div>
                </div>
              </div>
            {/if}
          </div>
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  .history-container {
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  .history-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 15px;
    padding-bottom: 10px;
    border-bottom: 2px solid #667eea;
  }

  .history-header h2 {
    margin: 0;
    font-size: 1.5rem;
    color: #333;
  }

  .header-actions {
    display: flex;
    gap: 8px;
  }

  .btn-small {
    padding: 6px 12px;
    font-size: 0.85rem;
    background: #667eea;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .btn-small:hover:not(:disabled) {
    background: #5568d3;
  }

  .btn-small:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-danger {
    background: #dc3545;
  }

  .btn-danger:hover:not(:disabled) {
    background: #c82333;
  }

  .loading-small {
    text-align: center;
    padding: 40px;
    color: #999;
  }

  .empty-state {
    text-align: center;
    padding: 60px 20px;
    color: #999;
  }

  .history-content {
    display: grid;
    grid-template-columns: 300px 1fr;
    gap: 20px;
    flex: 1;
    overflow: hidden;
  }

  .history-list {
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .history-item {
    background: #f8f9fa;
    border-radius: 6px;
    padding: 12px;
    cursor: pointer;
    border: 2px solid transparent;
    transition: all 0.2s;
  }

  .history-item:hover {
    border-color: #667eea;
    background: #fff;
  }

  .history-item.selected {
    border-color: #667eea;
    background: #f0f2ff;
  }

  .item-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .protocol-tag {
    background: #667eea;
    color: white;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 0.75rem;
    font-weight: 600;
  }

  .item-time {
    font-size: 0.75rem;
    color: #888;
  }

  .item-summary {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 8px;
    font-size: 0.85rem;
  }

  .metric {
    padding: 2px 6px;
    background: #e9ecef;
    border-radius: 3px;
    font-size: 0.75rem;
  }

  .metric.success {
    background: #d4edda;
    color: #155724;
  }

  .item-actions {
    display: flex;
    gap: 6px;
    margin-top: 8px;
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

  .btn-mini:hover {
    background: #5a6268;
  }

  .btn-mini.btn-danger {
    background: #dc3545;
  }

  .btn-mini.btn-danger:hover {
    background: #c82333;
  }

  .history-detail {
    background: #f8f9fa;
    border-radius: 8px;
    padding: 20px;
    overflow-y: auto;
  }

  .history-detail h3 {
    margin: 0 0 15px 0;
    color: #333;
  }

  .detail-content {
    display: flex;
    flex-direction: column;
    gap: 15px;
  }

  .detail-section h4 {
    margin: 0 0 10px 0;
    color: #666;
    font-size: 1rem;
  }

  .config-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
    font-size: 0.9rem;
  }

  .stats-mini {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
    font-size: 0.9rem;
  }
</style>
