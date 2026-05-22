import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import AccountStatusVisualCell from '../AccountStatusVisualCell.vue'
import type { Account } from '@/types'

enableAutoUnmount(afterEach)

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === 'common.time.countdown.minutes') return `${params?.m}m`
        if (key === 'common.time.countdown.hoursMinutes') return `${params?.h}h ${params?.m}m`
        if (key === 'common.time.countdown.daysHours') return `${params?.d}d ${params?.h}h`
        if (key === 'common.time.countdown.withSuffix') return `${params?.time} 后解除`
        return key
      }
    })
  }
})

const stubs = {
  ModelIcon: {
    props: ['model'],
    template: '<span data-test="model-icon">{{ model }}</span>'
  }
}

const makeAccount = (overrides: Partial<Account> = {}): Account => ({
  id: 1,
  name: 'Primary',
  platform: 'openai',
  type: 'apikey',
  proxy_id: null,
  concurrency: 3,
  current_concurrency: 1,
  priority: 1,
  status: 'active',
  error_message: null,
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: true,
  created_at: '2026-05-22T00:00:00Z',
  updated_at: '2026-05-22T00:00:00Z',
  schedulable: true,
  rate_limited_at: null,
  rate_limit_reset_at: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  session_window_start: null,
  session_window_end: null,
  session_window_status: null,
  lifecycle_state: 'normal',
  extra: {},
  ...overrides,
} as Account)

const mountVisual = (account: Account) => mount(AccountStatusVisualCell, {
  props: { account },
  global: {
    plugins: [createPinia()],
    stubs,
  },
})

describe('AccountStatusVisualCell', () => {
  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
    document.body.innerHTML = ''
  })

  it('renders the normal account with the migrated green status card', () => {
    const wrapper = mountVisual(makeAccount())

    expect(wrapper.get('[data-testid="account-status-visual-cell"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('admin.accounts.status.active')
    expect(wrapper.find('[data-testid="account-status-visual-countdown"]').exists()).toBe(false)
  })

  it('renders 429 limits with segmented countdown and resume copy', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-22T12:00:00Z'))

    const wrapper = mountVisual(makeAccount({
      rate_limited_at: '2026-05-22T11:55:00Z',
      rate_limit_reset_at: '2026-05-22T12:20:45Z',
      rate_limit_reason: 'rate_429',
    }))

    expect(wrapper.text()).toContain('admin.accounts.status.rateLimited')
    expect(wrapper.text()).toContain('429')
    expect(wrapper.get('[data-testid="account-status-visual-countdown"]').text()).toContain('00')
    expect(wrapper.text()).toContain('admin.accounts.status.visualAfterResume')
    expect(wrapper.text()).toContain('admin.accounts.status.rateLimitedAutoResume')
  })

  it('renders 5h and 7d usage limits with scoped labels', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-22T12:00:00Z'))

    const usage5h = mountVisual(makeAccount({
      rate_limited_at: '2026-05-22T11:00:00Z',
      rate_limit_reset_at: '2026-05-22T17:00:00Z',
      rate_limit_reason: 'usage_5h',
    }))
    const usage7d = mountVisual(makeAccount({
      rate_limited_at: '2026-05-20T12:00:00Z',
      rate_limit_reset_at: '2026-05-23T12:00:00Z',
      rate_limit_reason: 'usage_7d_all',
      extra: {
        codex_7d_reset_at: '2026-05-23T12:00:00Z',
        codex_spark_7d_reset_at: '2026-05-24T12:00:00Z',
      },
    }))

    expect(usage5h.text()).toContain('admin.accounts.status.usage5h')
    expect(usage5h.text()).toContain('admin.accounts.status.usage5hAutoResume')
    expect(usage7d.text()).toContain('admin.accounts.status.usage7d')
    expect(usage7d.text()).toContain('admin.accounts.status.usage7dAll')
    expect(usage7d.text()).toContain('Codex 7d')
    expect(usage7d.text()).toContain('Spark 7d')
  })

  it('renders overload with 529 countdown and release copy', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-22T12:00:00Z'))

    const wrapper = mountVisual(makeAccount({
      overload_until: '2026-05-22T12:20:45Z',
    }))

    expect(wrapper.text()).toContain('admin.accounts.status.overloaded')
    expect(wrapper.text()).toContain('529')
    expect(wrapper.text()).toContain('admin.accounts.status.visualAfterRelease')
    expect(wrapper.text()).toContain('20m 后解除')
  })

  it('keeps temp unschedulable clickable', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-05-22T12:00:00Z'))

    const wrapper = mountVisual(makeAccount({
      temp_unschedulable_until: '2026-05-22T12:45:00Z',
    }))

    await wrapper.get('button[type="button"]').trigger('click')

    expect(wrapper.text()).toContain('admin.accounts.status.tempUnschedulable')
    expect(wrapper.emitted('show-temp-unsched')).toEqual([[expect.objectContaining({ id: 1 })]])
  })

  it('renders paused and error states while preserving the error tooltip trigger', async () => {
    const paused = mountVisual(makeAccount({ schedulable: false }))
    const errored = mountVisual(makeAccount({
      status: 'error',
      error_message: 'Demo invalid credential state',
    }))

    expect(paused.text()).toContain('admin.accounts.status.paused')
    expect(errored.text()).toContain('admin.accounts.status.error')
    expect(errored.find('.error-info-trigger').exists()).toBe(true)
  })
})
