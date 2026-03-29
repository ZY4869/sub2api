import type { Ref } from 'vue'
import type { AddMethod } from '@/composables/useAccountOAuth'
import type {
  AccountPlatform,
  AccountType,
  GatewayAcceptedProtocol,
  GatewayClientProfile,
  GatewayClientRoute,
  GatewayProtocol
} from '@/types'
import { getModelsByPlatform } from '@/composables/useModelWhitelist'
import {
  DEFAULT_POOL_MODE_RETRY_COUNT,
  type AccountCustomErrorCodesState,
  type AccountPoolModeState,
  type ModelMapping
} from '@/utils/accountFormShared'
import {
  resetAccountCustomErrorCodesState,
  resetAccountPoolModeState
} from '@/utils/accountApiKeyAdvancedSettingsForm'
import { resolveAccountApiKeyDefaultBaseUrl } from '@/utils/accountApiKeyBasicSettings'
import { OPENAI_WS_MODE_OFF, type OpenAIWSMode } from '@/utils/openaiWsMode'
import type { GeminiOAuthType } from '@/utils/geminiAccount'

interface CreateAccountFormShape {
  name: string
  notes: string
  platform: AccountPlatform
  type: AccountType
  credentials: Record<string, unknown>
  proxy_id: number | null
  concurrency: number
  load_factor: number | null
  priority: number
  rate_multiplier: number
  group_ids: number[]
  expires_at: number | null
}

interface UseCreateAccountResetOptions {
  step: Ref<number>
  form: CreateAccountFormShape
  autoImportModels: Ref<boolean>
  accountCategory: Ref<'oauth-based' | 'apikey'>
  addMethod: Ref<AddMethod>
  gatewayProtocol: Ref<GatewayProtocol>
  apiKeyBaseUrl: Ref<string>
  apiKeyValue: Ref<string>
  grokSSOToken: Ref<string>
  grokTier: Ref<'basic' | 'super' | 'heavy'>
  editQuotaLimit: Ref<number | null>
  editQuotaDailyLimit: Ref<number | null>
  editQuotaWeeklyLimit: Ref<number | null>
  editQuotaDailyResetMode: Ref<'rolling' | 'fixed' | null>
  editQuotaDailyResetHour: Ref<number | null>
  editQuotaWeeklyResetMode: Ref<'rolling' | 'fixed' | null>
  editQuotaWeeklyResetDay: Ref<number | null>
  editQuotaWeeklyResetHour: Ref<number | null>
  editQuotaResetTimezone: Ref<string | null>
  modelMappings: Ref<ModelMapping[]>
  modelRestrictionMode: Ref<'whitelist' | 'mapping'>
  allowedModels: Ref<string[]>
  protocolGatewayProbedModels?: Ref<Array<Record<string, unknown>>>
  gatewayAcceptedProtocols?: Ref<GatewayAcceptedProtocol[]>
  gatewayClientProfiles?: Ref<GatewayClientProfile[]>
  gatewayClientRoutes?: Ref<GatewayClientRoute[]>
  claudeCodeMimicEnabled?: Ref<boolean>
  claudeTLSFingerprintEnabled?: Ref<boolean>
  claudeSessionIDMaskingEnabled?: Ref<boolean>
  loadAntigravityDefaultMappings: () => Promise<void>
  poolModeState: AccountPoolModeState
  customErrorCodesState: AccountCustomErrorCodesState
  interceptWarmupRequests: Ref<boolean>
  autoPauseOnExpired: Ref<boolean>
  openaiPassthroughEnabled: Ref<boolean>
  openaiOAuthResponsesWebSocketV2Mode: Ref<OpenAIWSMode>
  openaiAPIKeyResponsesWebSocketV2Mode: Ref<OpenAIWSMode>
  codexCLIOnlyEnabled: Ref<boolean>
  anthropicPassthroughEnabled: Ref<boolean>
  quotaControlReset: () => void
  antigravityAccountType: Ref<'oauth' | 'upstream'>
  upstreamBaseUrl: Ref<string>
  upstreamApiKey: Ref<string>
  geminiVertexProjectId: Ref<string>
  geminiVertexLocation: Ref<string>
  geminiVertexAccessToken: Ref<string>
  geminiVertexExpiresAtInput: Ref<string>
  geminiVertexBaseUrl: Ref<string>
  resetTempUnschedRules: () => void
  geminiOAuthType: Ref<GeminiOAuthType>
  geminiTierGoogleOne: Ref<'google_one_free' | 'google_ai_pro' | 'google_ai_ultra'>
  geminiTierGcp: Ref<'gcp_standard' | 'gcp_enterprise'>
  geminiTierAIStudio: Ref<'aistudio_free' | 'aistudio_paid'>
  oauthReset: () => void
  openaiOAuthReset: () => void
  soraOAuthReset: () => void
  geminiOAuthReset: () => void
  antigravityOAuthReset: () => void
  oauthFlowReset: () => void
  copilotFlowReset?: () => void
  kiroImportReset?: () => void
  resetMixedChannelRisk: () => void
}

