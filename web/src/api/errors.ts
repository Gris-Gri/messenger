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

export async function parseApiError(response: Response): Promise<ApiError> {
  try {
    const data = (await response.json()) as {
      error?: { code?: string; message?: string }
    }
    const err = data.error
    if (err?.code) {
      return new ApiError(
        response.status,
        err.code,
        err.message ?? response.statusText,
      )
    }
  } catch {
    // ignore JSON parse errors
  }
  return new ApiError(response.status, 'unknown', response.statusText)
}
