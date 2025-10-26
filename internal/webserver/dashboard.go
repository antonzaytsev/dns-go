package webserver

// dashboardHTML contains the HTML template for the dashboard
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <link rel="stylesheet" href="/static/dashboard.css">
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/chartjs-adapter-date-fns/dist/chartjs-adapter-date-fns.bundle.min.js"></script>
</head>
<body>
    <div class="container">
        <header class="header">
            <h1>{{.Title}}</h1>
            <div class="header-info">
                <span class="version">Version: {{.Version}}</span>
                <span class="status" id="status">●</span>
                <span class="last-updated" id="lastUpdated">Last updated: Never</span>
            </div>
        </header>

        <div class="metrics-grid">
            <!-- Overview Cards -->
            <div class="card overview-card">
                <h3>Total Requests</h3>
                <div class="metric-value" id="totalRequests">-</div>
                <div class="metric-subtitle" id="requestsPerSecond">- req/sec</div>
            </div>

            <div class="card overview-card">
                <h3>Cache Hit Rate</h3>
                <div class="metric-value" id="cacheHitRate">-%</div>
                <div class="metric-subtitle">Cache Performance</div>
            </div>

            <div class="card overview-card">
                <h3>Success Rate</h3>
                <div class="metric-value" id="successRate">-%</div>
                <div class="metric-subtitle">Query Success</div>
            </div>

            <div class="card overview-card">
                <h3>Avg Response Time</h3>
                <div class="metric-value" id="avgResponseTime">- ms</div>
                <div class="metric-subtitle">Performance</div>
            </div>

            <div class="card overview-card">
                <h3>Clients</h3>
                <div class="metric-value" id="clients">-</div>
                <div class="metric-subtitle">Last Hour</div>
            </div>

            <div class="card overview-card">
                <h3>Uptime</h3>
                <div class="metric-value" id="uptime">-</div>
                <div class="metric-subtitle">System Uptime</div>
            </div>

            <!-- Charts -->
            <div class="card chart-card">
                <h3>Requests per Minute (Last Hour)</h3>
                <canvas id="hourlyChart"></canvas>
            </div>

            <div class="card chart-card">
                <h3>Requests per Hour (Last Day)</h3>
                <canvas id="dailyChart"></canvas>
            </div>

            <!-- Query Types -->
            <div class="card">
                <h3>Query Types</h3>
                <div class="query-types" id="queryTypes">
                    <div class="loading">Loading...</div>
                </div>
            </div>

            <!-- Top Clients -->
            <div class="card">
                <h3>Top Clients</h3>
                <div class="clients-table" id="topClients">
                    <div class="loading">Loading...</div>
                </div>
            </div>

            <!-- Upstream Servers -->
            <div class="card">
                <h3>Upstream Servers</h3>
                <div class="upstream-servers" id="upstreamServers">
                    <div class="loading">Loading...</div>
                </div>
            </div>

            <!-- Recent Requests -->
            <div class="card recent-requests-card">
                <h3>Recent Requests</h3>
                <div class="recent-requests" id="recentRequests">
                    <div class="loading">Loading...</div>
                </div>
            </div>
        </div>
    </div>

    <script>
        const API_BASE = window.location.origin;
        const WEB_PORT = '{{.Port}}';
    </script>
    <script src="/static/dashboard.js"></script>
</body>
</html>`

// dashboardCSS contains the CSS styles for the dashboard
const dashboardCSS = `
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    min-height: 100vh;
    color: #333;
}

.container {
    max-width: 1400px;
    margin: 0 auto;
    padding: 20px;
}

.header {
    background: rgba(255, 255, 255, 0.95);
    backdrop-filter: blur(10px);
    border-radius: 15px;
    padding: 20px 30px;
    margin-bottom: 30px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    box-shadow: 0 8px 32px rgba(31, 38, 135, 0.37);
    border: 1px solid rgba(255, 255, 255, 0.18);
}

.header h1 {
    color: #2d3748;
    font-size: 2rem;
    font-weight: 700;
}

