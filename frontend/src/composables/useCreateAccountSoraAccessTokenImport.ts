import type { ComputedRef, Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { Account } from '@/types'

interface OpenAIOAuthClient {
  loading: Ref<boolean>
  error: Ref<string>
}

interface UseCreateAccountSoraAccessTokenImportOptions {
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
  buildSoraAccountExtra: () => Record<string, unknown> | undefined
  afterCreateImportModels: (accounts: Account[]) => Promise<void>
  emitCreated: () => void
  onClose: () => void
}

export function useCreateAccountSoraAccessTokenImport(
  options: UseCreateAccountSoraAccessTokenImportOptions
) {
  const { t } = useI18n()
  const appStore = useAppStore()

  const handleImportAccessToken = async (accessTokenInput: string) => {
    const oauthClient = options.oauthClient.value
    if (!accessTokenInput.trim()) return

    const accessTokens = accessTokenInput
      .split('\n')
      .map((at) => at.trim())
      .filter((at) => at)

    if (accessTokens.length === 0) {
      return
    }

    oauthClient.loading.value = true
    oauthClient.error.value = ''

    let successCount = 0
    let failedCount = 0
    const errors: string[] = []
    const createdAccounts: Account[] = []

    try {
      for (let i = 0; i < accessTokens.length; i++) {
        try {
          const credentials: Record<string, unknown> = {
            access_token: accessTokens[i]
          }
          const soraExtra = options.buildSoraAccountExtra()

          const accountName = accessTokens.length > 1 ? `${options.form.name} #${i + 1}` : options.form.name
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
          accessTokens.length > 1
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
    handleImportAccessToken
  }
}
