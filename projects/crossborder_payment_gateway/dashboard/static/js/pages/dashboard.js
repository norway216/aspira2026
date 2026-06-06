/* ===== Dashboard Page — Enhanced ===== */
var _dashboard = {
    tpsChart: null,
    volumeChart: null,
    refreshTimer: null,
    wsUnsubs: [],
    tpsHistory: [],
    tpsMaxPoints: 60,
    _initialized: false
};

function init_dashboard() {
    var d = _dashboard;

    // Reset stat card values for fresh animations
    StatCard.reset();

    // Load stats
    loadDashboardStats();

    // Init TPS chart (only once)
    if (!d._initialized) {
        d.tpsChart = Chart.create('chartTps', {
            tooltip: {
                trigger: 'axis',
                formatter: function (params) {
                    var p = params[0];
                    if (!p) return '';
                    return p.axisValue + '<br/>TPS: <strong>' + (p.value || 0).toFixed(1) + '</strong>';
                }
            },
            xAxis: {
                type: 'category',
                data: []
            },
            yAxis: {
                type: 'value',
                name: 'TPS',
                min: 0
            },
            series: [{
                name: 'TPS',
                type: 'line',
                smooth: true,
                showSymbol: false,
                lineStyle: {
                    color: '#FF2D55',
                    width: 2.5
                },
                areaStyle: {
                    color: {
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: 'rgba(255,45,85,0.35)' },
                            { offset: 1, color: 'rgba(255,45,85,0.02)' }
                        ]
                    }
                },
                data: []
            }],
            grid: { top: 15, right: 15, bottom: 25, left: 45 }
        });

        // Init Volume chart (only once)
        d.volumeChart = Chart.create('chartVolume', {
            tooltip: {
                trigger: 'axis',
                formatter: function (params) {
                    var p = params[0];
                    if (!p) return '';
                    return p.axisValue + '<br/>交易量 (USD): <strong>' + formatCurrency(p.value || 0, 'USD') + '</strong>';
                }
            },
            xAxis: {
                type: 'category',
                data: []
            },
            yAxis: {
                type: 'value',
                name: '交易量'
            },
            series: [{
                name: '交易量',
                type: 'bar',
                barWidth: '60%',
                itemStyle: {
                    borderRadius: [6, 6, 0, 0],
                    color: {
                        type: 'linear',
                        x: 0, y: 0, x2: 0, y2: 1,
                        colorStops: [
                            { offset: 0, color: '#64D2FF' },
                            { offset: 1, color: 'rgba(0,122,255,0.3)' }
                        ]
                    }
                },
                emphasis: {
                    itemStyle: {
                        color: '#007AFF'
                    }
                },
                data: []
            }],
            grid: { top: 15, right: 15, bottom: 25, left: 55 }
        });

        d._initialized = true;
    }

    // Load data
    loadTpsHistory();
    loadVolumeHistory();
    loadRecentTransactions();

    // WebSocket listeners
    setupDashboardWS();

    // Auto-refresh fallback every 5s
    if (d.refreshTimer) clearInterval(d.refreshTimer);
    d.refreshTimer = setInterval(function () {
        if (!WS.connected) {
            loadDashboardStats();
        }
    }, 5000);

    // Update last updated time
    updateLastUpdated();

    // Register cleanup
    App.setCleanup(cleanup_dashboard);
}

function cleanup_dashboard() {
    var d = _dashboard;
    if (d.refreshTimer) {
        clearInterval(d.refreshTimer);
        d.refreshTimer = null;
    }
    // Unsubscribe WS handlers
    d.wsUnsubs.forEach(function (fn) { if (typeof fn === 'function') fn(); });
    d.wsUnsubs = [];
    // Don't dispose charts on navigation - keep them for when we return
    StatCard.reset();
}

