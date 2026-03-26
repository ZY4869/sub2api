import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import AccountStatusIndicator from '../AccountStatusIndicator.vue'
import type { Account } from '@/types'
import { resetUiNowForTests, UI_NOW_TICK_MS } from '@/composables/useUiNow'

enableAutoUnmount(afterEach)

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function makeAccount(overrides: Partial<Account> = {}): Account {
  return {
    id: 1,
    name: 'account',
    platform: 'antigravity',
    type: 'oauth',
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: true,
    created_at: '2026-03-15T00:00:00Z',
    updated_at: '2026-03-15T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides
  } as Account
}

describe('AccountStatusIndicator', () => {
  beforeEach(() => {
    resetUiNowForTests()
  })

  afterEach(() => {
    resetUiNowForTests()
    vi.restoreAllMocks()
    vi.useRealTimers()
    document.body.innerHTML = ''
  })

  it('clears the rate-limited label once the reset time passes', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-03-13T12:00:00Z'))

    const wrapper = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          rate_limit_reset_at: '2026-03-13T12:02:00Z'
        })
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain('admin.accounts.status.rateLimited')

    vi.advanceTimersByTime(2 * 60 * 1000 + UI_NOW_TICK_MS)
    await flushPromises()

    expect(wrapper.text()).not.toContain('admin.accounts.status.rateLimited')
  })

  it('renders overage model tags without broken glyphs', () => {
    const wrapper = mount(AccountStatusIndicator, {
      props: {
        account: makeAccount({
          extra: {
            allow_overages: true,
            model_rate_limits: {
              'claude-sonnet-4-5': {
                rate_limited_at: '2026-03-15T00:00:00Z',
                rate_limit_reset_at: '2099-03-15T00:00:00Z'
              }
            }
          }
        })
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    expect(wrapper.text()).toContain('CSon45')
    expect(wrapper.text()).not.toContain('鉁')
  })

  it('shows the full error text inside the teleported tooltip', async () => {
    const wrapper = mount(AccountStatusIndicator, {
      attachTo: document.body,
      props: {
        account: makeAccount({
          status: 'error',
          error_message: 'Payment required (402): {"code":"deactivated_workspace"}'
        })
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.get('.error-info-trigger').trigger('mouseenter')
    await flushPromises()

    const tooltip = document.body.querySelector('.error-info-tooltip')
    expect(tooltip).not.toBeNull()
    expect(tooltip?.textContent).toContain('Payment required (402): {"code":"deactivated_workspace"}')
  })

  it('repositions the error tooltip to stay within the viewport', async () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 320 })
    Object.defineProperty(window, 'innerHeight', { configurable: true, value: 240 })

    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(function () {
      if ((this as HTMLElement).classList.contains('error-info-trigger')) {
        return {
          width: 20,
          height: 20,
          top: 210,
          right: 310,
          bottom: 230,
          left: 290,
          x: 290,
          y: 210,
          toJSON: () => ({})
        } as DOMRect
      }

      if ((this as HTMLElement).classList.contains('error-info-tooltip')) {
        return {
          width: 220,
          height: 80,
          top: 0,
          right: 220,
          bottom: 80,
          left: 0,
          x: 0,
          y: 0,
          toJSON: () => ({})
        } as DOMRect
      }

      return {
        width: 0,
        height: 0,
        top: 0,
        right: 0,
        bottom: 0,
        left: 0,
        x: 0,
        y: 0,
        toJSON: () => ({})
      } as DOMRect
    })

    const wrapper = mount(AccountStatusIndicator, {
      attachTo: document.body,
      props: {
        account: makeAccount({
          status: 'error',
          error_message: 'Very long upstream error message for viewport checks'
        })
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.get('.error-info-trigger').trigger('mouseenter')
    await flushPromises()

    const tooltip = document.body.querySelector('.error-info-tooltip') as HTMLElement | null
    expect(tooltip).not.toBeNull()
    expect(tooltip?.style.left).toBe('88px')
    expect(tooltip?.style.top).toBe('120px')
    expect(tooltip?.style.maxWidth).toBe('296px')
  })
})
