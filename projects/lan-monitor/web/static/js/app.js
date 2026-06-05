// LAN Monitor - Single Page Application
const API = '/api/v1';
let token = localStorage.getItem('token');
let currentUser = null;
let ws = null;
let wsReconnectTimer = null;
let currentPage = 'dashboard';
let editDeviceId = null;

// ==================== INIT ====================
document.addEventListener('DOMContentLoaded', () => {
  if (token) {
    checkAuth();
  }
  document.getElementById('loginForm').addEventListener('submit', handleLogin);
  document.getElementById('menuToggle').addEventListener('click', toggleSidebar);
  updateClock();
  setInterval(updateClock, 10000);
});

function updateClock() {
  const now = new Date();
  document.getElementById('headerTime').textContent = now.toLocaleString('zh-CN');
  const sc = document.getElementById('sidebarClock');
  if (sc) sc.textContent = now.toLocaleTimeString('zh-CN');
}

function toggleSidebar() {
  document.getElementById('sidebar').classList.toggle('open');
}

// ==================== AUTH ====================
async function handleLogin(e) {
  e.preventDefault();
  const username = document.getElementById('loginUser').value.trim();
  const password = document.getElementById('loginPass').value.trim();
  if (!username || !password) return showToast('请输入用户名和密码', 'error');

  try {
    const resp = await fetch(`${API}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    const data = await resp.json();
    if (!resp.ok) throw new Error(data.error || '登录失败');

    token = data.access_token;
    currentUser = data.user;
    localStorage.setItem('token', token);
    localStorage.setItem('user', JSON.stringify(data.user));

    showApp();
    connectWebSocket();
    navigateTo('dashboard');
  } catch (err) {
    showToast(err.message, 'error');
  }
}

async function checkAuth() {
  try {
    const resp = await fetch(`${API}/auth/profile`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!resp.ok) throw new Error('认证失败');
    currentUser = await resp.json();
    localStorage.setItem('user', JSON.stringify(currentUser));
    showApp();
    connectWebSocket();
    navigateTo('dashboard');
  } catch (err) {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    token = null;
    currentUser = null;
  }
}

function logout() {
  localStorage.removeItem('token');
  localStorage.removeItem('user');
  token = null;
  currentUser = null;
  if (ws) ws.close();
  document.getElementById('loginPage').style.display = 'flex';
  document.getElementById('appLayout').classList.remove('active');
}

function showApp() {
  document.getElementById('loginPage').style.display = 'none';
  document.getElementById('appLayout').classList.add('active');
  document.getElementById('headerUsername').textContent = currentUser.username;
  document.getElementById('headerRole').textContent = roleLabel(currentUser.role);
}

function roleLabel(role) {
  const map = { super_admin: '超级管理员', admin: '管理员', viewer: '只读用户' };
  return map[role] || role;
}

// ==================== API HELPER ====================
async function apiGet(path) {
  const resp = await fetch(`${API}${path}`, {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  if (resp.status === 401) { logout(); throw new Error('登录已过期'); }
  return resp.json();
}

async function apiPost(path, body = {}) {
  const resp = await fetch(`${API}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
    body: JSON.stringify(body)
  });
  if (resp.status === 401) { logout(); throw new Error('登录已过期'); }
  return resp.json();
}

