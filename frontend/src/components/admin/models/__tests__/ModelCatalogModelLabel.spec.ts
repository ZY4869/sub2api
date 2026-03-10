import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

const copyToClipboard = vi.fn()

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => {
      if (key === 'admin.models.copyModelIdSuccess') {
        return `Model ID copied: ${params?.model ?? ''}`
      }
      return key
    },
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard,
  }),
}))

import ModelCatalogModelLabel from '../ModelCatalogModelLabel.vue'

describe('ModelCatalogModelLabel', () => {
  it('renders display name and copies raw model id on click', async () => {
    const wrapper = mount(ModelCatalogModelLabel, {
      props: {
        model: 'gpt-4o-mini-2026-03-05',
        displayName: 'GPT-4o-mini',
        iconKey: 'chatgpt',
      },
    })

    expect(wrapper.text()).toContain('GPT-4o-mini')

    expect(wrapper.text()).toContain('gpt-4o-mini-2026-03-05')

    await wrapper.get('button').trigger('click')

    expect(copyToClipboard).toHaveBeenCalledWith(
      'gpt-4o-mini-2026-03-05',
      'Model ID copied: gpt-4o-mini-2026-03-05',
    )
  })
})
