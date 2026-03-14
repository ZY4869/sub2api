<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ProxySelector from '@/components/common/ProxySelector.vue'
import type { Proxy } from '@/types'
import {
  normalizeAccountConcurrency,
  normalizeAccountLoadFactor
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

const handleConcurrencyInput = () => {
  concurrency.value = normalizeAccountConcurrency(concurrency.value)
}

const handleLoadFactorInput = () => {
  loadFactor.value = normalizeAccountLoadFactor(loadFactor.value)
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
    <label class="input-label">{{ t('admin.accounts.expiresAt') }}</label>
    <input v-model="expiresAtInput" type="datetime-local" class="input" />
    <p class="input-hint">{{ t('admin.accounts.expiresAtHint') }}</p>
  </div>
</template>
