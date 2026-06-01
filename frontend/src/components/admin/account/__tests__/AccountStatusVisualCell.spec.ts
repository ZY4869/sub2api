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
        if (key === 'common.time.countdown.minutes') return `${params?.m}分钟`
        if (key === 'common.time.countdown.hoursMinutes') return `${params?.h}小时 ${params?.m}分钟`
        if (key === 'common.time.countdown.daysHours') return `${params?.d}天 ${params?.h}小时`
        if (key === 'common.time.countdown.compact.hoursMinutes') return `${params?.h}小时${params?.m}分`
        if (key === 'common.time.countdown.compact.minutesSeconds') return `${params?.m}分${params?.s}秒`
        if (key === 'common.time.countdown.compact.seconds') return `${params?.s}秒`
        if (key === 'common.time.countdown.withSuffix') return `${params?.time} 后解除`
        if (key === 'admin.accounts.status.visualAvailableTitle') return '可用'
        if (key === 'admin.accounts.status.visualAvailableTag') return '可调度'
        if (key === 'admin.accounts.status.visualBannedTitle') return '账号封禁'
        if (key === 'admin.accounts.status.window5h') return '5小时'
        if (key === 'admin.accounts.status.window7d') return '7天'
        if (key === 'admin.accounts.status.issueSummaries.offline') return '连接上游或代理超时，请检查网络与代理配置。'
        if (key === 'admin.accounts.status.issueSummaries.overdue') return '额度或账单已耗尽，请处理账单或充值后再恢复。'
        if (key === 'admin.accounts.status.issueSummaries.error') return '账号状态异常，请打开详情或重新测试排查。'
        if (key === 'admin.accounts.status.issueSummaries.credentials') return '凭证无效或已过期，请重新授权或更换凭证。'
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
  auto_renew_enabled: false,
  auto_renew_period: 'month',
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
    expect(wrapper.text()).toContain('可用')
    expect(wrapper.text()).toContain('可调度')
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
    expect(usage7d.text()).toContain('Codex 7天')
    expect(usage7d.text()).toContain('Spark 7天')
    expect(usage7d.text()).toContain('24小时0分')
    expect(usage7d.text()).not.toContain('Codex 7d')
    expect(usage7d.text()).not.toContain('Spark 7d')

    const badgeContainer = usage7d.get('[data-test="account-limit-badges"]')
    expect(badgeContainer.classes()).toContain('grid')
    expect(badgeContainer.classes()).toContain('gap-2')
    const badges = usage7d.findAll('[data-test="account-status-limit-badge"]')
    expect(badges).toHaveLength(2)
    expect(badges[0].find('span.inline-flex').classes()).toContain('w-full')
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
    expect(wrapper.text()).toContain('20分钟 后解除')
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

    expect(paused.text()).toContain('admin.accounts.status.visualPausedTitle')
    expect(errored.text()).toContain('admin.accounts.status.error')
    expect(errored.text()).not.toContain('凭证无效或已过期，请重新授权或更换凭证。')
    expect(errored.text()).not.toContain('Demo invalid credential state')
    const trigger = errored.get('.error-info-trigger')
    expect(trigger.exists()).toBe(true)

    await trigger.trigger('mouseenter')

    expect(document.body.textContent).toContain('凭证无效或已过期，请重新授权或更换凭证。')
  })

  it('renders Chinese issue summaries instead of upstream English details', () => {
    const offline = mountVisual(makeAccount({
      error_message: 'network timeout while connecting proxy',
    }))
    const overdue = mountVisual(makeAccount({
      error_message: 'quota exhausted, payment required',
    }))

    expect(offline.text()).not.toContain('连接上游或代理超时，请检查网络与代理配置。')
    expect(offline.text()).not.toContain('network timeout while connecting proxy')
    expect(overdue.text()).not.toContain('额度或账单已耗尽，请处理账单或充值后再恢复。')
    expect(overdue.text()).not.toContain('quota exhausted, payment required')
    expect(offline.find('.error-info-trigger').exists()).toBe(true)
    expect(overdue.find('.error-info-trigger').exists()).toBe(true)
  })

  it('covers the airy full status set from reliable reason signals', () => {
    const cases = [
      {
        account: makeAccount({ lifecycle_state: 'blacklisted' }),
        expected: '账号封禁',
      },
      {
        account: makeAccount({ error_message: 'security locked due suspicious login' }),
        expected: 'admin.accounts.status.visualLockedTitle',
      },
      {
        account: makeAccount({ error_message: 'scheduled maintenance window' }),
        expected: 'admin.accounts.status.visualMaintenanceTitle',
      },
      {
        account: makeAccount({ error_message: 'network timeout while connecting proxy' }),
        expected: 'admin.accounts.status.visualOfflineTitle',
      },
      {
        account: makeAccount({ error_message: 'quota exhausted, payment required' }),
        expected: 'admin.accounts.status.visualOverdueTitle',
      },
      {
        account: makeAccount({ session_window_status: 'allowed_warning' }),
        expected: 'admin.accounts.status.visualDegradedTitle',
      },
      {
        account: makeAccount({ error_message: 'captcha challenge required' }),
        expected: 'admin.accounts.status.visualCaptchaTitle',
      },
      {
        account: makeAccount({ auto_recovery_probe: { status: 'retry_scheduled' } }),
        expected: 'admin.accounts.status.visualSyncingTitle',
      },
      {
        account: makeAccount({ schedulable: false }),
        expected: 'admin.accounts.status.visualPausedTitle',
      },
      {
        account: makeAccount(),
        expected: '可用',
      },
    ]

    for (const item of cases) {
      const wrapper = mountVisual(item.account)
      expect(wrapper.text()).toContain(item.expected)
      wrapper.unmount()
    }
  })
})
