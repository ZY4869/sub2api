import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PanelRateLimitSettingsCard from '../PanelRateLimitSettingsCard.vue'

const mockState = vi.hoisted(() => ({
  getPanelRateLimitSettings: vi.fn(),
  updatePanelRateLimitSettings: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api', () => ({
  adminAPI: {
    settings: {
      getPanelRateLimitSettings: mockState.getPanelRateLimitSettings,
      updatePanelRateLimitSettings: mockState.updatePanelRateLimitSettings
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
  return mount(PanelRateLimitSettingsCard, {
    global: {
      stubs: {
        Toggle: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template:
            '<input type="checkbox" :checked="modelValue" @change="$emit(\'update:modelValue\', $event.target.checked)" />'
        }
      }
    }
  })
}

describe('PanelRateLimitSettingsCard', () => {
  beforeEach(() => {
    mockState.getPanelRateLimitSettings.mockReset()
    mockState.updatePanelRateLimitSettings.mockReset()
    mockState.showError.mockReset()
    mockState.showSuccess.mockReset()
  })

  it('loads disabled defaults and saves panel rate limit settings', async () => {
    mockState.getPanelRateLimitSettings.mockResolvedValue({
      enabled: false,
      user_rpm: 240,
      heavy_rpm: 60,
      public_ip_rpm: 300,
      exempt_admin: true
    })
    mockState.updatePanelRateLimitSettings.mockImplementation(async (payload) => payload)

    const wrapper = mountCard()
    await flushPromises()

    const toggles = wrapper.findAll('input[type="checkbox"]')
    expect((toggles[0].element as HTMLInputElement).checked).toBe(false)
    expect((toggles[1].element as HTMLInputElement).checked).toBe(true)

    await toggles[0].setValue(true)
    const inputs = wrapper.findAll('input[type="number"]')
    await inputs[0].setValue('120')
    await inputs[1].setValue('30')
    await inputs[2].setValue('600')
    await toggles[1].setValue(false)
    await wrapper.find('button.btn-primary').trigger('click')

    expect(mockState.updatePanelRateLimitSettings).toHaveBeenCalledWith({
      enabled: true,
      user_rpm: 120,
      heavy_rpm: 30,
      public_ip_rpm: 600,
      exempt_admin: false
    })
    expect(mockState.showSuccess).toHaveBeenCalledWith('admin.settings.panelRateLimit.saved')
  })
})
