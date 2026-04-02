import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string) => key
    })
  }
})

import QuotaLimitCard from '../QuotaLimitCard.vue'

describe('QuotaLimitCard', () => {
  it('renders zh timezone labels with the original code', () => {
    const wrapper = mount(QuotaLimitCard, {
      props: {
        totalLimit: 100,
        dailyLimit: 10,
        weeklyLimit: 50,
        dailyResetMode: 'fixed',
        dailyResetHour: 3,
        weeklyResetMode: null,
        weeklyResetDay: null,
        weeklyResetHour: null,
        resetTimezone: 'Asia/Shanghai'
      }
    })

    const timezoneOptions = wrapper.findAll('option').map((option) => option.text())
    expect(timezoneOptions).toContain('\u534f\u8c03\u4e16\u754c\u65f6\uff08UTC\uff09')
    expect(timezoneOptions).toContain('\u4e2d\u56fd\u4e0a\u6d77\uff08Asia/Shanghai\uff09')
  })

  it('clears all quota fields when toggled off', async () => {
    const wrapper = mount(QuotaLimitCard, {
      props: {
        totalLimit: 100,
        dailyLimit: 10,
        weeklyLimit: 50,
        dailyResetMode: 'fixed',
        dailyResetHour: 3,
        weeklyResetMode: 'fixed',
        weeklyResetDay: 1,
        weeklyResetHour: 4,
        resetTimezone: 'UTC'
      }
    })

    await wrapper.get('button[type="button"]').trigger('click')

    expect(wrapper.emitted('update:totalLimit')?.at(-1)).toEqual([null])
    expect(wrapper.emitted('update:dailyLimit')?.at(-1)).toEqual([null])
    expect(wrapper.emitted('update:weeklyLimit')?.at(-1)).toEqual([null])
    expect(wrapper.emitted('update:dailyResetMode')?.at(-1)).toEqual([null])
    expect(wrapper.emitted('update:dailyResetHour')?.at(-1)).toEqual([null])
    expect(wrapper.emitted('update:weeklyResetMode')?.at(-1)).toEqual([null])
    expect(wrapper.emitted('update:weeklyResetDay')?.at(-1)).toEqual([null])
    expect(wrapper.emitted('update:weeklyResetHour')?.at(-1)).toEqual([null])
    expect(wrapper.emitted('update:resetTimezone')?.at(-1)).toEqual([null])
  })
})