.header-info {
    display: flex;
    align-items: center;
    gap: 20px;
    font-size: 0.9rem;
    color: #666;
}

.status {
    color: #48bb78;
    font-size: 1.2rem;
    animation: pulse 2s infinite;
}

@keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
}

.metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 20px;
}

.card {
    background: rgba(255, 255, 255, 0.95);
    backdrop-filter: blur(10px);
    border-radius: 15px;
    padding: 25px;
    box-shadow: 0 8px 32px rgba(31, 38, 135, 0.37);
    border: 1px solid rgba(255, 255, 255, 0.18);
    transition: transform 0.3s ease, box-shadow 0.3s ease;
}

.card:hover {
    transform: translateY(-5px);
    box-shadow: 0 12px 40px rgba(31, 38, 135, 0.5);
}

.card h3 {
    color: #2d3748;
    margin-bottom: 15px;
    font-size: 1.1rem;
    font-weight: 600;
}

.overview-card {
    text-align: center;
    min-height: 120px;
    display: flex;
    flex-direction: column;
    justify-content: center;
}

.metric-value {
    font-size: 2.5rem;
    font-weight: 700;
    color: #4299e1;
    margin: 10px 0;
}

.metric-subtitle {
    color: #718096;
    font-size: 0.9rem;
}

.chart-card {
    grid-column: span 2;
    min-height: 300px;
}

.recent-requests-card {
    grid-column: span 3;
}

.query-types {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
    gap: 10px;
}

.query-type {
    background: #f7fafc;
    padding: 10px;
    border-radius: 8px;
    text-align: center;
    border: 2px solid #e2e8f0;
    transition: all 0.3s ease;
}

.query-type:hover {
    border-color: #4299e1;
    transform: scale(1.05);
}

.query-type-name {
    font-weight: 600;
    color: #2d3748;
    margin-bottom: 5px;
}

.query-type-count {
    font-size: 1.2rem;
    color: #4299e1;
    font-weight: 700;
}

.clients-table {
    overflow-x: auto;
}

.clients-table table {
    width: 100%;
    border-collapse: collapse;
}

.clients-table th,
.clients-table td {
    padding: 12px;
    text-align: left;
    border-bottom: 1px solid #e2e8f0;
}

.clients-table th {
    background: #f7fafc;
    font-weight: 600;
    color: #2d3748;
}

.clients-table tr:hover {
    background: #f7fafc;
}

.upstream-servers {
    display: grid;
    gap: 15px;
}

.upstream-server {
    background: #f7fafc;
    padding: 15px;
    border-radius: 8px;
    border-left: 4px solid #4299e1;
}

.upstream-server.failed {
    border-left-color: #f56565;
}

.upstream-name {
    font-weight: 600;
    color: #2d3748;
    margin-bottom: 8px;
}

.upstream-stats {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
    gap: 10px;
    font-size: 0.9rem;
    color: #666;
}

.recent-requests {
    max-height: 400px;
    overflow-y: auto;
}

.request-item {
    background: #f7fafc;
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 10px;
    border-left: 4px solid #4299e1;
    transition: all 0.3s ease;
}

.request-item:hover {
    background: #edf2f7;
    transform: translateX(5px);
}

.request-item.cache-hit {
    border-left-color: #48bb78;
}

.request-item.failed {
    border-left-color: #f56565;
}

.request-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 5px;
}

.request-query {
    font-weight: 600;
    color: #2d3748;
}

.request-status {
    padding: 2px 8px;
    border-radius: 12px;
    font-size: 0.8rem;
    font-weight: 500;
}

.request-status.success {
    background: #c6f6d5;
    color: #22543d;
}

.request-status.cache-hit {
    background: #bee3f8;
    color: #2a4365;
}

.request-status.failed {
    background: #fed7d7;
    color: #742a2a;
}

.request-details {
    font-size: 0.9rem;
    color: #666;
    display: flex;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 10px;
}

.loading {
    text-align: center;
    color: #666;
    padding: 20px;
}

.error {
    color: #f56565;
    text-align: center;
    padding: 20px;
}

