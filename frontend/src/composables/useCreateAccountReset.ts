import type { Ref } from 'vue'
import type { AddMethod } from '@/composables/useAccountOAuth'
import type { AccountPlatform, AccountType } from '@/types'
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
  apiKeyBaseUrl: Ref<string>
  apiKeyValue: Ref<string>
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
  resetTempUnschedRules: () => void
  geminiOAuthType: Ref<'google_one' | 'ai_studio' | 'code_assist'>
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
    options.apiKeyBaseUrl.value = resolveAccountApiKeyDefaultBaseUrl('anthropic')
    options.apiKeyValue.value = ''
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
