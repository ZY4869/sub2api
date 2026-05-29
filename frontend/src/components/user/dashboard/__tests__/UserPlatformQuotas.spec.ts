import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import UserPlatformQuotas from '../UserPlatformQuotas.vue'

const apiMocks = vi.hoisted(() => ({
  getPlatformQuotas: vi.fn()
}))

vi.mock('@/api/user', () => ({
  userAPI: apiMocks
}))

vi.mock('@/utils/format', () => ({
  formatCurrency: (value: number) => `$${Number(value || 0).toFixed(2)}`,
  formatDateTime: (value: string) => `date:${value}`
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/components/common/LoadingSpinner.vue', () => ({
  default: { template: '<div data-test="loading" />' }
}))

vi.mock('@/components/common/PlatformIcon.vue', () => ({
  default: { props: ['platform'], template: '<span class="platform-icon">{{ platform }}</span>' }
}))

describe('UserPlatformQuotas', () => {
  let consoleErrorSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    apiMocks.getPlatformQuotas.mockReset()
    consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterEach(() => {
    consoleErrorSpy.mockRestore()
  })

  it('loads and renders user platform quota usage', async () => {
    apiMocks.getPlatformQuotas.mockResolvedValue([
      {
        platform: 'openai',
        daily: { limit: 1, used: 0.25, reset_at: '2026-05-27T00:00:00Z' },
        weekly: { limit: null, used: 0.5, reset_at: null },
        monthly: { limit: 10, used: 2, reset_at: '2026-06-01T00:00:00Z' }
      }
    ])

    const wrapper = mount(UserPlatformQuotas)
    await flushPromises()

    expect(apiMocks.getPlatformQuotas).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('dashboard.platformQuotas')
    expect(wrapper.text()).toContain('OpenAI')
    expect(wrapper.text()).toContain('dashboard.unlimited')
  })

  it('shows failure state and retries loading', async () => {
    apiMocks.getPlatformQuotas
      .mockRejectedValueOnce(new Error('load failed'))
      .mockResolvedValueOnce([])

    const wrapper = mount(UserPlatformQuotas)
    await flushPromises()

    expect(wrapper.text()).toContain('dashboard.platformQuotasLoadFailed')

    await wrapper.find('button').trigger('click')
    await flushPromises()

    expect(apiMocks.getPlatformQuotas).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('dashboard.platformQuotasEmpty')
  })
})