async function apiPut(path, body = {}) {
  const resp = await fetch(`${API}${path}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
    body: JSON.stringify(body)
  });
  if (resp.status === 401) { logout(); throw new Error('登录已过期'); }
  return resp.json();
}

async function apiDelete(path) {
  const resp = await fetch(`${API}${path}`, {
    method: 'DELETE',
    headers: { 'Authorization': `Bearer ${token}` }
  });
  if (resp.status === 401) { logout(); throw new Error('登录已过期'); }
  return resp.json();
}

// ==================== ROUTING ====================
function navigateTo(page, params) {
  currentPage = page;
  document.querySelectorAll('.page-content').forEach(p => p.classList.remove('active'));
  document.querySelectorAll('.nav-item').forEach(n => n.classList.remove('active'));

  const pageMap = {
    'dashboard': '仪表盘',
    'devices': '在线设备',
    'device-detail': '设备详情',
    'traffic': '流量分析',
    'alerts': '告警中心',
    'users': '用户管理',
    'audit': '操作审计',
    'settings': '系统设置'
  };

  document.getElementById('pageTitle').textContent = pageMap[page] || page;

  const navItem = document.querySelector(`[data-page="${page.split('-')[0]}"]`);
  if (navItem) navItem.classList.add('active');

  const pageEl = document.getElementById(`page-${page}`);
  if (pageEl) pageEl.classList.add('active');

  // Also activate parent page for device-detail
  if (page === 'device-detail') {
    const devicesPage = document.getElementById('page-device-detail');
    if (devicesPage) devicesPage.classList.add('active');
  }

  switch (page) {
    case 'dashboard': loadDashboard(); break;
    case 'devices': loadDeviceList(); break;
    case 'device-detail': loadDeviceDetail(params); break;
    case 'traffic': loadTrafficPage(); break;
    case 'alerts': loadAlertList(); break;
    case 'users': loadUserList(); break;
    case 'audit': loadAuditLogs(); break;
    case 'settings': loadSettings(); break;
  }
}

// ==================== TOAST ====================
function showToast(msg, type = 'info') {
  const container = document.getElementById('toasts');
  const toast = document.createElement('div');
  toast.className = `toast toast-${type}`;
  toast.textContent = msg;
  container.appendChild(toast);
  setTimeout(() => { toast.remove(); }, 3000);
}

// ==================== MODAL ====================
function openModal(id) { document.getElementById(id).classList.add('active'); }
function closeModal(id) { document.getElementById(id).classList.remove('active'); }

// ==================== WEBSOCKET ====================
function connectWebSocket() {
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const wsUrl = `${protocol}//${location.host}/ws`;
  ws = new WebSocket(wsUrl);

  ws.onopen = () => {
    document.getElementById('wsStatus').innerHTML = '<i class="fa-solid fa-circle" style="color:#67C23A;font-size:8px"></i> 实时连接';
    if (wsReconnectTimer) { clearTimeout(wsReconnectTimer); wsReconnectTimer = null; }
  };

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data);
      handleWSMessage(msg);
    } catch (e) {}
  };

  ws.onclose = () => {
    document.getElementById('wsStatus').innerHTML = '<i class="fa-solid fa-circle" style="color:#F56C6C;font-size:8px"></i> 已断开';
    wsReconnectTimer = setTimeout(connectWebSocket, 5000);
  };

  ws.onerror = () => { ws.close(); };
}

function handleWSMessage(msg) {
  switch (msg.type) {
    case 'device_status':
      if (currentPage === 'devices') loadDeviceList();
      if (currentPage === 'dashboard') loadDashboard();
      break;
    case 'alert':
      showToast(`新告警: ${msg.data.title}`, 'warning');
      updateAlertBadge();
      break;
    case 'traffic':
      if (currentPage === 'device-detail') updateTrafficChart(msg.data);
      break;
    case 'scan_progress':
      break;
  }
}

async function updateAlertBadge() {
  try {
    const data = await apiGet('/alerts/stats');
    const badge = document.getElementById('alertBadge');
    if (data.total_open > 0) {
      badge.style.display = 'inline';
      badge.textContent = data.total_open;
    } else {
      badge.style.display = 'none';
    }
  } catch (e) {}
}

// ==================== DASHBOARD ====================
async function loadDashboard() {
  try {
    const data = await apiGet('/dashboard');
    renderDashboardStats(data);
    renderStatusChart(data.status_distribution);
    renderTypeChart(data.type_distribution);
    renderRecentAlerts(data.recent_alerts);
    renderRecentEvents(data.recent_events);
    updateAlertBadge();
  } catch (e) { showToast('加载仪表盘失败: ' + e.message, 'error'); }
}

function renderDashboardStats(data) {
  document.getElementById('dashboardStats').innerHTML = `
    <div class="stat-card"><div class="stat-icon blue"><i class="fa-solid fa-wifi"></i></div>
      <div><div class="stat-value">${data.online_count}</div><div class="stat-label">当前在线设备</div></div></div>
    <div class="stat-card"><div class="stat-icon green"><i class="fa-solid fa-server"></i></div>
      <div><div class="stat-value">${data.total_devices}</div><div class="stat-label">累计发现设备</div></div></div>
    <div class="stat-card"><div class="stat-icon orange"><i class="fa-solid fa-triangle-exclamation"></i></div>
      <div><div class="stat-value">${data.offline_count}</div><div class="stat-label">离线设备</div></div></div>
    <div class="stat-card"><div class="stat-icon red"><i class="fa-solid fa-bell"></i></div>
      <div><div class="stat-value">${data.open_alerts}</div><div class="stat-label">未解决告警</div></div></div>
    <div class="stat-card"><div class="stat-icon purple"><i class="fa-solid fa-circle-plus"></i></div>
      <div><div class="stat-value">${data.today_new}</div><div class="stat-label">今日新增设备</div></div></div>
    <div class="stat-card"><div class="stat-icon blue"><i class="fa-solid fa-magnifying-glass-chart"></i></div>
      <div><div class="stat-value">${data.scans_today}</div><div class="stat-label">今日扫描次数</div></div></div>
  `;
}

