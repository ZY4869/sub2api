import { describe, expect, it } from 'vitest'
import { resolveAccountModelImportErrorMessage } from '@/utils/accountModelImport'

const t = (key: string) => key

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
