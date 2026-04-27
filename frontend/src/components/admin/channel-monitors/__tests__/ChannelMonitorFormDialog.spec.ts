import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import ChannelMonitorFormDialog from '../ChannelMonitorFormDialog.vue'

const mocks = vi.hoisted(() => ({
  createMonitor: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    channelMonitors: {
      createMonitor: mocks.createMonitor,
      updateMonitor: vi.fn(),
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
      t: (key: string) => key,
    },
  }),
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

function mountDialog() {
  return mount(ChannelMonitorFormDialog, {
    props: {
      show: true,
      monitor: null,
      templates: [],
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>',
        },
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option value="openai">openai</option></select>',
        },
        Toggle: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<input type="checkbox" :checked="modelValue" @change="$emit(\'update:modelValue\', $event.target.checked)" />',
        },
        Icon: true,
      },
    },
  })
}

describe('ChannelMonitorFormDialog', () => {
  beforeEach(() => {
    mocks.createMonitor.mockReset()
    mocks.showError.mockReset()
    mocks.showSuccess.mockReset()
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
    await inputs[3].setValue('gpt-5.4')

    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(mocks.showError).toHaveBeenCalledWith('endpoint is unreachable')
  })
})
