import { computed, ref } from 'vue'
import type { AccountUsageDisplayMode } from '@/types'

const ACCOUNT_USAGE_DISPLAY_MODE_STORAGE_KEY = 'account-usage-display-mode'
const DEFAULT_ACCOUNT_USAGE_DISPLAY_MODE: AccountUsageDisplayMode = 'used'

function normalizeAccountUsageDisplayMode(value: string | null | undefined): AccountUsageDisplayMode {
  return value === 'remaining' ? 'remaining' : DEFAULT_ACCOUNT_USAGE_DISPLAY_MODE
}

export function getPersistedAccountUsageDisplayMode(): AccountUsageDisplayMode {
  try {
    return normalizeAccountUsageDisplayMode(localStorage.getItem(ACCOUNT_USAGE_DISPLAY_MODE_STORAGE_KEY))
  } catch {
    return DEFAULT_ACCOUNT_USAGE_DISPLAY_MODE
  }
}

export function setPersistedAccountUsageDisplayMode(mode: AccountUsageDisplayMode): void {
  try {
    localStorage.setItem(ACCOUNT_USAGE_DISPLAY_MODE_STORAGE_KEY, mode)
  } catch {
    // Ignore storage failures and keep the shared in-memory state.
  }
}

const accountUsageDisplayModeState = ref<AccountUsageDisplayMode>(getPersistedAccountUsageDisplayMode())

export function useAccountUsageDisplayMode() {
  const accountUsageDisplayMode = computed(() => accountUsageDisplayModeState.value)
  const isRemainingAccountUsageDisplayMode = computed(() => accountUsageDisplayModeState.value === 'remaining')

  const setAccountUsageDisplayMode = (mode: AccountUsageDisplayMode) => {
    accountUsageDisplayModeState.value = normalizeAccountUsageDisplayMode(mode)
    setPersistedAccountUsageDisplayMode(accountUsageDisplayModeState.value)
  }

  const toggleAccountUsageDisplayMode = () => {
    setAccountUsageDisplayMode(accountUsageDisplayModeState.value === 'remaining' ? 'used' : 'remaining')
  }

  return {
    accountUsageDisplayMode,
    isRemainingAccountUsageDisplayMode,
    setAccountUsageDisplayMode,
    toggleAccountUsageDisplayMode,
  }
}
