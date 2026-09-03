import type {
  AccountPlatform,
  CodexImageToolPolicy,
  OpenAIImageProtocolMode
} from '@/types'
import type { AccountCategory } from '@/components/account/createAccountModal/accountCategory'
import { isOpenAIWSModeEnabled, type OpenAIWSMode } from '@/utils/openaiWsMode'

export type AnthropicAPIKeyAuthScheme = 'x_api_key' | 'authorization_bearer'
export const DEFAULT_CODEX_IMAGE_TOOL_POLICY: CodexImageToolPolicy = 'follow_channel'

export function normalizeCodexImageToolPolicy(value: unknown): CodexImageToolPolicy {
  switch (String(value || '').trim()) {
    case 'force_inject':
      return 'force_inject'
    case 'no_inject':
      return 'no_inject'
    case 'block_all':
      return 'block_all'
    default:
      return DEFAULT_CODEX_IMAGE_TOOL_POLICY
  }
}

export const normalizeAnthropicAPIKeyAuthScheme = (
  value: unknown
): AnthropicAPIKeyAuthScheme => {
  return value === 'authorization_bearer' ? 'authorization_bearer' : 'x_api_key'
}

export function buildOpenAIExtra(options: {
  platform: AccountPlatform
  accountCategory: AccountCategory
  base?: Record<string, unknown>
  openaiOAuthResponsesWebSocketV2Mode: OpenAIWSMode
  openaiAPIKeyResponsesWebSocketV2Mode: OpenAIWSMode
  openaiPassthroughEnabled: boolean
  openaiAlphaSearchEnabled?: boolean
  codexCLIOnlyEnabled: boolean
  codexImageToolPolicy?: CodexImageToolPolicy
  openAIImageProtocolMode: OpenAIImageProtocolMode
  openAIImageCompatAllowed: boolean
  includeOpenAIImageProtocolMode?: boolean
}): Record<string, unknown> | undefined {
  if (options.platform !== 'openai') {
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

  if (options.openaiAlphaSearchEnabled === false) {
    extra.openai_alpha_search_enabled = false
  } else {
    delete extra.openai_alpha_search_enabled
  }

  if (options.accountCategory === 'oauth-based' && options.codexCLIOnlyEnabled) {
    extra.codex_cli_only = true
  } else {
    delete extra.codex_cli_only
  }

  const codexImageToolPolicy = normalizeCodexImageToolPolicy(options.codexImageToolPolicy)
  if (options.accountCategory === 'oauth-based' && codexImageToolPolicy !== DEFAULT_CODEX_IMAGE_TOOL_POLICY) {
    extra.codex_image_tool_policy = codexImageToolPolicy
  } else {
    delete extra.codex_image_tool_policy
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
  anthropicAPIKeyAuthScheme?: AnthropicAPIKeyAuthScheme
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
  if (options.anthropicAPIKeyAuthScheme === 'authorization_bearer') {
    extra.anthropic_apikey_auth_scheme = 'authorization_bearer'
  } else {
    delete extra.anthropic_apikey_auth_scheme
  }

  return Object.keys(extra).length > 0 ? extra : undefined
}
