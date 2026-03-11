import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { metaAPI, type ExchangeRateInfo } from '@/api/meta'

const EXCHANGE_RATE_CACHE_TTL = 30 * 60 * 1000

export const useExchangeRateStore = defineStore('exchangeRate', () => {
  const exchangeRate = ref<ExchangeRateInfo | null>(null)
  const loading = ref(false)
  const loadedAt = ref(0)

  const hasFreshRate = computed(() => {
    if (!exchangeRate.value || !loadedAt.value) {
      return false
    }
    return Date.now() - loadedAt.value < EXCHANGE_RATE_CACHE_TTL
  })

  async function fetchExchangeRate(force = false): Promise<ExchangeRateInfo | null> {
    if (loading.value) {
      return exchangeRate.value
    }
    if (!force && hasFreshRate.value) {
      return exchangeRate.value
    }

    loading.value = true
    try {
      exchangeRate.value = await metaAPI.getUSDCNYExchangeRate(force)
      loadedAt.value = Date.now()
      return exchangeRate.value
    } catch {
      return exchangeRate.value
    } finally {
      loading.value = false
    }
  }

  return {
    exchangeRate,
    loading,
    hasFreshRate,
    fetchExchangeRate
  }
})
