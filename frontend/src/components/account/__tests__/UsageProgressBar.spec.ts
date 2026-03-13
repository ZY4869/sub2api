import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import UsageProgressBar from '../UsageProgressBar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        const dict: Record<string, string> = {
          'admin.accounts.usageWindow.remainingLabel': '剩余',
          'admin.accounts.usageWindow.resetAtLabel': '重置于',
          'admin.accounts.usageWindow.now': '现在',
          'dates.today': '今天',
          'dates.tomorrow': '明天'
        }
        return dict[key] ?? key
      }
    })
  }
})

describe('UsageProgressBar', () => {
  it('renders detailed reset row and tooltip', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T08:00:00'))

    const wrapper = mount(UsageProgressBar, {
      props: {
        label: '5h',
        utilization: 42,
        resetsAt: '2026-03-13T10:30:00',
        color: 'indigo',
        detailedReset: true
      }
    })

    expect(wrapper.text()).toContain('5h')
    expect(wrapper.text()).toContain('42%')
    expect(wrapper.text()).toContain('剩余 2h 30m')
    expect(wrapper.text()).toContain('重置于 今天 10:30')
    expect(wrapper.attributes('title')).toBeUndefined()
    expect(wrapper.find('[title="2026-03-13 10:30:00"]').exists()).toBe(true)

    vi.useRealTimers()
  })

  it('uses remainingSeconds when resetsAt is missing', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T08:00:00'))

    const wrapper = mount(UsageProgressBar, {
      props: {
        label: '7d',
        utilization: 80,
        remainingSeconds: 1800,
        color: 'emerald',
        detailedReset: true
      }
    })

    expect(wrapper.text()).toContain('剩余 30m')
    expect(wrapper.text()).toContain('重置于 今天 08:30')

    vi.useRealTimers()
  })
})
