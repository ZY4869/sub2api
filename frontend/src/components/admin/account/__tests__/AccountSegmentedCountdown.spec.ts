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

    expect(wrapper.text()).toContain('7D00M03S')
    expect(wrapper.get('[data-test="account-segmented-countdown"]').attributes('aria-label')).toBe('7D 00M 03S')
    expect(wrapper.get('[data-test="account-segmented-countdown"]').attributes('title')).toBe('7D 00H 00M 03S')
    expect(wrapper.get('[data-test="account-segmented-countdown-prefix"]').text()).toBe('7D')
    expect(wrapper.get('[data-unit="M"]').classes()).toContain('bg-sky-100')
    expect(wrapper.get('[data-unit="S"]').classes()).toContain('bg-rose-100')
    expect(wrapper.text()).not.toContain(':')
    expect(wrapper.html()).not.toContain('backdrop-blur')
    expect(wrapper.html()).not.toContain('shadow-[')

    await vi.advanceTimersByTimeAsync(1000)

    expect(wrapper.text()).toContain('00M02S')
  })
})