function renderStatusChart(distribution) {
  const chart = echarts.init(document.getElementById('chartStatusDist'));
  const data = (distribution || []).map(d => ({ name: statusLabel(d.status), value: d.count }));
  chart.setOption({
    tooltip: { trigger: 'item' },
    legend: { bottom: '0%' },
    series: [{
      type: 'pie', radius: ['40%', '70%'], center: ['50%', '45%'],
      avoidLabelOverlap: false,
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
      label: { show: false },
      emphasis: { label: { show: true, fontSize: 16, fontWeight: 'bold' } },
      data: data
    }],
    color: ['#67C23A', '#F56C6C', '#E6A23C', '#909399', '#409EFF']
  });
  window.addEventListener('resize', () => chart.resize());
}

function renderTypeChart(distribution) {
  const chart = echarts.init(document.getElementById('chartTypeDist'));
  const data = (distribution || []).map(d => ({ name: d.device_type || 'Unknown', value: d.count }));
  chart.setOption({
    tooltip: { trigger: 'item' },
    legend: { bottom: '0%' },
    series: [{
      type: 'pie', radius: ['40%', '70%'], center: ['50%', '45%'],
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
      label: { show: false },
      emphasis: { label: { show: true, fontSize: 16, fontWeight: 'bold' } },
      data: data
    }],
    color: ['#5470C6', '#91CC75', '#FAC858', '#EE6666', '#73C0DE', '#3BA272', '#FC8452', '#9A60B4']
  });
  window.addEventListener('resize', () => chart.resize());
}

function renderRecentAlerts(alerts) {
  if (!alerts || alerts.length === 0) {
    document.getElementById('recentAlerts').innerHTML = '<div class="empty-state"><p>暂无告警</p></div>';
    return;
  }
  document.getElementById('recentAlerts').innerHTML = alerts.map(a => `
    <div style="padding:8px 0;border-bottom:1px solid #f0f0f0;display:flex;align-items:center;gap:8px">
      <span class="badge badge-${a.level}">${a.level}</span>
      <span style="flex:1;font-size:13px">${a.title}</span>
      <span style="font-size:11px;color:#999">${formatTime(a.created_at)}</span>
    </div>
  `).join('');
}

function renderRecentEvents(events) {
  if (!events || events.length === 0) {
    document.getElementById('recentEvents').innerHTML = '<div class="empty-state"><p>暂无事件</p></div>';
    return;
  }
  document.getElementById('recentEvents').innerHTML = events.map(e => `
    <div style="padding:8px 0;border-bottom:1px solid #f0f0f0;display:flex;align-items:center;gap:8px">
      <i class="fa-solid ${e.event_type === 'online' ? 'fa-circle-check' : e.event_type === 'offline' ? 'fa-circle-xmark' : 'fa-circle-info'}" style="color:${e.event_type === 'online' ? '#67C23A' : '#F56C6C'}"></i>
      <span style="flex:1;font-size:13px">${e.description}</span>
      <span style="font-size:11px;color:#999">${formatTime(e.event_time)}</span>
    </div>
  `).join('');
}

// ==================== DEVICES ====================
async function loadDeviceList(page = 1) {
  try {
    const search = document.getElementById('deviceSearch')?.value || '';
    const status = document.getElementById('deviceStatusFilter')?.value || '';
    const type = document.getElementById('deviceTypeFilter')?.value || '';
    const params = new URLSearchParams({ page: String(page), size: '20', search, status, type, sort: 'last_seen', order: 'desc' });
    const data = await apiGet(`/devices?${params}`);
    renderDeviceTable(data.data, data.total, data.page, data.size);
  } catch (e) { showToast('加载设备列表失败: ' + e.message, 'error'); }
}

function renderDeviceTable(devices, total, page, size) {
  if (!devices || devices.length === 0) {
    document.getElementById('deviceTableBody').innerHTML = '<tr><td colspan="10" style="text-align:center;padding:40px;color:#999">暂无设备数据</td></tr>';
    document.getElementById('devicePagination').innerHTML = '';
    return;
  }
  document.getElementById('deviceTableBody').innerHTML = devices.map(d => `
    <tr>
      <td><span class="badge badge-${d.status}">${statusLabel(d.status)}</span></td>
      <td><a href="javascript:void(0)" onclick="navigateTo('device-detail', ${d.id})" style="color:var(--primary);font-weight:500">${d.ip}</a></td>
      <td><code style="font-size:12px">${d.mac || '-'}</code></td>
      <td>${d.hostname || '-'}</td>
      <td>${d.vendor || 'Unknown'}</td>
      <td><span class="badge badge-info">${d.device_type || 'unknown'}</span></td>
      <td>${d.status === 'online' ? formatDuration(d.online_duration_seconds) : '-'}</td>
      <td>${d.label || '-'}</td>
      <td><span style="font-size:12px">${formatTime(d.last_seen)}</span></td>
      <td>
        <button class="btn btn-xs btn-primary" onclick="navigateTo('device-detail', ${d.id})"><i class="fa-solid fa-eye"></i></button>
        <button class="btn btn-xs btn-default" onclick="openEditDevice(${d.id})"><i class="fa-solid fa-pen"></i></button>
        <button class="btn btn-xs btn-warning" onclick="ignoreDevice(${d.id})"><i class="fa-solid fa-ban"></i></button>
      </td>
    </tr>
  `).join('');

  renderPagination('devicePagination', page, Math.ceil(total / size), loadDeviceList);
}

