import type { AccountPlatform, OpenAIImageProtocolMode } from '@/types'
import { isOpenAIWSModeEnabled, type OpenAIWSMode } from '@/utils/openaiWsMode'

export type AccountCategory = 'oauth-based' | 'apikey' | 'vertex_ai'

export function buildOpenAIExtra(options: {
  platform: AccountPlatform
  accountCategory: AccountCategory
  base?: Record<string, unknown>
  openaiOAuthResponsesWebSocketV2Mode: OpenAIWSMode
  openaiAPIKeyResponsesWebSocketV2Mode: OpenAIWSMode
  openaiPassthroughEnabled: boolean
  codexCLIOnlyEnabled: boolean
  openAIImageProtocolMode: OpenAIImageProtocolMode
  openAIImageCompatAllowed: boolean
  includeOpenAIImageProtocolMode?: boolean
}): Record<string, unknown> | undefined {
  if (options.platform !== 'openai' && options.platform !== 'copilot') {
    return options.base
  }

  const extra: Record<string, unknown> = { ...(options.base || {}) }
  if (options.accountCategory === 'oauth-based') {
    extra.openai_oauth_responses_websockets_v2_mode = options.openaiOAuthResponsesWebSocketV2Mode
    extra.openai_oauth_responses_websockets_v2_enabled = isOpenAIWSModeEnabled(
      options.openaiOAuthResponsesWebSocketV2Mode
    )
  } else {
    extra.openai_apikey_responses_websockets_v2_mode = options.openaiAPIKeyResponsesWebSocketV2Mode
    extra.openai_apikey_responses_websockets_v2_enabled = isOpenAIWSModeEnabled(
      options.openaiAPIKeyResponsesWebSocketV2Mode
    )
  }

  delete extra.responses_websockets_v2_enabled
  delete extra.openai_ws_enabled

  if (options.openaiPassthroughEnabled) {
    extra.openai_passthrough = true
  } else {
    delete extra.openai_passthrough
    delete extra.openai_oauth_passthrough
  }

  if (options.accountCategory === 'oauth-based' && options.codexCLIOnlyEnabled) {
    extra.codex_cli_only = true
  } else {
    delete extra.codex_cli_only
  }

  if (options.includeOpenAIImageProtocolMode !== false) {
    extra.image_protocol_mode = options.openAIImageProtocolMode
    if (options.accountCategory === 'oauth-based') {
      extra.image_compat_allowed = options.openAIImageCompatAllowed
    } else {
      delete extra.image_compat_allowed
    }
  } else {
    delete extra.image_protocol_mode
    delete extra.image_compat_allowed
  }

  return Object.keys(extra).length > 0 ? extra : undefined
}

export function buildAnthropicExtra(options: {
  platform: AccountPlatform
  accountCategory: AccountCategory
  base?: Record<string, unknown>
  anthropicPassthroughEnabled: boolean
}): Record<string, unknown> | undefined {
  if (options.platform !== 'anthropic' || options.accountCategory !== 'apikey') {
    return options.base
  }

  const extra: Record<string, unknown> = { ...(options.base || {}) }
  if (options.anthropicPassthroughEnabled) {
    extra.anthropic_passthrough = true
  } else {
    delete extra.anthropic_passthrough
  }

  return Object.keys(extra).length > 0 ? extra : undefined
}
