import apiClient from './client'

export interface Device {
  id: number
  device_id: string
  device_name: string
  hostname: string
  ip_address: string
  mac_address: string
  board_type: string
  arch: string
  os_name: string
  os_version: string
  kernel_version: string
  bsp_version: string
  agent_version: string
  status: string
  last_seen: string
  created_at: string
  updated_at: string
}

export interface DeviceMetric {
  cpu_usage: number
  memory_usage: number
  disk_usage: number
  temperature: number
  load_avg_1m: number
  network_rx_bytes: number
  network_tx_bytes: number
  created_at: string
}

export interface DeviceListResponse {
  devices: Device[]
  total: number
  page: number
  page_size: number
  online_count: number
  total_count: number
}

export interface DashboardStats {
  online_count: number
  offline_count: number
  total_count: number
}

export interface CommandTask {
  task_id: string
  device_id: string
  action: string
  params: string
  status: string
  stdout: string
  stderr: string
  exit_code: number
  created_by: string
  created_at: string
}

export const deviceApi = {
  getDevices(page = 1, pageSize = 20) {
    return apiClient.get<DeviceListResponse>('/devices', { params: { page, page_size: pageSize } })
  },
  getDevice(deviceId: string) {
    return apiClient.get<{ device: Device; metrics: DeviceMetric }>(`/devices/${deviceId}`)
  },
  getDashboardStats() {
    return apiClient.get<DashboardStats>('/devices/stats')
  },
  getDeviceMetrics(deviceId: string, limit = 60) {
    return apiClient.get(`/devices/${deviceId}/metrics`, { params: { limit } })
  },
  getCommands(deviceId: string, limit = 20) {
    return apiClient.get<{ commands: CommandTask[]; limit: number }>(`/devices/${deviceId}/commands`, { params: { limit } })
  },
  executeCommand(deviceId: string, action: string, params: Record<string, any> = {}) {
    return apiClient.post(`/devices/${deviceId}/commands`, { action, params })
  },
  getCommandActions() {
    return apiClient.get<{ actions: any[] }>('/commands/actions')
  },
  getAlerts(page = 1, pageSize = 20) {
    return apiClient.get('/alerts', { params: { page, page_size: pageSize } })
  },
  resolveAlert(alertId: number) {
    return apiClient.put(`/alerts/${alertId}/resolve`)
  },
}

export const authApi = {
  login(username: string, password: string) {
    return apiClient.post('/auth/login', { username, password })
  },
  me() {
    return apiClient.get('/auth/me')
  },
}