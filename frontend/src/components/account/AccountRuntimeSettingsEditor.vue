<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ProxySelector from '@/components/common/ProxySelector.vue'
import type { Proxy } from '@/types'
import {
  formatDateTimeLocalInput,
  parseDateTimeLocalInput,
} from '@/utils/format'
import {
  normalizeAccountConcurrency,
  normalizeAccountLoadFactor,
} from '@/utils/accountFormShared'

defineProps<{
  proxies: Proxy[]
}>()

const proxyId = defineModel<number | null>('proxyId', { required: true })
const concurrency = defineModel<number>('concurrency', { required: true })
const loadFactor = defineModel<number | null>('loadFactor', { required: true })
const priority = defineModel<number>('priority', { required: true })
const rateMultiplier = defineModel<number>('rateMultiplier', { required: true })
const expiresAtInput = defineModel<string>('expiresAtInput', { required: true })

const { t } = useI18n()

const expirationEnabled = computed({
  get: () => Boolean(expiresAtInput.value),
  set: (enabled: boolean) => {
    if (!enabled) {
      expiresAtInput.value = ''
      return
    }
    if (!expiresAtInput.value) {
      expiresAtInput.value = buildExpiryInput(1, 'month')
    }
  }
})

const expiresAtPreview = computed(() => {
  if (!expiresAtInput.value) return ''
  const timestamp = parseDateTimeLocalInput(expiresAtInput.value)
  if (!timestamp) return ''
  return formatDateTimeLocalInput(timestamp)
})

const handleConcurrencyInput = () => {
  concurrency.value = normalizeAccountConcurrency(concurrency.value)
}

const handleLoadFactorInput = () => {
  loadFactor.value = normalizeAccountLoadFactor(loadFactor.value)
}

function buildExpiryInput(amount: number, unit: 'month' | 'year'): string {
  const next = new Date()
  if (unit === 'month') {
    next.setMonth(next.getMonth() + amount)
  } else {
    next.setFullYear(next.getFullYear() + amount)
  }
  const year = next.getFullYear()
  const month = String(next.getMonth() + 1).padStart(2, '0')
  const day = String(next.getDate()).padStart(2, '0')
  const hours = String(next.getHours()).padStart(2, '0')
  const minutes = String(next.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hours}:${minutes}`
}

function applyQuickExpiry(unit: 'month' | 'year') {
  expirationEnabled.value = true
  expiresAtInput.value = buildExpiryInput(1, unit)
}
</script>

<template>
  <div>
    <label class="input-label">{{ t('admin.accounts.proxy') }}</label>
    <ProxySelector v-model="proxyId" :proxies="proxies" />
  </div>

  <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <div>
      <label class="input-label">{{ t('admin.accounts.concurrency') }}</label>
      <input
        v-model.number="concurrency"
        type="number"
        min="1"
        class="input"
        @input="handleConcurrencyInput"
      />
    </div>
    <div>
      <label class="input-label">{{ t('admin.accounts.loadFactor') }}</label>
      <input
        v-model.number="loadFactor"
        type="number"
        min="1"
        class="input"
        :placeholder="String(concurrency || 1)"
        @input="handleLoadFactorInput"
      />
      <p class="input-hint">{{ t('admin.accounts.loadFactorHint') }}</p>
    </div>
    <div>
      <label class="input-label">{{ t('admin.accounts.priority') }}</label>
      <input
        v-model.number="priority"
        type="number"
        min="1"
        class="input"
        data-tour="account-form-priority"
      />
      <p class="input-hint">{{ t('admin.accounts.priorityHint') }}</p>
    </div>
    <div>
      <label class="input-label">{{ t('admin.accounts.billingRateMultiplier') }}</label>
      <input v-model.number="rateMultiplier" type="number" min="0" step="0.001" class="input" />
      <p class="input-hint">{{ t('admin.accounts.billingRateMultiplierHint') }}</p>
    </div>
  </div>

  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="flex items-center justify-between gap-3">
      <label class="input-label mb-0">{{ t('admin.accounts.expiresAt') }}</label>
      <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
        <input v-model="expirationEnabled" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
        <span>{{ t('admin.accounts.expirationEnabled') }}</span>
      </label>
    </div>
    <div v-if="expirationEnabled" class="mt-3 space-y-3">
      <div class="flex flex-wrap gap-2">
        <button type="button" class="btn btn-secondary btn-sm" @click="applyQuickExpiry('month')">
          {{ t('admin.accounts.expirationQuickMonth') }}
        </button>
        <button type="button" class="btn btn-secondary btn-sm" @click="applyQuickExpiry('year')">
          {{ t('admin.accounts.expirationQuickYear') }}
        </button>
      </div>
      <input v-model="expiresAtInput" type="datetime-local" class="input" />
      <p v-if="expiresAtPreview" class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.expiresAtPreview', { value: expiresAtPreview.replace('T', ' ') }) }}
      </p>
    </div>
    <p class="input-hint">{{ t('admin.accounts.expiresAtHint') }}</p>
  </div>
</template>