/* Responsive design */
@media (max-width: 768px) {
    .container {
        padding: 10px;
    }
    
    .header {
        flex-direction: column;
        text-align: center;
        gap: 15px;
    }
    
    .metrics-grid {
        grid-template-columns: 1fr;
    }
    
    .chart-card,
    .recent-requests-card {
        grid-column: span 1;
    }
    
    .metric-value {
        font-size: 2rem;
    }
    
    .request-header {
        flex-direction: column;
        align-items: flex-start;
        gap: 5px;
    }
    
    .request-details {
        flex-direction: column;
    }
}

/* Dark mode support */
@media (prefers-color-scheme: dark) {
    body {
        background: linear-gradient(135deg, #1a202c 0%, #2d3748 100%);
        color: #e2e8f0;
    }
    
    .card,
    .header {
        background: rgba(45, 55, 72, 0.95);
        border: 1px solid rgba(255, 255, 255, 0.1);
    }
    
    .header h1,
    .card h3,
    .query-type-name,
    .upstream-name,
    .request-query {
        color: #e2e8f0;
    }
    
    .query-type,
    .upstream-server,
    .request-item {
        background: rgba(74, 85, 104, 0.5);
    }
    
    .clients-table th {
        background: rgba(74, 85, 104, 0.5);
        color: #e2e8f0;
    }
    
    .clients-table tr:hover,
    .request-item:hover {
        background: rgba(74, 85, 104, 0.7);
    }
}`

// dashboardJS contains the JavaScript code for the dashboard
const dashboardJS = `
class DNSDashboard {
    constructor() {
        this.charts = {};
        this.updateInterval = null;
        this.init();
    }

    init() {
        this.setupCharts();
        this.loadData();
        this.startAutoUpdate();
    }

    setupCharts() {
        // Hourly chart
        const hourlyCtx = document.getElementById('hourlyChart').getContext('2d');
        this.charts.hourly = new Chart(hourlyCtx, {
            type: 'line',
            data: {
                labels: [],
                datasets: [{
                    label: 'Requests per Minute',
                    data: [],
                    borderColor: '#4299e1',
                    backgroundColor: 'rgba(66, 153, 225, 0.1)',
                    fill: true,
                    tension: 0.4
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            precision: 0
                        }
                    },
                    x: {
                        ticks: {
                            maxTicksLimit: 10
                        }
                    }
                },
                plugins: {
                    legend: {
                        display: false
                    }
                }
            }
        });

        // Daily chart
        const dailyCtx = document.getElementById('dailyChart').getContext('2d');
        this.charts.daily = new Chart(dailyCtx, {
            type: 'bar',
            data: {
                labels: [],
                datasets: [{
                    label: 'Requests per Hour',
                    data: [],
                    backgroundColor: '#667eea',
                    borderColor: '#5a67d8',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            precision: 0
                        }
                    },
                    x: {
                        ticks: {
                            maxTicksLimit: 12
                        }
                    }
                },
                plugins: {
                    legend: {
                        display: false
                    }
                }
            }
        });
    }

    async loadData() {
        try {
            console.log('Fetching metrics from:', API_BASE + '/api/metrics');
            const response = await fetch(API_BASE + '/api/metrics');
            if (!response.ok) {
                throw new Error('Failed to fetch metrics: ' + response.status);
            }
            
            const data = await response.json();
            console.log('Received data:', data);
            this.updateDashboard(data);
            this.updateStatus('online');
            
        } catch (error) {
            console.error('Error loading data:', error);
            this.updateStatus('offline');
            this.showError('Failed to load metrics data: ' + error.message);
        }
    }

    updateDashboard(data) {
        this.updateOverview(data.overview);
        this.updateCharts(data.time_series);
        this.updateQueryTypes(data.query_types);
        this.updateTopClients(data.top_clients);
        this.updateUpstreamServers(data.upstream_servers);
        this.updateRecentRequests(data.recent_requests);
        this.updateLastUpdated();
    }

    updateOverview(overview) {
        console.log('Updating overview with data:', overview);
        
        const totalRequestsEl = document.getElementById('totalRequests');
        const requestsPerSecondEl = document.getElementById('requestsPerSecond');
        const cacheHitRateEl = document.getElementById('cacheHitRate');
        const successRateEl = document.getElementById('successRate');
        const avgResponseTimeEl = document.getElementById('avgResponseTime');
        const clientsEl = document.getElementById('clients');
        const uptimeEl = document.getElementById('uptime');
        
        if (totalRequestsEl) totalRequestsEl.textContent = this.formatNumber(overview.total_requests || 0);
        if (requestsPerSecondEl) requestsPerSecondEl.textContent = (overview.requests_per_second || 0).toFixed(2) + ' req/sec';
        if (cacheHitRateEl) cacheHitRateEl.textContent = (overview.cache_hit_rate || 0).toFixed(1) + '%';
        if (successRateEl) successRateEl.textContent = (overview.success_rate || 0).toFixed(1) + '%';
        if (avgResponseTimeEl) avgResponseTimeEl.textContent = (overview.average_response_time_ms || 0).toFixed(1) + ' ms';
        if (clientsEl) clientsEl.textContent = overview.clients || 0;
        if (uptimeEl) uptimeEl.textContent = overview.uptime || '-';
    }

    updateCharts(timeSeriesData) {
        // Update hourly chart
        const hourlyData = timeSeriesData.requests_last_hour || [];
        this.charts.hourly.data.labels = hourlyData.map(point => {
            const date = new Date(point.timestamp);
            return date.getHours().toString().padStart(2, '0') + ':' + 
                   date.getMinutes().toString().padStart(2, '0');
        });
        this.charts.hourly.data.datasets[0].data = hourlyData.map(point => point.value);
        this.charts.hourly.update('none');

        // Update daily chart
        const dailyData = timeSeriesData.requests_last_day || [];
        this.charts.daily.data.labels = dailyData.map(point => {
            const date = new Date(point.timestamp);
            return (date.getMonth() + 1) + '/' + date.getDate() + ' ' +
                   date.getHours().toString().padStart(2, '0') + ':00';
        });
        this.charts.daily.data.datasets[0].data = dailyData.map(point => point.value);
        this.charts.daily.update('none');
    }

    updateQueryTypes(queryTypes) {
        const container = document.getElementById('queryTypes');
        
        if (!queryTypes || Object.keys(queryTypes).length === 0) {
            container.innerHTML = '<div class="loading">No query data available</div>';
            return;
        }

        const html = Object.entries(queryTypes)
            .sort(([,a], [,b]) => b - a)
            .map(([type, count]) => 
                '<div class="query-type">' +
                '<div class="query-type-name">' + type + '</div>' +
                '<div class="query-type-count">' + this.formatNumber(count) + '</div>' +
                '</div>'
            ).join('');

        container.innerHTML = html;
    }

    updateTopClients(clients) {
        const container = document.getElementById('topClients');
        
        if (!clients || clients.length === 0) {
            container.innerHTML = '<div class="loading">No client data available</div>';
            return;
        }

        const html = '<table>' +
            '<thead>' +
                '<tr>' +
                    '<th>Client IP</th>' +
                    '<th>Requests</th>' +
                    '<th>Cache Hit Rate</th>' +
                    '<th>Success Rate</th>' +
                    '<th>Last Seen</th>' +
                '</tr>' +
            '</thead>' +
            '<tbody>' +
                clients.map(client => 
                    '<tr>' +
                        '<td>' + client.ip + '</td>' +
                        '<td>' + this.formatNumber(client.requests) + '</td>' +
                        '<td>' + client.cache_hit_rate.toFixed(1) + '%</td>' +
                        '<td>' + client.success_rate.toFixed(1) + '%</td>' +
                        '<td>' + this.formatTime(client.last_seen) + '</td>' +
                    '</tr>'
                ).join('') +
            '</tbody>' +
        '</table>';

        container.innerHTML = html;
    }

    updateUpstreamServers(servers) {
        const container = document.getElementById('upstreamServers');
        
        if (!servers || Object.keys(servers).length === 0) {
            container.innerHTML = '<div class="loading">No upstream server data available</div>';
            return;
        }

        const html = Object.entries(servers).map(([server, stats]) => {
            const successRate = stats.total_queries > 0 ? 
                (stats.successful_queries / stats.total_queries * 100).toFixed(1) : '0.0';
            const isHealthy = stats.successful_queries > stats.failed_queries;
            
            return '<div class="upstream-server ' + (isHealthy ? '' : 'failed') + '">' +
                '<div class="upstream-name">' + server + '</div>' +
                '<div class="upstream-stats">' +
                    '<div>Total: ' + this.formatNumber(stats.total_queries) + '</div>' +
                    '<div>Success: ' + successRate + '%</div>' +
                    '<div>Avg RTT: ' + stats.average_rtt.toFixed(1) + 'ms</div>' +
                    '<div>Last Used: ' + this.formatTime(stats.last_used) + '</div>' +
                '</div>' +
            '</div>';
        }).join('');

        container.innerHTML = html;
    }

    updateRecentRequests(requests) {
        const container = document.getElementById('recentRequests');
        
        if (!requests || requests.length === 0) {
            container.innerHTML = '<div class="loading">No recent requests</div>';
            return;
        }

        const html = requests.slice(0, 20).map(request => {
            const statusClass = this.getStatusClass(request.status);
            const statusText = this.getStatusText(request.status);
            
            return '<div class="request-item ' + statusClass + '">' +
                '<div class="request-header">' +
                    '<div class="request-query">' + request.request.query + ' (' + request.request.type + ')</div>' +
                    '<div class="request-status ' + statusClass + '">' + statusText + '</div>' +
                '</div>' +
                '<div class="request-details">' +
                    '<span>Client: ' + request.request.client + '</span>' +
                    '<span>Duration: ' + request.total_duration_ms.toFixed(1) + 'ms</span>' +
                    '<span>Time: ' + this.formatTime(request.timestamp) + '</span>' +
                '</div>' +
            '</div>';
        }).join('');

        container.innerHTML = html;
    }

    updateStatus(status) {
        const statusElement = document.getElementById('status');
        statusElement.className = 'status ' + status;
        statusElement.style.color = status === 'online' ? '#48bb78' : '#f56565';
    }

    updateLastUpdated() {
        document.getElementById('lastUpdated').textContent = 
            'Last updated: ' + new Date().toLocaleTimeString();
    }

    startAutoUpdate() {
        // Update every 5 seconds
        this.updateInterval = setInterval(() => {
            this.loadData();
        }, 5000);
    }

    stopAutoUpdate() {
        if (this.updateInterval) {
            clearInterval(this.updateInterval);
            this.updateInterval = null;
        }
    }

    // Utility methods
    formatNumber(num) {
        if (num >= 1000000) {
            return (num / 1000000).toFixed(1) + 'M';
        } else if (num >= 1000) {
            return (num / 1000).toFixed(1) + 'K';
        }
        return num.toString();
    }

    formatTime(timestamp) {
        return new Date(timestamp).toLocaleTimeString();
    }

    getStatusClass(status) {
        switch (status) {
            case 'success': return 'success';
            case 'cache_hit': return 'cache-hit';
            case 'all_upstreams_failed':
            case 'malformed_query': return 'failed';
            default: return '';
        }
    }

    getStatusText(status) {
        switch (status) {
            case 'success': return 'Success';
            case 'cache_hit': return 'Cache Hit';
            case 'all_upstreams_failed': return 'Failed';
            case 'malformed_query': return 'Malformed';
            default: return status;
        }
    }

    showError(message) {
        const containers = ['queryTypes', 'topClients', 'upstreamServers', 'recentRequests'];
        containers.forEach(id => {
            const element = document.getElementById(id);
            if (element) {
                element.innerHTML = '<div class="error">' + message + '</div>';
            }
        });
    }
}

// Initialize dashboard when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    new DNSDashboard();
});

// Handle page visibility changes to pause/resume updates
document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
        // Page is hidden, could pause updates
    } else {
        // Page is visible, ensure updates are running
    }
});`
