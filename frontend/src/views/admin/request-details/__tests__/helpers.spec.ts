import { describe, expect, it, vi } from 'vitest'

vi.mock('@/stores/modelRegistry', () => ({
  getModelRegistrySnapshot: () => ({
    etag: 'test',
    updated_at: '2026-04-04T00:00:00Z',
    models: [
      {
        id: 'claude-opus-4.1',
        display_name: 'Claude Opus 4.1',
        provider: 'anthropic',
        platforms: ['anthropic'],
        protocol_ids: ['claude-opus-4-1-20250805'],
        aliases: ['claude-opus-4-5'],
        pricing_lookup_ids: [],
        modalities: ['text'],
        capabilities: ['reasoning'],
        ui_priority: 0,
        exposed_in: ['whitelist']
      }
    ],
    presets: []
  })
}))

import {
  buildRequestTraceQuery,
  createDefaultRequestTraceFilter,
  getRequestTraceFinishReasonLabel,
  getRequestTraceStatusLabel,
  parseRequestTraceFilterFromQuery,
  resolveRequestTraceModelPresentation
} from '../helpers'

const translations: Record<string, string> = {
  'admin.requestDetails.presentation.status.success': '成功',
  'admin.requestDetails.presentation.finishReasons.stop': '正常结束'
}

const t = (key: string) => translations[key] ?? key

describe('request-details helpers', () => {
  it('translates known enum values and falls back to raw text for unknown values', () => {
    expect(getRequestTraceStatusLabel(t, 'success')).toBe('成功')
    expect(getRequestTraceFinishReasonLabel(t, 'stop')).toBe('正常结束')
    expect(getRequestTraceFinishReasonLabel(t, 'custom_reason')).toBe('custom_reason')
  })

  it('resolves model presentation by protocol id and aliases', () => {
    expect(resolveRequestTraceModelPresentation('claude-opus-4-1-20250805')).toEqual({
      modelId: 'claude-opus-4-1-20250805',
      displayName: 'Claude Opus 4.1',
      provider: 'anthropic'
    })

    expect(resolveRequestTraceModelPresentation('claude-opus-4-5')).toEqual({
      modelId: 'claude-opus-4-5',
      displayName: 'Claude Opus 4.1',
      provider: 'anthropic'
    })
  })

  it('parses and rebuilds gemini metadata filters', () => {
    const parsed = parseRequestTraceFilterFromQuery({
      gemini_surface: 'live',
      billing_rule_id: 'rule-live-1',
      probe_action: 'retry',
    })

    expect(parsed.gemini_surface).toBe('live')
    expect(parsed.billing_rule_id).toBe('rule-live-1')
    expect(parsed.probe_action).toBe('retry')

    const rebuilt = buildRequestTraceQuery({
      ...createDefaultRequestTraceFilter(),
      gemini_surface: 'compat',
      billing_rule_id: 'rule-compat-2',
      probe_action: 'blacklist',
    })

    expect(rebuilt.gemini_surface).toBe('compat')
    expect(rebuilt.billing_rule_id).toBe('rule-compat-2')
    expect(rebuilt.probe_action).toBe('blacklist')
  })
})
