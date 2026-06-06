/* ===== Transactions Page ===== */
var _txnState = {
    page: 1,
    pageSize: 20,
    total: 0,
    status: '',
    dateRange: 'week',
    search: '',
    dateFrom: '',
    dateTo: '',
    wsUnsub: null
};

function init_transactions() {
    var s = _txnState;
    s.page = 1;

    // Set up filter handlers
    setupTxnFilters();

    // Load data
    loadTransactions();

    // WS listener for new transactions
    if (s.wsUnsub) s.wsUnsub();
    s.wsUnsub = WS.on('transaction_update', function (data) {
        var txn = data.transaction || data;
        // If current filter matches, update table
        if (txn.status && s.status && txn.status !== s.status) return;
        // Reload current page to show update
        loadTransactions();
        // Show toast for completed/failed
        if (txn.status === 'completed') {
            Toast.info('新交易已完成: ' + truncateMiddle(txn.id || '', 12));
        } else if (txn.status === 'failed') {
            Toast.error('交易失败: ' + truncateMiddle(txn.id || '', 12));
        }
    });

    // Cleanup
    App.setCleanup(function () {
        if (s.wsUnsub) { s.wsUnsub(); s.wsUnsub = null; }
    });
}

function setupTxnFilters() {
    var statusEl = document.getElementById('txnStatusFilter');
    var rangeEl = document.getElementById('txnDateRange');
    var searchEl = document.getElementById('txnSearch');
    var dateFromEl = document.getElementById('txnDateFrom');
    var dateToEl = document.getElementById('txnDateTo');
    var customGroup = document.getElementById('customDateGroup');

    if (statusEl) {
        statusEl.onchange = function () {
            _txnState.status = this.value;
            _txnState.page = 1;
            loadTransactions();
        };
    }

    if (rangeEl) {
        rangeEl.onchange = function () {
            _txnState.dateRange = this.value;
            _txnState.page = 1;
            if (this.value === 'custom') {
                if (customGroup) customGroup.style.display = 'flex';
                // Set default dates
                var today = new Date();
                if (dateFromEl) {
                    var from = new Date(today);
                    from.setDate(from.getDate() - 7);
                    dateFromEl.value = from.toISOString().split('T')[0];
                }
                if (dateToEl) dateToEl.value = today.toISOString().split('T')[0];
            } else {
                if (customGroup) customGroup.style.display = 'none';
            }
            loadTransactions();
        };
    }

    if (searchEl) {
        var searchTimer = null;
        searchEl.oninput = function () {
            if (searchTimer) clearTimeout(searchTimer);
            searchTimer = setTimeout(function () {
                _txnState.search = searchEl.value.trim();
                _txnState.page = 1;
                loadTransactions();
            }, 400);
        };
    }

    if (dateFromEl) {
        dateFromEl.onchange = function () {
            _txnState.dateFrom = this.value;
            _txnState.page = 1;
            loadTransactions();
        };
    }

    if (dateToEl) {
        dateToEl.onchange = function () {
            _txnState.dateTo = this.value;
            _txnState.page = 1;
            loadTransactions();
        };
    }
}

function getDateRangeParams(range) {
    var now = new Date();
    var from = new Date(now);
    switch (range) {
        case 'today':
            from.setHours(0, 0, 0, 0);
            break;
        case 'week':
            from.setDate(from.getDate() - 7);
            break;
        case 'month':
            from.setMonth(from.getMonth() - 1);
            break;
        default:
            return {};
    }
    return { from: from.toISOString(), to: now.toISOString() };
}

async function loadTransactions() {
    var container = document.getElementById('transactionsTable');
    if (!container) return;

    var s = _txnState;
    container.innerHTML = '<div class="table-placeholder">加载中...</div>';

    try {
        var params = { page: s.page, size: s.pageSize };

        if (s.status) params.status = s.status;
        if (s.search) params.search = s.search;

        if (s.dateRange === 'custom') {
            if (s.dateFrom) params.from = s.dateFrom;
            if (s.dateTo) params.to = s.dateTo;
        } else if (s.dateRange) {
            var dp = getDateRangeParams(s.dateRange);
            if (dp.from) params.from = dp.from;
            if (dp.to) params.to = dp.to;
        }

        var query = Object.keys(params).map(function (k) {
            return encodeURIComponent(k) + '=' + encodeURIComponent(params[k]);
        }).join('&');

        var data = await API.get('/api/v1/transactions?' + query);
        var txns = data.data || data.transactions || [];
        s.total = data.total || data.count || data.total_count || txns.length;

        renderTransactionsTable(txns);

        // Pagination
        renderTxnPagination();
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><p>加载失败: ' + escapeHtml(e.message || '未知错误') + '</p></div>';
    }
}

