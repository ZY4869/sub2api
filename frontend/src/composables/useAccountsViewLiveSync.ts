import { computed, ref, toRaw, watch, type ComputedRef, type Ref } from 'vue'
import { adminAPI } from '@/api/admin'
import { useAccountsAutoRefresh } from '@/composables/useAccountsAutoRefresh'
import { useAccountsTodayStats } from '@/composables/useAccountsTodayStats'
import { shouldReplaceAutoRefreshRow, type AccountListRequestParams } from '@/utils/accountListSync'
import type { Account } from '@/types'

interface PaginationState {
  page: number
  page_size: number
  total: number
  pages: number
}

interface UseAccountsViewLiveSyncOptions {
  accounts: Ref<Account[]>
  loading: Ref<boolean>
  params: AccountListRequestParams
  pagination: PaginationState
  hiddenColumns: Pick<Set<string>, 'has'>
  baseLoad: () => Promise<void>
  baseReload: () => Promise<void>
  baseDebouncedReload: () => void
  baseHandlePageChange: (page: number) => void
  baseHandlePageSizeChange: (size: number) => void
  isAnyModalOpen: ComputedRef<boolean>
  isActionMenuOpen: ComputedRef<boolean>
  syncAccountRefs: (account: Account) => void
  onListChanged?: () => void | Promise<void>
}

export function useAccountsViewLiveSync({
  accounts,
  loading,
  params,
  pagination,
  hiddenColumns,
  baseLoad,
  baseReload,
  baseDebouncedReload,
  baseHandlePageChange,
  baseHandlePageSizeChange,
  isAnyModalOpen,
  isActionMenuOpen,
  syncAccountRefs,
  onListChanged
}: UseAccountsViewLiveSyncOptions) {
  const autoRefreshETag = ref<string | null>(null)
  const autoRefreshFetching = ref(false)
  const hasPendingListSync = ref(false)
  const pendingTodayStatsRefresh = ref(false)
  const isFirstLoad = ref(true)
  const {
    todayStatsByAccountId,
    todayStatsLoading,
    todayStatsError,
    refreshTodayStats
  } = useAccountsTodayStats({
    accounts,
    hiddenColumns
  })

  const resetAutoRefreshCache = () => {
    autoRefreshETag.value = null
  }

  const load = async () => {
    hasPendingListSync.value = false
    resetAutoRefreshCache()
    pendingTodayStatsRefresh.value = false
    if (isFirstLoad.value) {
      params.lite = '1'
    }
    await baseLoad()
    if (isFirstLoad.value) {
      isFirstLoad.value = false
      delete params.lite
    }
    await refreshTodayStats()
    await onListChanged?.()
  }

  const reload = async () => {
    hasPendingListSync.value = false
    resetAutoRefreshCache()
    pendingTodayStatsRefresh.value = false
    await baseReload()
    await refreshTodayStats()
    await onListChanged?.()
  }

  const debouncedReload = () => {
    hasPendingListSync.value = false
    resetAutoRefreshCache()
    pendingTodayStatsRefresh.value = true
    baseDebouncedReload()
  }

  const handlePageChange = (page: number) => {
    hasPendingListSync.value = false
    resetAutoRefreshCache()
    pendingTodayStatsRefresh.value = true
    baseHandlePageChange(page)
  }

  const handlePageSizeChange = (size: number) => {
    hasPendingListSync.value = false
    resetAutoRefreshCache()
    pendingTodayStatsRefresh.value = true
    baseHandlePageSizeChange(size)
  }

  watch(loading, (isLoading, wasLoading) => {
    if (wasLoading && !isLoading && pendingTodayStatsRefresh.value) {
      pendingTodayStatsRefresh.value = false
      refreshTodayStats().catch((error) => {
        console.error('Failed to refresh account today stats after table load:', error)
      })
    }
  })

  const mergeAccountsIncrementally = (nextRows: Account[]) => {
    const currentRows = accounts.value
    const currentByID = new Map(currentRows.map(row => [row.id, row]))
    let changed = nextRows.length !== currentRows.length

    const mergedRows = nextRows.map((nextRow) => {
      const currentRow = currentByID.get(nextRow.id)
      if (!currentRow) {
        changed = true
        return nextRow
      }
      if (shouldReplaceAutoRefreshRow(currentRow, nextRow)) {
        changed = true
        syncAccountRefs(nextRow)
        return nextRow
      }
      return currentRow
    })

    if (!changed) {
      for (let index = 0; index < mergedRows.length; index += 1) {
        if (mergedRows[index].id !== currentRows[index]?.id) {
          changed = true
          break
        }
      }
    }

    if (changed) {
      accounts.value = mergedRows
    }
  }

  const refreshAccountsIncrementally = async () => {
    if (autoRefreshFetching.value) return

    autoRefreshFetching.value = true
    try {
      const result = await adminAPI.accounts.listWithEtag(
        pagination.page,
        pagination.page_size,
        toRaw(params) as AccountListRequestParams,
        { etag: autoRefreshETag.value }
      )

      if (result.etag) {
        autoRefreshETag.value = result.etag
      }
      if (!result.notModified && result.data) {
        pagination.total = result.data.total || 0
        pagination.pages = result.data.pages || 0
        mergeAccountsIncrementally(result.data.items || [])
        hasPendingListSync.value = false
        await onListChanged?.()
      }

      await refreshTodayStats()
    } catch (error) {
      console.error('Auto refresh failed:', error)
    } finally {
      autoRefreshFetching.value = false
    }
  }

  const syncPendingListChanges = async () => {
    hasPendingListSync.value = false
    await load()
  }

  const isAutoRefreshBlocked = computed(() => {
    return loading.value || autoRefreshFetching.value || isAnyModalOpen.value || isActionMenuOpen.value
  })

  const {
    autoRefreshIntervals,
    autoRefreshEnabled,
    autoRefreshIntervalSeconds,
    autoRefreshCountdown,
    setAutoRefreshEnabled,
    handleAutoRefreshIntervalChange,
    enterAutoRefreshSilentWindow,
    pauseAutoRefresh,
    resumeAutoRefresh
  } = useAccountsAutoRefresh({
    isBlocked: isAutoRefreshBlocked,
    onRefresh: refreshAccountsIncrementally
  })

  return {
    autoRefreshIntervals,
    autoRefreshEnabled,
    autoRefreshIntervalSeconds,
    autoRefreshCountdown,
    hasPendingListSync,
    todayStatsByAccountId,
    todayStatsLoading,
    todayStatsError,
    load,
    reload,
    debouncedReload,
    handlePageChange,
    handlePageSizeChange,
    refreshTodayStats,
    syncPendingListChanges,
    setAutoRefreshEnabled,
    handleAutoRefreshIntervalChange,
    enterAutoRefreshSilentWindow,
    pauseAutoRefresh,
    resumeAutoRefresh
  }
}
