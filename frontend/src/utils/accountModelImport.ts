export type AccountModelImportTranslate = (
  key: string,
  named?: Record<string, unknown>
) => string

interface AccountModelImportErrorLike {
  message?: unknown
  response?: {
    data?: {
      detail?: unknown
      message?: unknown
    }
  }
}

function extractAccountModelImportErrorMessage(error: unknown): string {
  const err = (error || {}) as AccountModelImportErrorLike
  const detail = typeof err.response?.data?.detail === 'string'
    ? err.response.data.detail.trim()
    : ''
  if (detail) {
    return detail
  }

  const responseMessage = typeof err.response?.data?.message === 'string'
    ? err.response.data.message.trim()
    : ''
  if (responseMessage) {
    return responseMessage
  }

  return typeof err.message === 'string' ? err.message.trim() : ''
}

export function resolveAccountModelImportErrorMessage(
  t: AccountModelImportTranslate,
  error: unknown
): string {
  const message = extractAccountModelImportErrorMessage(error)
  if (message.includes('does not support real model probing') || message.includes('does not support model import')) {
    return t('admin.accounts.modelImportUnsupported')
  }
  return message || t('admin.accounts.modelImportFailed')
}
