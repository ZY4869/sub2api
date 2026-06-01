import { computed, type Ref } from 'vue'
import {
  formatDateTimeLocalInput,
  parseDateTimeLocalInput,
} from '@/utils/format'

export type AccountExpirationShortcutUnit = 'day' | 'month' | 'year'

export interface AccountExpirationShortcutsOptions {
  expiresAtInput: Ref<string>
  autoRenewEnabled: Ref<boolean>
}

export function useAccountExpirationShortcuts(options: AccountExpirationShortcutsOptions) {
  const expirationEnabled = computed({
    get: () => Boolean(options.expiresAtInput.value),
    set: (enabled: boolean) => {
      if (!enabled) {
        options.expiresAtInput.value = ''
        options.autoRenewEnabled.value = false
        return
      }
      if (!options.expiresAtInput.value) {
        options.expiresAtInput.value = buildExpiryInput(1, 'month')
      }
    }
  })

  const expiresAtPreview = computed(() => {
    if (!options.expiresAtInput.value) return ''
    const timestamp = parseDateTimeLocalInput(options.expiresAtInput.value)
    if (!timestamp) return ''
    return formatDateTimeLocalInput(timestamp)
  })

  function applyQuickExpiry(amount: number, unit: AccountExpirationShortcutUnit) {
    expirationEnabled.value = true
    options.expiresAtInput.value = buildExpiryInput(amount, unit)
  }

  return {
    expirationEnabled,
    expiresAtPreview,
    applyQuickExpiry
  }
}

export function buildExpiryInput(amount: number, unit: AccountExpirationShortcutUnit): string {
  const next = new Date()
  if (unit === 'day') {
    next.setDate(next.getDate() + amount)
  } else if (unit === 'month') {
    next.setMonth(next.getMonth() + amount)
  } else {
    next.setFullYear(next.getFullYear() + amount)
  }
  return formatLocalDateTimeForInput(next)
}

function formatLocalDateTimeForInput(value: Date): string {
  const year = value.getFullYear()
  const month = String(value.getMonth() + 1).padStart(2, '0')
  const day = String(value.getDate()).padStart(2, '0')
  const hours = String(value.getHours()).padStart(2, '0')
  const minutes = String(value.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hours}:${minutes}`
}