export function useCreateAccountReset(options: UseCreateAccountResetOptions) {
  const resetForm = () => {
    options.step.value = 1
    options.form.name = ''
    options.form.notes = ''
    options.form.platform = 'anthropic'
    options.form.type = 'oauth'
    options.form.credentials = {}
    options.autoImportModels.value = false
    options.form.proxy_id = null
    options.form.concurrency = 10
    options.form.load_factor = null
    options.form.priority = 1
    options.form.rate_multiplier = 1
    options.form.group_ids = []
    options.form.expires_at = null
    options.accountCategory.value = 'oauth-based'
    options.addMethod.value = 'oauth'
    options.gatewayProtocol.value = 'openai'
    options.apiKeyBaseUrl.value = resolveAccountApiKeyDefaultBaseUrl('anthropic')
    options.apiKeyValue.value = ''
    options.grokSSOToken.value = ''
    options.grokTier.value = 'basic'
    options.editQuotaLimit.value = null
    options.editQuotaDailyLimit.value = null
    options.editQuotaWeeklyLimit.value = null
    options.editQuotaDailyResetMode.value = null
    options.editQuotaDailyResetHour.value = null
    options.editQuotaWeeklyResetMode.value = null
    options.editQuotaWeeklyResetDay.value = null
    options.editQuotaWeeklyResetHour.value = null
    options.editQuotaResetTimezone.value = null
    options.modelMappings.value = []
    options.modelRestrictionMode.value = 'whitelist'
    options.allowedModels.value = [...getModelsByPlatform('anthropic', 'whitelist')]
    options.protocolGatewayProbedModels && (options.protocolGatewayProbedModels.value = [])
    options.gatewayAcceptedProtocols && (options.gatewayAcceptedProtocols.value = ['openai'])
    options.gatewayClientProfiles && (options.gatewayClientProfiles.value = [])
    options.gatewayClientRoutes && (options.gatewayClientRoutes.value = [])
    options.claudeCodeMimicEnabled && (options.claudeCodeMimicEnabled.value = false)
    options.claudeTLSFingerprintEnabled && (options.claudeTLSFingerprintEnabled.value = false)
    options.claudeSessionIDMaskingEnabled && (options.claudeSessionIDMaskingEnabled.value = false)

    options.loadAntigravityDefaultMappings()
    resetAccountPoolModeState(options.poolModeState, DEFAULT_POOL_MODE_RETRY_COUNT)
    resetAccountCustomErrorCodesState(options.customErrorCodesState)
    options.interceptWarmupRequests.value = false
    options.autoPauseOnExpired.value = true
    options.openaiPassthroughEnabled.value = false
    options.openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
    options.openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
    options.codexCLIOnlyEnabled.value = false
    options.anthropicPassthroughEnabled.value = false
    options.quotaControlReset()
    options.antigravityAccountType.value = 'oauth'
    options.upstreamBaseUrl.value = ''
    options.upstreamApiKey.value = ''
    options.geminiVertexProjectId.value = ''
    options.geminiVertexLocation.value = ''
    options.geminiVertexAccessToken.value = ''
    options.geminiVertexExpiresAtInput.value = ''
    options.geminiVertexBaseUrl.value = ''
    options.resetTempUnschedRules()
    options.geminiOAuthType.value = 'code_assist'
    options.geminiTierGoogleOne.value = 'google_one_free'
    options.geminiTierGcp.value = 'gcp_standard'
    options.geminiTierAIStudio.value = 'aistudio_free'
    options.oauthReset()
    options.openaiOAuthReset()
    options.soraOAuthReset()
    options.geminiOAuthReset()
    options.antigravityOAuthReset()
    options.oauthFlowReset()
    options.copilotFlowReset?.()
    options.kiroImportReset?.()
    options.resetMixedChannelRisk()
  }

  return {
    resetForm
  }
}