// ==================== DEVICE DETAIL ====================
async function loadDeviceDetail(id) {
  try {
    const data = await apiGet(`/devices/${id}`);
    renderDeviceDetail(data);
  } catch (e) { showToast('加载设备详情失败: ' + e.message, 'error'); }
}

function renderDeviceDetail(data) {
  const d = data.device;
  const events = data.events || [];
  const traffic = data.traffic || [];
  const alerts = data.alerts || [];
  const sessions = data.sessions || [];

  let html = `
  <div class="detail-grid" style="margin-bottom:20px">
    <div class="card">
      <div class="card-title" style="margin-bottom:16px"><i class="fa-solid fa-circle-info"></i> 设备信息</div>
      <div class="detail-item"><div class="detail-label">IP 地址</div><div class="detail-value">${d.ip}</div></div>
      <div class="detail-item"><div class="detail-label">MAC 地址</div><div class="detail-value"><code>${d.mac || '-'}</code></div></div>
      <div class="detail-item"><div class="detail-label">主机名</div><div class="detail-value">${d.hostname || '-'}</div></div>
      <div class="detail-item"><div class="detail-label">厂商</div><div class="detail-value">${d.vendor || 'Unknown'}</div></div>
      <div class="detail-item"><div class="detail-label">类型</div><div class="detail-value">${d.device_type || 'unknown'}</div></div>
      <div class="detail-item"><div class="detail-label">标签</div><div class="detail-value">${d.label || '-'}</div></div>
      <div class="detail-item"><div class="detail-label">负责人</div><div class="detail-value">${d.owner || '-'}</div></div>
      <div class="detail-item"><div class="detail-label">部门</div><div class="detail-value">${d.department || '-'}</div></div>
    </div>
    <div class="card">
      <div class="card-title" style="margin-bottom:16px"><i class="fa-solid fa-chart-simple"></i> 状态信息</div>
      <div class="detail-item"><div class="detail-label">当前状态</div><div class="detail-value"><span class="badge badge-${d.status}">${statusLabel(d.status)}</span></div></div>
      <div class="detail-item"><div class="detail-label">风险等级</div><div class="detail-value"><span class="badge badge-${d.risk_level}">${d.risk_level}</span></div></div>
      <div class="detail-item"><div class="detail-label">首次发现</div><div class="detail-value">${formatTime(d.first_seen)}</div></div>
      <div class="detail-item"><div class="detail-label">最近在线</div><div class="detail-value">${formatTime(d.last_seen)}</div></div>
      <div class="detail-item"><div class="detail-label">累计在线时长</div><div class="detail-value">${formatDuration(d.online_duration_seconds)}</div></div>
      <div class="detail-item"><div class="detail-label">离线次数</div><div class="detail-value">${d.offline_count}</div></div>
      <div class="detail-item"><div class="detail-label">开放端口</div><div class="detail-value">${d.open_ports || '-'}</div></div>
    </div>
  </div>

  <div class="charts-grid" style="margin-bottom:20px">
    <div class="card"><div class="card-header"><span class="card-title">实时流量</span></div>
      <div class="chart-container" id="chartDeviceTraffic"></div>
    </div>
    <div class="card"><div class="card-header"><span class="card-title">在线时间轴</span></div>
      <div class="chart-container" id="chartDeviceTimeline"></div>
    </div>
  </div>

  <div class="charts-grid">
    <div class="card"><div class="card-header"><span class="card-title">设备事件</span></div>
      <div style="max-height:300px;overflow-y:auto">
        ${events.length === 0 ? '<div class="empty-state"><p>暂无事件</p></div>' : events.map(e => `
          <div style="padding:8px 0;border-bottom:1px solid #f0f0f0;display:flex;align-items:center;gap:8px">
            <i class="fa-solid ${e.event_type === 'online' ? 'fa-circle-check' : e.event_type === 'offline' ? 'fa-circle-xmark' : 'fa-circle-info'}" style="color:${e.event_type === 'online' ? '#67C23A' : '#F56C6C'}"></i>
            <span style="flex:1;font-size:13px">${e.description || e.event_type}</span>
            <span style="font-size:11px;color:#999">${formatTime(e.event_time)}</span>
          </div>
        `).join('')}
      </div>
    </div>
    <div class="card"><div class="card-header"><span class="card-title">告警历史</span></div>
      <div style="max-height:300px;overflow-y:auto">
        ${alerts.length === 0 ? '<div class="empty-state"><p>暂无告警</p></div>' : alerts.map(a => `
          <div style="padding:8px 0;border-bottom:1px solid #f0f0f0;display:flex;align-items:center;gap:8px">
            <span class="badge badge-${a.level}">${a.level}</span>
            <span style="flex:1;font-size:13px">${a.title}</span>
            <span class="badge badge-${a.status}">${a.status}</span>
          </div>
        `).join('')}
      </div>
    </div>
  </div>`;

  document.getElementById('deviceDetailContent').innerHTML = html;

  // Render traffic chart
  setTimeout(() => {
renderDeviceTrafficChart(traffic);
renderDeviceTimelineChart(sessions, events);
  }, 100);
}

