/* ===== Reusable Data Table Renderer ===== */
function renderTable(containerId, columns, data, options) {
    const container = document.getElementById(containerId);
    if (!container) return;

    options = options || {};
    const pageSize = options.pageSize || 20;
    const currentPage = options.page || 1;
    const totalItems = options.total || (data ? data.length : 0);
    const totalPages = Math.ceil(totalItems / pageSize) || 1;
    const clickable = options.clickable !== false;
    const sortable = options.sortable || false;

    if (!data || data.length === 0) {
        container.innerHTML = '<div class="empty-state"><div class="empty-icon">&#128196;</div><h3>暂无数据</h3><p>' + (options.emptyText || '没有找到匹配的记录') + '</p></div>';
        return;
    }

    var html = '<table class="data-table">';
    // Header
    html += '<thead><tr>';
    for (var i = 0; i < columns.length; i++) {
        var col = columns[i];
        var sortClass = '';
        var sortAttr = '';
        if (sortable && col.sortable !== false) {
            sortClass = ' sortable';
            sortAttr = ' data-sort-key="' + (col.key || '') + '"';
        }
        html += '<th class="' + sortClass + '"' + sortAttr + ' style="' + (col.width ? 'width:' + col.width : '') + '">' + (col.label || '') + '</th>';
    }
    html += '</tr></thead>';

    // Body
    html += '<tbody>';
    var startIdx = (currentPage - 1) * pageSize;
    var endIdx = Math.min(startIdx + pageSize, data.length);
    if (options.paginate === false) {
        startIdx = 0;
        endIdx = data.length;
    }

    for (var i = startIdx; i < endIdx; i++) {
        var row = data[i];
        var rowId = row.id || row.ID || i;
        html += '<tr data-id="' + escapeHtml(String(rowId)) + '"' + (clickable ? ' class="clickable-row"' : '') + '>';
        for (var j = 0; j < columns.length; j++) {
            var col = columns[j];
            var val = resolveNestedValue(row, col.key);
            if (col.render && typeof col.render === 'function') {
                html += '<td>' + col.render(val, row) + '</td>';
            } else {
                html += '<td>' + (val !== undefined && val !== null ? escapeHtml(String(val)) : '-') + '</td>';
            }
        }
        html += '</tr>';
    }
    html += '</tbody></table>';

    container.innerHTML = html;

    // Click handler
    if (clickable && options.onRowClick) {
        container.querySelectorAll('.clickable-row').forEach(function (row) {
            row.addEventListener('click', function () {
                var id = this.getAttribute('data-id');
                var item = data.find(function (d) { return String(d.id || d.ID) === id; }) || data[parseInt(id)];
                options.onRowClick(item, id, this);
            });
        });
    }

    // Pagination
    if (options.showPagination !== false) {
        renderPaginationControls(container, currentPage, totalPages, totalItems, options.onPageChange);
    }
}

function renderPaginationControls(container, currentPage, totalPages, totalItems, onPageChange) {
    var paginationEl = container.querySelector('.pagination');
    if (!paginationEl) {
        paginationEl = document.createElement('div');
        paginationEl.className = 'pagination';
        container.appendChild(paginationEl);
    }

    if (totalPages <= 1) {
        paginationEl.innerHTML = '';
        return;
    }

    var html = '';

    // Prev button
    html += '<button ' + (currentPage <= 1 ? 'disabled' : '') + ' data-page="' + (currentPage - 1) + '">' +
        '<svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M9 3L5 7L9 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>' +
        '</button>';

    // Page numbers
    var start = Math.max(1, currentPage - 2);
    var end = Math.min(totalPages, currentPage + 2);
    if (start > 1) {
        html += '<button data-page="1">1</button>';
        if (start > 2) html += '<span class="page-info">...</span>';
    }
    for (var i = start; i <= end; i++) {
        html += '<button class="' + (i === currentPage ? 'active' : '') + '" data-page="' + i + '">' + i + '</button>';
    }
    if (end < totalPages) {
        if (end < totalPages - 1) html += '<span class="page-info">...</span>';
        html += '<button data-page="' + totalPages + '">' + totalPages + '</button>';
    }

    // Next button
    html += '<button ' + (currentPage >= totalPages ? 'disabled' : '') + ' data-page="' + (currentPage + 1) + '">' +
        '<svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M5 3L9 7L5 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>' +
        '</button>';

    // Total info
    html += '<span class="page-info">共 ' + totalItems + ' 条</span>';

    paginationEl.innerHTML = html;

    // Bind click events
    paginationEl.querySelectorAll('button[data-page]').forEach(function (btn) {
        btn.addEventListener('click', function () {
            if (this.disabled) return;
            var page = parseInt(this.getAttribute('data-page'));
            if (typeof onPageChange === 'function') onPageChange(page);
        });
    });
}

/* ===== Utility Functions ===== */
function escapeHtml(str) {
    if (typeof str !== 'string') return String(str || '');
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(str));
    return div.innerHTML;
}

function resolveNestedValue(obj, key) {
    if (!obj || !key) return undefined;
    var keys = key.split('.');
    var val = obj;
    for (var i = 0; i < keys.length; i++) {
        if (val === undefined || val === null) return undefined;
        val = val[keys[i]];
    }
    return val;
}

/* ===== Date/Time Formatting ===== */
function formatTime(t) {
    if (!t) return '-';
    var d = new Date(t);
    if (isNaN(d.getTime())) return String(t);
    return d.toLocaleString('zh-CN', {
        year: 'numeric', month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit', second: '2-digit'
    });
}

function formatTimeShort(t) {
    if (!t) return '-';
    var d = new Date(t);
    if (isNaN(d.getTime())) return String(t);
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function formatCurrency(amount, currency) {
    currency = currency || 'CNY';
    var symbolMap = { CNY: '¥', USD: '$', EUR: '€', GBP: '£', JPY: '¥', HKD: 'HK$', SGD: 'S$' };
    var symbol = symbolMap[currency] || currency + ' ';
    var val = parseFloat(amount) || 0;
    if (currency === 'JPY') {
        return symbol + Math.round(val).toLocaleString();
    }
    return symbol + val.toFixed(2).toLocaleString();
}

function formatPercent(val) {
    return (parseFloat(val) || 0).toFixed(1) + '%';
}

function formatNumber(val) {
    return (parseFloat(val) || 0).toLocaleString();
}

function truncateMiddle(str, maxLen) {
    if (!str) return '-';
    if (str.length <= maxLen) return str;
    var half = Math.floor((maxLen - 3) / 2);
    return str.substring(0, half) + '...' + str.substring(str.length - half);
}
