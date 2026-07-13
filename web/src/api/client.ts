import {
  clearTokens,
  getAccessToken,
  getRefreshToken,
  refresh,
} from './auth'
import { ApiError, parseApiError } from './errors'

const API_BASE = import.meta.env.VITE_API_URL ?? ''

type RequestOptions = Omit<RequestInit, 'headers'> & {
  headers?: Record<string, string>
  /** Внутренний флаг: не пытаться refresh при 401 */
  _retried?: boolean
  /** Не подставлять Authorization (для публичных ручек через client) */
  skipAuth?: boolean
}

let onSessionExpired: (() => void) | null = null
let refreshPromise: Promise<string> | null = null

export function configureClient(config: { onSessionExpired: () => void }): void {
  onSessionExpired = config.onSessionExpired
}

function tryRefresh(): Promise<string> {
  if (!getRefreshToken()) {
    return Promise.reject(
      new ApiError(401, 'unauthorized', 'Не авторизован'),
    )
  }
  if (!refreshPromise) {
    refreshPromise = refresh().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

function handleSessionExpired(): void {
  clearTokens()
  onSessionExpired?.()
}

export async function apiClient<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { _retried = false, skipAuth = false, headers = {}, ...init } = options

  const requestHeaders: Record<string, string> = {
    ...headers,
  }

  if (!skipAuth) {
    const token = getAccessToken()
    if (token) {
      requestHeaders.Authorization = `Bearer ${token}`
    }
  }

  if (init.body && !requestHeaders['Content-Type']) {
    requestHeaders['Content-Type'] = 'application/json'
  }

  let response: Response
  try {
    response = await fetch(`${API_BASE}${path}`, {
      ...init,
      headers: requestHeaders,
    })
  } catch {
    throw new ApiError(0, 'network_error', 'Не удалось связаться с сервером')
  }

  if (response.status === 401 && !skipAuth && !_retried) {
    try {
      await tryRefresh()
    } catch {
      handleSessionExpired()
      throw await parseApiError(response)
    }

    return apiClient<T>(path, { ...options, _retried: true })
  }

  if (response.status === 401 && !skipAuth) {
    handleSessionExpired()
    throw await parseApiError(response)
  }

  if (!response.ok) {
    throw await parseApiError(response)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return response.json() as Promise<T>
}
