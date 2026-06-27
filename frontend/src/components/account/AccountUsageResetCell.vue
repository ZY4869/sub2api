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
      :reset-zero="resetCreditsZero"
      :reset-credits-low="resetCreditsLow"
      :resetting="resetting"
      :refreshing="refreshingResetCredits"
      :reset-disabled="resetButtonDisabled"
      :show-refresh="false"
      :show-remaining="false"
      @reset="resetOpenAIQuota"
    />
  </div>

  <div v-else class="text-xs text-gray-400">-</div>
</template>

<script setup lang="ts">
import { useAccountUsagePresentation } from '@/composables/useAccountUsagePresentation'
import { useRealtimeCountdownNow } from '@/composables/useRealtimeCountdownNow'
import { useOpenAIResetCreditsControls } from '@/composables/useOpenAIResetCreditsControls'
import type { Account } from '@/types'
import AccountOpenAIResetCreditsControls from './AccountOpenAIResetCreditsControls.vue'
import AccountUsageResetRows from './AccountUsageResetRows.vue'

const props = defineProps<{
  account: Account
}>()

const { nowDate } = useRealtimeCountdownNow('accounts')
const { presentation } = useAccountUsagePresentation(() => props.account)
const {
  canResetOpenAIQuota,
  resetCreditsStatusLabel,
  resetCreditsUnsupported,
  resetCreditsZero,
  resetCreditsLow,
  resetting,
  refreshingResetCredits,
  resetButtonDisabled,
  resetOpenAIQuota,
} = useOpenAIResetCreditsControls(
  () => props.account,
  () => presentation.value,
)

</script>
