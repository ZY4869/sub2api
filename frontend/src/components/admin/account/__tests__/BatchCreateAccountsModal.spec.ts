import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BatchCreateAccountsModal from '../BatchCreateAccountsModal.vue'

const {
  checkMixedChannelRisk,
  batchCreateAccounts,
  showSuccess,
  showWarning,
  showError
} = vi.hoisted(() => ({
  checkMixedChannelRisk: vi.fn(),
  batchCreateAccounts: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      checkMixedChannelRisk,
      batchCreateAccounts
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showWarning,
    showError
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('BatchCreateAccountsModal', () => {
  beforeEach(() => {
    checkMixedChannelRisk.mockReset()
    batchCreateAccounts.mockReset()
    showSuccess.mockReset()
    showWarning.mockReset()
    showError.mockReset()

    checkMixedChannelRisk.mockResolvedValue({
      has_risk: false
    })

    Object.defineProperty(globalThis, 'localStorage', {
      value: {
        getItem: vi.fn(() => ''),
        setItem: vi.fn(),
        removeItem: vi.fn()
      },
      configurable: true
    })
  })

  it('retries batch create with confirm flag after mixed-channel 409', async () => {
    batchCreateAccounts
      .mockRejectedValueOnce({
        status: 409,
        error: 'mixed_channel_warning',
        message: 'mixed warning'
      })
      .mockResolvedValueOnce({
        created_count: 1,
        failed_count: 0,
        results: []
      })

    const wrapper = mount(BatchCreateAccountsModal, {
      props: {
        show: true,
        proxies: [],
        groups: []
      },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          ConfirmDialog: {
            props: ['show', 'message'],
            emits: ['confirm', 'cancel'],
            template: `
              <div v-if="show" class="confirm-dialog">
                <div class="confirm-message">{{ message }}</div>
                <button class="confirm-button" @click="$emit('confirm')" />
                <button class="cancel-button" @click="$emit('cancel')" />
              </div>
            `
          },
          GroupSelector: true,
          AccountRuntimeSettingsEditor: true
        }
      }
    })

    await wrapper.get('textarea.font-mono').setValue('sk-ant-sid01-test')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(checkMixedChannelRisk).toHaveBeenCalledTimes(1)
    expect(batchCreateAccounts).toHaveBeenCalledTimes(1)
    expect(batchCreateAccounts.mock.calls[0][0].confirm_mixed_channel_risk).toBeUndefined()
    expect(wrapper.find('.confirm-dialog').exists()).toBe(true)

    await wrapper.get('.confirm-button').trigger('click')
    await flushPromises()

    expect(batchCreateAccounts).toHaveBeenCalledTimes(2)
    expect(batchCreateAccounts.mock.calls[1][0]).toMatchObject({
      items: ['sk-ant-sid01-test'],
      confirm_mixed_channel_risk: true
    })
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.batchCreateSuccess')
  })
})
