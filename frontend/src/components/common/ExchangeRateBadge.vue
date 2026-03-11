<template>
  <div v-if="exchangeRate" :class="badgeClass">
    <span class="font-medium">{{ t('common.todayExchangeRate') }}</span>
    <span>{{ exchangeRate.base }}→{{ exchangeRate.quote }} {{ exchangeRate.rate.toFixed(4) }}</span>
    <span class="text-gray-500 dark:text-dark-400">· {{ exchangeRate.date }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useExchangeRateStore } from '@/stores/exchangeRate'

const props = withDefaults(
  defineProps<{
    variant?: 'header' | 'auth'
  }>(),
  {
    variant: 'header'
  }
)

const { t } = useI18n()
const exchangeRateStore = useExchangeRateStore()

const exchangeRate = computed(() => exchangeRateStore.exchangeRate)
const badgeClass = computed(() =>
  props.variant === 'auth'
    ? 'inline-flex flex-wrap items-center gap-1.5 rounded-full border border-white/60 bg-white/80 px-3 py-1 text-xs text-gray-700 shadow-sm backdrop-blur dark:border-dark-700 dark:bg-dark-900/80 dark:text-gray-200'
    : 'inline-flex flex-wrap items-center gap-1.5 rounded-xl bg-primary-50 px-3 py-1.5 text-xs text-primary-700 dark:bg-primary-900/20 dark:text-primary-300'
)

onMounted(() => {
  exchangeRateStore.fetchExchangeRate()
})
</script>