function setupDashboardWS() {
    var d = _dashboard;

    // Unsubscribe old
    d.wsUnsubs.forEach(function (fn) { if (typeof fn === 'function') fn(); });
    d.wsUnsubs = [];

    d.wsUnsubs.push(
        WS.on('transaction_update', function (data) {
            var txn = data.transaction || data;
            prependTransaction(txn);
            // Stats and TPS are updated via dashboard_stats and tps_update WS events
        })
    );

    d.wsUnsubs.push(
        WS.on('engine_health', function (data) {
            updateEnginePanel(data);
        })
    );

    d.wsUnsubs.push(
        WS.on('dashboard_stats', function (data) {
            updateStatCards(data);
            updateHeroStats(data);
            // Update volume chart in real-time
            updateVolumeChart(data);
            updateLastUpdated();
        })
    );

    d.wsUnsubs.push(
        WS.on('tps_update', function (data) {
            var tpsVal = data.tps || data.value || 0;
            // Update stat card
            StatCard.update('stat-tps', {
                value: tpsVal,
                label: '实时 TPS',
                format: 'tps'
            });
            // Update hero TPS value
            var heroTps = document.getElementById('heroTps');
            if (heroTps) {
                var newVal = tpsVal.toFixed(1);
                if (heroTps.textContent !== newVal) {
                    if (typeof Animations !== 'undefined' && Animations.animateValue) {
                        Animations.animateValue(heroTps, parseFloat(heroTps.textContent) || 0, tpsVal, 400,
                            function (v) { return v.toFixed(1); });
                    } else {
                        heroTps.textContent = newVal;
                    }
                }
            }
            // Update TPS chart
            if (d.tpsChart) {
                var time = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
                d.tpsHistory.push({ time: time, value: tpsVal });
                if (d.tpsHistory.length > d.tpsMaxPoints) {
                    d.tpsHistory.shift();
                }
                d.tpsChart.update({
                    xAxis: { data: d.tpsHistory.map(function (p) { return p.time; }) },
                    series: [{ data: d.tpsHistory.map(function (p) { return p.value; }) }]
                });
            }
        })
    );
}

function updateTpsFromTransaction(txn) {
    var d = _dashboard;
    if (d.tpsChart) {
        var time = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
        if (d.tpsHistory.length > 0) {
            var last = d.tpsHistory[d.tpsHistory.length - 1];
            if (last.time === time) {
                last.value = (parseFloat(last.value) || 0) + 1;
            } else {
                d.tpsHistory.push({ time: time, value: 1 });
            }
        } else {
            d.tpsHistory.push({ time: time, value: 1 });
        }
        if (d.tpsHistory.length > d.tpsMaxPoints) {
            d.tpsHistory.shift();
        }
        d.tpsChart.update({
            xAxis: { data: d.tpsHistory.map(function (p) { return p.time; }) },
            series: [{ data: d.tpsHistory.map(function (p) { return p.value; }) }]
        });
    }
}

async function loadDashboardStats() {
    try {
        var data = await API.get('/api/v1/dashboard');
        updateStatCards(data);
        updateHeroStats(data);
        if (data.engine) {
            updateEnginePanel(data.engine);
        }
        updateLastUpdated();
    } catch (e) {
        // Silent fail - WS may provide updates
    }
}

function updateStatCards(data) {
    StatCard.update('stat-tps', {
        value: data.tps || data.current_tps || 0,
        label: '实时 TPS',
        format: 'tps'
    });

    StatCard.update('stat-volume', {
        value: data.today_volume || data.volume || 0,
        label: '今日交易量 (USD)',
        format: 'usd'
    });

    var successRate = data.success_rate !== undefined ? data.success_rate : (data.rate || 0);
    StatCard.update('stat-success-rate', {
        value: successRate,
        label: '交易成功率',
        format: 'percent',
        trend: data.trend,
        trendUp: data.trend_up
    });
}

/* Update hero section stats */
function updateHeroStats(data) {
    var heroTps = document.getElementById('heroTps');
    var heroVolume = document.getElementById('heroVolume');
    var heroSuccess = document.getElementById('heroSuccess');

    if (heroTps) {
        var tps = parseFloat(data.tps || data.current_tps || 0);
        if (heroTps.textContent !== tps.toFixed(1)) {
            if (typeof Animations !== 'undefined' && Animations.animateValue) {
                Animations.animateValue(heroTps, parseFloat(heroTps.textContent) || 0, tps, 600,
                    function (v) { return v.toFixed(1); });
            } else {
                heroTps.textContent = tps.toFixed(1);
            }
        }
    }

    if (heroVolume) {
        var vol = parseFloat(data.today_volume || data.volume || 0);
        var volDisplay = formatHeroVolume(vol);
        if (heroVolume.textContent !== volDisplay) {
            heroVolume.textContent = volDisplay;
            Animations.pulse(heroVolume);
        }
    }

    if (heroSuccess) {
        var rate = parseFloat(data.success_rate || data.rate || 0);
        var rateDisplay = rate.toFixed(1) + '%';
        if (heroSuccess.textContent !== rateDisplay) {
            heroSuccess.textContent = rateDisplay;
            Animations.pulse(heroSuccess);
        }
    }
}

