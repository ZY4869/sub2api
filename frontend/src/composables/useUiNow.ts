import { computed, getCurrentScope, onScopeDispose, readonly, ref } from 'vue'

export const UI_NOW_TICK_MS = 5000

const nowMsState = ref(Date.now())

let subscriberCount = 0
let timer: ReturnType<typeof setInterval> | null = null

function syncNow(): void {
  nowMsState.value = Date.now()
}

function ensureTimer(): void {
  if (timer !== null) return
  syncNow()
  timer = setInterval(syncNow, UI_NOW_TICK_MS)
}

function releaseTimer(): void {
  if (subscriberCount > 0 || timer === null) return
  clearInterval(timer)
  timer = null
}

export function useUiNow() {
  subscriberCount += 1
  ensureTimer()

  if (getCurrentScope()) {
    onScopeDispose(() => {
      subscriberCount = Math.max(0, subscriberCount - 1)
      releaseTimer()
    })
  }

  return {
    nowMs: readonly(nowMsState),
    nowDate: computed(() => new Date(nowMsState.value)),
  }
}

export function resetUiNowForTests(): void {
  subscriberCount = 0
  if (timer !== null) {
    clearInterval(timer)
    timer = null
  }
  syncNow()
}
