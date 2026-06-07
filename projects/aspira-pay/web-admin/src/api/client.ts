// Aspira Pay V2 — API Client
// Simplified HTTP client for the admin dashboard.

const BASE_URL = '/api/v2'

let authToken = localStorage.getItem('auth_token') || ''

export function setToken(token: string) {
  authToken = token
  localStorage.setItem('auth_token', token)
}

export function clearToken() {
  authToken = ''
  localStorage.removeItem('auth_token')
}

async function request(path: string, options: RequestInit = {}): Promise<any> {
  const url = `${BASE_URL}${path}`
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  }

  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`
  }

  const res = await fetch(url, { ...options, headers })
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(error.error || `HTTP ${res.status}`)
  }

  return res.json()
}

export const api = {
  // Auth
  login: (username: string, password: string) =>
    request('/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) }),
  register: (username: string, email: string, password: string) =>
    request('/auth/register', { method: 'POST', body: JSON.stringify({ username, email, password }) }),

  // Dashboard
  getDashboard: () => request('/admin/dashboard'),

  // Payments
  getPayments: (params?: any) => {
    const qs = params ? '?' + new URLSearchParams(params).toString() : ''
    return request(`/payments${qs}`)
  },
  getPayment: (id: string) => request(`/payments/${id}`),
  createPayment: (data: any) =>
    request('/payments', { method: 'POST', body: JSON.stringify(data) }),

  // Users
  getUsers: () => request('/users'),
  getUser: (id: string) => request(`/users/${id}`),

  // FX
  getQuote: (data: any) =>
    request('/fx/quote', { method: 'POST', body: JSON.stringify(data) }),
  getRates: () => request('/fx/rates'),

  // Ledger
  getLedger: (paymentId: string) => request(`/ledger/${paymentId}`),

  // Settlement
  getBatches: () => request('/settlement/batches'),

  // Chain
  getBlocks: () => request('/chain/blocks'),
  getAudit: (paymentId: string) => request(`/chain/audit/${paymentId}`),
}
