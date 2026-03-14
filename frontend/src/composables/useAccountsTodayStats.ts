import { ref, type Ref } from 'vue'
import { adminAPI } from '@/api/admin'
import { buildDefaultTodayStats } from '@/utils/accountListSync'
import type { Account, WindowStats } from '@/types'

interface UseAccountsTodayStatsOptions {
  accounts: Ref<Account[]>
  hiddenColumns: Pick<Set<string>, 'has'>
}

export function useAccountsTodayStats({ accounts, hiddenColumns }: UseAccountsTodayStatsOptions) {
  const todayStatsByAccountId = ref<Record<string, WindowStats>>({})
  const todayStatsLoading = ref(false)
  const todayStatsError = ref<string | null>(null)
  const todayStatsReqSeq = ref(0)

  const refreshTodayStats = async () => {
    if (hiddenColumns.has('today_stats')) {
      todayStatsLoading.value = false
      todayStatsError.value = null
      return
    }

    const accountIDs = accounts.value.map(account => account.id)
    const reqSeq = ++todayStatsReqSeq.value
    if (accountIDs.length === 0) {
      todayStatsByAccountId.value = {}
      todayStatsError.value = null
      todayStatsLoading.value = false
      return
    }

    todayStatsLoading.value = true
    todayStatsError.value = null

    try {
      const result = await adminAPI.accounts.getBatchTodayStats(accountIDs)
      if (reqSeq !== todayStatsReqSeq.value) return

      const serverStats = result.stats ?? {}
      const nextStats: Record<string, WindowStats> = {}
      for (const accountID of accountIDs) {
        const key = String(accountID)
        nextStats[key] = serverStats[key] ?? buildDefaultTodayStats()
      }
      todayStatsByAccountId.value = nextStats
    } catch (error) {
      if (reqSeq !== todayStatsReqSeq.value) return
      todayStatsError.value = 'Failed'
      console.error('Failed to load account today stats:', error)
    } finally {
      if (reqSeq === todayStatsReqSeq.value) {
        todayStatsLoading.value = false
      }
    }
  }

  return {
    todayStatsByAccountId,
    todayStatsLoading,
    todayStatsError,
    refreshTodayStats
  }
}