function formatHeroVolume(val) {
    if (val >= 1000000) return '$' + (val / 1000000).toFixed(2) + 'M';
    if (val >= 1000) return '$' + (val / 1000).toFixed(1) + 'K';
    return '$' + val.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

/* Update volume chart with real-time data from dashboard_stats */
function updateVolumeChart(data) {
    var d = _dashboard;
    if (!d.volumeChart) return;
    var vol = data.today_volume || data.volume || 0;
    var cnt = data.today_count || 0;
    if (vol <= 0 && cnt <= 0) return;
    var time = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
    // Append a point to the volume chart for real-time effect
    var opt = d.volumeChart.getOption();
    var xData = opt.xAxis[0].data || [];
    var sData = opt.series[0].data || [];
    // Only add if the time slot is new
    if (xData.length === 0 || xData[xData.length - 1] !== time) {
        xData.push(time);
        sData.push(vol);
    } else {
        // Update the last point with latest volume
        sData[sData.length - 1] = vol;
    }
    // Keep max 24 points
    if (xData.length > 24) {
        xData.shift();
        sData.shift();
    }
    d.volumeChart.update({
        xAxis: { data: xData },
        series: [{ data: sData }]
    });
}

function updateEnginePanel(data) {
    if (!data) return;

    var engineEnabled = data.engine_enabled === true;
    var connected = data.connected || data.status === 'connected';

    // Determine status: enabled+connected, enabled+disconnected, or disabled(internal mode)
    var dotClass, statusText, labelText;
    if (engineEnabled && connected) {
        dotClass = 'status-connected';
        statusText = '已连接';
        labelText = '引擎运行中';
    } else if (engineEnabled && !connected) {
        dotClass = 'status-disconnected';
        statusText = '未连接';
        labelText = '引擎已断开';
    } else {
        // Engine is disabled — using internal fallback processing
        dotClass = 'status-processing';
        statusText = '内部处理';
        labelText = '内部处理模式';
    }

    var statusEl = document.getElementById('engineConnStatus');
    var queueEl = document.getElementById('engineQueueDepth');
    var workerEl = document.getElementById('engineWorkerCount');
    var processingEl = document.getElementById('engineProcessing');

    if (statusEl) {
        statusEl.innerHTML = '<span class="status-dot ' + dotClass + '"></span> ' + statusText;
    }
    if (queueEl) queueEl.textContent = data.queue_depth || data.queueDepth || 0;
    if (workerEl) workerEl.textContent = data.worker_count || data.workerCount || 0;
    if (processingEl) processingEl.textContent = data.processing || 0;

    // Update engine stat card
    var engineCard = document.getElementById('stat-engine');
    if (engineCard) {
        var dot = engineCard.querySelector('.status-dot');
        if (dot) {
            dot.className = 'status-dot ' + dotClass;
        }
        var valEl = engineCard.querySelector('.stat-value');
        if (valEl) {
            valEl.innerHTML = '<span class="status-dot ' + dotClass + '"></span>';
        }
        var labelEl = engineCard.querySelector('.stat-label');
        if (labelEl) {
            labelEl.textContent = labelText;
        }
    }
}

async function loadTpsHistory() {
    try {
        var data = await API.get('/api/v1/dashboard/tps-history');
        var points = data.data || data.points || data;
        if (Array.isArray(points) && points.length > 0) {
            _dashboard.tpsHistory = points.map(function (p) {
                return {
                    time: p.time || p.t || formatTimeShort(p.timestamp),
                    value: p.value || p.tps || p.v || 0
                };
            });
            if (_dashboard.tpsHistory.length > _dashboard.tpsMaxPoints) {
                _dashboard.tpsHistory = _dashboard.tpsHistory.slice(-_dashboard.tpsMaxPoints);
            }
            if (_dashboard.tpsChart) {
                _dashboard.tpsChart.update({
                    xAxis: { data: _dashboard.tpsHistory.map(function (p) { return p.time; }) },
                    series: [{ data: _dashboard.tpsHistory.map(function (p) { return p.value; }) }]
                });
            }
        }
    } catch (e) { }
}

async function loadVolumeHistory() {
    try {
        var data = await API.get('/api/v1/dashboard/volume-history');
        var points = data.data || data.points || data;
        if (Array.isArray(points) && points.length > 0) {
            if (_dashboard.volumeChart) {
                _dashboard.volumeChart.update({
                    xAxis: { data: points.map(function (p) { return p.time || p.hour || p.label || '-'; }) },
                    series: [{ data: points.map(function (p) { return p.value || p.volume || p.v || 0; }) }]
                });
            }
        }
    } catch (e) { }
}

async function loadRecentTransactions() {
    var container = document.getElementById('recentTransactions');
    if (!container) return;

    try {
        var data = await API.get('/api/v1/transactions?size=10&sort=created_at&order=desc');
        var txns = data.data || data.transactions || [];
        renderRecentTxns(txns);
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><div class="empty-icon">📋</div><p>加载失败</p></div>';
    }
}

function renderRecentTxns(txns) {
    var container = document.getElementById('recentTransactions');
    if (!container) return;

    if (!txns || txns.length === 0) {
        container.innerHTML = '<div class="empty-state"><div class="empty-icon">📋</div><h3>暂无交易</h3><p>还没有任何交易记录</p></div>';
        return;
    }

    var html = '<table class="data-table">';
    html += '<thead><tr>' +
        '<th>时间</th><th>交易ID</th><th>商户</th><th>金额</th><th>状态</th>' +
        '</tr></thead><tbody>';

    for (var i = 0; i < Math.min(txns.length, 10); i++) {
        var t = txns[i];
        var time = formatTimeShort(t.created_at || t.time || t.timestamp);
        var txnId = truncateMiddle(t.id || t.txn_id || t.transaction_id, 12);
        var merchant = t.merchant_name || t.merchant || '-';
        var sourceAmount = formatCurrency(t.source_amount || t.amount, t.source_currency || t.currency);
        var targetAmount = formatCurrency(t.target_amount, t.target_currency);
        var status = t.status || 'pending';
        var statusLabel = getStatusLabel(status);

        html += '<tr onclick="App.navigate(\'transaction-detail\', {id: \'' + escapeHtml(String(t.id || t.txn_id)) + '\'})" style="cursor:pointer">' +
            '<td class="time-cell">' + time + '</td>' +
            '<td class="txn-id">' + txnId + '</td>' +
            '<td>' + escapeHtml(merchant) + '</td>' +
            '<td class="amount-cell">' + sourceAmount + ' → ' + targetAmount + '</td>' +
            '<td><span class="badge badge-' + status + '">' + statusLabel + '</span></td>' +
            '</tr>';
    }

    html += '</tbody></table>';
    container.innerHTML = html;
}

function prependTransaction(txn) {
    if (!txn) return;
    var container = document.getElementById('recentTransactions');
    if (!container) return;
    var table = container.querySelector('.data-table');
    if (!table) {
        loadRecentTransactions();
        return;
    }

    var tbody = table.querySelector('tbody');
    if (!tbody) return;

    var time = formatTimeShort(txn.created_at || txn.time || txn.timestamp);
    var txnId = truncateMiddle(txn.id || txn.txn_id || txn.transaction_id, 12);
    var merchant = txn.merchant_name || txn.merchant || '-';
    var sourceAmount = formatCurrency(txn.source_amount || txn.amount, txn.source_currency || txn.currency);
    var targetAmount = formatCurrency(txn.target_amount, txn.target_currency);
    var status = txn.status || 'pending';
    var statusLabel = getStatusLabel(status);

    var row = document.createElement('tr');
    row.className = 'new-row';
    row.style.cursor = 'pointer';
    row.setAttribute('onclick', 'App.navigate(\'transaction-detail\', {id: \'' + escapeHtml(String(txn.id || txn.txn_id)) + '\'})');
    row.innerHTML =
        '<td class="time-cell">' + time + '</td>' +
        '<td class="txn-id">' + txnId + '</td>' +
        '<td>' + escapeHtml(merchant) + '</td>' +
        '<td class="amount-cell">' + sourceAmount + ' → ' + targetAmount + '</td>' +
        '<td><span class="badge badge-' + status + '">' + statusLabel + '</span></td>';

    // Remove last row if more than 10
    if (tbody.children.length >= 10) {
        tbody.removeChild(tbody.lastChild);
    }

    tbody.insertBefore(row, tbody.firstChild);
}

function getStatusLabel(status) {
    var map = {
        pending: '待处理',
        processing: '处理中',
        completed: '已完成',
        failed: '失败',
        refunded: '已退款',
        success: '成功',
        cancelled: '已取消'
    };
    return map[status] || status;
}

function updateLastUpdated() {
    var el = document.getElementById('lastUpdated');
    if (el) {
        el.textContent = '更新于 ' + new Date().toLocaleTimeString('zh-CN');
    }
}
