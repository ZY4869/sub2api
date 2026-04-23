import { describe, expect, it } from 'vitest'
import { OPENAI_WS_MODE_OFF, OPENAI_WS_MODE_PASSTHROUGH } from '@/utils/openaiWsMode'
import { buildAnthropicExtra, buildOpenAIExtra } from '@/utils/accountCreateExtras'

describe('accountCreateExtras', () => {
  describe('buildOpenAIExtra', () => {
    const baseOpenAIOptions = {
      openAIImageProtocolMode: 'native' as const,
      openAIImageCompatAllowed: true,
      includeOpenAIImageProtocolMode: true
    }

    it('keeps base untouched for non-openai platform', () => {
      const base = { foo: 1 }
      const out = buildOpenAIExtra({
        platform: 'anthropic',
        accountCategory: 'oauth-based',
        base,
        openaiOAuthResponsesWebSocketV2Mode: OPENAI_WS_MODE_OFF,
        openaiAPIKeyResponsesWebSocketV2Mode: OPENAI_WS_MODE_OFF,
        openaiPassthroughEnabled: false,
        codexCLIOnlyEnabled: false,
        ...baseOpenAIOptions
      })
      expect(out).toBe(base)
    })

    it('adds oauth ws fields and codex flag', () => {
      const out = buildOpenAIExtra({
        platform: 'openai',
        accountCategory: 'oauth-based',
        base: { responses_websockets_v2_enabled: true, openai_ws_enabled: true },
        openaiOAuthResponsesWebSocketV2Mode: OPENAI_WS_MODE_PASSTHROUGH,
        openaiAPIKeyResponsesWebSocketV2Mode: OPENAI_WS_MODE_OFF,
        openaiPassthroughEnabled: false,
        codexCLIOnlyEnabled: true,
        ...baseOpenAIOptions
      })

      expect(out).toMatchObject({
        openai_oauth_responses_websockets_v2_mode: OPENAI_WS_MODE_PASSTHROUGH,
        openai_oauth_responses_websockets_v2_enabled: true,
        codex_cli_only: true,
        image_protocol_mode: 'native',
        image_compat_allowed: true
      })
      expect(out).not.toHaveProperty('responses_websockets_v2_enabled')
      expect(out).not.toHaveProperty('openai_ws_enabled')
    })

    it('treats copilot as openai-family for extra flags', () => {
      const out = buildOpenAIExtra({
        platform: 'copilot',
        accountCategory: 'oauth-based',
        base: {},
        openaiOAuthResponsesWebSocketV2Mode: OPENAI_WS_MODE_PASSTHROUGH,
        openaiAPIKeyResponsesWebSocketV2Mode: OPENAI_WS_MODE_OFF,
        openaiPassthroughEnabled: false,
        codexCLIOnlyEnabled: true,
        ...baseOpenAIOptions
      })

      expect(out).toMatchObject({
        openai_oauth_responses_websockets_v2_mode: OPENAI_WS_MODE_PASSTHROUGH,
        openai_oauth_responses_websockets_v2_enabled: true,
        codex_cli_only: true
      })
    })

    it('sets openai_passthrough when enabled', () => {
      const out = buildOpenAIExtra({
        platform: 'openai',
        accountCategory: 'apikey',
        base: {},
        openaiOAuthResponsesWebSocketV2Mode: OPENAI_WS_MODE_OFF,
        openaiAPIKeyResponsesWebSocketV2Mode: OPENAI_WS_MODE_OFF,
        openaiPassthroughEnabled: true,
        codexCLIOnlyEnabled: false,
        ...baseOpenAIOptions
      })

      expect(out).toMatchObject({ openai_passthrough: true })
    })

    it('can skip direct image protocol keys for protocol-gateway flows', () => {
      const out = buildOpenAIExtra({
        platform: 'openai',
        accountCategory: 'apikey',
        base: {},
        openaiOAuthResponsesWebSocketV2Mode: OPENAI_WS_MODE_OFF,
        openaiAPIKeyResponsesWebSocketV2Mode: OPENAI_WS_MODE_OFF,
        openaiPassthroughEnabled: false,
        codexCLIOnlyEnabled: false,
        openAIImageProtocolMode: 'compat',
        openAIImageCompatAllowed: true,
        includeOpenAIImageProtocolMode: false
      })

      expect(out).not.toHaveProperty('image_protocol_mode')
      expect(out).not.toHaveProperty('image_compat_allowed')
    })
  })

  describe('buildAnthropicExtra', () => {
    it('sets anthropic_passthrough only for anthropic apikey', () => {
      const out = buildAnthropicExtra({
        platform: 'anthropic',
        accountCategory: 'apikey',
        base: {},
        anthropicPassthroughEnabled: true
      })
      expect(out).toMatchObject({ anthropic_passthrough: true })

      const untouched = buildAnthropicExtra({
        platform: 'openai',
        accountCategory: 'apikey',
        base: { anthropic_passthrough: true },
        anthropicPassthroughEnabled: false
      })
      expect(untouched).toEqual({ anthropic_passthrough: true })
    })
  })
})
