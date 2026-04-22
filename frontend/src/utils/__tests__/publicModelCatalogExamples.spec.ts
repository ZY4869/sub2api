import { describe, expect, it } from 'vitest'

import type { PublicModelCatalogDetailResponse } from '@/api/meta'

import { buildPublicModelExample } from '../publicModelCatalogExamples'

function buildDetail(
  model: string,
  protocol: string,
  overrideID: 'image-generation' | 'image-generation-tool'
): PublicModelCatalogDetailResponse {
  return {
    item: {
      model,
      provider: protocol,
      request_protocols: [protocol],
      mode: overrideID === 'image-generation-tool' ? 'chat' : 'image',
      currency: 'USD',
      price_display: { primary: [] },
      multiplier_summary: { enabled: false, kind: 'disabled' }
    },
    example_source: 'override_template',
    example_protocol: protocol,
    example_override_id: overrideID
  }
}

function getTabCode(detail: PublicModelCatalogDetailResponse, label: 'Python' | 'REST' | 'JavaScript'): string {
  const result = buildPublicModelExample(detail, 'sk-test', 'https://example.com')
  const code = result.group?.tabs.find((tab) => tab.label === label)?.code
  expect(code).toBeTruthy()
  return code ?? ''
}

describe('publicModelCatalogExamples', () => {
  it('uses the public OpenAI images endpoint for native OpenAI image models', () => {
    const code = getTabCode(buildDetail('gpt-image-2', 'openai', 'image-generation'), 'REST')

    expect(code).toContain('/v1/images/generations')
    expect(code).not.toContain('/grok/v1/images/generations')
  })

  it('uses the explicit Grok images endpoint for native Grok image models', () => {
    const code = getTabCode(buildDetail('grok-imagine-1.0', 'grok', 'image-generation'), 'REST')

    expect(code).toContain('/grok/v1/images/generations')
    expect(code).not.toContain('/v1beta/openai/images/generations')
  })

  it('uses the Gemini OpenAI compat images endpoint for native Gemini image models', () => {
    const code = getTabCode(buildDetail('gemini-2.5-flash-image', 'gemini', 'image-generation'), 'REST')

    expect(code).toContain('/v1beta/openai/images/generations')
    expect(code).not.toContain('/grok/v1/images/generations')
  })

  it('uses responses image_generation tool examples for tool-only image models', () => {
    const code = getTabCode(buildDetail('gpt-5.4-mini', 'openai', 'image-generation-tool'), 'JavaScript')

    expect(code).toContain('/v1/responses')
    expect(code).toContain('image_generation')
    expect(code).toContain('gpt-image-2')
  })
})
