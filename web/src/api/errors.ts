export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

const CODE_MESSAGES: Record<string, string> = {
  validation_error: 'Неверные данные',
  invalid_credentials: 'Неверные логин или пароль',
  unauthorized: 'Не авторизован',
  forbidden: 'Нет доступа',
  not_found: 'Не найдено',
  conflict: 'Конфликт',
  internal_error: 'Внутренняя ошибка сервера',
  network_error: 'Не удалось связаться с сервером',
  unknown: 'Ошибка запроса',
}

const STATUS_MESSAGES: Record<number, string> = {
  400: 'Неверные данные',
  401: 'Не авторизован',
  403: 'Нет доступа',
  404: 'Не найдено',
  409: 'Конфликт',
  500: 'Внутренняя ошибка сервера',
}

const NETWORK_MESSAGE_RE =
  /failed to fetch|networkerror|network request failed|load failed|fetch failed|aborted|timeout|econnrefused|enotfound/i

function hasCyrillic(text: string): boolean {
  return /[а-яё]/i.test(text)
}

export function messageForStatus(status: number, code?: string): string {
  if (code && CODE_MESSAGES[code]) {
    return CODE_MESSAGES[code]
  }
  return STATUS_MESSAGES[status] ?? 'Ошибка запроса'
}

export async function parseApiError(response: Response): Promise<ApiError> {
  try {
    const data = (await response.json()) as {
      error?: { code?: string; message?: string }
    }
    const err = data.error
    if (err?.code) {
      const message =
        err.message && hasCyrillic(err.message)
          ? err.message
          : messageForStatus(response.status, err.code)
      return new ApiError(response.status, err.code, message)
    }
  } catch {
    // ignore JSON parse errors
  }
  return new ApiError(
    response.status,
    'unknown',
    messageForStatus(response.status),
  )
}

export function isNetworkError(err: unknown): boolean {
  if (!(err instanceof Error)) {
    return false
  }
  if (err instanceof ApiError && err.code === 'network_error') {
    return true
  }
  if (err.name === 'TypeError' || err.name === 'NetworkError') {
    return true
  }
  return NETWORK_MESSAGE_RE.test(err.message)
}

/**
 * Сообщение для UI: API (русские тексты бэкенда), сеть, иначе fallback.
 * Англоязычные браузерные/HTTP statusText не показываем как есть.
 */
export function toUserMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError) {
    if (err.message && hasCyrillic(err.message)) {
      return err.message
    }
    return messageForStatus(err.status, err.code) || fallback
  }

  if (isNetworkError(err)) {
    return 'Не удалось связаться с сервером'
  }

  if (err instanceof Error) {
    const msg = err.message.trim()
    if (msg && hasCyrillic(msg)) {
      return msg
    }
  }

  return fallback
}
