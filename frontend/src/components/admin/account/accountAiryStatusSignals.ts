import type { Account } from '@/types'

export const AIRY_STATUS_KEYWORDS = {
  banned: ['blacklist', 'banned', 'ban', '封禁', '拉黑'],
  locked: ['locked', 'security', 'suspicious', '异地', '安全锁定', '安全保护'],
  maintenance: ['maintenance', 'maintain', '维护', '检修'],
  offline: [
    'offline',
    'timeout',
    'timed out',
    'network',
    'dial tcp',
    'connect',
    'connection refused',
    'proxy',
    'ping',
    '离线',
    '超时',
    '网络',
  ],
  overdue: [
    'quota',
    'credit',
    'billing',
    'payment',
    'insufficient',
    'exhausted',
    'overdue',
    '欠费',
    '额度',
    '余额',
    '充值',
  ],
  degraded: ['degraded', 'partial', 'limited model', '降级', '部分受限'],
  captcha: [
    'captcha',
    'challenge',
    'cloudflare',
    'turnstile',
    'verify',
    'verification',
    '人机',
    '验证',
    '风控',
  ],
  syncing: ['syncing', 'sync', 'synchronizing', '同步中', '拉取'],
} as const

export const includesAny = (
  text: string,
  keywords: readonly string[],
) => keywords.some((keyword) => text.includes(keyword))

export const hasFutureTimestamp = (
  value: string | null | undefined,
  nowMs: number,
) => {
  if (!value) return false
  const time = Date.parse(value)
  return Number.isFinite(time) && time > nowMs
}

export const createAccountAiryStatusSignal = (account: Account) => ({
  text: [
    account.lifecycle_reason_code,
    account.lifecycle_reason_message,
    account.temp_unschedulable_reason,
    account.error_message,
    account.auto_recovery_probe?.summary,
    account.auto_recovery_probe?.error_code,
  ].map((value) => String(value || '').toLowerCase()).join(' '),
  probeStatus: String(account.auto_recovery_probe?.status || '').toLowerCase(),
})
