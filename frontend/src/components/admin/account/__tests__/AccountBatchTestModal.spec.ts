import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountBatchTestModal from '../AccountBatchTestModal.vue'

const { getBatchTestModels, batchTestAccounts, showSuccess, showError } = vi.hoisted(() => ({
  getBatchTestModels: vi.fn(),
  batchTestAccounts: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getBatchTestModels,
      batchTestAccounts
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showError
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.accounts.batchTest.targetSingle') {
          return `single-${params?.name || ''}`
        }
        if (key === 'admin.accounts.batchTest.targetBatch') {
          return `batch-${params?.count || 0}`
        }
        return key
      }
    })
  }
})

function mountModal(props?: Record<string, unknown>) {
  return mount(AccountBatchTestModal, {
    props: {
      show: true,
      accounts: [
        {
          id: 101,
          name: 'OpenAI 101',
          platform: 'openai',
          type: 'apikey',
          status: 'active'
        }
      ],
      ...props
    } as any,
    global: {
      stubs: {
        BaseDialog: {
          props: ['show', 'title'],
          template: '<div v-if="show"><div class="dialog-title">{{ title }}</div><slot /><slot name="footer" /></div>'
        },
        TextArea: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<textarea class="textarea-stub" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
        },
        AccountTestModelSelectionFields: {
          props: ['emptyHint'],
          template: '<div class="model-selection-stub" :data-empty-hint="emptyHint"></div>'
        }
      }
    }
  })
}

describe('AccountBatchTestModal', () => {
  beforeEach(() => {
    getBatchTestModels.mockReset()
    batchTestAccounts.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
  })

  it('defaults to health_check and auto model strategy', async () => {
    batchTestAccounts.mockResolvedValueOnce({
      results: [
        {
          account_id: 101,
          account_name: 'OpenAI 101',
          status: 'success',
          resolved_model_id: 'gpt-5.4'
        }
      ]
    })

    const wrapper = mountModal()
    await flushPromises()
    await wrapper.get('[data-test="batch-test-submit"]').trigger('click')

    expect(batchTestAccounts).toHaveBeenCalledWith({
      account_ids: [101],
      model_input_mode: 'auto',
      test_mode: 'health_check'
    })
  })

  it('loads shared models for specified strategy and submits the first catalog model', async () => {
    getBatchTestModels.mockResolvedValueOnce([
      {
        id: 'gpt-5.4',
        display_name: 'GPT-5.4',
        source_protocol: 'openai'
      }
    ])
    batchTestAccounts.mockResolvedValueOnce({
      results: [
        {
          account_id: 101,
          account_name: 'OpenAI 101',
          status: 'success',
          resolved_model_id: 'gpt-5.4',
          resolved_source_protocol: 'openai'
        }
      ]
    })

    const wrapper = mountModal()
    await flushPromises()

    await wrapper.get('[data-test="batch-model-strategy-specified"]').trigger('click')
    await flushPromises()

    expect(getBatchTestModels).toHaveBeenCalledWith([101])

    await wrapper.get('[data-test="batch-test-submit"]').trigger('click')

    expect(batchTestAccounts).toHaveBeenCalledWith({
      account_ids: [101],
      model_id: 'gpt-5.4',
      model_input_mode: 'catalog',
      source_protocol: 'openai',
      test_mode: 'health_check'
    })
  })

  it('renders healthy and auto-blacklisted results after completion', async () => {
    batchTestAccounts.mockResolvedValueOnce({
      results: [
        {
          account_id: 101,
          account_name: 'OpenAI 101',
          status: 'success',
          resolved_model_id: 'gpt-5.4'
        },
        {
          account_id: 102,
          account_name: 'OpenAI 102',
          status: 'failed',
          error_message: 'Unauthorized',
          resolved_model_id: 'gpt-5.4',
          blacklist_advice_decision: 'auto_blacklisted',
          current_lifecycle_state: 'blacklisted'
        }
      ]
    })

    const wrapper = mountModal({
      accounts: [
        {
          id: 101,
          name: 'OpenAI 101',
          platform: 'openai',
          type: 'apikey',
          status: 'active'
        },
        {
          id: 102,
          name: 'OpenAI 102',
          platform: 'openai',
          type: 'apikey',
          status: 'active'
        }
      ]
    })
    await flushPromises()
    await wrapper.get('[data-test="batch-test-submit"]').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('completed')).toEqual([[]])
    expect(wrapper.text()).toContain('OpenAI 101')
    expect(wrapper.text()).toContain('OpenAI 102')
    expect(wrapper.text()).toContain('admin.accounts.batchTest.resultLabels.healthy')
    expect(wrapper.text()).toContain('admin.accounts.batchTest.resultLabels.autoBlacklisted')
  })
})
