import type { Account } from '@/types'
import {
  AIRY_STATUS_KEYWORDS,
  createAccountAiryStatusSignal,
  hasFutureTimestamp,
  includesAny,
} from './accountAiryStatusSignals'
import type { AiryStatusContext, AiryStatusVisual } from './accountAiryStatusTypes'

export const resolveAccountAiryStatus = (
  account: Account,
  context: AiryStatusContext,
): AiryStatusVisual => {
  const { text, probeStatus } = createAccountAiryStatusSignal(account)
  const probe = account.auto_recovery_probe

  if (
    account.lifecycle_state === 'blacklisted' ||
    probe?.blacklisted === true ||
    probeStatus === 'blacklisted' ||
    includesAny(text, AIRY_STATUS_KEYWORDS.banned)
  ) {
    return {
      kind: 'banned',
      titleKey: 'admin.accounts.status.visualBannedTitle',
      tagKey: 'admin.accounts.status.visualBannedTag',
      tone: 'red',
      iconName: 'ban',
      helper: account.lifecycle_reason_message || undefined,
    }
  }

  if (includesAny(text, AIRY_STATUS_KEYWORDS.locked)) {
    return {
      kind: 'locked',
      titleKey: 'admin.accounts.status.visualLockedTitle',
      tagKey: 'admin.accounts.status.visualLockedTag',
      tone: 'slate',
      iconName: 'lock',
      helper: account.lifecycle_reason_message || account.error_message || undefined,
    }
  }

  if (includesAny(text, AIRY_STATUS_KEYWORDS.maintenance)) {
    return {
      kind: 'maintenance',
      titleKey: 'admin.accounts.status.visualMaintenanceTitle',
      tagKey: 'admin.accounts.status.visualMaintenanceTag',
      tone: 'slate',
      iconName: 'cog',
      helper: account.lifecycle_reason_message || account.error_message || undefined,
    }
  }

  if (includesAny(text, AIRY_STATUS_KEYWORDS.offline)) {
    return {
      kind: 'offline',
      titleKey: 'admin.accounts.status.visualOfflineTitle',
      tagKey: 'admin.accounts.status.visualOfflineTag',
      tone: 'slate',
      iconName: 'cloud',
      helper: account.error_message || undefined,
    }
  }

  if (includesAny(text, AIRY_STATUS_KEYWORDS.overdue)) {
    return {
      kind: 'overdue',
      titleKey: 'admin.accounts.status.visualOverdueTitle',
      tagKey: 'admin.accounts.status.visualOverdueTag',
      tone: 'red',
      iconName: 'creditCard',
      helper: account.error_message || undefined,
    }
  }

  if (context.isOverloaded) {
    return {
      kind: 'overloaded',
      titleKey: 'admin.accounts.status.overloaded',
      tagKey: 'admin.accounts.status.visualOverloadedTag',
      tone: 'red',
      iconName: 'exclamationTriangle',
      tagFallback: '529',
    }
  }

  if (context.isRateLimited) {
    if (account.rate_limit_reason === 'usage_5h' || account.rate_limit_reason === 'rate_429') {
      return {
        kind: 'usage5h',
        titleKey: account.rate_limit_reason === 'usage_5h'
          ? 'admin.accounts.status.visualUsage5hTitle'
          : 'admin.accounts.status.rateLimited',
        tagKey: account.rate_limit_reason === 'usage_5h'
          ? 'admin.accounts.status.visualUsage5hTag'
          : 'admin.accounts.status.visualRateLimitedTag',
        tone: 'orange',
        iconName: 'clock',
        tagFallback: account.rate_limit_reason === 'usage_5h' ? '5H' : '429',
      }
    }
    if (account.rate_limit_reason === 'usage_7d' || account.rate_limit_reason === 'usage_7d_all') {
      return {
        kind: 'usage7d',
        titleKey: 'admin.accounts.status.visualUsage7dTitle',
        tagKey: account.rate_limit_reason === 'usage_7d_all'
          ? 'admin.accounts.status.visualUsage7dAllTag'
          : 'admin.accounts.status.visualUsage7dTag',
        tone: 'indigo',
        iconName: 'clock',
      }
    }
    return {
      kind: 'rateLimited',
      titleKey: 'admin.accounts.status.rateLimited',
      tagKey: 'admin.accounts.status.visualRateLimitedTag',
      tone: 'amber',
      iconName: 'clock',
      tagFallback: '429',
    }
  }

  if (
    context.activeLimitBadgeCount > 0 ||
    account.session_window_status === 'allowed_warning' ||
    includesAny(text, AIRY_STATUS_KEYWORDS.degraded)
  ) {
    return {
      kind: 'degraded',
      titleKey: 'admin.accounts.status.visualDegradedTitle',
      tagKey: 'admin.accounts.status.visualDegradedTag',
      tone: 'amber',
      iconName: 'exclamationTriangle',
    }
  }

  if (includesAny(text, AIRY_STATUS_KEYWORDS.captcha)) {
    return {
      kind: 'captcha',
      titleKey: 'admin.accounts.status.visualCaptchaTitle',
      tagKey: 'admin.accounts.status.visualCaptchaTag',
      tone: 'purple',
      iconName: 'shield',
      helper: account.error_message || account.temp_unschedulable_reason || undefined,
    }
  }

  if (context.isTempUnschedulable) {
    return {
      kind: 'tempUnschedulable',
      titleKey: 'admin.accounts.status.tempUnschedulable',
      tagKey: 'admin.accounts.status.tempUnschedulable',
      tone: 'sky',
      iconName: 'clock',
      helper: account.temp_unschedulable_reason || undefined,
    }
  }

  if (
    probeStatus === 'retry_scheduled' ||
    hasFutureTimestamp(probe?.next_retry_at, context.nowMs) ||
    includesAny(text, AIRY_STATUS_KEYWORDS.syncing)
  ) {
    return {
      kind: 'syncing',
      titleKey: 'admin.accounts.status.visualSyncingTitle',
      tagKey: 'admin.accounts.status.visualSyncingTag',
      tone: 'teal',
      iconName: 'sync',
    }
  }

  if (!account.schedulable || account.status === 'inactive' || account.lifecycle_state === 'archived') {
    return {
      kind: 'paused',
      titleKey: 'admin.accounts.status.visualPausedTitle',
      tagKey: 'admin.accounts.status.visualPausedTag',
      tone: 'slate',
      iconName: 'clock',
    }
  }

  if (context.hasError) {
    return {
      kind: 'error',
      titleKey: 'admin.accounts.status.error',
      tagKey: 'admin.accounts.status.error',
      tone: 'red',
      iconName: 'exclamationTriangle',
      helper: account.error_message || undefined,
    }
  }

  return {
    kind: 'available',
    titleKey: 'admin.accounts.status.visualAvailableTitle',
    tagKey: 'admin.accounts.status.visualAvailableTag',
    tone: 'emerald',
    iconName: 'checkCircle',
  }
}
