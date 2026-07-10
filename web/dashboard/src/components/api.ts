import type { AuthResponse, Team, User, Workspace } from '../types'

const API_BASE = import.meta.env.DEV ? '' : import.meta.env.PUBLIC_API_URL

function getToken(): string | null {
  return localStorage.getItem('nimbuscore_token')
}

function userId(): string | null {
  return localStorage.getItem('nimbuscore_user_id')
}

function setAuth(auth: AuthResponse) {
  localStorage.setItem('nimbuscore_token', auth.token)
  localStorage.setItem('nimbuscore_user_id', auth.user_id)
}

function clearAuth() {
  localStorage.removeItem('nimbuscore_token')
  localStorage.removeItem('nimbuscore_user_id')
}

function isAuthenticated(): boolean {
  return !!getToken()
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>)
  }
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch(`${API_BASE}${path}`, { ...options, headers })

  if (res.status === 401) {
    clearAuth()
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }

  if (!res.ok) {
    const text = await res.text()
    throw new Error(text || `HTTP ${res.status}`)
  }

  if (res.status === 204) return undefined as T
  return res.json()
}

export const api = {
  getToken,
  userId,
  setAuth,
  clearAuth,
  isAuthenticated,

  login() {
    window.location.href = `${API_BASE}/auth/login`
  },

  async handleCallback(code: string): Promise<AuthResponse> {
    const auth = await request<AuthResponse>(`/auth/callback?code=${code}`)
    setAuth(auth)
    return auth
  },

  me: {
    get: () => request<User>('/api/users/me')
  },

  users: {
    list: () => request<User[]>('/api/users'),
    get: (id: string) => request<User>(`/api/users/${id}`)
  },

  teams: {
    list: () => request<Team[]>('/api/teams'),
    create: (data: { name: string; slug?: string }) =>
      request<Team>('/api/teams', {
        method: 'POST',
        body: JSON.stringify(data),
        headers: { 'Content-Type': 'application/json' }
      }),
    get: (id: string) => request<Team>(`/api/teams/${id}`),
    update: (id: string, data: { name: string; slug?: string }) =>
      request<Team>(`/api/teams/${id}`, {
        method: 'PATCH',
        body: JSON.stringify(data),
        headers: { 'Content-Type': 'application/json' }
      }),
    delete: (id: string) =>
      request<void>(`/api/teams/${id}`, { method: 'DELETE' })
  },

  workspaces: {
    list: () => request<Workspace[]>('/api/workspaces'),
    get: (id: string) => request<Workspace>(`/api/workspaces/${id}`),
    create: (data: Partial<Workspace>) =>
      request<Workspace>('/api/workspaces', {
        method: 'POST',
        body: JSON.stringify(data),
        headers: { 'Content-Type': 'application/json' }
      }),
    delete: (id: string) =>
      request<void>(`/api/workspaces/${id}`, { method: 'DELETE' }),
    start: (id: string) =>
      request<void>(`/api/workspaces/${id}/start`, { method: 'POST' }),
    stop: (id: string) =>
      request<void>(`/api/workspaces/${id}/stop`, { method: 'POST' })
  }
}
