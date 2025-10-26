/*
Copyright (c) 2025 Girino Vey.

This software is licensed under Girino's Anarchist License (GAL).
See LICENSE file for full license text.
License available at: https://license.girino.org/

Statistics page functionality for Espelho de São Miguel web interface.
*/

// Stats page JavaScript
let refreshInterval;

function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function formatDuration(ms) {
  if (ms === 0) return '0 ms';
  if (ms < 1000) return ms.toFixed(1) + ' ms';
  return (ms / 1000).toFixed(2) + ' s';
}

function getHealthClass(state) {
  switch(state) {
    case 'GREEN': return 'health-green';
    case 'YELLOW': return 'health-yellow';
    case 'RED': return 'health-red';
    default: return 'health-green';
  }
}

function updateLastUpdated() {
  const now = new Date();
  document.getElementById('last-updated').textContent = now.toLocaleTimeString();
}

function showLoading() {
  document.getElementById('stats-content').style.display = 'none';
  document.getElementById('loading').style.display = 'block';
  document.getElementById('error').style.display = 'none';
}

function showError(message) {
  document.getElementById('stats-content').style.display = 'none';
  document.getElementById('loading').style.display = 'none';
  document.getElementById('error').style.display = 'block';
  document.getElementById('error-message').textContent = message;
}

function showStats() {
  document.getElementById('stats-content').style.display = 'block';
  document.getElementById('loading').style.display = 'none';
  document.getElementById('error').style.display = 'none';
}

function populateStats(data) {
  // Application stats
  document.getElementById('app-version').textContent = data.app?.version || '-';
  document.getElementById('app-uptime').textContent = `${Math.floor((data.app?.uptime || 0) / 60)}m ${Math.floor((data.app?.uptime || 0) % 60)}s`;
  document.getElementById('app-goroutines').textContent = data.app?.goroutines?.count || '-';
  document.getElementById('app-memory-used').textContent = formatBytes(data.app?.memory?.heap_alloc_bytes || 0);
  document.getElementById('app-memory-total').textContent = formatBytes(data.app?.memory?.sys_bytes || 0);
  document.getElementById('app-gc-cycles').textContent = data.app?.gc?.cycles || '-';

  // Relay operations
  document.getElementById('relay-publish-attempts').textContent = data.relay?.publish_attempts || '-';
  document.getElementById('relay-publish-successes').textContent = data.relay?.publish_successes || '-';
  document.getElementById('relay-publish-failures').textContent = data.relay?.publish_failures || '-';
  document.getElementById('relay-query-requests').textContent = data.relay?.query_requests || '-';
  document.getElementById('relay-query-events').textContent = data.relay?.query_events_returned || '-';
  document.getElementById('relay-count-requests').textContent = data.relay?.count_requests || '-';

  // Mirroring operations
  document.getElementById('relay-mirrored-events').textContent = data.mirror?.mirrored_events || '-';
  document.getElementById('relay-mirror-attempts').textContent = data.mirror?.mirror_attempts || '-';
  document.getElementById('relay-mirror-successes').textContent = data.mirror?.mirror_successes || '-';
  document.getElementById('relay-mirror-failures').textContent = data.mirror?.mirror_failures || '-';

  // Relay health
  document.getElementById('relay-live-count').textContent = data.mirror?.live_relays || '-';
  document.getElementById('relay-dead-count').textContent = data.mirror?.dead_relays || '-';
  document.getElementById('relay-total-count').textContent = (data.mirror?.live_relays || 0) + (data.mirror?.dead_relays || 0);

  // Health status
  const overallHealthEl = document.getElementById('health-overall');
  const overallHealthState = data.relay?.main_health_state || 'UNKNOWN';
  overallHealthEl.textContent = overallHealthState;
  overallHealthEl.className = `health-indicator ${getHealthClass(overallHealthState)}`;

  const publishHealthEl = document.getElementById('health-publish');
  const publishHealthState = data.relay?.publish_health_state || 'UNKNOWN';
  publishHealthEl.textContent = publishHealthState;
  publishHealthEl.className = `health-indicator ${getHealthClass(publishHealthState)}`;

  const queryHealthEl = document.getElementById('health-query');
  const queryHealthState = data.relay?.query_health_state || 'UNKNOWN';
  queryHealthEl.textContent = queryHealthState;
  queryHealthEl.className = `health-indicator ${getHealthClass(queryHealthState)}`;

  const mirrorHealthEl = document.getElementById('health-mirror');
  const mirrorHealthState = data.mirror?.mirror_health_state || 'UNKNOWN';
  mirrorHealthEl.textContent = mirrorHealthState;
  mirrorHealthEl.className = `health-indicator ${getHealthClass(mirrorHealthState)}`;

  const goroutineHealthEl = document.getElementById('health-goroutines');
  const goroutineHealthState = data.app?.goroutines?.health_state || 'UNKNOWN';
  goroutineHealthEl.textContent = goroutineHealthState;
  goroutineHealthEl.className = `health-indicator ${getHealthClass(goroutineHealthState)}`;

  document.getElementById('health-publish-failures').textContent = data.relay?.consecutive_publish_failures || '-';
  document.getElementById('health-query-failures').textContent = data.relay?.consecutive_query_failures || '-';
  document.getElementById('health-mirror-failures').textContent = data.mirror?.consecutive_mirror_failures || '-';

  // Performance
  document.getElementById('perf-publish-avg').textContent = formatDuration(data.relay?.average_publish_duration_ms || 0);
  document.getElementById('perf-query-avg').textContent = formatDuration(data.relay?.average_query_duration_ms || 0);
  document.getElementById('perf-count-avg').textContent = formatDuration(data.relay?.average_count_duration_ms || 0);
  document.getElementById('perf-publish-total').textContent = formatDuration(data.relay?.total_publish_duration_ms || 0);
  document.getElementById('perf-query-total').textContent = formatDuration(data.relay?.total_query_duration_ms || 0);
  document.getElementById('perf-count-total').textContent = formatDuration(data.relay?.total_count_duration_ms || 0);
}

async function loadStats() {
  try {
    showLoading();
    const response = await fetch('/api/v1/stats');
    if (!response.ok) throw new Error('Failed to fetch stats');
    const data = await response.json();
    
    populateStats(data);
    showStats();
    updateLastUpdated();
  } catch (error) {
    showError(`Failed to load statistics: ${error.message}`);
  }
}

// Load stats immediately
loadStats();

// Set up auto-refresh every 10 seconds
refreshInterval = setInterval(loadStats, 10000);

// Clean up on page unload
window.addEventListener('beforeunload', () => {
  if (refreshInterval) {
    clearInterval(refreshInterval);
  }
});