function renderDeviceTrafficChart(traffic) {
  const el = document.getElementById('chartDeviceTraffic');
  if (!el) return;
  const chart = echarts.init(el);
  const times = (traffic || []).map(t => formatTimeShort(t.timestamp));
  const rxData = (traffic || []).map(t => (t.rx_rate || 0).toFixed(2));
  const txData = (traffic || []).map(t => (t.tx_rate || 0).toFixed(2));

  chart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['入站速率', '出站速率'], bottom: 0 },
    xAxis: { type: 'category', data: times.length > 0 ? times : ['暂无数据'], boundaryGap: false },
    yAxis: { type: 'value', name: 'KB/s' },
    series: [
      { name: '入站速率', type: 'line', smooth: true, data: rxData.length > 0 ? rxData : [0], itemStyle: { color: '#409EFF' }, areaStyle: { color: 'rgba(64,158,255,0.1)' } },
      { name: '出站速率', type: 'line', smooth: true, data: txData.length > 0 ? txData : [0], itemStyle: { color: '#67C23A' }, areaStyle: { color: 'rgba(103,194,58,0.1)' } }
    ],
    grid: { top: 20, right: 20, bottom: 35, left: 55 }
  });
  window.addEventListener('resize', () => chart.resize());
}

function renderDeviceTimelineChart(sessions, events) {
  const el = document.getElementById('chartDeviceTimeline');
  if (!el) return;
  const chart = echarts.init(el);
  // Create timeline from events
  const data = (events || []).filter(e => e.event_type === 'online' || e.event_type === 'offline').reverse().map(e => ({
    time: formatTimeShort(e.event_time),
    status: e.event_type === 'online' ? 1 : 0
  }));

  chart.setOption({
    tooltip: { trigger: 'axis' },
    xAxis: { type: 'category', data: data.map(d => d.time).length > 0 ? data.map(d => d.time) : ['暂无数据'] },
    yAxis: { type: 'value', min: -0.5, max: 1.5, axisLabel: { formatter: v => v === 1 ? '在线' : '离线' } },
    series: [{
      name: '在线状态', type: 'line', step: 'end',
      data: data.map(d => d.status).length > 0 ? data.map(d => d.status) : [0],
      itemStyle: { color: '#67C23A' },
      areaStyle: { color: 'rgba(103,194,58,0.2)' }
    }],
    grid: { top: 20, right: 20, bottom: 30, left: 40 }
  });
  window.addEventListener('resize', () => chart.resize());
}

function updateTrafficChart(wsData) {
  // Real-time traffic update via WebSocket
  // Would update the active chart dynamically
}

