import { describe, expect, it } from 'vitest'
import {
  resolveAccountModelImportErrorMessage,
  resolveAccountModelImportProbeNoticeMessage
} from '@/utils/accountModelImport'

const t = (key: string, named?: Record<string, unknown>) => (
  named ? `${key}:${JSON.stringify(named)}` : key
)

describe('resolveAccountModelImportErrorMessage', () => {
  it('prefers response detail over generic error message', () => {
    const message = resolveAccountModelImportErrorMessage(t, {
      message: 'Request failed with status code 400',
      response: {
        data: {
          detail: 'account must be active to import models',
          message: 'bad request'
        }
      }
    })

    expect(message).toBe('account must be active to import models')
  })

  it('maps unsupported probing errors to localized copy', () => {
    const message = resolveAccountModelImportErrorMessage(t, {
      response: {
        data: {
          message: 'current Sora account type does not support real model probing'
        }
      }
    })

    expect(message).toBe('admin.accounts.modelImportUnsupported')
  })

  it('falls back to generic failure copy when no message exists', () => {
    expect(resolveAccountModelImportErrorMessage(t, {})).toBe('admin.accounts.modelImportFailed')
  })
})

describe('resolveAccountModelImportProbeNoticeMessage', () => {
  it('prefers explicit probe notice from backend', () => {
    expect(resolveAccountModelImportProbeNoticeMessage(t, {
      imported_count: 6,
      probe_source: 'gemini_cli_default_fallback',
      probe_notice: 'AI Studio model listing lacks required scopes; imported Gemini CLI default models instead'
    })).toBe('AI Studio model listing lacks required scopes; imported Gemini CLI default models instead')
  })

  it('maps Gemini CLI fallback source to localized copy', () => {
    expect(resolveAccountModelImportProbeNoticeMessage(t, {
      imported_count: 3,
      probe_source: 'gemini_cli_default_fallback'
    })).toBe('admin.accounts.modelImportGeminiFallback:{"count":3}')
  })

  it('returns empty string for upstream probe results', () => {
    expect(resolveAccountModelImportProbeNoticeMessage(t, {
      imported_count: 2,
      probe_source: 'upstream'
    })).toBe('')
  })
})
