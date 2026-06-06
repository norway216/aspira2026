/* ===== Transaction Detail Page ===== */
function init_transaction_detail(params) {
    var txnId = params.id;
    if (!txnId) {
        document.getElementById('transactionDetailContent').innerHTML = '<div class="empty-state"><p>未指定交易ID</p></div>';
        return;
    }

    var container = document.getElementById('transactionDetailContent');
    container.innerHTML = '<div class="table-placeholder">加载中...</div>';

    API.get('/api/v1/transactions/' + encodeURIComponent(txnId))
        .then(function (data) {
            var txn = data.transaction || data;
            renderTransactionDetail(txn);
        })
        .catch(function (e) {
            container.innerHTML = '<div class="empty-state"><p>加载失败: ' + escapeHtml(e.message || '未知错误') + '</p></div>';
        });
}

function renderTransactionDetail(txn) {
    var container = document.getElementById('transactionDetailContent');
    if (!container) return;

    var status = txn.status || 'pending';
    var statusLabel = getStatusLabel(status);

    var html = '<div class="detail-grid">';

    // Left column - Transaction info
    html += '<div class="card">';

    // Amount section
    html += '<div class="detail-amount-row">' +
        '<div class="detail-label">交易金额</div>' +
        '<div class="detail-amount">' +
        '<div class="detail-amount-main">' + formatCurrency(txn.source_amount || txn.amount, txn.source_currency || txn.currency) + '</div>' +
        '<div class="detail-amount-sub">' + (txn.source_currency || '') + ' → ' + (txn.target_currency || '') + ' @ ' + (txn.exchange_rate ? parseFloat(txn.exchange_rate).toFixed(4) : '-') + '</div>' +
        '</div></div>';

    // Target amount
    if (txn.target_amount) {
        html += '<div class="detail-amount-row" style="border-top:1px solid var(--color-border);">' +
            '<div class="detail-label">结算金额</div>' +
            '<div class="detail-amount">' +
            '<div class="detail-amount-main">' + formatCurrency(txn.target_amount, txn.target_currency) + '</div>' +
            '</div></div>';
    }

    html += '<div class="detail-section">';
    html += '<div class="detail-section-title">交易信息</div>';

    var fields = [
        { label: '交易ID', value: '<code>' + escapeHtml(txn.id || txn.txn_id || txn.transaction_id || '-') + '</code>' },
        { label: '参考ID', value: '<code>' + escapeHtml(txn.reference_id || txn.ref_id || '-') + '</code>' },
        { label: '状态', value: '<span class="badge badge-' + status + '">' + statusLabel + '</span>' },
        { label: '创建时间', value: formatTime(txn.created_at || txn.time || txn.timestamp) },
        { label: '完成时间', value: formatTime(txn.completed_at) || '-' },
        { label: '手续费', value: txn.fee ? formatCurrency(txn.fee, txn.fee_currency || txn.source_currency) : '-' }
    ];

    for (var i = 0; i < fields.length; i++) {
        html += '<div class="detail-row"><span class="detail-label">' + fields[i].label + '</span><span class="detail-value">' + fields[i].value + '</span></div>';
    }

    html += '</div>'; // detail-section

    html += '<div class="detail-section">';
    html += '<div class="detail-section-title">商户信息</div>';

    var merchantFields = [
        { label: '商户名称', value: escapeHtml(txn.merchant_name || txn.merchant || '-') },
        { label: '商户ID', value: '<code>' + escapeHtml(txn.merchant_id || '-') + '</code>' }
    ];

    for (var j = 0; j < merchantFields.length; j++) {
        html += '<div class="detail-row"><span class="detail-label">' + merchantFields[j].label + '</span><span class="detail-value">' + merchantFields[j].value + '</span></div>';
    }

    html += '</div>'; // detail-section
    html += '</div>'; // card

    // Right column - Source & Target details
    html += '<div class="card">';

    html += '<div class="detail-section">';
    html += '<div class="detail-section-title">来源信息</div>';

    if (txn.source_account || txn.source_currency || txn.source_amount) {
        html += '<div class="detail-row"><span class="detail-label">账户</span><span class="detail-value"><code>' + escapeHtml(txn.source_account || '-') + '</code></span></div>';
        html += '<div class="detail-row"><span class="detail-label">币种</span><span class="detail-value">' + escapeHtml(txn.source_currency || '-') + '</span></div>';
        html += '<div class="detail-row"><span class="detail-label">金额</span><span class="detail-value">' + formatCurrency(txn.source_amount || txn.amount, txn.source_currency || txn.currency) + '</span></div>';
        if (txn.source_balance_before !== undefined) {
            html += '<div class="detail-row"><span class="detail-label">此前余额</span><span class="detail-value">' + formatCurrency(txn.source_balance_before, txn.source_currency) + '</span></div>';
            html += '<div class="detail-row"><span class="detail-label">此后余额</span><span class="detail-value">' + formatCurrency(txn.source_balance_after, txn.source_currency) + '</span></div>';
        }
    } else {
        html += '<div class="empty-state" style="padding:20px"><p>来源信息不可用</p></div>';
    }

    html += '</div>';

    html += '<div class="detail-section">';
    html += '<div class="detail-section-title">目标信息</div>';

    if (txn.target_account || txn.target_currency || txn.target_amount) {
        html += '<div class="detail-row"><span class="detail-label">账户</span><span class="detail-value"><code>' + escapeHtml(txn.target_account || '-') + '</code></span></div>';
        html += '<div class="detail-row"><span class="detail-label">币种</span><span class="detail-value">' + escapeHtml(txn.target_currency || '-') + '</span></div>';
        html += '<div class="detail-row"><span class="detail-label">金额</span><span class="detail-value">' + formatCurrency(txn.target_amount, txn.target_currency) + '</span></div>';
        if (txn.target_balance_before !== undefined) {
            html += '<div class="detail-row"><span class="detail-label">此前余额</span><span class="detail-value">' + formatCurrency(txn.target_balance_before, txn.target_currency) + '</span></div>';
            html += '<div class="detail-row"><span class="detail-label">此后余额</span><span class="detail-value">' + formatCurrency(txn.target_balance_after, txn.target_currency) + '</span></div>';
        }
    } else {
        html += '<div class="empty-state" style="padding:20px"><p>目标信息不可用</p></div>';
    }

    html += '</div>';

    // Error info if failed
    if (status === 'failed' && (txn.error_message || txn.error)) {
        html += '<div class="detail-section">';
        html += '<div class="detail-section-title">错误信息</div>';
        html += '<div style="background:rgba(255,69,58,0.08);border-radius:8px;padding:12px;font-size:13px;color:var(--color-red);font-family:var(--font-mono)">';
        html += escapeHtml(txn.error_message || txn.error || '');
        html += '</div></div>';
    }

    html += '</div>'; // card
    html += '</div>'; // detail-grid

    // Actions
    html += '<div style="margin-top:20px;display:flex;gap:12px">';
    html += '<button class="btn btn-ghost" onclick="App.navigate(\'transactions\')">返回交易列表</button>';
    if (status === 'completed') {
        html += '<button class="btn btn-outline" onclick="showRefundConfirm(\'' + escapeHtml(String(txn.id || txn.txn_id)) + '\')">发起退款</button>';
    }
    html += '</div>';

    container.innerHTML = html;
}
