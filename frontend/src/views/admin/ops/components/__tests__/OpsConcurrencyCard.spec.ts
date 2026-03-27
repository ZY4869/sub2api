import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import OpsConcurrencyCard from '../OpsConcurrencyCard.vue'

const mockGetConcurrencyStats = vi.fn()
const mockGetAccountAvailabilityStats = vi.fn()
const mockGetUserConcurrencyStats = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getConcurrencyStats: (...args: any[]) => mockGetConcurrencyStats(...args),
    getAccountAvailabilityStats: (...args: any[]) => mockGetAccountAvailabilityStats(...args),
    getUserConcurrencyStats: (...args: any[]) => mockGetUserConcurrencyStats(...args)
  }
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('OpsConcurrencyCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetConcurrencyStats.mockResolvedValue({
      enabled: true,
      platform: {},
      group: {},
      account: {}
    })
    mockGetAccountAvailabilityStats.mockResolvedValue({
      enabled: true,
      platform: {},
      group: {},
      account: {}
    })
    mockGetUserConcurrencyStats.mockResolvedValue({
      user: {}
    })
  })

  it('waits for parent refresh token before first load and avoids duplicate mount fetches', async () => {
    const wrapper = mount(OpsConcurrencyCard, {
      props: {
        refreshToken: 0,
        platformFilter: '',
        groupIdFilter: null
      }
    })

    await flushPromises()
    expect(mockGetConcurrencyStats).not.toHaveBeenCalled()
    expect(mockGetAccountAvailabilityStats).not.toHaveBeenCalled()

    await wrapper.setProps({ refreshToken: 1 })
    await flushPromises()

    expect(mockGetConcurrencyStats).toHaveBeenCalledTimes(1)
    expect(mockGetAccountAvailabilityStats).toHaveBeenCalledTimes(1)
    expect(mockGetConcurrencyStats).toHaveBeenCalledWith('', null)
    expect(mockGetAccountAvailabilityStats).toHaveBeenCalledWith('', null)
  })

  it('loads exactly once when mounted after the parent has already produced the first refresh token', async () => {
    mount(OpsConcurrencyCard, {
      props: {
        refreshToken: 1,
        platformFilter: 'openai',
        groupIdFilter: 7
      }
    })

    await flushPromises()
    expect(mockGetConcurrencyStats).toHaveBeenCalledTimes(1)
    expect(mockGetAccountAvailabilityStats).toHaveBeenCalledTimes(1)
    expect(mockGetConcurrencyStats).toHaveBeenCalledWith('openai', 7)
    expect(mockGetAccountAvailabilityStats).toHaveBeenCalledWith('openai', 7)
  })
})