async function openEditDevice(id) {
  try {
    const data = await apiGet(`/devices/${id}`);
    const d = data.device;
    editDeviceId = id;
    document.getElementById('editDeviceForm').innerHTML = `
      <div class="form-group"><label class="form-label">主机名</label><input type="text" class="form-input" id="editHostname" value="${d.hostname || ''}"></div>
      <div class="form-group"><label class="form-label">标签</label><input type="text" class="form-input" id="editLabel" value="${d.label || ''}"></div>
      <div class="form-group"><label class="form-label">负责人</label><input type="text" class="form-input" id="editOwner" value="${d.owner || ''}"></div>
      <div class="form-group"><label class="form-label">部门</label><input type="text" class="form-input" id="editDepartment" value="${d.department || ''}"></div>
      <div class="form-group"><label class="form-label">设备类型</label>
        <select class="form-select" id="editDeviceType">
          <option value="unknown" ${d.device_type === 'unknown' ? 'selected' : ''}>未知</option>
          <option value="PC" ${d.device_type === 'PC' ? 'selected' : ''}>PC</option>
          <option value="Phone" ${d.device_type === 'Phone' ? 'selected' : ''}>手机</option>
          <option value="Server" ${d.device_type === 'Server' ? 'selected' : ''}>服务器</option>
          <option value="Router" ${d.device_type === 'Router' ? 'selected' : ''}>路由器</option>
          <option value="Printer" ${d.device_type === 'Printer' ? 'selected' : ''}>打印机</option>
          <option value="Embedded" ${d.device_type === 'Embedded' ? 'selected' : ''}>嵌入式</option>
          <option value="Camera" ${d.device_type === 'Camera' ? 'selected' : ''}>摄像头</option>
        </select>
      </div>
      <div class="form-group"><label class="form-label">风险等级</label>
        <select class="form-select" id="editRiskLevel">
          <option value="normal" ${d.risk_level === 'normal' ? 'selected' : ''}>正常</option>
          <option value="warning" ${d.risk_level === 'warning' ? 'selected' : ''}>警告</option>
          <option value="critical" ${d.risk_level === 'critical' ? 'selected' : ''}>严重</option>
        </select>
      </div>
      <div class="form-group"><label class="form-label">备注</label><textarea class="form-textarea" id="editNote">${d.note || ''}</textarea></div>
    `;
    openModal('editDeviceModal');
  } catch (e) { showToast('加载设备信息失败', 'error'); }
}

async function saveDeviceEdit() {
  const body = {
    hostname: document.getElementById('editHostname').value,
    label: document.getElementById('editLabel').value,
    owner: document.getElementById('editOwner').value,
    department: document.getElementById('editDepartment').value,
    device_type: document.getElementById('editDeviceType').value,
    risk_level: document.getElementById('editRiskLevel').value,
    note: document.getElementById('editNote').value
  };
  try {
    await apiPut(`/devices/${editDeviceId}`, body);
    closeModal('editDeviceModal');
    showToast('设备信息已更新', 'success');
    if (currentPage === 'device-detail') loadDeviceDetail(editDeviceId);
    else loadDeviceList();
  } catch (e) { showToast('更新失败: ' + e.message, 'error'); }
}

async function ignoreDevice(id) {
  if (!confirm('确定要忽略此设备吗？')) return;
  try {
    await apiPost(`/devices/${id}/ignore`);
    showToast('设备已忽略', 'success');
    loadDeviceList();
  } catch (e) { showToast('操作失败: ' + e.message, 'error'); }
}

// ==================== SCAN ====================
async function startScan() {
  try {
    await apiPost('/scans/start');
    showToast('扫描任务已启动', 'success');
    setTimeout(() => loadDeviceList(), 5000);
  } catch (e) { showToast('启动扫描失败: ' + e.message, 'error'); }
}

// ==================== TRAFFIC ====================
async function loadTrafficPage() {
  try {
    const rankData = await apiGet('/traffic/rank?limit=20');
    renderTrafficRank(rankData.data);
    renderSystemTrafficChart();
  } catch (e) { showToast('加载流量数据失败: ' + e.message, 'error'); }
}

function renderTrafficRank(ranks) {
  if (!ranks || ranks.length === 0) {
    document.getElementById('trafficRankTable').innerHTML = '<tr><td colspan="5" style="text-align:center;padding:40px;color:#999">暂无流量数据</td></tr>';
    return;
  }
  document.getElementById('trafficRankTable').innerHTML = ranks.map((r, i) => `
    <tr>
      <td><strong>#${i + 1}</strong></td>
      <td>${r.hostname || r.ip}</td>
      <td>${r.ip}</td>
      <td><code style="font-size:11px">${r.mac || '-'}</code></td>
      <td><strong>${formatRate(r.total_rate)}</strong></td>
    </tr>
  `).join('');
}

function renderSystemTrafficChart() {
  const el = document.getElementById('chartSystemTraffic');
  if (!el) return;
  const chart = echarts.init(el);
  chart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['系统入站', '系统出站'], bottom: 0 },
    xAxis: { type: 'category', data: ['暂无系统流量数据'] },
    yAxis: { type: 'value', name: 'KB/s' },
    series: [
      { name: '系统入站', type: 'line', smooth: true, data: [0], areaStyle: { color: 'rgba(64,158,255,0.1)' } },
      { name: '系统出站', type: 'line', smooth: true, data: [0], areaStyle: { color: 'rgba(103,194,58,0.1)' } }
    ],
    grid: { top: 20, right: 20, bottom: 35, left: 55 }
  });
  window.addEventListener('resize', () => chart.resize());
}

