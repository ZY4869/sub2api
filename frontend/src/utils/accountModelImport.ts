export type AccountModelImportTranslate = (
  key: string,
  named?: Record<string, unknown>
) => string

interface AccountModelImportResultLike {
  imported_count?: number
  probe_source?: string
  probe_notice?: string
}

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

export function resolveAccountModelImportProbeNoticeMessage(
  t: AccountModelImportTranslate,
  result: AccountModelImportResultLike | null | undefined
): string {
  const probeNotice = typeof result?.probe_notice === 'string'
    ? result.probe_notice.trim()
    : ''
  if (probeNotice) {
    return probeNotice
  }

  if (result?.probe_source === 'gemini_cli_default_fallback') {
    return t('admin.accounts.modelImportGeminiFallback', {
      count: Math.max(0, Number(result.imported_count) || 0)
    })
  }

  return ''
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
