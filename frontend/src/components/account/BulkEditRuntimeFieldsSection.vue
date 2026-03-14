<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ProxySelector from '@/components/common/ProxySelector.vue'
import type { Proxy } from '@/types'
import {
  normalizeAccountConcurrency,
  normalizeAccountLoadFactor
} from '@/utils/accountRuntimeSettings'

defineProps<{
  proxies: Proxy[]
}>()

const enableProxy = defineModel<boolean>('enableProxy', { required: true })
const proxyId = defineModel<number | null>('proxyId', { required: true })
const enableConcurrency = defineModel<boolean>('enableConcurrency', { required: true })
const concurrency = defineModel<number>('concurrency', { required: true })
const enableLoadFactor = defineModel<boolean>('enableLoadFactor', { required: true })
const loadFactor = defineModel<number | null>('loadFactor', { required: true })
const enablePriority = defineModel<boolean>('enablePriority', { required: true })
const priority = defineModel<number>('priority', { required: true })
const enableRateMultiplier = defineModel<boolean>('enableRateMultiplier', { required: true })
const rateMultiplier = defineModel<number>('rateMultiplier', { required: true })

const { t } = useI18n()

const handleConcurrencyInput = () => {
  concurrency.value = normalizeAccountConcurrency(concurrency.value)
}

const handleLoadFactorInput = () => {
  loadFactor.value = normalizeAccountLoadFactor(loadFactor.value)
}
</script>

<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="mb-3 flex items-center justify-between">
      <label
        id="bulk-edit-proxy-label"
        class="input-label mb-0"
        for="bulk-edit-proxy-enabled"
      >
        {{ t('admin.accounts.proxy') }}
      </label>
      <input
        v-model="enableProxy"
        id="bulk-edit-proxy-enabled"
        type="checkbox"
        aria-controls="bulk-edit-proxy-body"
        class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
      />
    </div>
    <div id="bulk-edit-proxy-body" :class="!enableProxy && 'pointer-events-none opacity-50'">
      <ProxySelector
        v-model="proxyId"
        :proxies="proxies"
        aria-labelledby="bulk-edit-proxy-label"
      />
    </div>
  </div>

  <div class="grid grid-cols-2 gap-4 border-t border-gray-200 pt-4 dark:border-dark-600 lg:grid-cols-4">
    <div>
      <div class="mb-3 flex items-center justify-between">
        <label
          id="bulk-edit-concurrency-label"
          class="input-label mb-0"
          for="bulk-edit-concurrency-enabled"
        >
          {{ t('admin.accounts.concurrency') }}
        </label>
        <input
          v-model="enableConcurrency"
          id="bulk-edit-concurrency-enabled"
          type="checkbox"
          aria-controls="bulk-edit-concurrency"
          class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        />
      </div>
      <input
        v-model.number="concurrency"
        id="bulk-edit-concurrency"
        type="number"
        min="1"
        :disabled="!enableConcurrency"
        class="input"
        :class="!enableConcurrency && 'cursor-not-allowed opacity-50'"
        aria-labelledby="bulk-edit-concurrency-label"
        @input="handleConcurrencyInput"
      />
    </div>

    <div>
      <div class="mb-3 flex items-center justify-between">
        <label
          id="bulk-edit-load-factor-label"
          class="input-label mb-0"
          for="bulk-edit-load-factor-enabled"
        >
          {{ t('admin.accounts.loadFactor') }}
        </label>
        <input
          v-model="enableLoadFactor"
          id="bulk-edit-load-factor-enabled"
          type="checkbox"
          aria-controls="bulk-edit-load-factor"
          class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        />
      </div>
      <input
        v-model.number="loadFactor"
        id="bulk-edit-load-factor"
        type="number"
        min="1"
        :disabled="!enableLoadFactor"
        class="input"
        :class="!enableLoadFactor && 'cursor-not-allowed opacity-50'"
        aria-labelledby="bulk-edit-load-factor-label"
        @input="handleLoadFactorInput"
      />
      <p class="input-hint">{{ t('admin.accounts.loadFactorHint') }}</p>
    </div>

    <div>
      <div class="mb-3 flex items-center justify-between">
        <label
          id="bulk-edit-priority-label"
          class="input-label mb-0"
          for="bulk-edit-priority-enabled"
        >
          {{ t('admin.accounts.priority') }}
        </label>
        <input
          v-model="enablePriority"
          id="bulk-edit-priority-enabled"
          type="checkbox"
          aria-controls="bulk-edit-priority"
          class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        />
      </div>
      <input
        v-model.number="priority"
        id="bulk-edit-priority"
        type="number"
        min="1"
        :disabled="!enablePriority"
        class="input"
        :class="!enablePriority && 'cursor-not-allowed opacity-50'"
        aria-labelledby="bulk-edit-priority-label"
      />
    </div>

    <div>
      <div class="mb-3 flex items-center justify-between">
        <label
          id="bulk-edit-rate-multiplier-label"
          class="input-label mb-0"
          for="bulk-edit-rate-multiplier-enabled"
        >
          {{ t('admin.accounts.billingRateMultiplier') }}
        </label>
        <input
          v-model="enableRateMultiplier"
          id="bulk-edit-rate-multiplier-enabled"
          type="checkbox"
          aria-controls="bulk-edit-rate-multiplier"
          class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
        />
      </div>
      <input
        v-model.number="rateMultiplier"
        id="bulk-edit-rate-multiplier"
        type="number"
        min="0"
        step="0.01"
        :disabled="!enableRateMultiplier"
        class="input"
        :class="!enableRateMultiplier && 'cursor-not-allowed opacity-50'"
        aria-labelledby="bulk-edit-rate-multiplier-label"
      />
      <p class="input-hint">{{ t('admin.accounts.billingRateMultiplierHint') }}</p>
    </div>
  </div>
</template>
