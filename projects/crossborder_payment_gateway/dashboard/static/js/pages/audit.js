/* ===== Audit Log Page ===== */
var _auditState = {
    page: 1,
    pageSize: 30,
    total: 0,
    action: '',
    dateRange: 'week',
    userSearch: '',
    autoRefresh: false,
    refreshTimer: null
};

function init_audit() {
    var s = _auditState;
    s.page = 1;

    setupAuditFilters();
    loadAuditLogs();

    // Auto-refresh toggle
    var toggleEl = document.getElementById('auditAutoRefresh');
    if (toggleEl) {
        toggleEl.onchange = function () {
            s.autoRefresh = this.checked;
            if (s.autoRefresh) {
                s.refreshTimer = setInterval(function () {
                    loadAuditLogs();
                }, 10000);
                Toast.info('自动刷新已开启 (10秒)');
            } else {
                if (s.refreshTimer) {
                    clearInterval(s.refreshTimer);
                    s.refreshTimer = null;
                }
            }
        };
    }

    App.setCleanup(function () {
        if (s.refreshTimer) {
            clearInterval(s.refreshTimer);
            s.refreshTimer = null;
        }
    });
}

function setupAuditFilters() {
    var actionEl = document.getElementById('auditActionFilter');
    var rangeEl = document.getElementById('auditDateRange');
    var searchEl = document.getElementById('auditUserSearch');

    if (actionEl) {
        actionEl.onchange = function () {
            _auditState.action = this.value;
            _auditState.page = 1;
            loadAuditLogs();
        };
    }

    if (rangeEl) {
        rangeEl.onchange = function () {
            _auditState.dateRange = this.value;
            _auditState.page = 1;
            loadAuditLogs();
        };
    }

    if (searchEl) {
        var timer = null;
        searchEl.oninput = function () {
            if (timer) clearTimeout(timer);
            timer = setTimeout(function () {
                _auditState.userSearch = searchEl.value.trim();
                _auditState.page = 1;
                loadAuditLogs();
            }, 400);
        };
    }
}

async function loadAuditLogs() {
    var container = document.getElementById('auditTable');
    if (!container) return;

    container.innerHTML = '<div class="table-placeholder">加载中...</div>';

    try {
        var s = _auditState;
        var params = { page: s.page, size: s.pageSize };

        if (s.action) params.action = s.action;
        if (s.userSearch) params.username = s.userSearch;

        var now = new Date();
        var from = new Date(now);
        switch (s.dateRange) {
            case 'today':
                from.setHours(0, 0, 0, 0);
                break;
            case 'week':
                from.setDate(from.getDate() - 7);
                break;
            case 'month':
                from.setMonth(from.getMonth() - 1);
                break;
        }
        if (s.dateRange !== 'all') {
            params.from = from.toISOString();
            params.to = now.toISOString();
        }

        var query = Object.keys(params).map(function (k) {
            return encodeURIComponent(k) + '=' + encodeURIComponent(params[k]);
        }).join('&');

        var data = await API.get('/api/v1/audit?' + query);
        var logs = data.data || data.logs || data.audit_logs || [];
        s.total = data.total || data.count || logs.length;

        renderAuditTable(logs);
        renderAuditPagination();
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><p>加载失败: ' + escapeHtml(e.message || '未知错误') + '</p></div>';
    }
}

function renderAuditTable(logs) {
    var container = document.getElementById('auditTable');
    if (!container) return;

    if (!logs || logs.length === 0) {
        container.innerHTML = '<div class="empty-state"><div class="empty-icon">&#128214;</div><h3>暂无审计日志</h3></div>';
        return;
    }

    var html = '<table class="data-table">';
    html += '<thead><tr>' +
        '<th>时间</th>' +
        '<th>用户</th>' +
        '<th>操作</th>' +
        '<th>资源</th>' +
        '<th>资源ID</th>' +
        '<th>IP地址</th>' +
        '<th>详情</th>' +
        '</tr></thead><tbody>';

    for (var i = 0; i < logs.length; i++) {
        var l = logs[i];
        var time = formatTime(l.created_at || l.timestamp || l.time);
        var username = l.username || l.user || l.user_name || '-';
        var action = l.action || '-';
        var resource = l.resource_type || l.resource || l.resourceType || '-';
        var resourceId = l.resource_id || l.resourceId || '-';
        var ip = l.ip || l.ip_address || l.client_ip || '-';
        var detail = l.detail || l.message || l.description || '-';
        var shortDetail = detail.length > 60 ? detail.substring(0, 60) + '...' : detail;

        html += '<tr>' +
            '<td class="time-cell">' + time + '</td>' +
            '<td>' + escapeHtml(username) + '</td>' +
            '<td><code style="font-size:11px">' + escapeHtml(action) + '</code></td>' +
            '<td>' + escapeHtml(resource) + '</td>' +
            '<td class="txn-id">' + escapeHtml(truncateMiddle(resourceId, 12)) + '</td>' +
            '<td style="font-family:var(--font-mono);font-size:11px;color:var(--color-text-tertiary)">' + escapeHtml(ip) + '</td>' +
            '<td style="max-width:200px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-size:12px">' + escapeHtml(shortDetail) + '</td>' +
            '</tr>';
    }

    html += '</tbody></table>';
    container.innerHTML = html;
}

function renderAuditPagination() {
    var container = document.getElementById('auditPagination');
    if (!container) return;

    var s = _auditState;
    var totalPages = Math.ceil(s.total / s.pageSize) || 1;
    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }

    var html = '';
    html += '<button ' + (s.page <= 1 ? 'disabled' : '') + ' onclick="goAuditPage(' + (s.page - 1) + ')">' +
        '<svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M9 3L5 7L9 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>' +
        '</button>';

    var start = Math.max(1, s.page - 2);
    var end = Math.min(totalPages, s.page + 2);

    if (start > 1) {
        html += '<button onclick="goAuditPage(1)">1</button>';
        if (start > 2) html += '<span class="page-info">...</span>';
    }
    for (var i = start; i <= end; i++) {
        html += '<button class="' + (i === s.page ? 'active' : '') + '" onclick="goAuditPage(' + i + ')">' + i + '</button>';
    }
    if (end < totalPages) {
        if (end < totalPages - 1) html += '<span class="page-info">...</span>';
        html += '<button onclick="goAuditPage(' + totalPages + ')">' + totalPages + '</button>';
    }

    html += '<button ' + (s.page >= totalPages ? 'disabled' : '') + ' onclick="goAuditPage(' + (s.page + 1) + ')">' +
        '<svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M5 3L9 7L5 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>' +
        '</button>';

    html += '<span class="page-info">共 ' + s.total + ' 条</span>';

    container.innerHTML = html;
}

function goAuditPage(page) {
    _auditState.page = page;
    loadAuditLogs();
}