// ==================== ALERTS ====================
async function loadAlertList(page = 1) {
  try {
    const status = document.getElementById('alertStatusFilter')?.value || '';
    const level = document.getElementById('alertLevelFilter')?.value || '';
    const params = new URLSearchParams({ page: String(page), size: '20', status, level });
    const data = await apiGet(`/alerts?${params}`);
    renderAlertTable(data.data, data.total, data.page);
  } catch (e) { showToast('加载告警列表失败: ' + e.message, 'error'); }
}

function renderAlertTable(alerts, total, page) {
  if (!alerts || alerts.length === 0) {
    document.getElementById('alertTableBody').innerHTML = '<tr><td colspan="7" style="text-align:center;padding:40px;color:#999">暂无告警</td></tr>';
    document.getElementById('alertPagination').innerHTML = '';
    return;
  }
  document.getElementById('alertTableBody').innerHTML = alerts.map(a => `
    <tr>
      <td><span class="badge badge-${a.level}">${a.level}</span></td>
      <td>${a.alert_type}</td>
      <td>${a.title}</td>
      <td><code style="font-size:11px">${a.device_mac || '-'}</code></td>
      <td><span class="badge badge-${a.status}">${a.status === 'open' ? '未解决' : '已解决'}</span></td>
      <td><span style="font-size:12px">${formatTime(a.created_at)}</span></td>
      <td>
        ${a.status === 'open' ? `<button class="btn btn-xs btn-success" onclick="resolveAlert(${a.id})"><i class="fa-solid fa-check"></i> 解决</button>` : '-'}
      </td>
    </tr>
  `).join('');
  renderPagination('alertPagination', page, Math.ceil(total / 20), loadAlertList);
}

async function resolveAlert(id) {
  try {
    await apiPost(`/alerts/${id}/resolve`);
    showToast('告警已解决', 'success');
    loadAlertList();
    updateAlertBadge();
  } catch (e) { showToast('操作失败: ' + e.message, 'error'); }
}

// ==================== USERS ====================
async function loadUserList(page = 1) {
  try {
    const data = await apiGet(`/users?page=${page}&size=20`);
    renderUserTable(data.data, data.total, data.page);
  } catch (e) { showToast('加载用户列表失败: ' + e.message, 'error'); }
}

function renderUserTable(users, total, page) {
  document.getElementById('userTableBody').innerHTML = users.map(u => `
    <tr>
      <td>${u.id}</td><td><strong>${u.username}</strong></td><td>${u.email || '-'}</td>
      <td><span class="badge badge-info">${roleLabel(u.role)}</span></td>
      <td><span class="badge badge-${u.status === 'active' ? 'online' : 'offline'}">${u.status}</span></td>
      <td><span style="font-size:12px">${u.created_at}</span></td>
      <td>
        <button class="btn btn-xs btn-danger" onclick="deleteUser(${u.id}, '${u.username}')"><i class="fa-solid fa-trash"></i></button>
      </td>
    </tr>
  `).join('');
  renderPagination('userPagination', page, Math.ceil(total / 20), loadUserList);
}

function showCreateUserModal() { openModal('createUserModal'); }

async function createUser() {
  const body = {
    username: document.getElementById('newUserName').value.trim(),
    password: document.getElementById('newUserPass').value,
    email: document.getElementById('newUserEmail').value.trim(),
    role: document.getElementById('newUserRole').value
  };
  if (!body.username || !body.password) return showToast('请输入用户名和密码', 'error');
  try {
    await apiPost('/users', body);
    closeModal('createUserModal');
    showToast('用户创建成功', 'success');
    loadUserList();
  } catch (e) { showToast('创建失败: ' + e.message, 'error'); }
}

async function deleteUser(id, username) {
  if (!confirm(`确定要删除用户 "${username}" 吗？此操作不可恢复！`)) return;
  try {
    await apiDelete(`/users/${id}`);
    showToast('用户已删除', 'success');
    loadUserList();
  } catch (e) { showToast('删除失败: ' + e.message, 'error'); }
}

// ==================== AUDIT ====================
async function loadAuditLogs(page = 1) {
  try {
    const data = await apiGet(`/audit?page=${page}&size=50`);
    renderAuditTable(data.data, data.total, data.page);
  } catch (e) { showToast('加载审计日志失败: ' + e.message, 'error'); }
}

