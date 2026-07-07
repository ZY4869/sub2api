import { describe, expect, it } from 'vitest'
import { getModelScopeWhitelistGroups } from '../accountModelScopeCandidates'

describe('accountModelScopeCandidates', () => {
  it('sorts whitelist entries by final provider display name instead of ui_priority', () => {
    const result = getModelScopeWhitelistGroups([
      {
        id: 'claude-sonnet-4-5',
        display_name: 'Claude Sonnet 4.5',
        provider: 'anthropic',
        platforms: ['anthropic'],
        protocol_ids: [],
        aliases: [],
        pricing_lookup_ids: [],
        modalities: ['text'],
        capabilities: [],
        ui_priority: 999,
        exposed_in: ['whitelist']
      },
      {
        id: 'claude-haiku-4-5',
        display_name: 'Claude Haiku 4.5',
        provider: 'anthropic',
        platforms: ['anthropic'],
        protocol_ids: [],
        aliases: [],
        pricing_lookup_ids: [],
        modalities: ['text'],
        capabilities: [],
        ui_priority: 1,
        exposed_in: ['whitelist']
      }
    ] as any, {
      platform: 'anthropic',
      selectedModelIds: new Set<string>(),
      query: '',
      showAllModels: true
    })

    expect(result.providerGroups[0]?.entries.map((entry) => entry.id)).toEqual([
      'claude-haiku-4-5',
      'claude-sonnet-4-5'
    ])
  })

  it('shows manually added registry models as searchable candidates without selecting them by default', () => {
    const registryModels = [
      {
        id: 'gpt-5.4-mini',
        display_name: 'GPT-5.4 Mini',
        provider: 'openai',
        platforms: ['openai'],
        protocol_ids: [],
        aliases: [],
        pricing_lookup_ids: [],
        modalities: ['text'],
        capabilities: [],
        ui_priority: 100,
        exposed_in: ['whitelist']
      }
    ] as any

    const defaultResult = getModelScopeWhitelistGroups(registryModels, {
      platform: 'openai',
      selectedModelIds: new Set<string>(),
      query: '',
      showAllModels: false
    })
    expect(defaultResult.providerGroups).toEqual([])

    const searchResult = getModelScopeWhitelistGroups(registryModels, {
      platform: 'openai',
      selectedModelIds: new Set<string>(),
      query: '5.4-mini',
      showAllModels: false
    })
    expect(searchResult.providerGroups[0]?.entries.map((entry) => entry.id)).toEqual(['gpt-5.4-mini'])
    expect(searchResult.providerGroups[0]?.selectedCount).toBe(0)
  })
})
