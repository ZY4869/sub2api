import { describe, expect, it } from 'vitest'
import {
  mergeAccountManualModelsIntoExtra,
  normalizeAccountManualModels,
  readAccountManualModelsFromExtra
} from '../accountProbeDraft'

describe('accountProbeDraft manual models', () => {
  it('normalizes provider metadata while preserving source protocol behavior', () => {
    expect(
      normalizeAccountManualModels(
        [
          {
            model_id: '  Custom-Model  ',
            request_alias: '  Alias  ',
            provider: '  Grok  ',
            source_protocol: 'openai'
          }
        ],
        true
      )
    ).toEqual([
      {
        model_id: 'Custom-Model',
        request_alias: 'Alias',
        provider: 'grok',
        source_protocol: 'openai'
      }
    ])
  })

  it('reads and writes manual model provider into extra payloads', () => {
    const extra = mergeAccountManualModelsIntoExtra(
      undefined,
      [
        {
          model_id: 'custom-model',
          request_alias: 'alias-model',
          provider: 'grok'
        }
      ],
      false
    )

    expect(extra).toEqual({
      manual_models: [
        {
          model_id: 'custom-model',
          request_alias: 'alias-model',
          provider: 'grok'
        }
      ]
    })

    expect(readAccountManualModelsFromExtra(extra, false)).toEqual([
      {
        model_id: 'custom-model',
        request_alias: 'alias-model',
        provider: 'grok'
      }
    ])
  })
})
