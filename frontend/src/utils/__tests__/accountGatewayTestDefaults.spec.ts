import { describe, expect, it } from 'vitest'
import {
  ensureOpenAIOAuthGatewayTestDefaults,
  findDefaultGatewayTestModel,
  resolveCatalogTargetFromModel,
  resolveGatewayTestSelectedModelKey
} from '../accountGatewayTestDefaults'

const models = [
  {
    id: 'gpt-5.4-preview',
    canonical_id: 'gpt-5.4',
    provider: 'openai',
    source_protocol: 'openai',
    display_name: 'GPT-5.4'
  },
  {
    id: 'claude-sonnet-4-5-20250929',
    canonical_id: 'claude-sonnet-4.5',
    provider: 'anthropic',
    source_protocol: 'anthropic',
    display_name: 'Claude Sonnet 4.5'
  }
] as any

describe('accountGatewayTestDefaults', () => {
  it('fills OpenAI OAuth test target defaults without overwriting explicit values', () => {
    expect(ensureOpenAIOAuthGatewayTestDefaults()).toEqual({
      gateway_test_provider: 'openai',
      gateway_test_model_id: 'gpt-5.4'
    })
    expect(ensureOpenAIOAuthGatewayTestDefaults({
      gateway_test_provider: 'openai',
      gateway_test_model_id: 'gpt-4.1-mini'
    })).toEqual({
      gateway_test_provider: 'openai',
      gateway_test_model_id: 'gpt-4.1-mini'
    })
  })

  it('uses a single account stored default when the catalog entry is still available', () => {
    const selected = findDefaultGatewayTestModel([
      {
        extra: {
          gateway_test_provider: 'anthropic',
          gateway_test_model_id: 'claude-sonnet-4.5'
        }
      } as any
    ], models)

    expect(selected?.id).toBe('claude-sonnet-4-5-20250929')
    expect(resolveGatewayTestSelectedModelKey([
      {
        extra: {
          gateway_test_provider: 'anthropic',
          gateway_test_model_id: 'claude-sonnet-4.5'
        }
      } as any
    ], models)).toBe('anthropic::claude-sonnet-4-5-20250929')
  })

  it('only auto-selects shared defaults for multi-account batches', () => {
    const sharedKey = resolveGatewayTestSelectedModelKey([
      {
        extra: {
          gateway_test_provider: 'openai',
          gateway_test_model_id: 'gpt-5.4'
        }
      } as any,
      {
        extra: {
          gateway_test_provider: 'openai',
          gateway_test_model_id: 'gpt-5.4'
        }
      } as any
    ], models, false)
    const mismatchedKey = resolveGatewayTestSelectedModelKey([
      {
        extra: {
          gateway_test_provider: 'openai',
          gateway_test_model_id: 'gpt-5.4'
        }
      } as any,
      {
        extra: {
          gateway_test_provider: 'anthropic',
          gateway_test_model_id: 'claude-sonnet-4.5'
        }
      } as any
    ], models, false)

    expect(sharedKey).toBe('openai::gpt-5.4-preview')
    expect(mismatchedKey).toBe('')
  })

  it('derives catalog target payload from the selected model entry', () => {
    expect(resolveCatalogTargetFromModel(models[0] as any)).toEqual({
      sourceProtocol: 'openai',
      targetProvider: 'openai',
      targetModelId: 'gpt-5.4'
    })
  })
})
