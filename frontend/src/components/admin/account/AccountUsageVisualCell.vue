<template>
  <AccountUsageCell
    :account="account"
    :today-stats="todayStats"
    :today-stats-loading="todayStatsLoading"
    :manual-refresh-token="manualRefreshToken"
    :visual-variant="visualStyle === 'airy' ? 'glass' : 'default'"
    :white-surface-enabled="whiteSurfaceEnabled"
  />
</template>

<script setup lang="ts">
import type { Account, AccountVisualStyle, WindowStats } from '@/types'
import AccountUsageCell from '@/components/account/AccountUsageCell.vue'

withDefaults(defineProps<{
  account: Account
  todayStats?: WindowStats | null
  todayStatsLoading?: boolean
  manualRefreshToken?: number
  visualStyle?: AccountVisualStyle
  whiteSurfaceEnabled?: boolean
}>(), {
  todayStats: null,
  todayStatsLoading: false,
  manualRefreshToken: 0,
  visualStyle: 'airy',
  whiteSurfaceEnabled: false
})
</script>