function renderTransactionsTable(txns) {
    var container = document.getElementById('transactionsTable');
    if (!container) return;

    if (!txns || txns.length === 0) {
        container.innerHTML = '<div class="empty-state"><div class="empty-icon">&#128203;</div><h3>暂无交易记录</h3><p>试试调整筛选条件</p></div>';
        return;
    }

    var html = '<table class="data-table">';
    html += '<thead><tr>' +
        '<th>交易ID</th>' +
        '<th>时间</th>' +
        '<th>商户</th>' +
        '<th>来源金额</th>' +
        '<th>目标金额</th>' +
        '<th>手续费</th>' +
        '<th>状态</th>' +
        '<th>操作</th>' +
        '</tr></thead><tbody>';

    for (var i = 0; i < txns.length; i++) {
        var t = txns[i];
        var txnId = truncateMiddle(t.id || t.txn_id || t.transaction_id, 16);
        var time = formatTime(t.created_at || t.time || t.timestamp);
        var merchant = t.merchant_name || t.merchant || '-';
        var srcAmt = formatCurrency(t.source_amount || t.amount, t.source_currency || t.currency);
        var tgtAmt = formatCurrency(t.target_amount, t.target_currency);
        var fee = t.fee ? formatCurrency(t.fee, t.fee_currency || t.source_currency) : '-';
        var status = t.status || 'pending';
        var statusLabel = getStatusLabel(status);

        html += '<tr onclick="App.navigate(\'transaction-detail\', {id: \'' + escapeHtml(String(t.id || t.txn_id)) + '\'})" style="cursor:pointer">' +
            '<td class="txn-id">' + txnId + '</td>' +
            '<td class="time-cell">' + formatTimeShort(t.created_at || t.time || t.timestamp) + '</td>' +
            '<td>' + escapeHtml(merchant) + '</td>' +
            '<td class="amount-cell">' + srcAmt + '</td>' +
            '<td class="amount-cell">' + tgtAmt + '</td>' +
            '<td>' + fee + '</td>' +
            '<td><span class="badge badge-' + status + '">' + statusLabel + '</span></td>' +
            '<td class="actions-cell">' +
            '<button class="btn btn-xs btn-ghost" onclick="event.stopPropagation();App.navigate(\'transaction-detail\', {id:\'' + escapeHtml(String(t.id || t.txn_id)) + '\'})">详情</button>' +
            (status === 'completed' ? '<button class="btn btn-xs btn-outline" onclick="event.stopPropagation();showRefundConfirm(\'' + escapeHtml(String(t.id || t.txn_id)) + '\')">退款</button>' : '') +
            '</td></tr>';
    }

    html += '</tbody></table>';
    container.innerHTML = html;
}

function renderTxnPagination() {
    var container = document.getElementById('txnPagination');
    if (!container) return;

    var s = _txnState;
    var totalPages = Math.ceil(s.total / s.pageSize) || 1;

    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }

    var html = '';
    html += '<button ' + (s.page <= 1 ? 'disabled' : '') + ' onclick="goTxnPage(' + (s.page - 1) + ')">' +
        '<svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M9 3L5 7L9 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>' +
        '</button>';

    var start = Math.max(1, s.page - 2);
    var end = Math.min(totalPages, s.page + 2);

    if (start > 1) {
        html += '<button onclick="goTxnPage(1)">1</button>';
        if (start > 2) html += '<span class="page-info">...</span>';
    }
    for (var i = start; i <= end; i++) {
        html += '<button class="' + (i === s.page ? 'active' : '') + '" onclick="goTxnPage(' + i + ')">' + i + '</button>';
    }
    if (end < totalPages) {
        if (end < totalPages - 1) html += '<span class="page-info">...</span>';
        html += '<button onclick="goTxnPage(' + totalPages + ')">' + totalPages + '</button>';
    }

    html += '<button ' + (s.page >= totalPages ? 'disabled' : '') + ' onclick="goTxnPage(' + (s.page + 1) + ')">' +
        '<svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M5 3L9 7L5 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>' +
        '</button>';

    html += '<span class="page-info">共 ' + s.total + ' 条</span>';

    container.innerHTML = html;
}

function goTxnPage(page) {
    _txnState.page = page;
    loadTransactions();
    // Scroll to top of table
    var el = document.getElementById('transactionsTable');
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

function showRefundConfirm(txnId) {
    showModal('确认退款', '<p>确定要对交易 <code>' + escapeHtml(txnId) + '</code> 执行退款操作吗？</p>', {
        confirmText: '确认退款',
        confirmClass: 'btn btn-danger',
        onConfirm: async function () {
            try {
                await API.post('/api/v1/transactions/' + encodeURIComponent(txnId) + '/refund');
                Toast.success('退款请求已提交');
                loadTransactions();
            } catch (e) {
                Toast.error('退款失败: ' + (e.message || '未知错误'));
                throw e;
            }
        },
        onCancel: function () { }
    });
}
