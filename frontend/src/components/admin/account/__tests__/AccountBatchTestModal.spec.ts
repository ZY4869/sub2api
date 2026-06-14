import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountBatchTestModal from '../AccountBatchTestModal.vue'

const { getBatchTestModels, batchTestAccounts, listAccounts, showSuccess, showError } = vi.hoisted(() => ({
  getBatchTestModels: vi.fn(),
  batchTestAccounts: vi.fn(),
  listAccounts: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getBatchTestModels,
      batchTestAccounts,
      list: listAccounts
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
        if (key === 'admin.accounts.batchTest.summary.success') {
          return `summary-success-${params?.count || 0}`
        }
        if (key === 'admin.accounts.batchTest.summary.failed') {
          return `summary-failed-${params?.count || 0}`
        }
        if (key === 'admin.accounts.batchTest.summary.autoBlacklisted') {
          return `summary-auto-blacklisted-${params?.count || 0}`
        }
        if (key === 'admin.accounts.batchTest.summary.needsReauth') {
          return `summary-needs-reauth-${params?.count || 0}`
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
    listAccounts.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
    listAccounts.mockResolvedValue({ total: 0, items: [] })
  })

  it('defaults to health_check and auto model strategy', async () => {
    batchTestAccounts.mockResolvedValueOnce({
      success_count: 7,
      failed_count: 8,
      auto_blacklisted_count: 9,
      needs_reauth_count: 10,
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
      test_mode: 'health_check',
      execution_mode: 'concurrent',
      concurrency: 4
    })
  })

  it('uses a single-account stored default when loading specified catalog models', async () => {
    getBatchTestModels.mockResolvedValueOnce([
      {
        id: 'friendly-gpt',
        display_name: 'GPT-5.4',
        target_model_id: 'gpt-5.4',
        provider: 'openai',
        source_protocol: 'openai'
      },
      {
        id: 'friendly-sonnet',
        display_name: 'Claude Sonnet 4.5',
        target_model_id: 'claude-sonnet-4.5',
        provider: 'anthropic',
        source_protocol: 'anthropic'
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

    const wrapper = mountModal({
      accounts: [
        {
          id: 101,
          name: 'OpenAI 101',
          platform: 'protocol_gateway',
          type: 'apikey',
          status: 'active',
          extra: {
            gateway_test_provider: 'anthropic',
            gateway_test_model_id: 'claude-sonnet-4.5'
          }
        }
      ]
    })
    await flushPromises()

    await wrapper.get('[data-test="batch-model-strategy-specified"]').trigger('click')
    await flushPromises()

    expect(getBatchTestModels).toHaveBeenCalledWith({ account_ids: [101] })

    await wrapper.get('[data-test="batch-test-submit"]').trigger('click')

    expect(batchTestAccounts).toHaveBeenCalledWith({
      account_ids: [101],
      model_id: 'friendly-sonnet',
      model_input_mode: 'catalog',
      source_protocol: 'anthropic',
      target_provider: 'anthropic',
      target_model_id: 'claude-sonnet-4.5',
      test_mode: 'health_check',
      execution_mode: 'concurrent',
      concurrency: 4
    })
  })

  it('falls back to the first catalog model when multi-account defaults are not shared', async () => {
    getBatchTestModels.mockResolvedValueOnce([
      {
        id: 'friendly-gpt',
        display_name: 'GPT-5.4',
        target_model_id: 'gpt-5.4',
        provider: 'openai',
        source_protocol: 'openai'
      },
      {
        id: 'friendly-sonnet',
        display_name: 'Claude Sonnet 4.5',
        target_model_id: 'claude-sonnet-4.5',
        provider: 'anthropic',
        source_protocol: 'anthropic'
      }
    ])
    batchTestAccounts.mockResolvedValueOnce({ results: [] })

    const wrapper = mountModal({
      accounts: [
        {
          id: 101,
          name: 'Gateway 101',
          platform: 'protocol_gateway',
          type: 'apikey',
          status: 'active',
          extra: {
            gateway_test_provider: 'openai',
            gateway_test_model_id: 'gpt-5.4'
          }
        },
        {
          id: 102,
          name: 'Gateway 102',
          platform: 'protocol_gateway',
          type: 'apikey',
          status: 'active',
          extra: {
            gateway_test_provider: 'anthropic',
            gateway_test_model_id: 'claude-sonnet-4.5'
          }
        }
      ]
    })
    await flushPromises()

    await wrapper.get('[data-test="batch-model-strategy-specified"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-test="batch-test-submit"]').trigger('click')

    expect(batchTestAccounts).toHaveBeenCalledWith({
      account_ids: [101, 102],
      model_id: 'friendly-gpt',
      model_input_mode: 'catalog',
      source_protocol: 'openai',
      target_provider: 'openai',
      target_model_id: 'gpt-5.4',
      test_mode: 'health_check',
      execution_mode: 'concurrent',
      concurrency: 4
    })
  })

  it('uses filters for one-click current preview and can clear platform scope', async () => {
    listAccounts.mockResolvedValue({ total: 2, items: [] })
    batchTestAccounts.mockResolvedValueOnce({
      results: [
        {
          account_id: 201,
          account_name: 'OpenAI 201',
          platform: 'openai',
          status: 'success'
        }
      ]
    })

    const wrapper = mountModal({
      accounts: [],
      filters: {
        platform: 'openai',
        status: 'active',
        search: ''
      },
      filtersTotal: 2,
      defaultTestMode: 'real_forward'
    })
    await flushPromises()

    const scopeButtons = wrapper.findAll('button')
    const allPlatformsButton = scopeButtons.find((button) =>
      button.text().includes('admin.accounts.batchTest.scopeAllPlatforms')
    )
    expect(allPlatformsButton).toBeTruthy()
    await allPlatformsButton?.trigger('click')
    await flushPromises()

    expect(listAccounts).toHaveBeenLastCalledWith(1, 1, {
      status: 'active'
    })

    await wrapper.get('[data-test="batch-test-submit"]').trigger('click')

    expect(batchTestAccounts).toHaveBeenCalledWith({
      filters: {
        status: 'active'
      },
      model_input_mode: 'auto',
      test_mode: 'real_forward',
      execution_mode: 'concurrent',
      concurrency: 4,
      prompt: 'Output exactly: OK'
    })
  })

  it('renders healthy and auto-blacklisted results after completion', async () => {
    batchTestAccounts.mockResolvedValueOnce({
      success_count: 7,
      failed_count: 8,
      auto_blacklisted_count: 9,
      needs_reauth_count: 10,
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
          blacklist_advice_reason_code: 'credentials_need_reauth',
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
    expect(wrapper.text()).toContain('admin.accounts.status.needsReauth')
    expect(wrapper.text()).toContain('summary-success-7')
    expect(wrapper.text()).toContain('summary-failed-8')
    expect(wrapper.text()).toContain('summary-auto-blacklisted-9')
    expect(wrapper.text()).toContain('summary-needs-reauth-10')
  })
})
