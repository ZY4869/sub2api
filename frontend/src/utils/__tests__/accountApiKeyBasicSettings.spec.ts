import { describe, expect, it } from 'vitest'
import {
  resolveAccountApiKeyBaseUrlHintKey,
  resolveAccountApiKeyDefaultBaseUrl,
  resolveAccountApiKeyHintKey,
  resolveAccountApiKeyPlaceholder
} from '../accountApiKeyBasicSettings'

describe('accountApiKeyBasicSettings', () => {
  it('resolves platform defaults and placeholders', () => {
    expect(resolveAccountApiKeyDefaultBaseUrl('anthropic')).toBe('https://api.anthropic.com')
    expect(resolveAccountApiKeyDefaultBaseUrl('openai')).toBe('https://api.openai.com')
    expect(resolveAccountApiKeyDefaultBaseUrl('sora')).toBe('https://api.openai.com')
    expect(resolveAccountApiKeyDefaultBaseUrl('gemini')).toBe('https://generativelanguage.googleapis.com')
    expect(resolveAccountApiKeyDefaultBaseUrl('antigravity')).toBe('https://cloudcode-pa.googleapis.com')

    expect(resolveAccountApiKeyPlaceholder('anthropic')).toBe('sk-ant-...')
    expect(resolveAccountApiKeyPlaceholder('openai')).toBe('sk-proj-...')
    expect(resolveAccountApiKeyPlaceholder('sora')).toBe('sk-proj-...')
    expect(resolveAccountApiKeyPlaceholder('gemini')).toBe('AIza...')
    expect(resolveAccountApiKeyPlaceholder('antigravity')).toBe('sk-...')
  })

  it('resolves mode-aware hint keys', () => {
    expect(resolveAccountApiKeyBaseUrlHintKey('sora', 'create')).toBe('admin.accounts.soraUpstreamBaseUrlHint')
    expect(resolveAccountApiKeyBaseUrlHintKey('openai', 'edit')).toBe('admin.accounts.openai.baseUrlHint')
    expect(resolveAccountApiKeyBaseUrlHintKey('gemini', 'create')).toBe('admin.accounts.gemini.baseUrlHint')
    expect(resolveAccountApiKeyBaseUrlHintKey('antigravity', 'edit')).toBe('admin.accounts.upstream.baseUrlHint')

    expect(resolveAccountApiKeyHintKey('anthropic', 'create')).toBe('admin.accounts.apiKeyHint')
    expect(resolveAccountApiKeyHintKey('openai', 'create')).toBe('admin.accounts.openai.apiKeyHint')
    expect(resolveAccountApiKeyHintKey('gemini', 'create')).toBe('admin.accounts.gemini.apiKeyHint')
    expect(resolveAccountApiKeyHintKey('antigravity', 'create')).toBe('admin.accounts.upstream.apiKeyHint')
    expect(resolveAccountApiKeyHintKey('openai', 'edit')).toBe('admin.accounts.leaveEmptyToKeep')
  })
})
