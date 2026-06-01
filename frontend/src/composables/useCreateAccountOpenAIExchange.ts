import type { ComputedRef, Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { buildModelMappingObject } from '@/composables/useModelWhitelist'
import type { Account, AccountAutoRenewPeriod } from '@/types'
import type { ModelMapping } from '@/utils/accountFormShared'
import {
  applyOpenAIOAuthDefaultModelState,
  buildOpenAIOAuthCreateExtra,
} from '@/utils/openaiOAuthDefaults'

interface OpenAIOAuthClient {
  sessionId: Ref<string>
  oauthState: Ref<string>
  loading: Ref<boolean>
  error: Ref<string>
  exchangeAuthCode: (
    code: string,
    sessionId: string,
    oauthState: string,
    proxyId: number | null
  ) => Promise<any>
  buildCredentials: (tokenInfo: any) => Record<string, unknown>
  buildExtraInfo: (tokenInfo: any) => Record<string, unknown> | undefined
}

interface UseCreateAccountOpenAIExchangeOptions {
  oauthClient: ComputedRef<OpenAIOAuthClient>
  getOAuthState: () => string
  form: {
    platform: string
    name: string
    notes: string
    proxy_id: number | null
    concurrency: number
    load_factor: number | null
    priority: number
    rate_multiplier: number
    group_ids: number[]
    expires_at: number | null
  }
  autoPauseOnExpired: Ref<boolean>
  autoRenewEnabled: Ref<boolean>
  autoRenewPeriod: Ref<AccountAutoRenewPeriod>
  applyTempUnschedConfig: (credentials: Record<string, unknown>) => boolean
  isOpenAIModelRestrictionDisabled: ComputedRef<boolean>
  modelRestrictionEnabled: Ref<boolean>
  modelRestrictionMode: Ref<'whitelist' | 'mapping'>
  allowedModels: Ref<string[]>
  modelMappings: Ref<ModelMapping[]>
  hasCustomizedOpenAIDefaults: Ref<boolean>
  buildAccountExtra: (base?: Record<string, unknown>) => Record<string, unknown> | undefined
  applyOpenAIImageProtocolDefaults?: (planType?: string | null) => void
  afterCreateImportModels: (accounts: Account[]) => Promise<void>
  emitCreated: () => void
  onClose: () => void
}

export function useCreateAccountOpenAIExchange(options: UseCreateAccountOpenAIExchangeOptions) {
  const { t } = useI18n()
  const appStore = useAppStore()

  const handleOpenAIExchange = async (authCode: string) => {
    const oauthClient = options.oauthClient.value
    if (!authCode.trim() || !oauthClient.sessionId.value) return

    oauthClient.loading.value = true
    oauthClient.error.value = ''

    try {
      const stateToUse = options.getOAuthState()
      if (!stateToUse) {
        oauthClient.error.value = t('admin.accounts.oauth.authFailed')
        appStore.showError(oauthClient.error.value)
        return
      }

      const tokenInfo = await oauthClient.exchangeAuthCode(
        authCode.trim(),
        oauthClient.sessionId.value,
        stateToUse,
        options.form.proxy_id
      )
      if (!tokenInfo) return

      const nextDefaults = applyOpenAIOAuthDefaultModelState({
        planType: String(tokenInfo.plan_type || ''),
        proMultiplier:
          typeof tokenInfo.pro_multiplier === 'number' ? tokenInfo.pro_multiplier : null,
        currentAllowedModels: options.allowedModels.value,
        currentModelMappings: options.modelMappings.value,
        modelRestrictionMode: options.modelRestrictionMode.value,
        userCustomized: options.hasCustomizedOpenAIDefaults.value,
      })
      options.allowedModels.value = nextDefaults.allowedModels
      options.modelMappings.value = nextDefaults.modelMappings
      options.applyOpenAIImageProtocolDefaults?.(String(tokenInfo.plan_type || ''))

      const credentials = oauthClient.buildCredentials(tokenInfo)
      const oauthExtra = oauthClient.buildExtraInfo(tokenInfo)
      const extra = buildOpenAIOAuthCreateExtra(options.buildAccountExtra(oauthExtra), {
        modelRestrictionEnabled:
          options.modelRestrictionEnabled.value &&
          !options.isOpenAIModelRestrictionDisabled.value,
        modelRestrictionMode: options.modelRestrictionMode.value,
        allowedModels: options.allowedModels.value,
        modelMappings: options.modelMappings.value,
      })

      if (
        options.modelRestrictionEnabled.value &&
        !options.isOpenAIModelRestrictionDisabled.value
      ) {
        const modelMapping = buildModelMappingObject(
          options.modelRestrictionMode.value,
          options.allowedModels.value,
          options.modelMappings.value
        )
        if (modelMapping) {
          credentials.model_mapping = modelMapping
        }
      }

      if (!options.applyTempUnschedConfig(credentials)) {
        return
      }

      const createdAccounts: Account[] = []
      const openaiAccount = await adminAPI.accounts.create({
        name: options.form.name,
        notes: options.form.notes,
        platform: 'openai',
        type: 'oauth',
        credentials,
        extra,
        proxy_id: options.form.proxy_id,
        concurrency: options.form.concurrency,
        load_factor: options.form.load_factor ?? undefined,
        priority: options.form.priority,
        rate_multiplier: options.form.rate_multiplier,
        group_ids: options.form.group_ids,
        expires_at: options.form.expires_at,
        auto_pause_on_expired: options.autoPauseOnExpired.value,
        auto_renew_enabled: options.autoRenewEnabled.value,
        auto_renew_period: options.autoRenewPeriod.value
      })
      createdAccounts.push(openaiAccount)
      appStore.showSuccess(t('admin.accounts.accountCreated'))

      await options.afterCreateImportModels(createdAccounts)
      options.emitCreated()
      options.onClose()
    } catch (error: any) {
      oauthClient.error.value = error?.message || t('admin.accounts.oauth.authFailed')
      appStore.showError(oauthClient.error.value)
    } finally {
      oauthClient.loading.value = false
    }
  }

  return {
    handleOpenAIExchange
  }
}
