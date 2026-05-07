import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import CustomMenuSettingsCard from '../CustomMenuSettingsCard.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === 'admin.settings.customMenu.itemLabel') {
          return `Item #${params?.n}`
        }
        return key
      },
    }),
  }
})

const InputStub = {
  props: ['modelValue', 'label', 'placeholder', 'hint'],
  emits: ['update:modelValue'],
  template: `
    <label>
      <span>{{ label }}</span>
      <input
        :value="modelValue"
        :placeholder="placeholder"
        @input="$emit('update:modelValue', $event.target.value)"
      />
    </label>
  `,
}

const TextAreaStub = {
  props: ['modelValue', 'label', 'placeholder'],
  emits: ['update:modelValue'],
  template: `
    <label>
      <span>{{ label }}</span>
      <textarea
        :value="modelValue"
        :placeholder="placeholder"
        @input="$emit('update:modelValue', $event.target.value)"
      />
    </label>
  `,
}

describe('CustomMenuSettingsCard', () => {
  it('normalizes markdown slugs on input', async () => {
    const items = [
      {
        id: 'page-1',
        label: 'Guide',
        icon_svg: '',
        url: '',
        visibility: 'user' as const,
        sort_order: 0,
        page_mode: 'markdown' as const,
        page_slug: '',
        page_content: '# Draft',
        page_public: false,
        page_published: true,
      },
    ]

    const wrapper = mount(CustomMenuSettingsCard, {
      props: {
        modelValue: items,
      },
      global: {
        stubs: {
          Input: InputStub,
          TextArea: TextAreaStub,
          Toggle: true,
          ImageUpload: true,
        },
      },
    })

    const slugInput = wrapper.findAll('input').at(1)
    await slugInput!.setValue(' Help Center / Intro ')

    expect(items[0].page_slug).toBe('help-center-intro')
  })

  it('clears markdown fields when switching back to iframe mode', async () => {
    const items = [
      {
        id: 'page-1',
        label: 'Guide',
        icon_svg: '',
        url: '',
        visibility: 'user' as const,
        sort_order: 0,
        page_mode: 'markdown' as const,
        page_slug: 'guide',
        page_content: '# Draft',
        page_public: false,
        page_published: true,
      },
    ]

    const wrapper = mount(CustomMenuSettingsCard, {
      props: {
        modelValue: items,
      },
      global: {
        stubs: {
          Input: InputStub,
          TextArea: TextAreaStub,
          Toggle: true,
          ImageUpload: true,
        },
      },
    })

    const modeSelect = wrapper.findAll('select').at(1)
    await modeSelect!.setValue('iframe')

    expect(items[0].page_mode).toBe('iframe')
    expect(items[0].page_slug).toBe('')
    expect(items[0].page_content).toBe('')
    expect(items[0].page_published).toBe(false)
  })
})