function renderAuditTable(logs, total, page) {
  if (!logs || logs.length === 0) {
    document.getElementById('auditTableBody').innerHTML = '<tr><td colspan="6" style="text-align:center;padding:40px;color:#999">暂无审计日志</td></tr>';
    return;
  }
  document.getElementById('auditTableBody').innerHTML = logs.map(l => `
    <tr>
      <td><span style="font-size:12px">${formatTime(l.created_at)}</span></td>
      <td>${l.username || '-'}</td>
      <td><code style="font-size:11px">${l.action}</code></td>
      <td>${l.resource_type || '-'}</td>
      <td>${l.ip}</td>
      <td><span style="font-size:11px">${(l.detail || '-').substring(0, 80)}</span></td>
    </tr>
  `).join('');
  renderPagination('auditPagination', page, Math.ceil(total / 50), loadAuditLogs);
}

// ==================== SETTINGS ====================
async function loadSettings() {
  try {
    const cfg = await apiGet('/scans/config');
    document.getElementById('cfgScanEnabled').value = cfg.enabled ? 'true' : 'false';
    document.getElementById('cfgScanInterval').value = cfg.interval_seconds;
    document.getElementById('cfgWorkerCount').value = cfg.worker_count;
    document.getElementById('cfgTimeoutMs').value = cfg.timeout_ms;
    document.getElementById('cfgOfflineThreshold').value = cfg.offline_threshold;
  } catch (e) {}

  try {
    const info = await apiGet('/system/info');
    document.getElementById('systemInfo').innerHTML = `
      <div class="detail-grid">
        <div class="detail-item"><div class="detail-label">Go 版本</div><div class="detail-value">${info.go_version}</div></div>
        <div class="detail-item"><div class="detail-label">Goroutines</div><div class="detail-value">${info.goroutines}</div></div>
        <div class="detail-item"><div class="detail-label">CPU 核心</div><div class="detail-value">${info.cpu_cores}</div></div>
        <div class="detail-item"><div class="detail-label">内存使用</div><div class="detail-value">${info.memory_alloc} MB</div></div>
        <div class="detail-item"><div class="detail-label">运行时间</div><div class="detail-value">${formatDuration(info.uptime_seconds)}</div></div>
        <div class="detail-item"><div class="detail-label">服务器时间</div><div class="detail-value">${info.server_time}</div></div>
      </div>
    `;
  } catch (e) {}
}

async function saveScanConfig() {
  const body = {
    enabled: document.getElementById('cfgScanEnabled').value === 'true',
    interval_seconds: parseInt(document.getElementById('cfgScanInterval').value),
    worker_count: parseInt(document.getElementById('cfgWorkerCount').value),
    timeout_ms: parseInt(document.getElementById('cfgTimeoutMs').value),
    offline_threshold: parseInt(document.getElementById('cfgOfflineThreshold').value),
    subnets: JSON.stringify(['auto']),
    methods: JSON.stringify(['arp', 'ping', 'tcp']),
    tcp_ports: JSON.stringify([22, 80, 443, 8080, 3389])
  };
  try {
    await apiPut('/scans/config', body);
    showToast('配置已保存', 'success');
  } catch (e) { showToast('保存失败: ' + e.message, 'error'); }
}

// ==================== UTILS ====================
function formatTime(t) {
  if (!t) return '-';
  const d = new Date(t);
  return d.toLocaleString('zh-CN');
}

function formatTimeShort(t) {
  if (!t) return '-';
  const d = new Date(t);
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function formatDuration(seconds) {
  if (!seconds || seconds < 0) return '-';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return `${h}时${m}分`;
  if (m > 0) return `${m}分${s}秒`;
  return `${s}秒`;
}

function formatRate(bytesPerSec) {
  if (!bytesPerSec || bytesPerSec < 0) return '0 B/s';
  if (bytesPerSec > 1024 * 1024) return (bytesPerSec / 1024 / 1024).toFixed(2) + ' MB/s';
  if (bytesPerSec > 1024) return (bytesPerSec / 1024).toFixed(2) + ' KB/s';
  return bytesPerSec.toFixed(2) + ' B/s';
}

function statusLabel(status) {
  const map = {
    online: '在线', offline: '离线', warning: '异常',
    unknown: '未知', ignored: '已忽略', blocked: '已屏蔽'
  };
  return map[status] || status;
}

function renderPagination(id, current, total, callback) {
  if (total <= 1) {
    document.getElementById(id).innerHTML = '';
    return;
  }
  let html = `<button ${current <= 1 ? 'disabled' : ''} onclick="${callback.name}(${current - 1})">上一页</button>`;
  const start = Math.max(1, current - 2);
  const end = Math.min(total, current + 2);
  for (let i = start; i <= end; i++) {
    html += `<button class="${i === current ? 'active' : ''}" onclick="${callback.name}(${i})">${i}</button>`;
  }
  html += `<button ${current >= total ? 'disabled' : ''} onclick="${callback.name}(${current + 1})">下一页</button>`;
  html += `<span>共 ${total} 页</span>`;
  document.getElementById(id).innerHTML = html;
}
