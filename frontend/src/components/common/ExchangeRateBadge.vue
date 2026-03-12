<template>
  <div v-if="exchangeRate" :class="badgeClass">
    <span class="font-medium">{{ t('common.todayExchangeRate') }}</span>
    <span>{{ exchangeRate.base }}/{{ exchangeRate.quote }} {{ exchangeRate.rate.toFixed(4) }}</span>
    <span class="text-gray-500 dark:text-dark-400">{{ exchangeRate.date }}</span>
    <button
      type="button"
      class="inline-flex h-5 w-5 items-center justify-center rounded-full text-current/70 transition hover:bg-black/5 hover:text-current disabled:cursor-not-allowed disabled:opacity-60 dark:hover:bg-white/10"
      :title="t('common.refresh')"
      :disabled="exchangeRateStore.loading"
      @click="handleRefresh"
    >
      <Icon name="refresh" size="xs" :class="exchangeRateStore.loading ? 'animate-spin' : ''" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
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

async function handleRefresh() {
  await exchangeRateStore.fetchExchangeRate(true)
}

onMounted(() => {
  exchangeRateStore.fetchExchangeRate()
})
</script>
