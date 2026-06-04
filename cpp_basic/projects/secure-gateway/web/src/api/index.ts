import axios from 'axios'
import type { AxiosInstance, InternalAxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'

const BASE_URL = import.meta.env.DEV ? '/api/v1' : '/api/v1'

// Create Axios instance
const api: AxiosInstance = axios.create({
  baseURL: BASE_URL,
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Request interceptor - attach JWT token
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('access_token')
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// Response interceptor - handle auth errors
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config

    if (error.response?.status === 401 && !originalRequest._retry) {
      const refreshToken = localStorage.getItem('refresh_token')
      if (refreshToken) {
        originalRequest._retry = true
        try {
          const res = await axios.post(`${BASE_URL}/auth/refresh`, {
            refresh_token: refreshToken
          })
          if (res.data.code === 200) {
            const newToken = res.data.data.access_token
            localStorage.setItem('access_token', newToken)
            originalRequest.headers.Authorization = `Bearer ${newToken}`
            return api(originalRequest)
          }
        } catch {
          // Refresh failed, redirect to login
          localStorage.clear()
          window.location.hash = '#/login'
          ElMessage.error('登录已过期，请重新登录')
        }
      } else {
        localStorage.clear()
        window.location.hash = '#/login'
        ElMessage.error('登录已过期，请重新登录')
      }
    }

    const msg = error.response?.data?.message || error.message || '请求失败'
    if (error.response?.status !== 401) {
      ElMessage.error(msg)
    }
    return Promise.reject(error)
  }
)

// Auth APIs
export const authAPI = {
  login(username: string, password: string) {
    return api.post('/auth/login', { username, password })
  },
  refresh(refreshToken: string) {
    return api.post('/auth/refresh', { refresh_token: refreshToken })
  },
  logout() {
    return api.post('/auth/logout')
  },
  getProfile() {
    return api.get('/auth/profile')
  }
}

// Dashboard APIs
export const dashboardAPI = {
  getStats() {
    return api.get('/dashboard/stats')
  },
  getDistribution() {
    return api.get('/dashboard/distribution')
  },
  getTrafficHistory() {
    return api.get('/dashboard/traffic-history')
  },
  getTopUsers() {
    return api.get('/dashboard/top-users')
  }
}

// User APIs
export const userAPI = {
  list() {
    return api.get('/users')
  },
  create(data: { username: string; password: string; role?: string; quota_bytes?: number }) {
    return api.post('/users', data)
  },
  update(id: number, data: Record<string, any>) {
    return api.put(`/users/${id}`, data)
  },
  delete(id: number) {
    return api.delete(`/users/${id}`)
  }
}

// Node APIs
export const nodeAPI = {
  list() {
    return api.get('/nodes')
  },
  getMetrics(nodeId: string) {
    return api.get(`/nodes/${nodeId}/metrics`)
  },
  getMetricsHistory(nodeId: string) {
    return api.get(`/nodes/${nodeId}/metrics/history`)
  }
}

// Traffic APIs
export const trafficAPI = {
  list(params?: { user_id?: number; limit?: number }) {
    return api.get('/traffic', { params })
  },
  getSummary() {
    return api.get('/traffic/summary')
  }
}

// Audit log APIs
export const auditAPI = {
  list() {
    return api.get('/audit-logs')
  }
}

export default api