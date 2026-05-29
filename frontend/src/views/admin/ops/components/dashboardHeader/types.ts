import type { OpsDashboardOverview, OpsMetricThresholds } from '@/api/admin/ops'
import type { OpsRequestDetailsPreset } from '../opsRequestDetailsTypes'

export type RealtimeWindow = '1min' | '5min' | '30min' | '1h'

export interface Props {
  overview?: OpsDashboardOverview | null
  platform: string
  groupId: number | null
  channelId: number | null
  timeRange: string
  queryMode: string
  loading: boolean
  lastUpdated: Date | null
  thresholds?: OpsMetricThresholds | null
  autoRefreshEnabled?: boolean
  autoRefreshCountdown?: number
  refreshToken?: number
  fullscreen?: boolean
  customStartTime?: string | null
  customEndTime?: string | null
}

export interface Emits {
  (e: 'update:platform', value: string): void
  (e: 'update:group', value: number | null): void
  (e: 'update:channel', value: number | null): void
  (e: 'update:timeRange', value: string): void
  (e: 'update:queryMode', value: string): void
  (e: 'update:customTimeRange', startTime: string, endTime: string): void
  (e: 'refresh'): void
  (e: 'openRequestDetails', preset?: OpsRequestDetailsPreset): void
  (e: 'openErrorDetails', kind: 'request' | 'upstream'): void
  (e: 'openSettings'): void
  (e: 'openAlertRules'): void
  (e: 'enterFullscreen'): void
  (e: 'exitFullscreen'): void
}
