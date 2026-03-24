import { ref, watch } from 'vue'
import type { AccountViewMode } from '@/types'

const VIEW_MODE_STORAGE_KEY = 'account-view-mode'
const GROUP_VIEW_STORAGE_KEY = 'account-group-view-enabled'

const readStoredViewMode = (): AccountViewMode => {
  if (typeof window === 'undefined') {
    return 'table'
  }
  const value = window.localStorage.getItem(VIEW_MODE_STORAGE_KEY)
  return value === 'card' ? 'card' : 'table'
}

const readStoredGroupView = () => {
  if (typeof window === 'undefined') {
    return false
  }
  return window.localStorage.getItem(GROUP_VIEW_STORAGE_KEY) === 'true'
}

export function useAccountViewMode() {
  const viewMode = ref<AccountViewMode>(readStoredViewMode())
  const groupViewEnabled = ref(readStoredGroupView())

  watch(viewMode, (value) => {
    if (typeof window === 'undefined') {
      return
    }
    window.localStorage.setItem(VIEW_MODE_STORAGE_KEY, value)
  })

  watch(groupViewEnabled, (value) => {
    if (typeof window === 'undefined') {
      return
    }
    window.localStorage.setItem(GROUP_VIEW_STORAGE_KEY, String(value))
  })

  return {
    viewMode,
    groupViewEnabled
  }
}
