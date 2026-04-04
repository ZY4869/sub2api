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
  getRequestTraceFinishReasonLabel,
  getRequestTraceStatusLabel,
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
})
