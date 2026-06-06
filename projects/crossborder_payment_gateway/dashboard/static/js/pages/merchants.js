/* ===== Merchants Page ===== */
var _merchantState = {
    page: 1,
    pageSize: 20,
    total: 0
};

function init_merchants() {
    _merchantState.page = 1;
    loadMerchants();
    App.setCleanup(cleanup_merchants);
}

function cleanup_merchants() {
    // Nothing to clean
}

async function loadMerchants() {
    var container = document.getElementById('merchantsTable');
    if (!container) return;

    container.innerHTML = '<div class="table-placeholder">加载中...</div>';

    try {
        var s = _merchantState;
        var data = await API.get('/api/v1/merchants?page=' + s.page + '&size=' + s.pageSize);
        var merchants = data.data || data.merchants || [];
        s.total = data.total || data.count || data.total_count || merchants.length;
        renderMerchantsTable(merchants);
        renderMerchantPagination();
    } catch (e) {
        container.innerHTML = '<div class="empty-state"><p>加载失败: ' + escapeHtml(e.message || '未知错误') + '</p></div>';
    }
}

function renderMerchantsTable(merchants) {
    var container = document.getElementById('merchantsTable');
    if (!container) return;

    if (!merchants || merchants.length === 0) {
        container.innerHTML = '<div class="empty-state"><div class="empty-icon">&#128100;</div><h3>暂无商户</h3><p>点击右上角创建商户</p></div>';
        return;
    }

    var html = '<table class="data-table">';
    html += '<thead><tr>' +
        '<th>商户名称</th>' +
        '<th>API Key</th>' +
        '<th>状态</th>' +
        '<th>日限额</th>' +
        '<th>月限额</th>' +
        '<th>创建时间</th>' +
        '<th>操作</th>' +
        '</tr></thead><tbody>';

    for (var i = 0; i < merchants.length; i++) {
        var m = merchants[i];
        var apiKey = m.api_key || m.apiKey || '-';
        var maskedKey = apiKey.length > 8 ? apiKey.substring(0, 4) + '****' + apiKey.substring(apiKey.length - 4) : '****';
        var status = m.status || 'active';
        var dailyLimit = m.daily_limit ? formatCurrency(m.daily_limit) : '-';
        var monthlyLimit = m.monthly_limit ? formatCurrency(m.monthly_limit) : '-';
        var created = formatTime(m.created_at || m.createdAt || m.create_time);

        html += '<tr>' +
            '<td><strong>' + escapeHtml(m.name || '-') + '</strong></td>' +
            '<td><code style="font-size:11px">' + maskedKey + '</code></td>' +
            '<td><span class="badge badge-' + status + '">' + (status === 'active' ? '活跃' : '停用') + '</span></td>' +
            '<td>' + dailyLimit + '</td>' +
            '<td>' + monthlyLimit + '</td>' +
            '<td class="time-cell">' + created + '</td>' +
            '<td class="actions-cell">' +
            '<button class="btn btn-xs btn-ghost" onclick="editMerchant(\'' + escapeHtml(String(m.id)) + '\')">编辑</button>' +
            '<button class="btn btn-xs btn-' + (status === 'active' ? 'outline' : 'primary') + '" onclick="toggleMerchantStatus(\'' + escapeHtml(String(m.id)) + '\', \'' + status + '\')">' +
            (status === 'active' ? '停用' : '启用') + '</button>' +
            '</td></tr>';
    }

    html += '</tbody></table>';
    container.innerHTML = html;
}

function renderMerchantPagination() {
    var container = document.getElementById('merchantPagination');
    if (!container) return;

    var s = _merchantState;
    var totalPages = Math.ceil(s.total / s.pageSize) || 1;
    if (totalPages <= 1) {
        container.innerHTML = '';
        return;
    }

    var html = '';
    html += '<button ' + (s.page <= 1 ? 'disabled' : '') + ' onclick="goMerchantPage(' + (s.page - 1) + ')">上一页</button>';
    for (var i = 1; i <= totalPages; i++) {
        html += '<button class="' + (i === s.page ? 'active' : '') + '" onclick="goMerchantPage(' + i + ')">' + i + '</button>';
    }
    html += '<button ' + (s.page >= totalPages ? 'disabled' : '') + ' onclick="goMerchantPage(' + (s.page + 1) + ')">下一页</button>';
    html += '<span class="page-info">共 ' + s.total + ' 条</span>';

    container.innerHTML = html;
}

