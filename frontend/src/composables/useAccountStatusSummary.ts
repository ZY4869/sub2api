import { computed, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { useDebounceFn } from '@vueuse/core'
import { adminAPI } from '@/api/admin'
import type { AccountListRequestParams } from '@/utils/accountListSync'
import type { AccountStatusSummary } from '@/types'

const createEmptySummary = (): AccountStatusSummary => ({
  total: 0,
  by_status: {
    active: 0,
    inactive: 0,
    error: 0
  },
  rate_limited: 0,
  temp_unschedulable: 0,
  overloaded: 0,
  paused: 0,
  in_use: 0,
  by_platform: {},
  limited_breakdown: {
    total: 0,
    rate_429: 0,
    usage_5h: 0,
    usage_7d: 0
  }
})

export function useAccountStatusSummary(
  paramsSource: MaybeRefOrGetter<AccountListRequestParams>,
  options: {
    debounceMs?: number
  } = {}
) {
  const summary = ref<AccountStatusSummary>(createEmptySummary())
  const loading = ref(false)
  const error = ref<string | null>(null)
  const debounceMs = options.debounceMs ?? 250
  let requestId = 0

  const requestParams = computed(() => {
    const params = toValue(paramsSource)
    return {
      platform: String(params.platform || ''),
      type: String(params.type || ''),
      group: String(params.group || ''),
      search: String(params.search || ''),
      lifecycle: String(params.lifecycle || ''),
      limited_view: String(params.limited_view || ''),
      limited_reason: String(params.limited_reason || ''),
      runtime_view: String(params.runtime_view || '')
    }
  })

  const refresh = async () => {
    const currentRequestId = ++requestId
    loading.value = true
    error.value = null
    try {
      const nextSummary = await adminAPI.accounts.getStatusSummary(requestParams.value)
      if (currentRequestId !== requestId) {
        return
      }
      summary.value = nextSummary
    } catch (err: any) {
      if (currentRequestId !== requestId) {
        return
      }
      error.value = err?.message || 'Failed to load account summary'
    } finally {
      if (currentRequestId === requestId) {
        loading.value = false
      }
    }
  }

  const debouncedRefresh = useDebounceFn(refresh, debounceMs)

  watch(requestParams, () => {
    debouncedRefresh()
  }, { immediate: true })

  return {
    summary,
    loading,
    error,
    refresh,
    debouncedRefresh
  }
}
