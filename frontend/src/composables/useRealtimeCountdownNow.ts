import { computed, getCurrentScope, onScopeDispose, readonly, ref, watch } from 'vue'
import { getActivePinia, storeToRefs } from 'pinia'
import { useAuthStore } from '@/stores/auth'
import { useUiNow } from '@/composables/useUiNow'

export type RealtimeCountdownScope = 'global' | 'accounts'

const frozenNowByScope = {
  global: ref<number | null>(null),
  accounts: ref<number | null>(null),
} as const

export function useRealtimeCountdownNow(scope: RealtimeCountdownScope) {
  const activePinia = getActivePinia()
  const user = activePinia
    ? storeToRefs(useAuthStore()).user
    : ref<{
        global_realtime_countdown_enabled?: boolean
        account_realtime_countdown_enabled?: boolean
      } | null>(null)
  const { nowMs } = useUiNow()

  const enabled = computed(() => {
    if (scope === 'accounts') {
      return user.value?.account_realtime_countdown_enabled !== false
    }
    return user.value?.global_realtime_countdown_enabled === true
  })

  if (enabled.value) {
    frozenNowByScope[scope].value = nowMs.value
  } else if (frozenNowByScope[scope].value === null) {
    frozenNowByScope[scope].value = nowMs.value
  }

  const stop = watch(
    [enabled, nowMs],
    ([isEnabled, liveNowMs]) => {
      if (isEnabled) {
        frozenNowByScope[scope].value = liveNowMs
        return
      }
      if (frozenNowByScope[scope].value === null) {
        frozenNowByScope[scope].value = liveNowMs
      }
    },
    { immediate: true }
  )

  if (getCurrentScope()) {
    onScopeDispose(stop)
  }

  const scopedNowMs = computed(() => {
    if (enabled.value) {
      return nowMs.value
    }
    return frozenNowByScope[scope].value ?? nowMs.value
  })

  return {
    enabled,
    nowMs: readonly(scopedNowMs),
    nowDate: computed(() => new Date(scopedNowMs.value)),
  }
}
