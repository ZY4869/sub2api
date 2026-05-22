import { computed, ref, type ComputedRef } from 'vue'
import { useIntervalFn } from '@vueuse/core'
import { useRealtimeCountdownNow } from '@/composables/useRealtimeCountdownNow'

const AUTO_REFRESH_STORAGE_KEY = 'account-auto-refresh'
const AUTO_REFRESH_SILENT_WINDOW_MS = 15000

export const accountAutoRefreshIntervals = [5, 10, 15, 30] as const

type AutoRefreshInterval = (typeof accountAutoRefreshIntervals)[number]

interface UseAccountsAutoRefreshOptions {
  isBlocked: ComputedRef<boolean>
  onRefresh: () => Promise<void>
}

const isValidAutoRefreshInterval = (value: number): value is AutoRefreshInterval => {
  return accountAutoRefreshIntervals.some(interval => interval === value)
}

export function useAccountsAutoRefresh({ isBlocked, onRefresh }: UseAccountsAutoRefreshOptions) {
  const autoRefreshEnabled = ref(false)
  const autoRefreshIntervalSeconds = ref<AutoRefreshInterval>(30)
  const autoRefreshSilentUntil = ref(0)
  const nextRefreshAtMs = ref(0)
  const { nowMs: displayNowMs } = useRealtimeCountdownNow('accounts')

  const loadSavedAutoRefresh = () => {
    try {
      const saved = localStorage.getItem(AUTO_REFRESH_STORAGE_KEY)
      if (!saved) return

      const parsed = JSON.parse(saved) as { enabled?: boolean; interval_seconds?: number }
      const savedInterval = Number(parsed.interval_seconds)
      autoRefreshEnabled.value = parsed.enabled === true
      if (isValidAutoRefreshInterval(savedInterval)) {
        autoRefreshIntervalSeconds.value = savedInterval
      }
    } catch (error) {
      console.error('Failed to load saved auto refresh settings:', error)
    }
  }

  const saveAutoRefreshToStorage = () => {
    try {
      localStorage.setItem(
        AUTO_REFRESH_STORAGE_KEY,
        JSON.stringify({
          enabled: autoRefreshEnabled.value,
          interval_seconds: autoRefreshIntervalSeconds.value
        })
      )
    } catch (error) {
      console.error('Failed to save auto refresh settings:', error)
    }
  }

  const enterAutoRefreshSilentWindow = () => {
    autoRefreshSilentUntil.value = Date.now() + AUTO_REFRESH_SILENT_WINDOW_MS
    nextRefreshAtMs.value = autoRefreshSilentUntil.value
  }

  const inAutoRefreshSilentWindow = () => {
    return Date.now() < autoRefreshSilentUntil.value
  }

  const resetNextRefreshAt = () => {
    nextRefreshAtMs.value = Date.now() + autoRefreshIntervalSeconds.value * 1000
  }

  const autoRefreshCountdown = computed(() => {
    if (!autoRefreshEnabled.value || nextRefreshAtMs.value <= 0) {
      return 0
    }
    return Math.max(0, Math.ceil((nextRefreshAtMs.value - displayNowMs.value) / 1000))
  })

  const { pause: pauseAutoRefresh, resume: resumeAutoRefresh } = useIntervalFn(
    async () => {
      if (!autoRefreshEnabled.value) return
      if (typeof document !== 'undefined' && document.hidden) return
      if (isBlocked.value) return
      if (inAutoRefreshSilentWindow()) {
        return
      }

      if (nextRefreshAtMs.value <= 0) {
        resetNextRefreshAt()
        return
      }

      if (Date.now() >= nextRefreshAtMs.value) {
        resetNextRefreshAt()
        await onRefresh()
        return
      }
    },
    1000,
    { immediate: false }
  )

  const setAutoRefreshEnabled = (enabled: boolean) => {
    autoRefreshEnabled.value = enabled
    saveAutoRefreshToStorage()
    if (enabled) {
      resetNextRefreshAt()
      resumeAutoRefresh()
    } else {
      pauseAutoRefresh()
      nextRefreshAtMs.value = 0
    }
  }

  const setAutoRefreshInterval = (seconds: AutoRefreshInterval) => {
    autoRefreshIntervalSeconds.value = seconds
    saveAutoRefreshToStorage()
    if (autoRefreshEnabled.value) {
      resetNextRefreshAt()
    }
  }

  const handleAutoRefreshIntervalChange = (seconds: number) => {
    if (isValidAutoRefreshInterval(seconds)) {
      setAutoRefreshInterval(seconds)
    }
  }

  if (typeof window !== 'undefined') {
    loadSavedAutoRefresh()
  }

  return {
    autoRefreshIntervals: accountAutoRefreshIntervals,
    autoRefreshEnabled,
    autoRefreshIntervalSeconds,
    autoRefreshCountdown,
    nextRefreshAtMs,
    setAutoRefreshEnabled,
    handleAutoRefreshIntervalChange,
    enterAutoRefreshSilentWindow,
    pauseAutoRefresh,
    resumeAutoRefresh
  }
}
