const API_BASE = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080') + '/api/v1'

export interface APIEnvelope<T = unknown> {
  code: number
  message: string
  data: T
}

export function getAccessToken(): string {
  return localStorage.getItem('welfare_access_token') || ''
}

export function setAccessToken(token: string): void {
  localStorage.setItem('welfare_access_token', token)
}

export function clearAccessToken(): void {
  localStorage.removeItem('welfare_access_token')
}

export async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers || {})
  headers.set('Content-Type', 'application/json')
  const token = getAccessToken()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  const resp = await fetch(`${API_BASE}${path}`, { ...init, headers })
  const body = (await resp.json()) as APIEnvelope<T>
  if (!resp.ok || body.code !== 0) {
    throw new Error(body.message || `request failed: ${resp.status}`)
  }
  return body.data
}

export function linuxdoLoginURL(redirectPath: string): string {
  const q = new URLSearchParams({ redirect: redirectPath })
  return `${API_BASE}/auth/linuxdo/start?${q.toString()}`
}
