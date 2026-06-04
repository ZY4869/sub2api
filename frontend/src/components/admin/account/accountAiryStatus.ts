import type { Account } from '@/types'
import { resolveAccountAiryIssueSummary } from './accountAiryIssueText'
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
      ...resolveAccountAiryIssueSummary(
        'banned',
        [account.lifecycle_reason_message, account.error_message, account.lifecycle_reason_code],
        { defaultWhenEmpty: true },
      ),
    }
  }

  if (includesAny(text, AIRY_STATUS_KEYWORDS.locked)) {
    return {
      kind: 'locked',
      titleKey: 'admin.accounts.status.visualLockedTitle',
      tagKey: 'admin.accounts.status.visualLockedTag',
      tone: 'slate',
      iconName: 'lock',
      ...resolveAccountAiryIssueSummary('locked', [
        account.lifecycle_reason_message,
        account.error_message,
      ]),
    }
  }

  if (includesAny(text, AIRY_STATUS_KEYWORDS.maintenance)) {
    return {
      kind: 'maintenance',
      titleKey: 'admin.accounts.status.visualMaintenanceTitle',
      tagKey: 'admin.accounts.status.visualMaintenanceTag',
      tone: 'slate',
      iconName: 'cog',
      ...resolveAccountAiryIssueSummary('maintenance', [
        account.lifecycle_reason_message,
        account.error_message,
      ]),
    }
  }

  if (includesAny(text, AIRY_STATUS_KEYWORDS.offline)) {
    return {
      kind: 'offline',
      titleKey: 'admin.accounts.status.visualOfflineTitle',
      tagKey: 'admin.accounts.status.visualOfflineTag',
      tone: 'slate',
      iconName: 'cloud',
      ...resolveAccountAiryIssueSummary('offline', [
        account.error_message,
        account.lifecycle_reason_message,
      ]),
    }
  }

  if (includesAny(text, AIRY_STATUS_KEYWORDS.overdue)) {
    return {
      kind: 'overdue',
      titleKey: 'admin.accounts.status.visualOverdueTitle',
      tagKey: 'admin.accounts.status.visualOverdueTag',
      tone: 'red',
      iconName: 'creditCard',
      ...resolveAccountAiryIssueSummary('overdue', [
        account.error_message,
        account.lifecycle_reason_message,
      ]),
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
        tone: 'emerald',
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
    const degradedSummary = resolveAccountAiryIssueSummary(
      'degraded',
      [
        account.lifecycle_reason_message,
        account.error_message,
        account.auto_recovery_probe?.summary,
      ],
      {
        defaultWhenEmpty:
          account.session_window_status === 'allowed_warning' ||
          includesAny(text, AIRY_STATUS_KEYWORDS.degraded),
      },
    )

    return {
      kind: 'degraded',
      titleKey: 'admin.accounts.status.visualDegradedTitle',
      tagKey: 'admin.accounts.status.visualDegradedTag',
      tone: 'amber',
      iconName: 'exclamationTriangle',
      ...degradedSummary,
    }
  }

  if (includesAny(text, AIRY_STATUS_KEYWORDS.captcha)) {
    return {
      kind: 'captcha',
      titleKey: 'admin.accounts.status.visualCaptchaTitle',
      tagKey: 'admin.accounts.status.visualCaptchaTag',
      tone: 'purple',
      iconName: 'shield',
      ...resolveAccountAiryIssueSummary('captcha', [
        account.error_message,
        account.temp_unschedulable_reason,
      ]),
    }
  }

  if (context.isTempUnschedulable) {
    return {
      kind: 'tempUnschedulable',
      titleKey: 'admin.accounts.status.tempUnschedulable',
      tagKey: 'admin.accounts.status.tempUnschedulable',
      tone: 'sky',
      iconName: 'clock',
      ...resolveAccountAiryIssueSummary('tempUnschedulable', [
        account.temp_unschedulable_reason,
        account.error_message,
      ]),
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
      ...resolveAccountAiryIssueSummary(
        'syncing',
        [
          account.auto_recovery_probe?.summary,
          account.auto_recovery_probe?.error_code,
          account.lifecycle_reason_message,
        ],
        { defaultWhenEmpty: true },
      ),
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
      ...resolveAccountAiryIssueSummary(
        'error',
        [account.error_message, account.lifecycle_reason_message, account.lifecycle_reason_code],
        { defaultWhenEmpty: true },
      ),
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
