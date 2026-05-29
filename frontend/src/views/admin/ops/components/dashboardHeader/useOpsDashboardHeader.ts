import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api'
import { opsAPI, type OpsRealtimeTrafficSummary } from '@/api/admin/ops'
import { useAdminSettingsStore } from '@/stores'
import type { SelectOption } from '@/types'
import { formatNumber } from '@/utils/format'
import { loadAllAdminChannelOptions } from '@/utils/adminChannelOptions'
import { FILTER_PLATFORM_ORDER } from '@/utils/platformBranding'
import type { Emits, Props, RealtimeWindow } from './types'
import type { OpsRequestDetailsPreset } from '../opsRequestDetailsTypes'

export function useOpsDashboardHeader(props: Props, emit: Emits) {
  const { t } = useI18n()
  const adminSettingsStore = useAdminSettingsStore()

  const realtimeWindow = ref<RealtimeWindow>('1min')

  const overview = computed(() => props.overview ?? null)
  const systemMetrics = computed(() => overview.value?.system_metrics ?? null)

  const REALTIME_WINDOW_MINUTES: Record<RealtimeWindow, number> = {
    '1min': 1,
    '5min': 5,
    '30min': 30,
    '1h': 60
  }

  const TOOLBAR_RANGE_MINUTES: Record<string, number> = {
    '5m': 5,
    '30m': 30,
    '1h': 60,
    '6h': 6 * 60,
    '24h': 24 * 60
  }

  const availableRealtimeWindows = computed(() => {
    const toolbarMinutes = TOOLBAR_RANGE_MINUTES[props.timeRange] ?? 60
    return (['1min', '5min', '30min', '1h'] as const).filter((w) => REALTIME_WINDOW_MINUTES[w] <= toolbarMinutes)
  })

  watch(
    () => props.timeRange,
    () => {
      // The realtime window must be inside the toolbar window; reset to keep UX predictable.
      realtimeWindow.value = '1min'
      // Keep realtime traffic consistent with toolbar changes even when the window is already 1min.
      loadRealtimeTrafficSummary()
    }
  )

  // --- Filters ---

  const showCustomTimeRangeDialog = ref(false)
  const customStartTimeInput = ref('')
  const customEndTimeInput = ref('')

  function formatCustomTimeRangeLabel(startTime: string, endTime: string): string {
    const start = new Date(startTime)
    const end = new Date(endTime)
    const formatDate = (d: Date) => {
      const month = String(d.getMonth() + 1).padStart(2, '0')
      const day = String(d.getDate()).padStart(2, '0')
      const hour = String(d.getHours()).padStart(2, '0')
      const minute = String(d.getMinutes()).padStart(2, '0')
      return `${month}-${day} ${hour}:${minute}`
    }
    return `${formatDate(start)} ~ ${formatDate(end)}`
  }

  const groups = ref<Array<{ id: number; name: string; platform: string }>>([])
  const channelOptions = ref<SelectOption[]>([{ value: null, label: t('admin.ops.allChannels') }])

  const platformOptions = computed(() => [
    { value: '', label: t('common.all') },
    ...FILTER_PLATFORM_ORDER.map((platform) => ({
      value: platform,
      label: t(`admin.accounts.platforms.${platform}`)
    }))
  ])

  const timeRangeOptions = computed(() => [
    { value: '5m', label: t('admin.ops.timeRange.5m') },
    { value: '30m', label: t('admin.ops.timeRange.30m') },
    { value: '1h', label: t('admin.ops.timeRange.1h') },
    { value: '6h', label: t('admin.ops.timeRange.6h') },
    { value: '24h', label: t('admin.ops.timeRange.24h') },
    {
      value: 'custom',
      label: props.timeRange === 'custom' && props.customStartTime && props.customEndTime
        ? `${t('admin.ops.timeRange.custom')} (${formatCustomTimeRangeLabel(props.customStartTime, props.customEndTime)})`
        : t('admin.ops.timeRange.custom')
    }
  ])

  const queryModeOptions = computed(() => [
    { value: 'auto', label: t('admin.ops.queryMode.auto') },
    { value: 'raw', label: t('admin.ops.queryMode.raw') },
    { value: 'preagg', label: t('admin.ops.queryMode.preagg') }
  ])

  const groupOptions = computed(() => {
    const filtered = props.platform ? groups.value.filter((g) => g.platform === props.platform) : groups.value
    return [{ value: null, label: t('common.all') }, ...filtered.map((g) => ({ value: g.id, label: g.name }))]
  })

  watch(
    () => props.platform,
    (newPlatform) => {
      if (!newPlatform) return
      const currentGroup = groups.value.find((g) => g.id === props.groupId)
      if (currentGroup && currentGroup.platform !== newPlatform) {
        emit('update:group', null)
      }
    }
  )

  onMounted(async () => {
    const [groupsResult, channelsResult] = await Promise.allSettled([
      adminAPI.groups.getAll(),
      loadAllAdminChannelOptions()
    ])

    if (groupsResult.status === 'fulfilled') {
      groups.value = groupsResult.value.map((g) => ({ id: g.id, name: g.name, platform: g.platform }))
    } else {
      console.error('[OpsDashboardHeader] Failed to load groups', groupsResult.reason)
      groups.value = []
    }

    if (channelsResult.status === 'fulfilled') {
      channelOptions.value = [{ value: null, label: t('admin.ops.allChannels') }, ...channelsResult.value]
    } else {
      console.error('[OpsDashboardHeader] Failed to load channels', channelsResult.reason)
      channelOptions.value = [{ value: null, label: t('admin.ops.allChannels') }]
    }
  })

  function handlePlatformChange(val: string | number | boolean | null) {
    emit('update:platform', String(val || ''))
  }

  function handleGroupChange(val: string | number | boolean | null) {
    if (val === null || val === '' || typeof val === 'boolean') {
      emit('update:group', null)
      return
    }
    const id = typeof val === 'number' ? val : Number.parseInt(String(val), 10)
    emit('update:group', Number.isFinite(id) && id > 0 ? id : null)
  }

  function handleChannelChange(val: string | number | boolean | null) {
    if (val === null || val === '' || typeof val === 'boolean') {
      emit('update:channel', null)
      return
    }
    const id = typeof val === 'number' ? val : Number.parseInt(String(val), 10)
    emit('update:channel', Number.isFinite(id) && id > 0 ? id : null)
  }

  function handleTimeRangeChange(val: string | number | boolean | null) {
    const newValue = String(val || '1h')
    if (newValue === 'custom') {
      // 初始化为最近1小时
      const now = new Date()
      const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000)
      customStartTimeInput.value = oneHourAgo.toISOString().slice(0, 16)
      customEndTimeInput.value = now.toISOString().slice(0, 16)
      showCustomTimeRangeDialog.value = true
    } else {
      emit('update:timeRange', newValue)
    }
  }

  function handleCustomTimeRangeConfirm() {
    if (!customStartTimeInput.value || !customEndTimeInput.value) return
    const startTime = new Date(customStartTimeInput.value).toISOString()
    const endTime = new Date(customEndTimeInput.value).toISOString()
    // Emit custom time range first so the parent can build correct API params
    // when it reacts to timeRange switching to "custom".
    emit('update:customTimeRange', startTime, endTime)
    emit('update:timeRange', 'custom')
    showCustomTimeRangeDialog.value = false
  }

  function handleCustomTimeRangeCancel() {
    showCustomTimeRangeDialog.value = false
    // 如果当前不是 custom，不需要做任何事
    // 如果当前是 custom，保持不变
  }

  function handleQueryModeChange(val: string | number | boolean | null) {
    emit('update:queryMode', String(val || 'auto'))
  }

  function openDetails(preset?: OpsRequestDetailsPreset) {
    emit('openRequestDetails', preset)
  }

  function openErrorDetails(kind: 'request' | 'upstream') {
    emit('openErrorDetails', kind)
  }

  // --- Threshold checking helpers ---
  type ThresholdLevel = 'normal' | 'warning' | 'critical'

  function getSLAThresholdLevel(slaPercent: number | null): ThresholdLevel {
    if (slaPercent == null) return 'normal'
    const threshold = props.thresholds?.sla_percent_min
    if (threshold == null) return 'normal'

    // SLA is "higher is better":
    // - below threshold => critical
    // - within +0.1% buffer => warning
    const warningBuffer = 0.1

    if (slaPercent < threshold) return 'critical'
    if (slaPercent < threshold + warningBuffer) return 'warning'
    return 'normal'
  }

  function getTTFTThresholdLevel(ttftMs: number | null): ThresholdLevel {
    if (ttftMs == null) return 'normal'
    const threshold = props.thresholds?.ttft_p99_ms_max
    if (threshold == null) return 'normal'
    if (ttftMs >= threshold) return 'critical'
    if (ttftMs >= threshold * 0.8) return 'warning'
    return 'normal'
  }

  function getRequestErrorRateThresholdLevel(errorRatePercent: number | null): ThresholdLevel {
    if (errorRatePercent == null) return 'normal'
    const threshold = props.thresholds?.request_error_rate_percent_max
    if (threshold == null) return 'normal'
    if (errorRatePercent >= threshold) return 'critical'
    if (errorRatePercent >= threshold * 0.8) return 'warning'
    return 'normal'
  }

  function getUpstreamErrorRateThresholdLevel(upstreamErrorRatePercent: number | null): ThresholdLevel {
    if (upstreamErrorRatePercent == null) return 'normal'
    const threshold = props.thresholds?.upstream_error_rate_percent_max
    if (threshold == null) return 'normal'
    if (upstreamErrorRatePercent >= threshold) return 'critical'
    if (upstreamErrorRatePercent >= threshold * 0.8) return 'warning'
    return 'normal'
  }

  function getThresholdColorClass(level: ThresholdLevel): string {
    switch (level) {
      case 'critical':
        return 'text-red-600 dark:text-red-400'
      case 'warning':
        return 'text-yellow-600 dark:text-yellow-400'
      default:
        return 'text-green-600 dark:text-green-400'
    }
  }

  // --- Realtime / Overview labels ---

  const totalRequestsLabel = computed(() => formatNumber(overview.value?.request_count_total ?? 0))
  const totalTokensLabel = computed(() => formatNumber(overview.value?.token_consumed ?? 0))

  const realtimeTrafficSummary = ref<OpsRealtimeTrafficSummary | null>(null)
  const realtimeTrafficLoading = ref(false)

  function makeZeroRealtimeTrafficSummary(): OpsRealtimeTrafficSummary {
    const now = new Date().toISOString()
    return {
      window: realtimeWindow.value,
      start_time: now,
      end_time: now,
      platform: props.platform,
      group_id: props.groupId,
      channel_id: props.channelId,
      qps: { current: 0, peak: 0, avg: 0 },
      tps: { current: 0, peak: 0, avg: 0 }
    }
  }

  async function loadRealtimeTrafficSummary() {
    if (realtimeTrafficLoading.value) return
    if (!adminSettingsStore.opsRealtimeMonitoringEnabled) {
      realtimeTrafficSummary.value = makeZeroRealtimeTrafficSummary()
      return
    }
    realtimeTrafficLoading.value = true
    try {
      const res = await opsAPI.getRealtimeTrafficSummary(
        realtimeWindow.value,
        props.platform,
        props.groupId,
        props.channelId
      )
      if (res && res.enabled === false) {
        adminSettingsStore.setOpsRealtimeMonitoringEnabledLocal(false)
      }
      realtimeTrafficSummary.value = res?.summary ?? null
    } catch (err) {
      console.error('[OpsDashboardHeader] Failed to load realtime traffic summary', err)
      realtimeTrafficSummary.value = null
    } finally {
      realtimeTrafficLoading.value = false
    }
  }

  watch(
    () => [realtimeWindow.value, props.platform, props.groupId, props.channelId] as const,
    () => {
      loadRealtimeTrafficSummary()
    },
    { immediate: true }
  )

  watch(
    () => adminSettingsStore.opsRealtimeMonitoringEnabled,
    (enabled) => {
      if (!enabled) {
        // Keep UI stable when realtime monitoring is turned off.
        realtimeTrafficSummary.value = makeZeroRealtimeTrafficSummary()
      } else {
        loadRealtimeTrafficSummary()
      }
    },
    { immediate: true }
  )

  // Realtime traffic refresh follows the parent (OpsDashboard) refresh cadence.
  watch(
    () => props.refreshToken,
    (refreshToken, previousRefreshToken) => {
      if (!props.autoRefreshEnabled) return
      if (props.loading) return
      if (refreshToken === undefined || refreshToken === previousRefreshToken) return
      loadRealtimeTrafficSummary()
    }
  )

  // no-op: parent controls refresh cadence

  const displayRealTimeQps = computed(() => {
    const v = realtimeTrafficSummary.value?.qps?.current
    return typeof v === 'number' && Number.isFinite(v) ? v : 0
  })

  const displayRealTimeTps = computed(() => {
    const v = realtimeTrafficSummary.value?.tps?.current
    return typeof v === 'number' && Number.isFinite(v) ? v : 0
  })

  const realtimeQpsPeakLabel = computed(() => {
    const v = realtimeTrafficSummary.value?.qps?.peak
    return typeof v === 'number' && Number.isFinite(v) ? v.toFixed(1) : '-'
  })
  const realtimeTpsPeakLabel = computed(() => {
    const v = realtimeTrafficSummary.value?.tps?.peak
    return typeof v === 'number' && Number.isFinite(v) ? v.toFixed(1) : '-'
  })
  const realtimeQpsAvgLabel = computed(() => {
    const v = realtimeTrafficSummary.value?.qps?.avg
    return typeof v === 'number' && Number.isFinite(v) ? v.toFixed(1) : '-'
  })
  const realtimeTpsAvgLabel = computed(() => {
    const v = realtimeTrafficSummary.value?.tps?.avg
    return typeof v === 'number' && Number.isFinite(v) ? v.toFixed(1) : '-'
  })

  const qpsAvgLabel = computed(() => {
    const v = overview.value?.qps?.avg
    if (typeof v !== 'number') return '-'
    return v.toFixed(1)
  })

  const tpsAvgLabel = computed(() => {
    const v = overview.value?.tps?.avg
    if (typeof v !== 'number') return '-'
    return v.toFixed(1)
  })

  const slaPercent = computed(() => {
    const v = overview.value?.sla
    if (typeof v !== 'number') return null
    return v * 100
  })

  const errorRatePercent = computed(() => {
    const v = overview.value?.error_rate
    if (typeof v !== 'number') return null
    return v * 100
  })

  const upstreamErrorRatePercent = computed(() => {
    const v = overview.value?.upstream_error_rate
    if (typeof v !== 'number') return null
    return v * 100
  })

  const durationP99Ms = computed(() => overview.value?.duration?.p99_ms ?? null)
  const durationP95Ms = computed(() => overview.value?.duration?.p95_ms ?? null)
  const durationP90Ms = computed(() => overview.value?.duration?.p90_ms ?? null)
  const durationP50Ms = computed(() => overview.value?.duration?.p50_ms ?? null)
  const durationAvgMs = computed(() => overview.value?.duration?.avg_ms ?? null)
  const durationMaxMs = computed(() => overview.value?.duration?.max_ms ?? null)

  const ttftP99Ms = computed(() => overview.value?.ttft?.p99_ms ?? null)
  const ttftP95Ms = computed(() => overview.value?.ttft?.p95_ms ?? null)
  const ttftP90Ms = computed(() => overview.value?.ttft?.p90_ms ?? null)
  const ttftP50Ms = computed(() => overview.value?.ttft?.p50_ms ?? null)
  const ttftAvgMs = computed(() => overview.value?.ttft?.avg_ms ?? null)
  const ttftMaxMs = computed(() => overview.value?.ttft?.max_ms ?? null)

  // --- Health Score & Diagnosis (primary) ---

  const isSystemIdle = computed(() => {
    const ov = overview.value
    if (!ov) return true
    const qps = ov.qps?.current
    const errorRate = ov.error_rate ?? 0
    return (qps ?? 0) === 0 && errorRate === 0
  })

  const healthScoreValue = computed<number | null>(() => {
    const v = overview.value?.health_score
    return typeof v === 'number' && Number.isFinite(v) ? v : null
  })

  const healthScoreColor = computed(() => {
    if (isSystemIdle.value) return '#9ca3af' // gray-400
    const score = healthScoreValue.value
    if (score == null) return '#9ca3af'
    if (score >= 90) return '#10b981' // green
    if (score >= 60) return '#f59e0b' // yellow
    return '#ef4444' // red
  })

  const healthScoreClass = computed(() => {
    if (isSystemIdle.value) return 'text-gray-400'
    const score = healthScoreValue.value
    if (score == null) return 'text-gray-400'
    if (score >= 90) return 'text-green-500'
    if (score >= 60) return 'text-yellow-500'
    return 'text-red-500'
  })

  const circleSize = computed(() => props.fullscreen ? 140 : 100)
  const strokeWidth = computed(() => props.fullscreen ? 10 : 8)
  const radius = computed(() => (circleSize.value - strokeWidth.value) / 2)
  const circumference = computed(() => 2 * Math.PI * radius.value)
  const dashOffset = computed(() => {
    if (isSystemIdle.value) return 0
    if (healthScoreValue.value == null) return 0
    const score = Math.max(0, Math.min(100, healthScoreValue.value))
    return circumference.value - (score / 100) * circumference.value
  })

  interface DiagnosisItem {
    type: 'critical' | 'warning' | 'info'
    message: string
    impact: string
    action?: string
  }

  const diagnosisReport = computed<DiagnosisItem[]>(() => {
    const ov = overview.value
    if (!ov) return []

    const report: DiagnosisItem[] = []

    if (isSystemIdle.value) {
      report.push({
        type: 'info',
        message: t('admin.ops.diagnosis.idle'),
        impact: t('admin.ops.diagnosis.idleImpact')
      })
      return report
    }

    // Resource diagnostics (highest priority)
    const sm = ov.system_metrics
    if (sm) {
      if (sm.db_ok === false) {
        report.push({
          type: 'critical',
          message: t('admin.ops.diagnosis.dbDown'),
          impact: t('admin.ops.diagnosis.dbDownImpact'),
          action: t('admin.ops.diagnosis.dbDownAction')
        })
      }
      if (sm.redis_ok === false) {
        report.push({
          type: 'warning',
          message: t('admin.ops.diagnosis.redisDown'),
          impact: t('admin.ops.diagnosis.redisDownImpact'),
          action: t('admin.ops.diagnosis.redisDownAction')
        })
      }

      const cpuPct = sm.cpu_usage_percent ?? 0
      if (cpuPct > 90) {
        report.push({
          type: 'critical',
          message: t('admin.ops.diagnosis.cpuCritical', { usage: cpuPct.toFixed(1) }),
          impact: t('admin.ops.diagnosis.cpuCriticalImpact'),
          action: t('admin.ops.diagnosis.cpuCriticalAction')
        })
      } else if (cpuPct > 80) {
        report.push({
          type: 'warning',
          message: t('admin.ops.diagnosis.cpuHigh', { usage: cpuPct.toFixed(1) }),
          impact: t('admin.ops.diagnosis.cpuHighImpact'),
          action: t('admin.ops.diagnosis.cpuHighAction')
        })
      }

      const memPct = sm.memory_usage_percent ?? 0
      if (memPct > 90) {
        report.push({
          type: 'critical',
          message: t('admin.ops.diagnosis.memoryCritical', { usage: memPct.toFixed(1) }),
          impact: t('admin.ops.diagnosis.memoryCriticalImpact'),
          action: t('admin.ops.diagnosis.memoryCriticalAction')
        })
      } else if (memPct > 85) {
        report.push({
          type: 'warning',
          message: t('admin.ops.diagnosis.memoryHigh', { usage: memPct.toFixed(1) }),
          impact: t('admin.ops.diagnosis.memoryHighImpact'),
          action: t('admin.ops.diagnosis.memoryHighAction')
        })
      }
    }

    const ttftP99 = ov.ttft?.p99_ms ?? 0
    if (ttftP99 > 500) {
      report.push({
        type: 'warning',
        message: t('admin.ops.diagnosis.ttftHigh', { ttft: ttftP99.toFixed(0) }),
        impact: t('admin.ops.diagnosis.ttftHighImpact'),
        action: t('admin.ops.diagnosis.ttftHighAction')
      })
    }

    // Error rate diagnostics (adjusted thresholds)
    const upstreamRatePct = (ov.upstream_error_rate ?? 0) * 100
    if (upstreamRatePct > 5) {
      report.push({
        type: 'critical',
        message: t('admin.ops.diagnosis.upstreamCritical', { rate: upstreamRatePct.toFixed(2) }),
        impact: t('admin.ops.diagnosis.upstreamCriticalImpact'),
        action: t('admin.ops.diagnosis.upstreamCriticalAction')
      })
    } else if (upstreamRatePct > 2) {
      report.push({
        type: 'warning',
        message: t('admin.ops.diagnosis.upstreamHigh', { rate: upstreamRatePct.toFixed(2) }),
        impact: t('admin.ops.diagnosis.upstreamHighImpact'),
        action: t('admin.ops.diagnosis.upstreamHighAction')
      })
    }

    const errorPct = (ov.error_rate ?? 0) * 100
    if (errorPct > 3) {
      report.push({
        type: 'critical',
        message: t('admin.ops.diagnosis.errorHigh', { rate: errorPct.toFixed(2) }),
        impact: t('admin.ops.diagnosis.errorHighImpact'),
        action: t('admin.ops.diagnosis.errorHighAction')
      })
    } else if (errorPct > 0.5) {
      report.push({
        type: 'warning',
        message: t('admin.ops.diagnosis.errorElevated', { rate: errorPct.toFixed(2) }),
        impact: t('admin.ops.diagnosis.errorElevatedImpact'),
        action: t('admin.ops.diagnosis.errorElevatedAction')
      })
    }

    // SLA diagnostics
    const slaPct = (ov.sla ?? 0) * 100
    if (slaPct < 90) {
      report.push({
        type: 'critical',
        message: t('admin.ops.diagnosis.slaCritical', { sla: slaPct.toFixed(2) }),
        impact: t('admin.ops.diagnosis.slaCriticalImpact'),
        action: t('admin.ops.diagnosis.slaCriticalAction')
      })
    } else if (slaPct < 98) {
      report.push({
        type: 'warning',
        message: t('admin.ops.diagnosis.slaLow', { sla: slaPct.toFixed(2) }),
        impact: t('admin.ops.diagnosis.slaLowImpact'),
        action: t('admin.ops.diagnosis.slaLowAction')
      })
    }

    // Health score diagnostics (lowest priority)
    if (healthScoreValue.value != null) {
      if (healthScoreValue.value < 60) {
        report.push({
          type: 'critical',
          message: t('admin.ops.diagnosis.healthCritical', { score: healthScoreValue.value }),
          impact: t('admin.ops.diagnosis.healthCriticalImpact'),
          action: t('admin.ops.diagnosis.healthCriticalAction')
        })
      } else if (healthScoreValue.value < 90) {
        report.push({
          type: 'warning',
          message: t('admin.ops.diagnosis.healthLow', { score: healthScoreValue.value }),
          impact: t('admin.ops.diagnosis.healthLowImpact'),
          action: t('admin.ops.diagnosis.healthLowAction')
        })
      }
    }

    if (report.length === 0) {
      report.push({
        type: 'info',
        message: t('admin.ops.diagnosis.healthy'),
        impact: t('admin.ops.diagnosis.healthyImpact')
      })
    }

    return report
  })

  // --- System health (secondary) ---

  function formatTimeShort(ts?: string | null): string {
    if (!ts) return '-'
    const d = new Date(ts)
    if (Number.isNaN(d.getTime())) return '-'
    return d.toLocaleTimeString()
  }

  const cpuPercentValue = computed<number | null>(() => {
    const v = systemMetrics.value?.cpu_usage_percent
    return typeof v === 'number' && Number.isFinite(v) ? v : null
  })

  const cpuPercentClass = computed(() => {
    const v = cpuPercentValue.value
    if (v == null) return 'text-gray-900 dark:text-white'
    if (v >= 95) return 'text-rose-600 dark:text-rose-400'
    if (v >= 80) return 'text-yellow-600 dark:text-yellow-400'
    return 'text-emerald-600 dark:text-emerald-400'
  })

  const memPercentValue = computed<number | null>(() => {
    const v = systemMetrics.value?.memory_usage_percent
    return typeof v === 'number' && Number.isFinite(v) ? v : null
  })

  const memPercentClass = computed(() => {
    const v = memPercentValue.value
    if (v == null) return 'text-gray-900 dark:text-white'
    if (v >= 95) return 'text-rose-600 dark:text-rose-400'
    if (v >= 85) return 'text-yellow-600 dark:text-yellow-400'
    return 'text-emerald-600 dark:text-emerald-400'
  })

  const dbConnActiveValue = computed<number | null>(() => {
    const v = systemMetrics.value?.db_conn_active
    return typeof v === 'number' && Number.isFinite(v) ? v : null
  })

  const dbConnIdleValue = computed<number | null>(() => {
    const v = systemMetrics.value?.db_conn_idle
    return typeof v === 'number' && Number.isFinite(v) ? v : null
  })

  const dbConnWaitingValue = computed<number | null>(() => {
    const v = systemMetrics.value?.db_conn_waiting
    return typeof v === 'number' && Number.isFinite(v) ? v : null
  })

  const dbConnOpenValue = computed<number | null>(() => {
    if (dbConnActiveValue.value == null || dbConnIdleValue.value == null) return null
    return dbConnActiveValue.value + dbConnIdleValue.value
  })

  const dbMaxOpenConnsValue = computed<number | null>(() => {
    const v = systemMetrics.value?.db_max_open_conns
    return typeof v === 'number' && Number.isFinite(v) ? v : null
  })

  const dbUsagePercent = computed<number | null>(() => {
    if (dbConnOpenValue.value == null || dbMaxOpenConnsValue.value == null || dbMaxOpenConnsValue.value <= 0) return null
    return Math.min(100, Math.max(0, (dbConnOpenValue.value / dbMaxOpenConnsValue.value) * 100))
  })

  const dbMiddleLabel = computed(() => {
    if (systemMetrics.value?.db_ok === false) return 'FAIL'
    if (dbUsagePercent.value != null) return `${dbUsagePercent.value.toFixed(0)}%`
    if (systemMetrics.value?.db_ok === true) return t('admin.ops.ok')
    return t('admin.ops.noData')
  })

  const dbMiddleClass = computed(() => {
    if (systemMetrics.value?.db_ok === false) return 'text-rose-600 dark:text-rose-400'
    if (dbUsagePercent.value != null) {
      if (dbUsagePercent.value >= 90) return 'text-rose-600 dark:text-rose-400'
      if (dbUsagePercent.value >= 70) return 'text-yellow-600 dark:text-yellow-400'
      return 'text-emerald-600 dark:text-emerald-400'
    }
    if (systemMetrics.value?.db_ok === true) return 'text-emerald-600 dark:text-emerald-400'
    return 'text-gray-900 dark:text-white'
  })

  const redisConnTotalValue = computed<number | null>(() => {
    const v = systemMetrics.value?.redis_conn_total
    return typeof v === 'number' && Number.isFinite(v) ? v : null
  })

  const redisConnIdleValue = computed<number | null>(() => {
    const v = systemMetrics.value?.redis_conn_idle
    return typeof v === 'number' && Number.isFinite(v) ? v : null
  })

  const redisConnActiveValue = computed<number | null>(() => {
    if (redisConnTotalValue.value == null || redisConnIdleValue.value == null) return null
    return Math.max(redisConnTotalValue.value - redisConnIdleValue.value, 0)
  })

  const redisPoolSizeValue = computed<number | null>(() => {
    const v = systemMetrics.value?.redis_pool_size
    return typeof v === 'number' && Number.isFinite(v) ? v : null
  })

  const redisUsagePercent = computed<number | null>(() => {
    if (redisConnTotalValue.value == null || redisPoolSizeValue.value == null || redisPoolSizeValue.value <= 0) return null
    return Math.min(100, Math.max(0, (redisConnTotalValue.value / redisPoolSizeValue.value) * 100))
  })

  const redisMiddleLabel = computed(() => {
    if (systemMetrics.value?.redis_ok === false) return 'FAIL'
    if (redisUsagePercent.value != null) return `${redisUsagePercent.value.toFixed(0)}%`
    if (systemMetrics.value?.redis_ok === true) return t('admin.ops.ok')
    return t('admin.ops.noData')
  })

  const redisMiddleClass = computed(() => {
    if (systemMetrics.value?.redis_ok === false) return 'text-rose-600 dark:text-rose-400'
    if (redisUsagePercent.value != null) {
      if (redisUsagePercent.value >= 90) return 'text-rose-600 dark:text-rose-400'
      if (redisUsagePercent.value >= 70) return 'text-yellow-600 dark:text-yellow-400'
      return 'text-emerald-600 dark:text-emerald-400'
    }
    if (systemMetrics.value?.redis_ok === true) return 'text-emerald-600 dark:text-emerald-400'
    return 'text-gray-900 dark:text-white'
  })

  const goroutineCountValue = computed<number | null>(() => {
    const v = systemMetrics.value?.goroutine_count
    return typeof v === 'number' && Number.isFinite(v) ? v : null
  })

  const goroutinesWarnThreshold = 8_000
  const goroutinesCriticalThreshold = 15_000

  const goroutineStatus = computed<'ok' | 'warning' | 'critical' | 'unknown'>(() => {
    const n = goroutineCountValue.value
    if (n == null) return 'unknown'
    if (n >= goroutinesCriticalThreshold) return 'critical'
    if (n >= goroutinesWarnThreshold) return 'warning'
    return 'ok'
  })

  const goroutineStatusLabel = computed(() => {
    switch (goroutineStatus.value) {
      case 'ok':
        return t('admin.ops.ok')
      case 'warning':
        return t('common.warning')
      case 'critical':
        return t('common.critical')
      default:
        return t('admin.ops.noData')
    }
  })

  const goroutineStatusClass = computed(() => {
    switch (goroutineStatus.value) {
      case 'ok':
        return 'text-emerald-600 dark:text-emerald-400'
      case 'warning':
        return 'text-yellow-600 dark:text-yellow-400'
      case 'critical':
        return 'text-rose-600 dark:text-rose-400'
      default:
        return 'text-gray-900 dark:text-white'
    }
  })

  const jobHeartbeats = computed(() => overview.value?.job_heartbeats ?? [])

  const jobsStatus = computed<'ok' | 'warn' | 'unknown'>(() => {
    const list = jobHeartbeats.value
    if (!list.length) return 'unknown'
    for (const hb of list) {
      if (!hb) continue
      if (hb.last_error_at && (!hb.last_success_at || hb.last_error_at > hb.last_success_at)) return 'warn'
    }
    return 'ok'
  })

  const jobsWarnCount = computed(() => {
    let warn = 0
    for (const hb of jobHeartbeats.value) {
      if (!hb) continue
      if (hb.last_error_at && (!hb.last_success_at || hb.last_error_at > hb.last_success_at)) warn++
    }
    return warn
  })

  const jobsStatusLabel = computed(() => {
    switch (jobsStatus.value) {
      case 'ok':
        return t('admin.ops.ok')
      case 'warn':
        return t('common.warning')
      default:
        return t('admin.ops.noData')
    }
  })

  const jobsStatusClass = computed(() => {
    switch (jobsStatus.value) {
      case 'ok':
        return 'text-emerald-600 dark:text-emerald-400'
      case 'warn':
        return 'text-yellow-600 dark:text-yellow-400'
      default:
        return 'text-gray-900 dark:text-white'
    }
  })

  const showJobsDetails = ref(false)

  function openJobsDetails() {
    showJobsDetails.value = true
  }

  function handleToolbarRefresh() {
    loadRealtimeTrafficSummary()
    emit('refresh')
  }

  return {
    t,
    formatNumber,
    realtimeWindow,
    overview,
    systemMetrics,
    availableRealtimeWindows,
    showCustomTimeRangeDialog,
    customStartTimeInput,
    customEndTimeInput,
    formatCustomTimeRangeLabel,
    groups,
    channelOptions,
    platformOptions,
    timeRangeOptions,
    queryModeOptions,
    groupOptions,
    handlePlatformChange,
    handleGroupChange,
    handleChannelChange,
    handleTimeRangeChange,
    handleCustomTimeRangeConfirm,
    handleCustomTimeRangeCancel,
    handleQueryModeChange,
    openDetails,
    openErrorDetails,
    getSLAThresholdLevel,
    getTTFTThresholdLevel,
    getRequestErrorRateThresholdLevel,
    getUpstreamErrorRateThresholdLevel,
    getThresholdColorClass,
    totalRequestsLabel,
    totalTokensLabel,
    realtimeTrafficSummary,
    realtimeTrafficLoading,
    loadRealtimeTrafficSummary,
    displayRealTimeQps,
    displayRealTimeTps,
    realtimeQpsPeakLabel,
    realtimeTpsPeakLabel,
    realtimeQpsAvgLabel,
    realtimeTpsAvgLabel,
    qpsAvgLabel,
    tpsAvgLabel,
    slaPercent,
    errorRatePercent,
    upstreamErrorRatePercent,
    durationP99Ms,
    durationP95Ms,
    durationP90Ms,
    durationP50Ms,
    durationAvgMs,
    durationMaxMs,
    ttftP99Ms,
    ttftP95Ms,
    ttftP90Ms,
    ttftP50Ms,
    ttftAvgMs,
    ttftMaxMs,
    isSystemIdle,
    healthScoreValue,
    healthScoreColor,
    healthScoreClass,
    circleSize,
    strokeWidth,
    radius,
    circumference,
    dashOffset,
    diagnosisReport,
    formatTimeShort,
    cpuPercentValue,
    cpuPercentClass,
    memPercentValue,
    memPercentClass,
    dbConnActiveValue,
    dbConnIdleValue,
    dbConnWaitingValue,
    dbConnOpenValue,
    dbMaxOpenConnsValue,
    dbUsagePercent,
    dbMiddleLabel,
    dbMiddleClass,
    redisConnTotalValue,
    redisConnIdleValue,
    redisConnActiveValue,
    redisPoolSizeValue,
    redisUsagePercent,
    redisMiddleLabel,
    redisMiddleClass,
    goroutineCountValue,
    goroutinesWarnThreshold,
    goroutinesCriticalThreshold,
    goroutineStatus,
    goroutineStatusLabel,
    goroutineStatusClass,
    jobHeartbeats,
    jobsStatus,
    jobsWarnCount,
    jobsStatusLabel,
    jobsStatusClass,
    showJobsDetails,
    openJobsDetails,
    handleToolbarRefresh,
  }
}
