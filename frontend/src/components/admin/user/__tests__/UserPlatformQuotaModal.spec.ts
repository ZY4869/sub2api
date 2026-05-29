import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import UserPlatformQuotaModal from '../UserPlatformQuotaModal.vue'

const apiMocks = vi.hoisted(() => ({
  getUserPlatformQuotas: vi.fn(),
  updateUserPlatformQuotas: vi.fn()
}))

const appStoreMocks = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: apiMocks
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStoreMocks
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

vi.mock('@/components/common/BaseDialog.vue', () => ({
  default: {
    name: 'BaseDialog',
    props: ['show', 'title'],
    emits: ['close'],
    template: '<section v-if="show"><h1>{{ title }}</h1><slot /><footer><slot name="footer" /></footer></section>'
  }
}))

vi.mock('@/components/common/LoadingSpinner.vue', () => ({
  default: { template: '<div data-test="loading" />' }
}))

vi.mock('@/components/common/PlatformIcon.vue', () => ({
  default: { props: ['platform'], template: '<span class="platform-icon">{{ platform }}</span>' }
}))

describe('UserPlatformQuotaModal', () => {
  const user = {
    id: 123,
    email: 'user@example.com'
  }

  beforeEach(() => {
    apiMocks.getUserPlatformQuotas.mockReset()
    apiMocks.updateUserPlatformQuotas.mockReset()
    appStoreMocks.showError.mockReset()
    appStoreMocks.showSuccess.mockReset()
  })

  it('loads quotas and saves normalized limits', async () => {
    apiMocks.getUserPlatformQuotas.mockResolvedValue([
      {
        platform: 'openai',
        daily: { limit: 1, used: 0.25, reset_at: '2026-05-27T00:00:00Z' },
        weekly: { limit: null, used: 0, reset_at: null },
        monthly: { limit: 10, used: 2, reset_at: '2026-06-01T00:00:00Z' }
      }
    ])
    apiMocks.updateUserPlatformQuotas.mockResolvedValue([])

    const wrapper = mount(UserPlatformQuotaModal, {
      props: { show: false, user: user as any }
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(apiMocks.getUserPlatformQuotas).toHaveBeenCalledWith(123)
    expect(wrapper.text()).toContain('OpenAI')

    const openAIDailyInput = wrapper
      .findAll('input[type="number"]')
      .find((input) => (input.element as HTMLInputElement).value === '1')
    expect(openAIDailyInput).toBeTruthy()
    await openAIDailyInput!.setValue('2.5')

    await wrapper.find('form').trigger('submit.prevent')
    await flushPromises()

    expect(apiMocks.updateUserPlatformQuotas).toHaveBeenCalledWith(
      123,
      expect.arrayContaining([
        expect.objectContaining({
          platform: 'openai',
          daily_limit_usd: 2.5,
          monthly_limit_usd: 10
        })
      ])
    )
    expect(appStoreMocks.showSuccess).toHaveBeenCalledWith('admin.users.platformQuotasSaved')
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('shows validation and load errors without saving invalid limits', async () => {
    apiMocks.getUserPlatformQuotas.mockRejectedValueOnce(new Error('load failed'))

    const wrapper = mount(UserPlatformQuotaModal, {
      props: { show: false, user: user as any }
    })
    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(appStoreMocks.showError).toHaveBeenCalledWith('admin.users.platformQuotasLoadFailed')

    const firstInput = wrapper.find('input[type="number"]')
    await firstInput.setValue('-1')
    await wrapper.find('form').trigger('submit.prevent')

    expect(appStoreMocks.showError).toHaveBeenCalledWith('admin.users.quotaInvalid')
    expect(apiMocks.updateUserPlatformQuotas).not.toHaveBeenCalled()
  })
})
