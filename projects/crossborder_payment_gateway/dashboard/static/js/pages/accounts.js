/* ===== Accounts Page ===== */
var _acctState = {
    merchantFilter: '',
    currencyFilter: ''
};

function init_accounts() {
    var s = _acctState;

    // Load merchants for filter
    loadMerchantFilter();

    // Setup filter handlers
    setupAcctFilters();

    // Load accounts
    loadAccounts();

    App.setCleanup(cleanup_accounts);
}

function cleanup_accounts() {
    // No timers to clean up for accounts
}

function setupAcctFilters() {
    var merchantEl = document.getElementById('acctMerchantFilter');
    var currencyEl = document.getElementById('acctCurrencyFilter');

    if (merchantEl) {
        merchantEl.onchange = function () {
            _acctState.merchantFilter = this.value;
            loadAccounts();
        };
    }

    if (currencyEl) {
        currencyEl.onchange = function () {
            _acctState.currencyFilter = this.value;
            loadAccounts();
        };
    }
}

async function loadMerchantFilter() {
    try {
        var data = await API.get('/api/v1/merchants?size=100');
        var merchants = data.data || data.merchants || [];
        var el = document.getElementById('acctMerchantFilter');
        if (!el) return;
        var html = '<option value="">全部商户</option>';
        for (var i = 0; i < merchants.length; i++) {
            html += '<option value="' + escapeHtml(String(merchants[i].id)) + '">' + escapeHtml(merchants[i].name || merchants[i].id) + '</option>';
        }
        el.innerHTML = html;
    } catch (e) { }
}

async function loadAccounts() {
    var container = document.getElementById('accountGrid');
    if (!container) return;

    container.innerHTML = '<div class="table-placeholder">加载中...</div>';

    try {
        var params = {};
        if (_acctState.merchantFilter) params.merchant_id = _acctState.merchantFilter;
        if (_acctState.currencyFilter) params.currency = _acctState.currencyFilter;

        var query = Object.keys(params).map(function (k) {
            return encodeURIComponent(k) + '=' + encodeURIComponent(params[k]);
        }).join('&');

        var data = await API.get('/api/v1/accounts' + (query ? '?' + query : ''));
        var accounts = data.data || data.accounts || [];
        renderAccountCards(accounts);
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><p>加载失败: ' + escapeHtml(e.message || '未知错误') + '</p></div>';
    }
}

function renderAccountCards(accounts) {
    var container = document.getElementById('accountGrid');
    if (!container) return;

    if (!accounts || accounts.length === 0) {
        container.innerHTML = '<div class="empty-state"><div class="empty-icon">&#128179;</div><h3>暂无账户</h3></div>';
        return;
    }

    var html = '';
    for (var i = 0; i < accounts.length; i++) {
        var a = accounts[i];
        var currency = a.currency || 'USD';
        var currencyClass = 'currency-' + currency.toLowerCase();
        var currencySymbol = getCurrencySymbol(currency);
        var balance = parseFloat(a.balance || a.available_balance || 0);
        var reserved = parseFloat(a.reserved || a.reserved_balance || 0);
        var totalBalance = balance + reserved;
        var dailyUsed = parseFloat(a.daily_usage || a.daily_used || 0);
        var dailyLimit = parseFloat(a.daily_limit || 0);
        var usagePercent = dailyLimit > 0 ? Math.min(100, (dailyUsed / dailyLimit) * 100) : 0;
        var status = a.status || 'active';

        html += '<div class="account-card" onclick="showAccountDetail(\'' + escapeHtml(String(a.id)) + '\')">';
        html += '<div class="account-card-header">';
        html += '<div class="account-currency">';
        html += '<div class="currency-icon ' + currencyClass + '">' + currencySymbol + '</div>';
        html += '<div><div class="currency-name">' + escapeHtml(currency) + ' 账户</div>';
        html += '<div class="currency-id">' + truncateMiddle(a.id || '', 16) + '</div></div></div>';
        html += '<span class="badge badge-' + status + '">' + (status === 'active' ? '活跃' : '冻结') + '</span>';
        html += '</div>';

        html += '<div class="account-balance-section">';
        html += '<div class="account-balance">' + formatCurrency(balance, currency) + '</div>';
        html += '<div class="account-balance-label">可用余额</div>';
        html += '<div class="balance-bar"><div class="balance-bar-fill" style="width:' + (totalBalance > 0 ? (balance / totalBalance * 100) : 0) + '%"></div></div>';
        html += '</div>';

        html += '<div class="account-details">';
        html += '<div class="account-detail-item"><span class="account-detail-label">总余额</span><span class="account-detail-value">' + formatCurrency(totalBalance, currency) + '</span></div>';
        html += '<div class="account-detail-item"><span class="account-detail-label">冻结金额</span><span class="account-detail-value">' + formatCurrency(reserved, currency) + '</span></div>';
        if (dailyLimit > 0) {
            html += '<div class="account-detail-item" style="grid-column:span 2">';
            html += '<span class="account-detail-label">日使用量 (' + formatCurrency(dailyUsed, currency) + ' / ' + formatCurrency(dailyLimit, currency) + ')</span>';
            html += '<div class="balance-bar"><div class="balance-bar-fill" style="width:' + usagePercent + '%;' + (usagePercent > 80 ? 'background:var(--gradient-orange)' : '') + (usagePercent > 95 ? 'background:var(--gradient-pink)' : '') + '"></div></div>';
            html += '</div>';
        }
        html += '</div>'; // account-details
        html += '</div>';
    }

    container.innerHTML = html;
}

function getCurrencySymbol(currency) {
    var map = {
        USD: '$', CNY: '¥', EUR: '€', JPY: '¥', GBP: '£',
        HKD: 'HK$', SGD: 'S$', KRW: '₩', THB: '฿', AUD: 'A$',
        CAD: 'C$', CHF: 'Fr', MXN: 'MX$', BRL: 'R$'
    };
    return map[currency] || currency.charAt(0);
}

async function showAccountDetail(accountId) {
    try {
        var data = await API.get('/api/v1/accounts/' + encodeURIComponent(accountId));
        var acct = data.account || data;
        var currency = acct.currency || 'USD';
        var balance = parseFloat(acct.balance || acct.available_balance || 0);
        var reserved = parseFloat(acct.reserved || acct.reserved_balance || 0);

        var html = '<div style="display:grid;gap:16px">';
        html += '<div><strong>账户ID:</strong> <code>' + escapeHtml(acct.id || '') + '</code></div>';
        html += '<div><strong>币种:</strong> ' + escapeHtml(currency) + '</div>';
        html += '<div><strong>可用余额:</strong> ' + formatCurrency(balance, currency) + '</div>';
        html += '<div><strong>冻结金额:</strong> ' + formatCurrency(reserved, currency) + '</div>';
        html += '<div><strong>总余额:</strong> ' + formatCurrency(balance + reserved, currency) + '</div>';
        html += '<div><strong>状态:</strong> <span class="badge badge-' + (acct.status || 'active') + '">' + (acct.status === 'active' ? '活跃' : '冻结') + '</span></div>';
        if (acct.merchant_name) {
            html += '<div><strong>商户:</strong> ' + escapeHtml(acct.merchant_name) + '</div>';
        }
        html += '</div>';

        showModal('账户详情 - ' + escapeHtml(currency), html, {
            showConfirm: false,
            cancelText: '关闭'
        });
    } catch (e) {
        Toast.error('加载账户详情失败');
    }
}
