import { beforeEach, describe, expect, it, vi } from 'vitest'

const snapshot = {
  etag: 'test-etag',
  updated_at: '2026-05-06T12:00:00Z',
  provider_labels: {},
  presets: [],
  models: [
    {
      id: 'deepseek-v4-pro',
      display_name: 'DeepSeek V4 Pro',
      provider: 'deepseek',
      platforms: ['deepseek'],
      protocol_ids: ['deepseek-v4-pro'],
      aliases: [],
      pricing_lookup_ids: ['deepseek-v4-pro'],
      context_window_tokens: 1_048_576,
      modalities: ['text'],
      capabilities: ['reasoning'],
      ui_priority: 1,
      exposed_in: ['runtime'],
    },
    {
      id: 'doubao-pro-256k',
      display_name: 'Doubao Pro 256K',
      provider: 'doubao',
      platforms: ['doubao'],
      protocol_ids: ['doubao-pro-256k'],
      aliases: [],
      pricing_lookup_ids: ['doubao-pro-256k'],
      context_window_tokens: 262_144,
      modalities: ['text'],
      capabilities: [],
      ui_priority: 1,
      exposed_in: ['runtime'],
    },
  ],
}

vi.mock('@/stores/modelRegistry', () => ({
  getModelRegistrySnapshot: () => snapshot,
}))

import {
  buildUsageModelLinePresentation,
  buildUsageModelPresentation,
  formatContextWindowLabel,
  normalizeUsageModelDisplayMode,
  normalizeUsageContextBadgeDisplayMode,
  resolveUsageContextBadge,
  resolveContextWindowTier,
} from '../usageModelPresentation'

describe('usageModelPresentation', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('normalizes invalid display modes to model_only', () => {
    expect(normalizeUsageModelDisplayMode('display_only')).toBe('display_only')
    expect(normalizeUsageModelDisplayMode('bad-mode')).toBe('model_only')
  })

  it('formats known context windows using marketing labels', () => {
    expect(formatContextWindowLabel(262_144)).toBe('256K')
    expect(formatContextWindowLabel(1_048_576)).toBe('1M')
    expect(formatContextWindowLabel(2_097_152)).toBe('2M')
  })

  it('maps context windows into visual tiers', () => {
    expect(resolveContextWindowTier(4_096)).toBe('4k')
    expect(resolveContextWindowTier(262_144)).toBe('200k')
    expect(resolveContextWindowTier(1_048_576)).toBe('1m')
  })

  it('shows request badge when million-context was requested and effective', () => {
    const presentation = buildUsageModelLinePresentation('deepseek-v4-pro', 'display_only', {
      million_context_requested: true,
      million_context_effective: true,
    })

    expect(presentation.primaryText).toBe('DeepSeek V4 Pro')
    expect(presentation.requestBadge?.label).toBe('1M')
    expect(presentation.requestBadge?.muted).toBe(false)
    expect(presentation.nativeContextBadge?.label).toBe('1M')
  })

  it('renders million-context requested but not effective as muted request badge', () => {
    const presentation = buildUsageModelLinePresentation('deepseek-v4-pro', 'display_only', {
      million_context_requested: true,
      million_context_effective: false,
    })

    expect(presentation.requestBadge?.label).toBe('usage.contextBadgeRequested1M')
    expect(presentation.requestBadge?.labelKey).toBe('usage.contextBadgeRequested1M')
    expect(presentation.requestBadge?.muted).toBe(true)
    expect(presentation.requestBadge?.titleKey).toBe('usage.contextBadgeRequested1MPending')
  })

  it('does not fall back to registry context as request badge when no million-context flag is present', () => {
    const presentation = buildUsageModelLinePresentation('doubao-pro-256k', 'display_and_model')

    expect(presentation.primaryText).toBe('Doubao Pro 256K')
    expect(presentation.secondaryText).toBe('doubao-pro-256k')
    expect(presentation.requestBadge).toBeNull()
    expect(presentation.nativeContextBadge?.label).toBe('256K')
    expect(presentation.nativeContextBadge?.title).toBeUndefined()
    expect(presentation.nativeContextBadge?.titleKey).toBe('usage.nativeContextTooltip')
    expect(presentation.nativeContextBadge?.titleParams).toEqual({ context: '256K' })
  })

  it('falls back to model id when display name lookup misses', () => {
    const presentation = buildUsageModelLinePresentation('unknown-model', 'display_only')

    expect(presentation.primaryText).toBe('unknown-model')
    expect(presentation.requestBadge).toBeNull()
    expect(presentation.nativeContextBadge).toBeNull()
  })

  it('builds upstream second line only when upstream model differs', () => {
    const presentation = buildUsageModelPresentation(
      {
        model: 'deepseek-v4-pro',
        upstream_model: 'doubao-pro-256k',
        million_context_requested: false,
        million_context_effective: false,
      },
      'display_and_model'
    )

    expect(presentation.requested.primaryText).toBe('DeepSeek V4 Pro')
    expect(presentation.upstream?.primaryText).toBe('Doubao Pro 256K')
  })

  it('normalizes context badge display mode and resolves badge by mode', () => {
    const presentation = buildUsageModelLinePresentation('doubao-pro-256k', 'display_only', {
      million_context_requested: false,
      million_context_effective: false,
    })

    expect(normalizeUsageContextBadgeDisplayMode('native_only')).toBe('native_only')
    expect(normalizeUsageContextBadgeDisplayMode('bad-mode')).toBe('request_only')
    expect(resolveUsageContextBadge(presentation, 'request_only')).toBeNull()
    expect(resolveUsageContextBadge(presentation, 'native_only')?.label).toBe('256K')
    expect(resolveUsageContextBadge(presentation, 'both')?.label).toBe('256K')
  })
})
