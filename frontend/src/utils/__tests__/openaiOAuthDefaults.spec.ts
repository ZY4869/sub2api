import { describe, expect, it } from 'vitest'
import {
  OPENAI_OAUTH_PRO_SPARK_MODEL,
  applyOpenAIOAuthDefaultModelState,
  resolveOpenAIOAuthDefaultAllowedModels,
} from '../openaiOAuthDefaults'

describe('openaiOAuthDefaults', () => {
  it('uses free defaults without gpt-image-2', () => {
    expect(resolveOpenAIOAuthDefaultAllowedModels({ planType: 'free' })).toEqual([
      'gpt-5.2',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-5.5',
    ])
  })

  it('keeps paid defaults for plus tiers', () => {
    expect(resolveOpenAIOAuthDefaultAllowedModels({ planType: 'plus' })).toEqual([
      'gpt-image-2',
      'gpt-5.2',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-5.5',
    ])
  })

  it('adds spark for pro tiers with a valid multiplier', () => {
    expect(
      resolveOpenAIOAuthDefaultAllowedModels({
        planType: 'pro',
        proMultiplier: 20,
      })
    ).toEqual([
      'gpt-image-2',
      'gpt-5.2',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-5.5',
      OPENAI_OAUTH_PRO_SPARK_MODEL,
    ])
  })

  it('preserves user customizations when defaults are already edited', () => {
    expect(
      applyOpenAIOAuthDefaultModelState({
        planType: 'free',
        proMultiplier: null,
        currentAllowedModels: ['gpt-5.4'],
        currentModelMappings: [{ from: 'friendly', to: 'gpt-5.4' }],
        modelRestrictionMode: 'mapping',
        userCustomized: true,
      })
    ).toEqual({
      allowedModels: ['gpt-5.4'],
      modelMappings: [{ from: 'friendly', to: 'gpt-5.4' }],
    })
  })
})
