import type { Ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { applyInterceptWarmup } from '@/components/account/credentialsBuilder'
import type { Account, AccountPlatform, AccountType } from '@/types'

interface AccountOAuthClient {
  sessionId: Ref<string>
  loading: Ref<boolean>
  error: Ref<string>
  buildExtraInfo: (tokenInfo: any) => Record<string, unknown> | undefined
}

interface QuotaControlClient {
  buildExtra: (base: Record<string, unknown>) => Record<string, unknown> | undefined
}

interface UseCreateAccountAnthropicExchangeOptions {
  oauthClient: AccountOAuthClient
  platform: Ref<AccountPlatform>
  addMethod: Ref<'oauth' | 'setup-token'>
  proxyId: Ref<number | null>
  interceptWarmupRequests: Ref<boolean>
  quotaControl: QuotaControlClient
  createAccountAndFinish: (
    platform: AccountPlatform,
    type: AccountType,
    credentials: Record<string, unknown>,
    extra?: Record<string, unknown>
  ) => Promise<Account | null>
}

export function useCreateAccountAnthropicExchange(options: UseCreateAccountAnthropicExchangeOptions) {
  const { t } = useI18n()
  const appStore = useAppStore()

  const handleAnthropicExchange = async (authCode: string) => {
    const oauthClient = options.oauthClient
    if (!authCode.trim() || !oauthClient.sessionId.value) return

    oauthClient.loading.value = true
    oauthClient.error.value = ''

    try {
      const proxyConfig = options.proxyId.value ? { proxy_id: options.proxyId.value } : {}
      const endpoint =
        options.addMethod.value === 'oauth'
          ? '/admin/accounts/exchange-code'
          : '/admin/accounts/exchange-setup-token-code'

      const tokenInfo = await adminAPI.accounts.exchangeCode(endpoint, {
        session_id: oauthClient.sessionId.value,
        code: authCode.trim(),
        ...proxyConfig
      })

      const baseExtra = oauthClient.buildExtraInfo(tokenInfo) || {}
      const extra = options.quotaControl.buildExtra(baseExtra)

      const credentials: Record<string, unknown> = { ...tokenInfo }
      applyInterceptWarmup(credentials, options.interceptWarmupRequests.value, 'create')
      await options.createAccountAndFinish(
        options.platform.value,
        options.addMethod.value as AccountType,
        credentials,
        extra
      )
    } catch (error: any) {
      oauthClient.error.value = error?.message || t('admin.accounts.oauth.authFailed')
      appStore.showError(oauthClient.error.value)
    } finally {
      oauthClient.loading.value = false
    }
  }

  return {
    handleAnthropicExchange
  }
}

