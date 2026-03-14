import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Account } from '@/types'
import {
  buildAnthropicQuotaControlExtra,
  createDefaultAnthropicQuotaControlState,
  readAnthropicQuotaControlState
} from '@/utils/accountQuotaControl'

export const useAnthropicQuotaControl = () => {
  const { t } = useI18n()
  const state = reactive(createDefaultAnthropicQuotaControlState())

  const reset = () => {
    Object.assign(state, createDefaultAnthropicQuotaControlState())
  }

  const loadFromAccount = (account: Account | null | undefined) => {
    Object.assign(state, readAnthropicQuotaControlState(account))
  }

  const buildExtra = (base?: Record<string, unknown>) =>
    buildAnthropicQuotaControlExtra(state, base)

  const umqModeOptions = computed(() => [
    { value: '', label: t('admin.accounts.quotaControl.rpmLimit.umqModeOff') },
    { value: 'throttle', label: t('admin.accounts.quotaControl.rpmLimit.umqModeThrottle') },
    { value: 'serialize', label: t('admin.accounts.quotaControl.rpmLimit.umqModeSerialize') }
  ])

  return {
    state,
    reset,
    loadFromAccount,
    buildExtra,
    umqModeOptions
  }
}
