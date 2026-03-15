import type { Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { applyInterceptWarmup } from '@/components/account/credentialsBuilder'
import type { Account, AccountPlatform } from '@/types'

interface AccountOAuthClient {
  loading: Ref<boolean>
  error: Ref<string>
  parseSessionKeys: (input: string) => string[]
  buildExtraInfo: (tokenInfo: any) => Record<string, unknown> | undefined
}

interface QuotaControlClient {
  buildExtra: (base: Record<string, unknown>) => Record<string, unknown> | undefined
}

interface UseCreateAccountAnthropicCookieAuthOptions {
  oauthClient: AccountOAuthClient
  platform: Ref<AccountPlatform>
  addMethod: Ref<'oauth' | 'setup-token'>
  proxyId: Ref<number | null>
  form: {
    name: string
    notes: string
    concurrency: number
    load_factor: number | null
    priority: number
    rate_multiplier: number
    group_ids: number[]
    expires_at: number | null
  }
  autoPauseOnExpired: Ref<boolean>
  interceptWarmupRequests: Ref<boolean>
  quotaControl: QuotaControlClient
  tempUnschedEnabled: Ref<boolean>
  buildTempUnschedPayload: () => unknown[]
  afterCreateImportModels: (accounts: Account[]) => Promise<void>
  emitCreated: () => void
  onClose: () => void
}

export function useCreateAccountAnthropicCookieAuth(options: UseCreateAccountAnthropicCookieAuthOptions) {
  const { t } = useI18n()
  const appStore = useAppStore()

  const handleCookieAuth = async (sessionKey: string) => {
    const oauthClient = options.oauthClient
    oauthClient.loading.value = true
    oauthClient.error.value = ''

    try {
      const proxyConfig = options.proxyId.value ? { proxy_id: options.proxyId.value } : {}
      const keys = oauthClient.parseSessionKeys(sessionKey)

      if (keys.length === 0) {
        oauthClient.error.value = t('admin.accounts.oauth.pleaseEnterSessionKey')
        return
      }

      const tempUnschedPayload = options.tempUnschedEnabled.value
        ? options.buildTempUnschedPayload()
        : []
      if (options.tempUnschedEnabled.value && tempUnschedPayload.length === 0) {
        appStore.showError(t('admin.accounts.tempUnschedulable.rulesInvalid'))
        return
      }

      const endpoint =
        options.addMethod.value === 'oauth'
          ? '/admin/accounts/cookie-auth'
          : '/admin/accounts/setup-token-cookie-auth'

      let successCount = 0
      let failedCount = 0
      const errors: string[] = []
      const createdAccounts: Account[] = []

      for (let i = 0; i < keys.length; i++) {
        try {
          const tokenInfo = await adminAPI.accounts.exchangeCode(endpoint, {
            session_id: '',
            code: keys[i],
            ...proxyConfig
          })

          const baseExtra = oauthClient.buildExtraInfo(tokenInfo) || {}
          const extra = options.quotaControl.buildExtra(baseExtra)

          const accountName = keys.length > 1 ? `${options.form.name} #${i + 1}` : options.form.name

          const credentials: Record<string, unknown> = { ...tokenInfo }
          applyInterceptWarmup(credentials, options.interceptWarmupRequests.value, 'create')
          if (options.tempUnschedEnabled.value) {
            credentials.temp_unschedulable_enabled = true
            credentials.temp_unschedulable_rules = tempUnschedPayload
          }

          const createdAccount = await adminAPI.accounts.create({
            name: accountName,
            notes: options.form.notes,
            platform: options.platform.value,
            type: options.addMethod.value,
            credentials,
            extra,
            proxy_id: options.proxyId.value,
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
          errors.push(
            t('admin.accounts.oauth.keyAuthFailed', {
              index: i + 1,
              error: error?.message || t('admin.accounts.oauth.authFailed')
            })
          )
        }
      }

      if (successCount > 0) {
        appStore.showSuccess(t('admin.accounts.oauth.successCreated', { count: successCount }))
        await options.afterCreateImportModels(createdAccounts)
        options.emitCreated()
        if (failedCount === 0) {
          options.onClose()
        }
      }

      if (failedCount > 0) {
        oauthClient.error.value = errors.join('\n')
      }
    } catch (error: any) {
      oauthClient.error.value = error?.message || t('admin.accounts.oauth.cookieAuthFailed')
    } finally {
      oauthClient.loading.value = false
    }
  }

  return {
    handleCookieAuth
  }
}

