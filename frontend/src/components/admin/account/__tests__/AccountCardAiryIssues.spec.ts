import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import AccountCard from '../AccountCard.vue'
import type { Account } from '@/types'

vi.mock('@/composables/useRealtimeCountdownNow', () => ({
  useRealtimeCountdownNow: () => ({
    nowMs: ref(Date.parse('2026-05-22T12:00:00Z')),
    nowDate: ref(new Date('2026-05-22T12:00:00Z')),
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'admin.accounts.status.issueSummaries.banned': '上游判定账号异常，已进入封禁状态。',
    'admin.accounts.status.issueSummaries.locked': '检测到异地或异常登录，账号已被安全锁定。',
    'admin.accounts.status.issueSummaries.maintenance': '上游服务维护中，请稍后重试。',
    'admin.accounts.status.issueSummaries.offline': '连接上游或代理超时，请检查网络与代理配置。',
    'admin.accounts.status.issueSummaries.overdue': '额度或账单已耗尽，请处理账单或充值后再恢复。',
    'admin.accounts.status.issueSummaries.degraded': '当前仅部分模型能力可用，账号处于降级运行状态。',
    'admin.accounts.status.issueSummaries.captcha': '触发上游风控验证，需要完成验证后恢复。',
    'admin.accounts.status.issueSummaries.syncing': '等待下一次自动恢复探测或配置刷新。',
    'admin.accounts.status.viewIssueDetails': '查看账号问题详情',
    'admin.accounts.autoRecoveryProbe.headline': '恢复探测：稍后重试',
    'admin.accounts.autoRecoveryProbe.statuses.retry_scheduled': '稍后重试',
  }
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string) => messages[key] || key,
    }),
  }
})

vi.mock('@/utils/format', () => ({
  formatRelativeTime: () => '1 day ago',
  formatDateTime: () => '2026/05/22 12:00:00',
  formatTime: () => '12:00',
}))

const makeAccount = (overrides: Partial<Account> = {}): Account => ({
  id: 1,
  name: 'Primary Account',
  platform: 'openai',
  type: 'apikey',
  proxy_id: null,
  concurrency: 4,
  current_concurrency: 1,
  priority: 1,
  status: 'active',
  error_message: null,
  last_used_at: '2026-05-22T00:00:00Z',
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

const mountAiryCard = (account: Account) => mount(AccountCard, {
  attachTo: document.body,
  props: {
    account,
    selected: false,
    togglingSchedulable: null,
    todayStatsByAccountId: {},
    todayStatsLoading: false,
    usageManualRefreshToken: 0,
    visualStyle: 'airy',
  },
  global: {
    stubs: {
      AccountCapacityCell: true,
      AccountGroupsCell: true,
      AccountUsageVisualCell: true,
      AccountSegmentedCountdown: true,
      AccountStatusLimitBadge: true,
      AccountServiceAuthVisualCell: true,
      AccountsViewAiryRowActions: true,
      PlatformIcon: true,
      Icon: true,
    },
  },
})

describe('AccountCard airy issue visibility', () => {
  it('shows supported airy account issue states with a details trigger', async () => {
    const cases = [
      {
        account: makeAccount({
          lifecycle_state: 'blacklisted',
          lifecycle_reason_message: 'Account permanently banned by upstream',
        }),
        title: 'admin.accounts.status.visualBannedTitle',
        detail: '上游判定账号异常，已进入封禁状态。',
        raw: 'Account permanently banned by upstream',
      },
      {
        account: makeAccount({ error_message: 'security locked due suspicious login' }),
        title: 'admin.accounts.status.visualLockedTitle',
        detail: '检测到异地或异常登录，账号已被安全锁定。',
        raw: 'security locked due suspicious login',
      },
      {
        account: makeAccount({ error_message: 'scheduled maintenance window' }),
        title: 'admin.accounts.status.visualMaintenanceTitle',
        detail: '上游服务维护中，请稍后重试。',
        raw: 'scheduled maintenance window',
      },
      {
        account: makeAccount({ error_message: 'network timeout while connecting proxy' }),
        title: 'admin.accounts.status.visualOfflineTitle',
        detail: '连接上游或代理超时，请检查网络与代理配置。',
        raw: 'network timeout while connecting proxy',
      },
      {
        account: makeAccount({ error_message: 'quota exhausted, payment required' }),
        title: 'admin.accounts.status.visualOverdueTitle',
        detail: '额度或账单已耗尽，请处理账单或充值后再恢复。',
        raw: 'quota exhausted, payment required',
      },
      {
        account: makeAccount({
          session_window_status: 'allowed_warning',
          lifecycle_reason_message: 'limited model set is currently available',
        }),
        title: 'admin.accounts.status.visualDegradedTitle',
        detail: '当前仅部分模型能力可用，账号处于降级运行状态。',
        raw: 'limited model set is currently available',
      },
      {
        account: makeAccount({ error_message: 'captcha challenge required' }),
        title: 'admin.accounts.status.visualCaptchaTitle',
        detail: '触发上游风控验证，需要完成验证后恢复。',
        raw: 'captcha challenge required',
      },
      {
        account: makeAccount({
          auto_recovery_probe: {
            status: 'retry_scheduled',
            summary: 'Waiting for the next config refresh',
          },
        }),
        title: 'admin.accounts.status.visualSyncingTitle',
        detail: '等待下一次自动恢复探测或配置刷新。',
        raw: 'Waiting for the next config refresh',
      },
    ]

    for (const item of cases) {
      const wrapper = mountAiryCard(item.account)

      expect(wrapper.text()).toContain(item.title)
      expect(wrapper.text()).not.toContain(item.detail)
      expect(wrapper.text()).not.toContain(item.raw)
      const trigger = wrapper.get('.error-info-trigger')
      expect(trigger.attributes('aria-label')).toBe('查看账号问题详情')
      expect(trigger.attributes('class')).toContain('text-rose-500')
      expect(wrapper.find('[data-testid="account-status-visual-cell"]').exists()).toBe(true)
      expect(wrapper.find('.status-classic-stub').exists()).toBe(false)
      await trigger.trigger('mouseenter')
      await flushPromises()
      expect(document.body.textContent).toContain(item.detail)
      expect(document.body.textContent).not.toContain(item.raw)
      wrapper.unmount()
      document.body.innerHTML = ''
    }
  })
})
