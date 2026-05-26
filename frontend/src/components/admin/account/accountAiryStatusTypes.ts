import type { AccountGlassTone } from './accountVisualGlass'

export type AiryStatusKind =
  | 'banned'
  | 'locked'
  | 'maintenance'
  | 'offline'
  | 'overdue'
  | 'usage7d'
  | 'usage5h'
  | 'degraded'
  | 'captcha'
  | 'syncing'
  | 'paused'
  | 'available'
  | 'error'
  | 'overloaded'
  | 'rateLimited'
  | 'tempUnschedulable'

export type AiryStatusIconName =
  | 'ban'
  | 'lock'
  | 'cog'
  | 'cloud'
  | 'creditCard'
  | 'clock'
  | 'exclamationTriangle'
  | 'shield'
  | 'sync'
  | 'checkCircle'

export type AiryStatusVisual = {
  kind: AiryStatusKind
  titleKey: string
  tagKey: string
  tone: AccountGlassTone
  iconName: AiryStatusIconName
  tagFallback?: string
  helper?: string
  helperKey?: string
}

export type AiryStatusContext = {
  nowMs: number
  isRateLimited: boolean
  isOverloaded: boolean
  isTempUnschedulable: boolean
  hasError: boolean
  activeLimitBadgeCount: number
}
