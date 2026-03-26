import { computed, onMounted, onUnmounted, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { adminAPI } from '@/api/admin'
import type { AccountListRequestParams } from '@/utils/accountListSync'
import type { AccountRuntimeSummary } from '@/types'

const POLL_INTERVAL_MS = 5000

const createEmptyRuntimeSummary = (): AccountRuntimeSummary => ({
  in_use: 0
})

export function useAccountsRuntimeSummary(
  paramsSource: MaybeRefOrGetter<AccountListRequestParams>,
  options: {
    enabled?: MaybeRefOrGetter<boolean>
    onSummaryChanged?: (next: AccountRuntimeSummary, previous: AccountRuntimeSummary) => void | Promise<void>
  } = {}
) {
  const summary = ref<AccountRuntimeSummary>(createEmptyRuntimeSummary())
  const loading = ref(false)
  const error = ref<string | null>(null)
  const etag = ref<string | null>(null)
  let requestId = 0

  const enabled = computed(() => toValue(options.enabled) !== false)
  const isDocumentVisible = () => typeof document === 'undefined' || !document.hidden

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

  const refresh = async (force = false) => {
    if (!enabled.value || !isDocumentVisible()) {
      return
    }

    const currentRequestId = ++requestId
    loading.value = true
    error.value = null

    try {
      const result = await adminAPI.accounts.getRuntimeSummaryWithEtag(
        requestParams.value,
        {
          etag: force ? null : etag.value
        }
      )
      if (currentRequestId !== requestId) {
        return
      }
      if (result.etag) {
        etag.value = result.etag
      }
      if (result.notModified || !result.data) {
        return
      }

      const previous = summary.value
      summary.value = result.data
      if (previous.in_use !== result.data.in_use) {
        await options.onSummaryChanged?.(result.data, previous)
      }
    } catch (err: any) {
      if (currentRequestId !== requestId) {
        return
      }
      error.value = err?.message || 'Failed to load runtime summary'
    } finally {
      if (currentRequestId === requestId) {
        loading.value = false
      }
    }
  }

  const reset = () => {
    etag.value = null
  }

  const { pause, resume } = useIntervalFn(() => {
    void refresh()
  }, POLL_INTERVAL_MS, { immediate: false })

  const handleVisibilityChange = () => {
    if (!enabled.value) {
      pause()
      return
    }
    if (isDocumentVisible()) {
      void refresh(true)
      resume()
      return
    }
    pause()
  }

  watch(requestParams, () => {
    reset()
    if (!enabled.value) {
      return
    }
    void refresh(true)
  }, { immediate: true })

  watch(enabled, (nextEnabled) => {
    if (!nextEnabled) {
      pause()
      return
    }
    if (isDocumentVisible()) {
      void refresh(true)
      resume()
    }
  })

  onMounted(() => {
    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', handleVisibilityChange)
    }
  })

  onUnmounted(() => {
    pause()
    if (typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  })

  return {
    summary,
    loading,
    error,
    refresh
  }
}
