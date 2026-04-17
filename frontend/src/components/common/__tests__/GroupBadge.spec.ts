import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import GroupBadge from '../GroupBadge.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('GroupBadge', () => {
  it('exposes the full group name through hover metadata', () => {
    const wrapper = mount(GroupBadge, {
      props: {
        name: 'Very Long Group Name For Hover Preview',
        platform: 'gemini',
        showRate: false
      },
      global: {
        stubs: {
          PlatformIcon: true
        }
      }
    })

    expect(wrapper.attributes('title')).toBe('Very Long Group Name For Hover Preview')
    expect(wrapper.attributes('aria-label')).toBe('Very Long Group Name For Hover Preview')
  })
})