function goMerchantPage(page) {
    _merchantState.page = page;
    loadMerchants();
}

function showCreateMerchant() {
    var html = '<div class="form-group"><label>商户名称</label><input type="text" class="form-input" id="merchantName" placeholder="例如: 某某科技"></div>' +
        '<div class="form-group"><label>每日限额 (CNY)</label><input type="number" class="form-input" id="merchantDailyLimit" placeholder="例如: 100000" value="100000"></div>' +
        '<div class="form-group"><label>每月限额 (CNY)</label><input type="number" class="form-input" id="merchantMonthlyLimit" placeholder="例如: 3000000" value="3000000"></div>' +
        '<div class="form-group"><label>回调 URL</label><input type="url" class="form-input" id="merchantCallbackUrl" placeholder="https://example.com/callback"></div>';

    showModal('创建商户', html, {
        confirmText: '创建',
        onConfirm: async function () {
            var name = document.getElementById('merchantName').value.trim();
            if (!name) {
                Toast.warning('请输入商户名称');
                throw new Error('Validation failed');
            }
            var body = {
                name: name,
                daily_limit: parseFloat(document.getElementById('merchantDailyLimit').value) || 0,
                monthly_limit: parseFloat(document.getElementById('merchantMonthlyLimit').value) || 0,
                callback_url: document.getElementById('merchantCallbackUrl').value.trim()
            };
            try {
                await API.post('/api/v1/merchants', body);
                Toast.success('商户创建成功');
                loadMerchants();
            } catch (e) {
                Toast.error('创建失败: ' + (e.message || '未知错误'));
                throw e;
            }
        },
        onCancel: function () { }
    });
}

async function editMerchant(id) {
    try {
        var data = await API.get('/api/v1/merchants/' + encodeURIComponent(id));
        var m = data.merchant || data;

        var html = '<div class="form-group"><label>商户名称</label><input type="text" class="form-input" id="editMerchantName" value="' + escapeHtml(m.name || '') + '"></div>' +
            '<div class="form-group"><label>每日限额 (CNY)</label><input type="number" class="form-input" id="editMerchantDailyLimit" value="' + (m.daily_limit || 0) + '"></div>' +
            '<div class="form-group"><label>每月限额 (CNY)</label><input type="number" class="form-input" id="editMerchantMonthlyLimit" value="' + (m.monthly_limit || 0) + '"></div>' +
            '<div class="form-group"><label>回调 URL</label><input type="url" class="form-input" id="editMerchantCallbackUrl" value="' + escapeHtml(m.callback_url || '') + '"></div>';

        showModal('编辑商户', html, {
            confirmText: '保存',
            onConfirm: async function () {
                var name = document.getElementById('editMerchantName').value.trim();
                if (!name) {
                    Toast.warning('请输入商户名称');
                    throw new Error('Validation failed');
                }
                var body = {
                    name: name,
                    daily_limit: parseFloat(document.getElementById('editMerchantDailyLimit').value) || 0,
                    monthly_limit: parseFloat(document.getElementById('editMerchantMonthlyLimit').value) || 0,
                    callback_url: document.getElementById('editMerchantCallbackUrl').value.trim()
                };
                try {
                    await API.put('/api/v1/merchants/' + encodeURIComponent(id), body);
                    Toast.success('商户信息已更新');
                    loadMerchants();
                } catch (e) {
                    Toast.error('更新失败: ' + (e.message || '未知错误'));
                    throw e;
                }
            },
            onCancel: function () { }
        });
    } catch (e) {
        Toast.error('加载商户信息失败');
    }
}

async function toggleMerchantStatus(id, currentStatus) {
    var newStatus = currentStatus === 'active' ? 'inactive' : 'active';
    var actionLabel = newStatus === 'active' ? '启用' : '停用';

    showModal('确认' + actionLabel, '<p>确定要' + actionLabel + '该商户吗？</p>', {
        confirmText: '确认' + actionLabel,
        confirmClass: newStatus === 'active' ? 'btn btn-primary' : 'btn btn-outline',
        onConfirm: async function () {
            try {
                await API.put('/api/v1/merchants/' + encodeURIComponent(id), { status: newStatus });
                Toast.success('商户已' + actionLabel);
                loadMerchants();
            } catch (e) {
                Toast.error('操作失败: ' + (e.message || '未知错误'));
                throw e;
            }
        },
        onCancel: function () { }
    });
}
