import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { resetUiNowForTests } from '@/composables/useUiNow'
import AccountSegmentedCountdown from '../AccountSegmentedCountdown.vue'

enableAutoUnmount(afterEach)

describe('AccountSegmentedCountdown', () => {
  afterEach(() => {
    resetUiNowForTests()
    vi.useRealTimers()
  })

  it('keeps ticking only the countdown text without heavy glass classes', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T12:00:00Z'))

    const wrapper = mount(AccountSegmentedCountdown, {
      props: {
        resetAt: '2026-03-13T12:00:03Z',
        tone: 'amber',
        prefix: '7D'
      },
      global: {
        plugins: [createPinia()]
      },
    })

    expect(wrapper.text()).toContain('7D00:00:03')
    expect(wrapper.get('[data-test="account-segmented-countdown-prefix"]').text()).toBe('7D')
    expect(wrapper.html()).not.toContain('backdrop-blur')
    expect(wrapper.html()).not.toContain('shadow-[')

    await vi.advanceTimersByTimeAsync(1000)

    expect(wrapper.text()).toContain('00:00:02')
  })
})
