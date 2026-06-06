/* ===== Settings Page ===== */
function init_settings() {
    loadExchangeRates();
    loadSystemInfo();
    App.setCleanup(cleanup_settings);
}

function cleanup_settings() {
    // No timers
}

/* ===== Exchange Rates ===== */
async function loadExchangeRates() {
    var container = document.getElementById('exchangeRatesTable');
    if (!container) return;

    container.innerHTML = '<div class="table-placeholder">加载中...</div>';

    try {
        var data = await API.get('/api/v1/exchange-rates');
        var rates = data.data || data.rates || data.exchange_rates || [];
        renderExchangeRates(rates);
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><p>加载失败: ' + escapeHtml(e.message || '未知错误') + '</p></div>';
    }
}

function renderExchangeRates(rates) {
    var container = document.getElementById('exchangeRatesTable');
    if (!container) return;

    if (!rates || rates.length === 0) {
        container.innerHTML = '<div class="empty-state"><div class="empty-icon">&#128176;</div><h3>暂无汇率数据</h3><p>点击"添加汇率"手动添加</p></div>';
        return;
    }

    var html = '';
    for (var i = 0; i < rates.length; i++) {
        var r = rates[i];
        var source = r.source_currency || r.from || r.base_currency || '-';
        var target = r.target_currency || r.to || r.quote_currency || '-';
        var rate = parseFloat(r.rate || r.exchange_rate || r.price || 0);
        var bid = parseFloat(r.bid || 0);
        var ask = parseFloat(r.ask || 0);
        var updated = formatTimeShort(r.updated_at || r.updated || r.updatedAt || r.timestamp);
        var src = r.source || r.provider || 'manual';

        html += '<div class="rate-row">' +
            '<div class="rate-pair">' +
            '<span>' + escapeHtml(source) + '</span>' +
            '<span class="rate-arrow">&#8594;</span>' +
            '<span>' + escapeHtml(target) + '</span>' +
            '</div>' +
            '<div style="text-align:right">' +
            '<div class="rate-value">' + rate.toFixed(6) + '</div>' +
            '<div class="rate-bid-ask">' +
            (bid ? '<span>买: ' + bid.toFixed(6) + '</span>' : '') +
            (ask ? '<span>卖: ' + ask.toFixed(6) + '</span>' : '') +
            '</div>' +
            '<div style="display:flex;align-items:center;gap:8px;margin-top:4px;justify-content:flex-end">' +
            '<span class="rate-source">' + escapeHtml(src) + '</span>' +
            '<span style="font-size:10px;color:var(--color-text-tertiary)">' + updated + '</span>' +
            '</div></div></div>';
    }

    container.innerHTML = html;
}

function showAddRate() {
    var html =
        '<div class="form-group"><label>来源币种</label>' +
        '<select class="form-select" id="rateSourceCurrency">' +
        '<option value="USD">USD - 美元</option>' +
        '<option value="CNY" selected>CNY - 人民币</option>' +
        '<option value="EUR">EUR - 欧元</option>' +
        '<option value="JPY">JPY - 日元</option>' +
        '<option value="GBP">GBP - 英镑</option>' +
        '<option value="HKD">HKD - 港币</option>' +
        '<option value="SGD">SGD - 新加坡元</option>' +
        '</select></div>' +
        '<div class="form-group"><label>目标币种</label>' +
        '<select class="form-select" id="rateTargetCurrency">' +
        '<option value="USD">USD - 美元</option>' +
        '<option value="CNY">CNY - 人民币</option>' +
        '<option value="EUR">EUR - 欧元</option>' +
        '<option value="JPY">JPY - 日元</option>' +
        '<option value="GBP">GBP - 英镑</option>' +
        '<option value="HKD">HKD - 港币</option>' +
        '<option value="SGD">SGD - 新加坡元</option>' +
        '</select></div>' +
        '<div class="form-group"><label>汇率</label><input type="number" class="form-input" id="rateValue" step="0.000001" placeholder="例如: 7.250000"></div>' +
        '<div class="form-group"><label>买入价 (可选)</label><input type="number" class="form-input" id="rateBid" step="0.000001" placeholder="例如: 7.240000"></div>' +
        '<div class="form-group"><label>卖出价 (可选)</label><input type="number" class="form-input" id="rateAsk" step="0.000001" placeholder="例如: 7.260000"></div>';

    showModal('添加汇率', html, {
        confirmText: '添加',
        onConfirm: async function () {
            var source = document.getElementById('rateSourceCurrency').value;
            var target = document.getElementById('rateTargetCurrency').value;
            var rate = parseFloat(document.getElementById('rateValue').value);
            var bid = parseFloat(document.getElementById('rateBid').value) || 0;
            var ask = parseFloat(document.getElementById('rateAsk').value) || 0;

            if (!source || !target || !rate || rate <= 0) {
                Toast.warning('请输入有效的汇率值');
                throw new Error('Validation failed');
            }
            if (source === target) {
                Toast.warning('来源和目标币种不能相同');
                throw new Error('Validation failed');
            }

            var body = {
                source_currency: source,
                target_currency: target,
                rate: rate,
                bid: bid || undefined,
                ask: ask || undefined
            };

            try {
                await API.post('/api/v1/exchange-rates', body);
                Toast.success('汇率已添加');
                loadExchangeRates();
            } catch (e) {
                Toast.error('添加失败: ' + (e.message || '未知错误'));
                throw e;
            }
        },
        onCancel: function () { }
    });
}

/* ===== System Info ===== */
async function loadSystemInfo() {
    var container = document.getElementById('systemInfo');
    if (!container) return;

    try {
        var data = await API.get('/api/v1/system/info');
        container.innerHTML = '<div class="sys-info-grid">' +
            '<div class="sys-info-row"><span class="sys-info-label">网关版本</span><span class="sys-info-value">' + escapeHtml(data.version || data.gateway_version || '-') + '</span></div>' +
            '<div class="sys-info-row"><span class="sys-info-label">引擎状态</span><span class="sys-info-value"><span class="status-dot ' + (data.engine_connected ? 'status-connected' : (data.engine_enabled ? 'status-disconnected' : 'status-processing')) + '"></span>' + (data.engine_connected ? '已连接' : (data.engine_enabled ? '未连接' : '内部处理')) + '</span></div>' +
            '<div class="sys-info-row"><span class="sys-info-label">数据库</span><span class="sys-info-value">' + escapeHtml(data.db_driver || data.database || data.db || '-') + '</span></div>' +
            '<div class="sys-info-row"><span class="sys-info-label">运行时间</span><span class="sys-info-value">' + (data.uptime || data.uptime_seconds ? formatDuration(data.uptime_seconds || data.uptime) : '-') + '</span></div>' +
            '<div class="sys-info-row"><span class="sys-info-label">Go 版本</span><span class="sys-info-value">' + escapeHtml(data.go_version || data.go || '-') + '</span></div>' +
            '<div class="sys-info-row"><span class="sys-info-label">Goroutines</span><span class="sys-info-value">' + (data.goroutines || '-') + '</span></div>' +
            '</div>';
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><p>系统信息不可用</p></div>';
    }
}

function formatDuration(seconds) {
    if (!seconds || seconds < 0) return '-';
    var h = Math.floor(seconds / 3600);
    var m = Math.floor((seconds % 3600) / 60);
    var s = seconds % 60;
    if (h > 24) {
        var d = Math.floor(h / 24);
        h = h % 24;
        return d + '天 ' + h + '小时 ' + m + '分';
    }
    if (h > 0) return h + '小时 ' + m + '分';
    if (m > 0) return m + '分 ' + s + '秒';
    return s + '秒';
}
