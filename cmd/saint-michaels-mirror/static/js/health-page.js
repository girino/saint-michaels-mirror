// Health page JavaScript
let refreshInterval;

function getHealthClass(state) {
  switch(state) {
    case 'GREEN': return 'health-green';
    case 'YELLOW': return 'health-yellow';
    case 'RED': return 'health-red';
    default: return 'health-green';
  }
}

function getStatusClass(status) {
  switch(status) {
    case 'healthy': return 'status-healthy';
    case 'degraded': return 'status-degraded';
    case 'unhealthy': return 'status-unhealthy';
    default: return 'status-healthy';
  }
}

function updateLastUpdated() {
  const now = new Date();
  document.getElementById('last-updated').textContent = now.toLocaleTimeString();
}

function showLoading() {
  document.getElementById('overall-status').style.display = 'none';
  document.getElementById('health-content').style.display = 'none';
  document.getElementById('loading').style.display = 'block';
  document.getElementById('error').style.display = 'none';
}

function showError(message) {
  document.getElementById('overall-status').style.display = 'none';
  document.getElementById('health-content').style.display = 'none';
  document.getElementById('loading').style.display = 'none';
  document.getElementById('error').style.display = 'block';
  document.getElementById('error-message').textContent = message;
}

function showHealth() {
  document.getElementById('overall-status').style.display = 'block';
  document.getElementById('health-content').style.display = 'block';
  document.getElementById('loading').style.display = 'none';
  document.getElementById('error').style.display = 'none';
}

function populateHealth(data) {
  // Update overall status
  const overallStatusEl = document.getElementById('overall-status');
  overallStatusEl.innerHTML = `
    <h2>
      <span class="health-indicator ${getStatusClass(data.status)}">
        ${data.status.toUpperCase()}
      </span>
    </h2>
    <p>${data.service} v${data.version}</p>
  `;

  // Service information
  document.getElementById('service-name').textContent = data.service;
  document.getElementById('service-version').textContent = data.version;
  
  const serviceStatusEl = document.getElementById('service-status');
  serviceStatusEl.textContent = data.status;
  serviceStatusEl.className = `health-indicator ${getStatusClass(data.status)}`;

  // Health indicators
  const mainHealthEl = document.getElementById('main-health');
  mainHealthEl.textContent = data.main_health_state;
  mainHealthEl.className = `health-indicator ${getHealthClass(data.main_health_state)}`;

  const publishHealthEl = document.getElementById('publish-health');
  publishHealthEl.textContent = data.publish_health_state;
  publishHealthEl.className = `health-indicator ${getHealthClass(data.publish_health_state)}`;

  const queryHealthEl = document.getElementById('query-health');
  queryHealthEl.textContent = data.query_health_state;
  queryHealthEl.className = `health-indicator ${getHealthClass(data.query_health_state)}`;

  // Failure tracking
  document.getElementById('publish-failures').textContent = data.consecutive_publish_failures;
  document.getElementById('query-failures').textContent = data.consecutive_query_failures;
}

async function loadHealth() {
  try {
    showLoading();
    const response = await fetch('/api/v1/health');
    if (!response.ok) throw new Error('Failed to fetch health status');
    const data = await response.json();
    
    populateHealth(data);
    showHealth();
    updateLastUpdated();
  } catch (error) {
    showError(`Failed to load health status: ${error.message}`);
  }
}

// Load health immediately
loadHealth();

// Set up auto-refresh every 10 seconds
refreshInterval = setInterval(loadHealth, 10000);

// Clean up on page unload
window.addEventListener('beforeunload', () => {
  if (refreshInterval) {
    clearInterval(refreshInterval);
  }
});
