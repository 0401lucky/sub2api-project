const FALLBACK_API_BASE = 'http://localhost:8080'
const GENERIC_NETWORK_ERROR_MESSAGE = '网络异常，请稍后重试'
const GENERIC_SERVER_ERROR_MESSAGE = '服务暂时不可用，请稍后重试'
const GENERIC_AUTH_EXPIRED_MESSAGE = '登录状态已过期，请重新登录'
const GENERIC_REQUEST_ERROR_MESSAGE = '请求失败，请稍后重试'

export const AUTH_EXPIRED_EVENT = 'welfare-auth-expired'

export interface APIEnvelope<T = unknown> {
  code: number
  message: string
  data: T
}

function isLocalHostname(hostname: string): boolean {
  const normalized = hostname.toLowerCase()
  return normalized === 'localhost' || normalized === '127.0.0.1' || normalized === '::1'
}

function resolveApiBase(): string {
  const rawBase = String(import.meta.env.VITE_API_BASE_URL || FALLBACK_API_BASE).trim()
  try {
    const url = new URL(rawBase, window.location.origin)
    if (!isLocalHostname(url.hostname)) {
      url.protocol = 'https:'
    }
    return url.toString().replace(/\/+$/, '')
  } catch {
    return FALLBACK_API_BASE
  }
}

const API_BASE = `${resolveApiBase()}/api/v1`

function toErrorMessage(status: number, backendMessage: string): string {
  if (status >= 500) {
    return GENERIC_SERVER_ERROR_MESSAGE
  }
  return backendMessage || `请求失败（${status}）`
}

async function parseResponseEnvelope<T>(resp: Response): Promise<APIEnvelope<T> | null> {
  try {
    const parsed = (await resp.json()) as Partial<APIEnvelope<T>>
    if (!parsed || typeof parsed !== 'object') {
      return null
    }
    return {
      code: typeof parsed.code === 'number' ? parsed.code : -1,
      message: typeof parsed.message === 'string' ? parsed.message : '',
      data: parsed.data as T
    }
  } catch {
    return null
  }
}

export async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers ?? {})
  if (init?.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  let resp: Response
  try {
    resp = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers,
      credentials: init?.credentials ?? 'include'
    })
  } catch {
    throw new Error(GENERIC_NETWORK_ERROR_MESSAGE)
  }

  const body = await parseResponseEnvelope<T>(resp)
  const backendMessage = body?.message || ''

  if (resp.status === 401 || body?.code === 401) {
    window.dispatchEvent(new CustomEvent(AUTH_EXPIRED_EVENT))
    throw new Error(GENERIC_AUTH_EXPIRED_MESSAGE)
  }

  if (!resp.ok) {
    throw new Error(toErrorMessage(resp.status, backendMessage))
  }

  if (!body) {
    throw new Error(GENERIC_REQUEST_ERROR_MESSAGE)
  }

  if (body.code !== 0) {
    throw new Error(toErrorMessage(resp.status, body.message || GENERIC_REQUEST_ERROR_MESSAGE))
  }

  return body.data
}

export function linuxdoLoginURL(redirectPath: string): string {
  const q = new URLSearchParams({ redirect: redirectPath })
  return `${API_BASE}/auth/linuxdo/start?${q.toString()}`
}
