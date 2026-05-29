import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import ContentModerationThresholdsEditor from '../ContentModerationThresholdsEditor.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
    te: (key: string) => key.endsWith('.violence')
  })
}))

describe('ContentModerationThresholdsEditor', () => {
  it('renders known thresholds and emits clamped updates', async () => {
    const wrapper = mount(ContentModerationThresholdsEditor, {
      props: {
        modelValue: {
          violence: 0.42
        }
      }
    })

    expect(wrapper.text()).toContain('admin.settings.moderation.thresholdsTitle')
    expect(wrapper.text()).toContain('admin.settings.moderation.thresholdCategories.violence')

    const violenceNumber = wrapper
      .findAll('input[type="number"]')
      .find((input) => input.attributes('aria-label') === 'admin.settings.moderation.thresholdCategories.violence')
    expect(violenceNumber?.element).toBeTruthy()
    expect((violenceNumber!.element as HTMLInputElement).value).toBe('0.42')

    await violenceNumber!.setValue('1.5')

    const updates = wrapper.emitted('update:modelValue')
    expect(updates).toBeTruthy()
    expect((updates!.at(-1)![0] as Record<string, number>).violence).toBe(1)
  })
})
