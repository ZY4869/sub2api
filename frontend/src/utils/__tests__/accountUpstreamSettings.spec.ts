import { describe, expect, it } from 'vitest'
import {
  ACCOUNT_UPSTREAM_API_KEY_PLACEHOLDER,
  ACCOUNT_UPSTREAM_BASE_URL_PLACEHOLDER,
  resolveAccountUpstreamApiKeyHintKey
} from '../accountUpstreamSettings'

describe('accountUpstreamSettings', () => {
  it('exposes stable placeholders', () => {
    expect(ACCOUNT_UPSTREAM_BASE_URL_PLACEHOLDER).toBe('https://cloudcode-pa.googleapis.com')
    expect(ACCOUNT_UPSTREAM_API_KEY_PLACEHOLDER).toBe('sk-...')
  })

  it('returns mode-aware api key hint keys', () => {
    expect(resolveAccountUpstreamApiKeyHintKey('create')).toBe('admin.accounts.upstream.apiKeyHint')
    expect(resolveAccountUpstreamApiKeyHintKey('edit')).toBe('admin.accounts.leaveEmptyToKeep')
  })
})
