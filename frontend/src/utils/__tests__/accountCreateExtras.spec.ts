import { describe, expect, it } from 'vitest'
import { OPENAI_WS_MODE_OFF, OPENAI_WS_MODE_PASSTHROUGH } from '@/utils/openaiWsMode'
import { buildAnthropicExtra, buildOpenAIExtra, buildSoraExtra } from '@/utils/accountCreateExtras'

describe('accountCreateExtras', () => {
  describe('buildOpenAIExtra', () => {
    it('keeps base untouched for non-openai platform', () => {
      const base = { foo: 1 }
      const out = buildOpenAIExtra({
        platform: 'anthropic',
        accountCategory: 'oauth-based',
        base,
        openaiOAuthResponsesWebSocketV2Mode: OPENAI_WS_MODE_OFF,
        openaiAPIKeyResponsesWebSocketV2Mode: OPENAI_WS_MODE_OFF,
        openaiPassthroughEnabled: false,
        codexCLIOnlyEnabled: false
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
        codexCLIOnlyEnabled: true
      })

      expect(out).toMatchObject({
        openai_oauth_responses_websockets_v2_mode: OPENAI_WS_MODE_PASSTHROUGH,
        openai_oauth_responses_websockets_v2_enabled: true,
        codex_cli_only: true
      })
      expect(out).not.toHaveProperty('responses_websockets_v2_enabled')
      expect(out).not.toHaveProperty('openai_ws_enabled')
    })

    it('sets openai_passthrough when enabled', () => {
      const out = buildOpenAIExtra({
        platform: 'openai',
        accountCategory: 'apikey',
        base: {},
        openaiOAuthResponsesWebSocketV2Mode: OPENAI_WS_MODE_OFF,
        openaiAPIKeyResponsesWebSocketV2Mode: OPENAI_WS_MODE_OFF,
        openaiPassthroughEnabled: true,
        codexCLIOnlyEnabled: false
      })

      expect(out).toMatchObject({ openai_passthrough: true })
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

  describe('buildSoraExtra', () => {
    it('links openai account id and strips openai-only flags', () => {
      const out = buildSoraExtra({
        base: {
          openai_passthrough: true,
          codex_cli_only: true,
          openai_oauth_responses_websockets_v2_mode: OPENAI_WS_MODE_PASSTHROUGH
        },
        linkedOpenAIAccountId: 123
      })

      expect(out).toMatchObject({ linked_openai_account_id: '123' })
      expect(out).not.toHaveProperty('openai_passthrough')
      expect(out).not.toHaveProperty('codex_cli_only')
      expect(out).not.toHaveProperty('openai_oauth_responses_websockets_v2_mode')
    })
  })
})

