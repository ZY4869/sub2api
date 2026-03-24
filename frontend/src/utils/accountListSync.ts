import { buildOpenAIUsageRefreshKey } from '@/utils/accountUsageRefresh'
import type { Account, WindowStats } from '@/types'

export interface AccountListFilters {
  platform?: string
  type?: string
  status?: string
  group?: string
  search?: string
  lifecycle?: string
}

export type AccountListRequestParams = AccountListFilters & {
  lite?: string
}

export const buildDefaultTodayStats = (): WindowStats => ({
  requests: 0,
  tokens: 0,
  cost: 0,
  standard_cost: 0,
  user_cost: 0
})

const hasFutureTimestamp = (value: string | null | undefined, now: number) => {
  if (!value) return false
  const timestamp = new Date(value).getTime()
  return Number.isFinite(timestamp) && timestamp > now
}

const matchesGroupFilter = (account: Account, groupFilter: string) => {
  if (!groupFilter) return true
  if (groupFilter === 'ungrouped') {
    const hasGroupIDs = Array.isArray(account.group_ids) && account.group_ids.length > 0
    const hasGroups = Array.isArray(account.groups) && account.groups.length > 0
    return !hasGroupIDs && !hasGroups
  }

  const groupID = Number(groupFilter)
  if (!Number.isFinite(groupID) || groupID <= 0) return true

  if (account.group_ids?.includes(groupID)) return true
  return account.groups?.some(group => group.id === groupID) ?? false
}

const resolveAccountLifecycle = (account: Account) => account.lifecycle_state || 'normal'

export const accountMatchesFilters = (
  account: Account,
  filters: AccountListFilters,
  now: number = Date.now()
) => {
  if (filters.platform && account.platform !== filters.platform) return false
  if (filters.type && account.type !== filters.type) return false
  if (filters.lifecycle && filters.lifecycle !== 'all' && resolveAccountLifecycle(account) !== filters.lifecycle) {
    return false
  }
  if (!matchesGroupFilter(account, filters.group || '')) return false

  if (filters.status) {
    if (filters.status === 'rate_limited') {
      if (!hasFutureTimestamp(account.rate_limit_reset_at, now)) return false
    } else if (filters.status === 'paused') {
      if (account.schedulable) return false
    } else if (filters.status === 'temp_unschedulable') {
      if (!hasFutureTimestamp(account.temp_unschedulable_until, now)) return false
    } else if (account.status !== filters.status) {
      return false
    }
  }

  const search = String(filters.search || '').trim().toLowerCase()
  if (search && !account.name.toLowerCase().includes(search)) return false
  return true
}

export const mergeRuntimeFields = (oldAccount: Account, updatedAccount: Account): Account => ({
  ...updatedAccount,
  current_concurrency: updatedAccount.current_concurrency ?? oldAccount.current_concurrency,
  current_window_cost: updatedAccount.current_window_cost ?? oldAccount.current_window_cost,
  active_sessions: updatedAccount.active_sessions ?? oldAccount.active_sessions
})

export const shouldReplaceAutoRefreshRow = (current: Account, next: Account) => {
  return (
    current.updated_at !== next.updated_at ||
    current.lifecycle_state !== next.lifecycle_state ||
    current.lifecycle_reason_code !== next.lifecycle_reason_code ||
    current.lifecycle_reason_message !== next.lifecycle_reason_message ||
    current.blacklisted_at !== next.blacklisted_at ||
    current.blacklist_purge_at !== next.blacklist_purge_at ||
    current.current_concurrency !== next.current_concurrency ||
    current.current_window_cost !== next.current_window_cost ||
    current.active_sessions !== next.active_sessions ||
    current.schedulable !== next.schedulable ||
    current.status !== next.status ||
    current.rate_limit_reset_at !== next.rate_limit_reset_at ||
    current.overload_until !== next.overload_until ||
    current.temp_unschedulable_until !== next.temp_unschedulable_until ||
    buildOpenAIUsageRefreshKey(current) !== buildOpenAIUsageRefreshKey(next)
  )
}
