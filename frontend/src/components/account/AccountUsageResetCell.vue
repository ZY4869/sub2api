<template>
  <div v-if="presentation.state === 'loading'" class="space-y-1.5">
    <div v-for="index in presentation.meta.loadingRows" :key="index" class="flex items-center gap-2">
      <div class="h-3 w-[32px] animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
      <div class="h-3 w-20 animate-pulse rounded bg-gray-200 dark:bg-gray-700"></div>
    </div>
  </div>

  <div v-else-if="presentation.state === 'error'" class="text-xs text-red-500">
    {{ presentation.error }}
  </div>

  <div v-else-if="presentation.resetRows.length > 0" class="max-w-full space-y-1.5 overflow-visible">
    <AccountUsageResetRows
      :rows="presentation.resetRows"
      :now-date="nowDate"
    />
    <AccountOpenAIResetCreditsControls
      v-if="canResetOpenAIQuota"
      :reset-status-label="resetCreditsStatusLabel"
      :reset-unsupported="resetCreditsUnsupported"
      :resetting="resetting"
      :refreshing="refreshingResetCredits"
      :reset-disabled="resetButtonDisabled"
      @refresh="refreshOpenAIResetCredits"
      @reset="resetOpenAIQuota"
    />
  </div>

  <div v-else class="text-xs text-gray-400">-</div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  invalidateAccountUsagePresentationCache,
  refreshAccountUsagePresentation,
  useAccountUsagePresentation,
} from '@/composables/useAccountUsagePresentation'
import { useRealtimeCountdownNow } from '@/composables/useRealtimeCountdownNow'
import type { Account } from '@/types'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api'
import { useAppStore } from '@/stores'
import { getRuntimePlatform } from '@/composables/accountUsagePresentation/support'
import AccountOpenAIResetCreditsControls from './AccountOpenAIResetCreditsControls.vue'
import AccountUsageResetRows from './AccountUsageResetRows.vue'

const props = defineProps<{
  account: Account
}>()

const { t } = useI18n()
const appStore = useAppStore()
const { nowDate } = useRealtimeCountdownNow('accounts')
const { presentation } = useAccountUsagePresentation(() => props.account)
const resetting = ref(false)
const refreshingResetCredits = ref(false)

const canResetOpenAIQuota = computed(() => {
  return getRuntimePlatform(props.account) === 'openai' && props.account.type === 'oauth'
})

const openAIQuotaResetRemaining = computed(() => {
  return presentation.value.meta.openAIResetCreditsAvailableCount ?? null
})

const resetCreditsUnsupported = computed(() => {
  return presentation.value.meta.openAIResetCreditsStatus === 'unsupported'
})

const resetButtonDisabled = computed(() => {
  return resetting.value || refreshingResetCredits.value || resetCreditsUnsupported.value
})

const resetCreditsStatusLabel = computed(() => {
  if (resetCreditsUnsupported.value) {
    return (
      presentation.value.meta.openAIResetCreditsUnsupportedReason ||
      t('admin.accounts.usageWindow.resetQuotaUnsupported')
    )
  }
  return t('admin.accounts.usageWindow.resetQuotaRemaining', {
    count: formatOpenAIQuotaResetRemaining(openAIQuotaResetRemaining.value),
  })
})

function formatOpenAIQuotaResetRemaining(value: number | null): string {
  if (value === null) return '--'
  const normalized = Number.isFinite(Number(value)) ? Math.max(0, Math.floor(Number(value))) : null
  if (normalized === null) return '--'
  return String(normalized).padStart(2, '0')
}

async function resetOpenAIQuota() {
  if (!canResetOpenAIQuota.value || resetButtonDisabled.value) return
  if (!window.confirm(t('admin.accounts.usageWindow.resetQuotaConfirm'))) return
  resetting.value = true
  try {
    await adminAPI.accounts.resetAccountQuota(props.account.id)
    invalidateAccountUsagePresentationCache([props.account.id])
    await refreshAccountUsagePresentation([props.account], { force: true, source: 'active' })
    appStore.showSuccess(t('admin.accounts.usageWindow.resetQuotaSuccess'))
  } catch (error: any) {
    invalidateAccountUsagePresentationCache([props.account.id])
    await refreshAccountUsagePresentation([props.account], { force: true, source: 'active' }).catch(() => {})
    appStore.showError(resolveResetQuotaErrorMessage(error))
  } finally {
    resetting.value = false
  }
}

async function refreshOpenAIResetCredits() {
  if (!canResetOpenAIQuota.value || refreshingResetCredits.value || resetting.value) return
  refreshingResetCredits.value = true
  try {
    invalidateAccountUsagePresentationCache([props.account.id])
    const result = await refreshAccountUsagePresentation([props.account], {
      force: true,
      source: 'active',
    })
    if (result.failed > 0) {
      appStore.showError(t('admin.accounts.usageWindow.refreshResetCreditsFailed'))
      return
    }
    appStore.showSuccess(t('admin.accounts.usageWindow.refreshResetCreditsSuccess'))
  } catch (error: any) {
    appStore.showError(resolveRefreshResetCreditsErrorMessage(error))
  } finally {
    refreshingResetCredits.value = false
  }
}

function resolveRefreshResetCreditsErrorMessage(error: any): string {
  const responseData = error?.response?.data ?? {}
  return (
    responseData.detail ||
    responseData.message ||
    error?.message ||
    t('admin.accounts.usageWindow.refreshResetCreditsFailed')
  )
}

function resolveResetQuotaErrorMessage(error: any): string {
  const responseData = error?.response?.data ?? {}
  const reason = String(
    responseData.reason ||
      responseData.error ||
      responseData.code ||
      responseData.error_code ||
      '',
  )

  if (reason === 'OPENAI_RESET_CREDITS_NO_CREDIT') {
    return t('admin.accounts.usageWindow.resetQuotaNoCredit')
  }
  if (reason === 'OPENAI_RESET_CREDITS_NOTHING_TO_RESET') {
    return t('admin.accounts.usageWindow.resetQuotaNothingToReset')
  }

  return (
    responseData.detail ||
    responseData.message ||
    error?.message ||
    t('admin.accounts.usageWindow.resetQuotaFailed')
  )
}

</script>
