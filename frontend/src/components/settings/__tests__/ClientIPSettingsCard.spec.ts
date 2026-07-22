import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ClientIPSettingsCard from '../ClientIPSettingsCard.vue'

const mockState = vi.hoisted(() => ({
  getClientIPSettings: vi.fn(),
  updateClientIPSettings: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api', () => ({
  adminAPI: {
    settings: {
      getClientIPSettings: mockState.getClientIPSettings,
      updateClientIPSettings: mockState.updateClientIPSettings
    }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: mockState.showError,
    showSuccess: mockState.showSuccess
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

function mountCard() {
  return mount(ClientIPSettingsCard, {
    global: {
      stubs: {
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template:
            '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>'
        }
      }
    }
  })
}

describe('ClientIPSettingsCard', () => {
  beforeEach(() => {
    mockState.getClientIPSettings.mockReset()
    mockState.updateClientIPSettings.mockReset()
    mockState.showError.mockReset()
    mockState.showSuccess.mockReset()
  })

  it('loads and saves trusted header client IP settings', async () => {
    mockState.getClientIPSettings.mockResolvedValue({
      mode: 'headers',
      headers: ['CF-Connecting-IP'],
      xff_hop_index: 1
    })
    mockState.updateClientIPSettings.mockImplementation(async (payload) => payload)

    const wrapper = mountCard()
    await flushPromises()

    expect((wrapper.find('select').element as HTMLSelectElement).value).toBe('headers')
    const textarea = wrapper.find('textarea')
    expect(textarea.element.value).toBe('CF-Connecting-IP')

    await textarea.setValue('X-Real-IP\nX-Forwarded-For, CF-Connecting-IP')
    await wrapper.find('input[type="number"]').setValue('2')
    await wrapper.find('button.btn-primary').trigger('click')

    expect(mockState.updateClientIPSettings).toHaveBeenCalledWith({
      mode: 'headers',
      headers: ['X-Real-IP', 'X-Forwarded-For', 'CF-Connecting-IP'],
      xff_hop_index: 2
    })
    expect(mockState.showSuccess).toHaveBeenCalledWith('admin.settings.clientIP.saved')
  })
})
