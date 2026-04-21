import type { ComputedRef, Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { buildModelMappingObject } from '@/composables/useModelWhitelist'
import type { Account } from '@/types'
import type { ModelMapping } from '@/utils/accountFormShared'

interface OpenAIOAuthClient {
  loading: Ref<boolean>
  error: Ref<string>
  validateRefreshToken: (refreshToken: string, proxyId: number | null) => Promise<any>
  buildCredentials: (tokenInfo: any) => Record<string, unknown>
  buildExtraInfo: (tokenInfo: any) => Record<string, unknown> | undefined
}

interface UseCreateAccountOpenAIRefreshTokenValidationOptions {
  oauthClient: ComputedRef<OpenAIOAuthClient>
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
  isOpenAIModelRestrictionDisabled: ComputedRef<boolean>
  modelRestrictionEnabled: Ref<boolean>
  modelRestrictionMode: Ref<'whitelist' | 'mapping'>
  allowedModels: Ref<string[]>
  modelMappings: Ref<ModelMapping[]>
  buildAccountExtra: (base?: Record<string, unknown>) => Record<string, unknown> | undefined
  afterCreateImportModels: (accounts: Account[]) => Promise<void>
  emitCreated: () => void
  onClose: () => void
}

export function useCreateAccountOpenAIRefreshTokenValidation(
  options: UseCreateAccountOpenAIRefreshTokenValidationOptions
) {
  const { t } = useI18n()
  const appStore = useAppStore()

  const handleOpenAIValidateRT = async (refreshTokenInput: string) => {
    const oauthClient = options.oauthClient.value
    if (!refreshTokenInput.trim()) return

    const refreshTokens = refreshTokenInput
      .split('\n')
      .map((rt) => rt.trim())
      .filter((rt) => rt)

    if (refreshTokens.length === 0) {
      oauthClient.error.value = t('admin.accounts.oauth.openai.pleaseEnterRefreshToken')
      return
    }

    oauthClient.loading.value = true
    oauthClient.error.value = ''

    let successCount = 0
    let failedCount = 0
    const errors: string[] = []
    const createdAccounts: Account[] = []

    try {
      for (let i = 0; i < refreshTokens.length; i++) {
        try {
          const tokenInfo = await oauthClient.validateRefreshToken(refreshTokens[i], options.form.proxy_id)
          if (!tokenInfo) {
            failedCount++
            errors.push(`#${i + 1}: ${oauthClient.error.value || 'Validation failed'}`)
            oauthClient.error.value = ''
            continue
          }

          const credentials = oauthClient.buildCredentials(tokenInfo)
          const oauthExtra = oauthClient.buildExtraInfo(tokenInfo)
          const extra = options.buildAccountExtra(oauthExtra)

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

          const accountName = refreshTokens.length > 1 ? `${options.form.name} #${i + 1}` : options.form.name

          const openaiAccount = await adminAPI.accounts.create({
            name: accountName,
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
            auto_pause_on_expired: options.autoPauseOnExpired.value
          })
          createdAccounts.push(openaiAccount)

          successCount++
        } catch (error: any) {
          failedCount++
          const errMsg = error?.message || 'Unknown error'
          errors.push(`#${i + 1}: ${errMsg}`)
        }
      }

      if (successCount > 0 && failedCount === 0) {
        appStore.showSuccess(
          refreshTokens.length > 1
            ? t('admin.accounts.oauth.batchSuccess', { count: successCount })
            : t('admin.accounts.accountCreated')
        )
        await options.afterCreateImportModels(createdAccounts)
        options.emitCreated()
        options.onClose()
      } else if (successCount > 0 && failedCount > 0) {
        appStore.showWarning(
          t('admin.accounts.oauth.batchPartialSuccess', { success: successCount, failed: failedCount })
        )
        await options.afterCreateImportModels(createdAccounts)
        oauthClient.error.value = errors.join('\n')
        options.emitCreated()
      } else {
        oauthClient.error.value = errors.join('\n')
        appStore.showError(t('admin.accounts.oauth.batchFailed'))
      }
    } finally {
      oauthClient.loading.value = false
    }
  }

  return {
    handleOpenAIValidateRT
  }
}
