import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useExchangeRateStore } from '@/stores/exchangeRate'

const mockGetUSDCNYExchangeRate = vi.fn()

vi.mock('@/api/meta', () => ({
  metaAPI: {
    getUSDCNYExchangeRate: (...args: any[]) => mockGetUSDCNYExchangeRate(...args)
  }
}))

describe('useExchangeRateStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('caches successful exchange rate responses', async () => {
    mockGetUSDCNYExchangeRate.mockResolvedValue({
      base: 'USD',
      quote: 'CNY',
      rate: 7.2,
      date: '2026-03-11'
    })
    const store = useExchangeRateStore()

    const first = await store.fetchExchangeRate()
    const second = await store.fetchExchangeRate()

    expect(first?.rate).toBe(7.2)
    expect(second?.rate).toBe(7.2)
    expect(mockGetUSDCNYExchangeRate).toHaveBeenCalledTimes(1)
    expect(mockGetUSDCNYExchangeRate).toHaveBeenCalledWith(false)
  })

  it('keeps previous exchange rate when force refresh fails', async () => {
    mockGetUSDCNYExchangeRate.mockResolvedValueOnce({
      base: 'USD',
      quote: 'CNY',
      rate: 7.2,
      date: '2026-03-11'
    })
    mockGetUSDCNYExchangeRate.mockRejectedValueOnce(new Error('network error'))
    const store = useExchangeRateStore()

    await store.fetchExchangeRate()
    const result = await store.fetchExchangeRate(true)

    expect(result?.rate).toBe(7.2)
    expect(store.exchangeRate?.rate).toBe(7.2)
    expect(mockGetUSDCNYExchangeRate).toHaveBeenCalledTimes(2)
    expect(mockGetUSDCNYExchangeRate).toHaveBeenNthCalledWith(1, false)
    expect(mockGetUSDCNYExchangeRate).toHaveBeenNthCalledWith(2, true)
  })
})
