import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import ChannelMonitorFormDialog from '../ChannelMonitorFormDialog.vue'

const mocks = vi.hoisted(() => ({
  createMonitor: vi.fn(),
  getBatchTestModels: vi.fn(),
  listAccounts: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  updateMonitor: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getBatchTestModels: mocks.getBatchTestModels,
      list: mocks.listAccounts,
    },
    channelMonitors: {
      createMonitor: mocks.createMonitor,
      updateMonitor: mocks.updateMonitor,
    },
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: mocks.showError,
    showSuccess: mocks.showSuccess,
  }),
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      locale: {
        value: 'zh-CN',
      },
      t: (key: string, params?: Record<string, unknown>) => params ? `${key}:${JSON.stringify(params)}` : key,
    },
  }),
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => params ? `${key}:${JSON.stringify(params)}` : key,
  }),
}))

function mountDialog(props: Partial<InstanceType<typeof ChannelMonitorFormDialog>['$props']> = {}) {
  return mount(ChannelMonitorFormDialog, {
    attachTo: document.body,
    props: {
      show: true,
      monitor: null,
      templates: [],
      ...props,
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
        ModelIcon: true,
        ModelPlatformIcon: true,
        Icon: true,
        Select: {
          props: ['modelValue', 'options', 'disabled'],
          emits: ['update:modelValue', 'change'],
          template: `
            <select
              :value="modelValue ?? ''"
              :disabled="disabled"
              @change="$emit('update:modelValue', value($event)); $emit('change', value($event), option(value($event)))"
            >
              <option v-for="item in options" :key="String(item.value)" :value="item.value ?? ''">
                {{ item.label }}
              </option>
            </select>
          `,
          methods: {
            value(event: Event) {
              const raw = (event.target as HTMLSelectElement).value
              return raw === '' ? null : raw
            },
            option(value: string | null) {
              return this.options.find((item: any) => String(item.value ?? '') === String(value ?? '')) || null
            },
          },
        },
        Toggle: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<input type="checkbox" :checked="modelValue" @change="$emit(\'update:modelValue\', $event.target.checked)" />',
        },
      },
    },
  })
}

describe('ChannelMonitorFormDialog', () => {
  beforeEach(() => {
    mocks.createMonitor.mockReset()
    mocks.getBatchTestModels.mockReset()
    mocks.listAccounts.mockReset()
    mocks.showError.mockReset()
    mocks.showSuccess.mockReset()
    mocks.updateMonitor.mockReset()
    mocks.listAccounts.mockResolvedValue({
      items: [
        { id: 101, name: 'alpha', platform: 'protocol_gateway', status: 'active' },
        { id: 102, name: 'beta', platform: 'protocol_gateway', status: 'active' },
      ],
    })
    mocks.getBatchTestModels.mockResolvedValue([
      {
        id: 'claude-sonnet',
        type: 'model',
        display_name: 'Claude Sonnet',
        created_at: '',
        provider: 'anthropic',
        provider_label: 'Anthropic',
        source_protocol: 'anthropic',
        availability_state: 'verified',
        status: 'stable',
      },
      {
        id: 'gemini-pro',
        type: 'model',
        display_name: 'Gemini Pro',
        created_at: '',
        provider: 'gemini',
        provider_label: 'Google',
        source_protocol: 'gemini',
        availability_state: 'verified',
        status: 'stable',
      },
    ])
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('shows backend detail when saving a new monitor fails', async () => {
    mocks.createMonitor.mockRejectedValue({
      response: {
        data: {
          detail: 'endpoint is unreachable',
        },
      },
    })

    const wrapper = mountDialog()
    const inputs = wrapper.findAll('input.input')
    await inputs[0].setValue('Primary OpenAI monitor')
    await inputs[1].setValue('https://example.test/v1/chat/completions')
    await inputs[4].setValue('gpt-5.4')

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(mocks.showError).toHaveBeenCalledWith('endpoint is unreachable')
  })

  it('submits direct create payload from the footer save button', async () => {
    mocks.createMonitor.mockResolvedValue({ id: 1 })

    const wrapper = mountDialog()
    const inputs = wrapper.findAll('input.input')
    await inputs[0].setValue('Primary OpenAI monitor')
    await inputs[1].setValue('https://example.test/v1/chat/completions')
    await inputs[4].setValue('gpt-5.4')

    wrapper.find('button[type="submit"][form="channel-monitor-form"]').element.click()
    await flushPromises()

    expect(mocks.createMonitor).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Primary OpenAI monitor',
      probe_mode: 'direct',
      endpoint: 'https://example.test/v1/chat/completions',
      primary_model_id: 'gpt-5.4',
      jitter_seconds: 0,
      model_probe_strategy: 'all_selected',
      model_source_protocols: {},
      enabled: false,
    }))
    expect(wrapper.emitted('saved')).toBeTruthy()
  })

  it('loads shared account models and submits account-pool payload with template sync', async () => {
    mocks.createMonitor.mockResolvedValue({ id: 2 })

    const wrapper = mountDialog()
    const selects = wrapper.findAll('select')
    await selects[0].setValue('account_pool')
    await flushPromises()

    const inputs = wrapper.findAll('input.input')
    await inputs[0].setValue('Pool health')
    await wrapper.findAll('input[type="checkbox"]')[0].setValue(true)
    await wrapper.findAll('input[type="checkbox"]')[1].setValue(true)
    await flushPromises()

    expect(mocks.getBatchTestModels).toHaveBeenCalledWith({ account_ids: [101, 102] })

    const refreshedSelects = wrapper.findAll('select')
    await refreshedSelects[6].setValue('claude-sonnet')
    await refreshedSelects[7].setValue('gemini-pro')
    await flushPromises()

    const textarea = wrapper.find('textarea.input')
    await textarea.setValue('请只回复 {{challenge}}')
    const checkboxes = wrapper.findAll('input[type="checkbox"]')
    await checkboxes[3].setValue(true)
    await wrapper.findAll('input.input').at(-1)!.setValue('Pool template')

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(mocks.createMonitor).toHaveBeenCalledWith(expect.objectContaining({
      name: 'Pool health',
      probe_mode: 'account_pool',
      endpoint: '',
      account_ids: [101, 102],
      primary_model_id: 'claude-sonnet',
      additional_model_ids: ['gemini-pro'],
      model_probe_strategy: 'primary_only',
      model_source_protocols: {
        'claude-sonnet': 'anthropic',
        'gemini-pro': 'gemini',
      },
      test_prompt_template: '请只回复 {{challenge}}',
      save_as_template: true,
      template_name: 'Pool template',
    }))
  })
})
