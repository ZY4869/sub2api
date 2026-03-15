import type { Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { Account, AccountType, CreateAccountRequest } from '@/types'
import type { ModelMapping } from '@/utils/accountFormShared'
import { buildModelMappingObject } from '@/composables/useModelWhitelist'
import { applyInterceptWarmup } from '@/components/account/credentialsBuilder'

interface AntigravityOAuthClient {
  sessionId: Ref<string>
  state: Ref<string>
  loading: Ref<boolean>
  error: Ref<string>
  validateRefreshToken: (refreshToken: string, proxyId: number | null) => Promise<any>
  exchangeAuthCode: (payload: {
    code: string
    sessionId: string
    state: string
    proxyId: number | null
  }) => Promise<any>
  buildCredentials: (tokenInfo: any) => Record<string, unknown>
}

interface UseCreateAccountAntigravityHandlersOptions {
  oauthClient: AntigravityOAuthClient
  getOAuthState: () => string
  withConfirmFlag: <TPayload extends object>(payload: TPayload) => TPayload
  form: {
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
  interceptWarmupRequests: Ref<boolean>
  antigravityModelMappings: Ref<ModelMapping[]>
  mixedScheduling: Ref<boolean>
  createAccountAndFinish: (
    platform: 'antigravity',
    type: AccountType,
    credentials: Record<string, unknown>,
    extra?: Record<string, unknown>
  ) => Promise<Account | null>
  afterCreateImportModels: (accounts: Account[]) => Promise<void>
  emitCreated: () => void
  onClose: () => void
}

export function useCreateAccountAntigravityHandlers(
  options: UseCreateAccountAntigravityHandlersOptions
) {
  const { t } = useI18n()
  const appStore = useAppStore()

  const handleAntigravityValidateRT = async (refreshTokenInput: string) => {
    const oauthClient = options.oauthClient
    if (!refreshTokenInput.trim()) return

    const refreshTokens = refreshTokenInput
      .split('\n')
      .map((rt) => rt.trim())
      .filter((rt) => rt)

    if (refreshTokens.length === 0) {
      oauthClient.error.value = t('admin.accounts.oauth.antigravity.pleaseEnterRefreshToken')
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
          const accountName = refreshTokens.length > 1 ? `${options.form.name} #${i + 1}` : options.form.name

          const createPayload: CreateAccountRequest = options.withConfirmFlag({
            name: accountName,
            notes: options.form.notes,
            platform: 'antigravity',
            type: 'oauth',
            credentials,
            extra: {},
            proxy_id: options.form.proxy_id,
            concurrency: options.form.concurrency,
            load_factor: options.form.load_factor ?? undefined,
            priority: options.form.priority,
            rate_multiplier: options.form.rate_multiplier,
            group_ids: options.form.group_ids,
            expires_at: options.form.expires_at,
            auto_pause_on_expired: options.autoPauseOnExpired.value
          })
          const createdAccount = await adminAPI.accounts.create(createPayload)
          createdAccounts.push(createdAccount)
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
        return
      }

      if (successCount > 0 && failedCount > 0) {
        appStore.showWarning(
          t('admin.accounts.oauth.batchPartialSuccess', { success: successCount, failed: failedCount })
        )
        await options.afterCreateImportModels(createdAccounts)
        oauthClient.error.value = errors.join('\n')
        options.emitCreated()
        return
      }

      oauthClient.error.value = errors.join('\n')
      appStore.showError(t('admin.accounts.oauth.batchFailed'))
    } finally {
      oauthClient.loading.value = false
    }
  }

  const handleAntigravityExchange = async (authCode: string) => {
    const oauthClient = options.oauthClient
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

      const tokenInfo = await oauthClient.exchangeAuthCode({
        code: authCode.trim(),
        sessionId: oauthClient.sessionId.value,
        state: stateToUse,
        proxyId: options.form.proxy_id
      })
      if (!tokenInfo) return

      const credentials = oauthClient.buildCredentials(tokenInfo)
      applyInterceptWarmup(credentials, options.interceptWarmupRequests.value, 'create')

      const antigravityModelMapping = buildModelMappingObject(
        'mapping',
        [],
        options.antigravityModelMappings.value
      )
      if (antigravityModelMapping) {
        credentials.model_mapping = antigravityModelMapping
      }

      const extra = options.mixedScheduling.value ? { mixed_scheduling: true } : undefined
      await options.createAccountAndFinish('antigravity', 'oauth', credentials, extra)
    } catch (error: any) {
      oauthClient.error.value = error?.message || t('admin.accounts.oauth.authFailed')
      appStore.showError(oauthClient.error.value)
    } finally {
      oauthClient.loading.value = false
    }
  }

  return {
    handleAntigravityValidateRT,
    handleAntigravityExchange
  }
}

