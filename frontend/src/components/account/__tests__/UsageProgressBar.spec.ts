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
          'admin.accounts.usageWindow.remainingLabel': 'Remaining',
          'admin.accounts.usageWindow.resetAtLabel': 'Reset at',
          'admin.accounts.usageWindow.now': 'Now',
          'dates.today': 'Today',
          'dates.tomorrow': 'Tomorrow',
        }
        return dict[key] ?? key
      },
    }),
  }
})

describe('UsageProgressBar', () => {
  it('renders inline remaining text on the same row', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T12:29:00'))

    const wrapper = mount(UsageProgressBar, {
      props: {
        label: '5h',
        utilization: 78,
        resetsAt: '2026-03-13T15:22:00',
        color: 'indigo',
        inlineReset: true,
      },
    })

    expect(wrapper.text()).toContain('5h')
    expect(wrapper.text()).toContain('78%')
    expect(wrapper.text()).toContain('Remaining 2h 53m')
    expect(wrapper.text()).not.toContain('Reset at')
    expect(wrapper.find('.text-amber-700').exists()).toBe(true)

    vi.useRealTimers()
  })

  it('keeps detailed reset mode backward compatible', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T08:00:00'))

    const wrapper = mount(UsageProgressBar, {
      props: {
        label: '7d',
        utilization: 80,
        remainingSeconds: 1800,
        color: 'emerald',
        detailedReset: true,
      },
    })

    expect(wrapper.text()).toContain('Remaining 30m')
    expect(wrapper.text()).toContain('Reset at Today 08:30')
    expect(wrapper.find('[title="2026-03-13 08:30:00"]').exists()).toBe(true)

    vi.useRealTimers()
  })
})
