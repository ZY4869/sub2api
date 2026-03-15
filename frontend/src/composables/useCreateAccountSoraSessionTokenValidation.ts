import type { ComputedRef, Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { Account } from '@/types'

interface OpenAIOAuthClient {
  loading: Ref<boolean>
  error: Ref<string>
  validateSessionToken: (sessionToken: string, proxyId: number | null) => Promise<any>
  buildCredentials: (tokenInfo: any) => Record<string, unknown>
  buildExtraInfo: (tokenInfo: any) => Record<string, unknown> | undefined
}

interface UseCreateAccountSoraSessionTokenValidationOptions {
  oauthClient: ComputedRef<OpenAIOAuthClient>
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
  buildSoraAccountExtra: (base?: Record<string, unknown>) => Record<string, unknown> | undefined
  afterCreateImportModels: (accounts: Account[]) => Promise<void>
  emitCreated: () => void
  onClose: () => void
}

export function useCreateAccountSoraSessionTokenValidation(
  options: UseCreateAccountSoraSessionTokenValidationOptions
) {
  const { t } = useI18n()
  const appStore = useAppStore()

  const handleSoraValidateST = async (sessionTokenInput: string) => {
    const oauthClient = options.oauthClient.value
    if (!sessionTokenInput.trim()) return

    const sessionTokens = sessionTokenInput
      .split('\n')
      .map((st) => st.trim())
      .filter((st) => st)

    if (sessionTokens.length === 0) {
      oauthClient.error.value = t('admin.accounts.oauth.openai.pleaseEnterSessionToken')
      return
    }

    oauthClient.loading.value = true
    oauthClient.error.value = ''

    let successCount = 0
    let failedCount = 0
    const errors: string[] = []
    const createdAccounts: Account[] = []

    try {
      for (let i = 0; i < sessionTokens.length; i++) {
        try {
          const tokenInfo = await oauthClient.validateSessionToken(sessionTokens[i], options.form.proxy_id)
          if (!tokenInfo) {
            failedCount++
            errors.push(`#${i + 1}: ${oauthClient.error.value || 'Validation failed'}`)
            oauthClient.error.value = ''
            continue
          }

          const credentials = oauthClient.buildCredentials(tokenInfo)
          credentials.session_token = sessionTokens[i]
          const oauthExtra = oauthClient.buildExtraInfo(tokenInfo)
          const soraExtra = options.buildSoraAccountExtra(oauthExtra)

          const accountName = sessionTokens.length > 1 ? `${options.form.name} #${i + 1}` : options.form.name
          const createdAccount = await adminAPI.accounts.create({
            name: accountName,
            notes: options.form.notes,
            platform: 'sora',
            type: 'oauth',
            credentials,
            extra: soraExtra,
            proxy_id: options.form.proxy_id,
            concurrency: options.form.concurrency,
            load_factor: options.form.load_factor ?? undefined,
            priority: options.form.priority,
            rate_multiplier: options.form.rate_multiplier,
            group_ids: options.form.group_ids,
            expires_at: options.form.expires_at,
            auto_pause_on_expired: options.autoPauseOnExpired.value
          })
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
          sessionTokens.length > 1
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
    handleSoraValidateST
  }
}
