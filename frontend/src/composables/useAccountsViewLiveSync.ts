import { computed, ref, toRaw, watch, type ComputedRef, type Ref } from 'vue'
import { adminAPI } from '@/api/admin'
import { useAccountsAutoRefresh } from '@/composables/useAccountsAutoRefresh'
import { useAccountsTodayStats } from '@/composables/useAccountsTodayStats'
import { useAccountsPagePrefetch } from '@/composables/useAccountsPagePrefetch'
import { shouldReplaceAutoRefreshRow, type AccountListRequestParams } from '@/utils/accountListSync'
import type { Account, AccountTodayStatsCycleMode } from '@/types'

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
  accountTodayStatsCycleMode?: Ref<AccountTodayStatsCycleMode>
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
  accountTodayStatsCycleMode,
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
  const pagePrefetch = useAccountsPagePrefetch()
  const {
    todayStatsByAccountId,
    todayStatsLoading,
    todayStatsError,
    refreshTodayStats
  } = useAccountsTodayStats({
    accounts,
    hiddenColumns,
    cycleMode: accountTodayStatsCycleMode
  })

  const resetAutoRefreshCache = () => {
    autoRefreshETag.value = null
  }

  const cloneParams = () => ({ ...(toRaw(params) as AccountListRequestParams) })

  const prefetchNextPage = async () => {
    if (loading.value || autoRefreshFetching.value) return
    if (pagination.pages <= 1) return
    if (pagination.page >= pagination.pages) return

    await pagePrefetch.prefetchPage(
      pagination.page + 1,
      pagination.page_size,
      cloneParams(),
    )
  }

  const hydratePageFromCache = (page: number, pageSize: number) => {
    const cached = pagePrefetch.getCachedPage(page, pageSize, cloneParams())
    if (!cached) return false

    accounts.value = cached.items || []
    pagination.page = cached.page || page
    pagination.page_size = cached.page_size || pageSize
    pagination.total = cached.total || 0
    pagination.pages = cached.pages || 0
    hasPendingListSync.value = false
    pendingTodayStatsRefresh.value = true
    resetAutoRefreshCache()
    return true
  }

  const load = async () => {
    hasPendingListSync.value = false
    resetAutoRefreshCache()
    pendingTodayStatsRefresh.value = false
    pagePrefetch.clear()
    if (isFirstLoad.value) {
      params.lite = '1'
    }
    await baseLoad()
    pagePrefetch.storePageSnapshot({
      items: accounts.value,
      total: pagination.total,
      page: pagination.page,
      page_size: pagination.page_size,
      pages: pagination.pages,
    }, cloneParams())
    if (isFirstLoad.value) {
      isFirstLoad.value = false
      delete params.lite
    }
    await refreshTodayStats()
    await onListChanged?.()
    await prefetchNextPage()
  }

  const reload = async () => {
    hasPendingListSync.value = false
    resetAutoRefreshCache()
    pendingTodayStatsRefresh.value = false
    pagePrefetch.clear()
    await baseReload()
    pagePrefetch.storePageSnapshot({
      items: accounts.value,
      total: pagination.total,
      page: pagination.page,
      page_size: pagination.page_size,
      pages: pagination.pages,
    }, cloneParams())
    await refreshTodayStats()
    await onListChanged?.()
    await prefetchNextPage()
  }

  const debouncedReload = () => {
    hasPendingListSync.value = false
    resetAutoRefreshCache()
    pendingTodayStatsRefresh.value = true
    pagePrefetch.clear()
    baseDebouncedReload()
  }

  const handlePageChange = (page: number) => {
    hasPendingListSync.value = false
    if (hydratePageFromCache(page, pagination.page_size)) {
      refreshTodayStats()
        .then(() => {
          pendingTodayStatsRefresh.value = false
        })
        .then(() => onListChanged?.())
        .then(() => prefetchNextPage())
        .catch((error) => {
          console.error('Failed to refresh prefetched account page today stats:', error)
        })
      return
    }

    pagePrefetch.clear()
    resetAutoRefreshCache()
    pendingTodayStatsRefresh.value = true
    baseHandlePageChange(page)
  }

  const handlePageSizeChange = (size: number) => {
    hasPendingListSync.value = false
    resetAutoRefreshCache()
    pendingTodayStatsRefresh.value = true
    pagePrefetch.clear()
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
        pagePrefetch.storePageSnapshot({
          items: result.data.items || [],
          total: result.data.total || 0,
          page: result.data.page || pagination.page,
          page_size: result.data.page_size || pagination.page_size,
          pages: result.data.pages || 0,
        }, cloneParams())
        hasPendingListSync.value = false
        await onListChanged?.()
      }

      await refreshTodayStats()
      await prefetchNextPage()
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
    refreshAccountsIncrementally,
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
