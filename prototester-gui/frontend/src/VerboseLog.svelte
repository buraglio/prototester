<script>
  import { onMount, onDestroy } from 'svelte';
  import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime.js';

  export let visible = false;

  let logs = [];
  let autoScroll = true;
  let logContainer;

  onMount(() => {
    EventsOn('verbose', (message) => {
      logs = [...logs, message];
      if (logs.length > 500) {
        logs = logs.slice(-500);
      }
      if (autoScroll && logContainer) {
        setTimeout(() => {
          logContainer.scrollTop = logContainer.scrollHeight;
        }, 10);
      }
    });
  });

  onDestroy(() => {
    EventsOff('verbose');
  });

  function clearLogs() {
    logs = [];
  }

  function toggleAutoScroll() {
    autoScroll = !autoScroll;
  }

  function formatTime(timestamp) {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', { hour12: false });
  }

  function getLogClass(type) {
    switch (type) {
      case 'success':
        return 'log-success';
      case 'error':
        return 'log-error';
      case 'info':
      default:
        return 'log-info';
    }
  }
</script>

{#if visible}
  <div class="verbose-container">
    <div class="verbose-header">
      <h3>📋 Verbose Output</h3>
      <div class="header-controls">
        <label class="checkbox-label">
          <input type="checkbox" bind:checked={autoScroll} on:change={toggleAutoScroll} />
          Auto-scroll
        </label>
        <button class="btn-mini" on:click={clearLogs}>Clear</button>
      </div>
    </div>
    <div class="verbose-logs" bind:this={logContainer}>
      {#if logs.length === 0}
        <div class="empty-logs">
          <p>No logs yet. Enable verbose mode and run a test.</p>
        </div>
      {:else}
        {#each logs as log (log.timestamp)}
          <div class="log-entry {getLogClass(log.type)}">
            <span class="log-time">[{formatTime(log.timestamp)}]</span>
            <span class="log-message">{log.message}</span>
          </div>
        {/each}
      {/if}
    </div>
  </div>
{/if}

<style>
  .verbose-container {
    background: #1e1e1e;
    border-radius: 8px;
    padding: 15px;
    margin-top: 15px;
    box-shadow: 0 4px 8px rgba(0,0,0,0.2);
    max-height: 300px;
    display: flex;
    flex-direction: column;
  }

  .verbose-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;
    padding-bottom: 8px;
    border-bottom: 1px solid #444;
  }

  .verbose-header h3 {
    margin: 0;
    color: #fff;
    font-size: 1rem;
  }

  .header-controls {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 5px;
    color: #ccc;
    font-size: 0.85rem;
    cursor: pointer;
  }

  .checkbox-label input[type="checkbox"] {
    cursor: pointer;
  }

  .btn-mini {
    padding: 4px 10px;
    font-size: 0.75rem;
    background: #444;
    color: #fff;
    border: none;
    border-radius: 3px;
    cursor: pointer;
  }

  .btn-mini:hover {
    background: #555;
  }

  .verbose-logs {
    flex: 1;
    overflow-y: auto;
    font-family: 'Courier New', monospace;
    font-size: 0.85rem;
    line-height: 1.6;
  }

  .empty-logs {
    text-align: center;
    padding: 40px 20px;
    color: #888;
  }

  .log-entry {
    padding: 4px 8px;
    margin-bottom: 2px;
    border-radius: 3px;
    display: flex;
    gap: 10px;
  }

  .log-time {
    color: #888;
    flex-shrink: 0;
  }

  .log-message {
    flex: 1;
  }

  .log-info {
    color: #9cdcfe;
  }

  .log-success {
    color: #4ec9b0;
    background: rgba(78, 201, 176, 0.1);
  }

  .log-error {
    color: #f48771;
    background: rgba(244, 135, 113, 0.1);
  }

  .verbose-logs::-webkit-scrollbar {
    width: 8px;
  }

  .verbose-logs::-webkit-scrollbar-track {
    background: #2a2a2a;
  }

  .verbose-logs::-webkit-scrollbar-thumb {
    background: #555;
    border-radius: 4px;
  }

  .verbose-logs::-webkit-scrollbar-thumb:hover {
    background: #666;
  }
</style>
