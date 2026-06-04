// TrafficLab Proxy Control Platform - Common JavaScript

// Format bytes to human-readable string
function formatBytes(bytes) {
    if (bytes === undefined || bytes === null) return '0 B';
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Auto-refresh node status on dashboard (every 5 seconds)
if (document.querySelector('#node-status-list')) {
    setInterval(async () => {
        try {
            const resp = await fetch('/api/v1/nodes');
            const data = await resp.json();
            if (data.success) {
                const list = document.querySelector('#node-status-list');
                if (data.data.length === 0) {
                    list.innerHTML = '<p class="text-muted">No nodes registered yet.</p>';
                    return;
                }
                list.innerHTML = data.data.map(n => `
                    <div class="node-item">
                        <span class="status-badge status-${n.status}">${n.status}</span>
                        <strong>${n.node_id}</strong> (${n.name || ''})
                        <span class="text-muted">${n.ip}:${n.proxy_port}</span>
                    </div>
                `).join('');
            }
        } catch (e) {
            // Silently ignore refresh errors
        }
    }, 5000);
}

// Auto-refresh nodes page
if (document.querySelector('#nodes-table')) {
    setInterval(() => {
        if (document.querySelector('#nodes-table')) location.reload();
    }, 30000);
}

// ESC to close modals
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        document.querySelectorAll('.modal').forEach(m => m.classList.add('hidden'));
    }
});
