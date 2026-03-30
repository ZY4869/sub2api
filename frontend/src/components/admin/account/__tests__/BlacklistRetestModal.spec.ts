import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BlacklistRetestModal from '../BlacklistRetestModal.vue'

const { getBlacklistRetestModels } = vi.hoisted(() => ({
  getBlacklistRetestModels: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getBlacklistRetestModels
    }
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        if (key === 'admin.accounts.blacklist.retestTargetSingle') {
          return `single-${params?.name || ''}`
        }
        if (key === 'admin.accounts.blacklist.retestTargetBatch') {
          return `batch-${params?.count || 0}`
        }
        return key
      }
    })
  }
})

function mountModal(props?: Record<string, unknown>) {
  return mount(BlacklistRetestModal, {
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
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>'
        },
        Select: {
          props: ['modelValue', 'options'],
          template: `
            <div class="select-stub">
              <div data-test="selected-option">
                <slot name="selected" :option="options.find((opt) => (opt.key || opt.id) === modelValue) || null" />
              </div>
              <div v-for="option in options" :key="option.key || option.id" data-test="option">
                <slot name="option" :option="option" :selected="(option.key || option.id) === modelValue" />
              </div>
            </div>
          `
        },
        Icon: true
      }
    }
  })
}

describe('BlacklistRetestModal', () => {
  beforeEach(() => {
    getBlacklistRetestModels.mockReset()
  })

  it('loads retest models on open and defaults to the first catalog model', async () => {
    getBlacklistRetestModels.mockResolvedValueOnce([
      {
        id: 'gpt-5.4',
        type: 'model',
        display_name: 'GPT-5.4',
        created_at: '',
        source_protocol: 'openai'
      },
      {
        id: 'gpt-4.1',
        type: 'model',
        display_name: 'GPT-4.1',
        created_at: '',
        source_protocol: 'openai'
      }
    ])

    const wrapper = mountModal()
    await flushPromises()

    expect(getBlacklistRetestModels).toHaveBeenCalledWith([101])

    await wrapper.get('[data-test="blacklist-retest-confirm"]').trigger('click')

    expect(wrapper.emitted('confirm')).toEqual([[
      {
        account_ids: [101],
        model_input_mode: 'catalog',
        model_id: 'gpt-5.4',
        source_protocol: 'openai'
      }
    ]])
  })

  it('submits manual model id and source protocol for protocol gateway targets', async () => {
    getBlacklistRetestModels.mockResolvedValueOnce([])

    const wrapper = mountModal({
      accounts: [
        {
          id: 202,
          name: 'Gateway 202',
          platform: 'protocol_gateway',
          type: 'apikey',
          status: 'active'
        }
      ]
    })
    await flushPromises()

    await wrapper.get('[data-test="model-input-mode-manual"]').trigger('click')
    await wrapper.get('[data-test="manual-model-id"]').setValue('claude-sonnet-4-5')
    await wrapper.get('[data-test="manual-source-protocol"]').setValue('anthropic')
    await wrapper.get('[data-test="blacklist-retest-confirm"]').trigger('click')

    expect(wrapper.emitted('confirm')).toEqual([[
      {
        account_ids: [202],
        model_input_mode: 'manual',
        manual_model_id: 'claude-sonnet-4-5',
        source_protocol: 'anthropic'
      }
    ]])
  })
})
