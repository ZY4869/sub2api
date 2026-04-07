import { apiClient } from '../../client'
import type { OpsDashboardOverview, OpsDashboardSnapshotV2Response, OpsErrorDistributionResponse, OpsErrorTrendResponse, OpsLatencyHistogramResponse, OpsOpenAITokenStatsParams, OpsOpenAITokenStatsResponse, OpsQueryMode, OpsRequestOptions, OpsThroughputTrendResponse } from './types'

export async function getDashboardOverview(
  params: {
  time_range?: '5m' | '30m' | '1h' | '6h' | '24h'
  start_time?: string
  end_time?: string
  platform?: string
  group_id?: number | null
  channel_id?: number | null
  mode?: OpsQueryMode
  },
  options: OpsRequestOptions = {}
): Promise<OpsDashboardOverview> {
  const { data } = await apiClient.get<OpsDashboardOverview>('/admin/ops/dashboard/overview', {
    params,
    signal: options.signal
  })
  return data
}

export async function getDashboardSnapshotV2(
  params: {
  time_range?: '5m' | '30m' | '1h' | '6h' | '24h'
  start_time?: string
  end_time?: string
  platform?: string
  group_id?: number | null
  channel_id?: number | null
  mode?: OpsQueryMode
  },
  options: OpsRequestOptions = {}
): Promise<OpsDashboardSnapshotV2Response> {
  const { data } = await apiClient.get<OpsDashboardSnapshotV2Response>('/admin/ops/dashboard/snapshot-v2', {
    params,
    signal: options.signal
  })
  return data
}

export async function getThroughputTrend(
  params: {
  time_range?: '5m' | '30m' | '1h' | '6h' | '24h'
  start_time?: string
  end_time?: string
  platform?: string
  group_id?: number | null
  channel_id?: number | null
  mode?: OpsQueryMode
  },
  options: OpsRequestOptions = {}
): Promise<OpsThroughputTrendResponse> {
  const { data } = await apiClient.get<OpsThroughputTrendResponse>('/admin/ops/dashboard/throughput-trend', {
    params,
    signal: options.signal
  })
  return data
}

export async function getLatencyHistogram(
  params: {
  time_range?: '5m' | '30m' | '1h' | '6h' | '24h'
  start_time?: string
  end_time?: string
  platform?: string
  group_id?: number | null
  channel_id?: number | null
  mode?: OpsQueryMode
  },
  options: OpsRequestOptions = {}
): Promise<OpsLatencyHistogramResponse> {
  const { data } = await apiClient.get<OpsLatencyHistogramResponse>('/admin/ops/dashboard/latency-histogram', {
    params,
    signal: options.signal
  })
  return data
}

export async function getErrorTrend(
  params: {
  time_range?: '5m' | '30m' | '1h' | '6h' | '24h'
  start_time?: string
  end_time?: string
  platform?: string
  group_id?: number | null
  channel_id?: number | null
  mode?: OpsQueryMode
  },
  options: OpsRequestOptions = {}
): Promise<OpsErrorTrendResponse> {
  const { data } = await apiClient.get<OpsErrorTrendResponse>('/admin/ops/dashboard/error-trend', {
    params,
    signal: options.signal
  })
  return data
}

export async function getErrorDistribution(
  params: {
  time_range?: '5m' | '30m' | '1h' | '6h' | '24h'
  start_time?: string
  end_time?: string
  platform?: string
  group_id?: number | null
  channel_id?: number | null
  mode?: OpsQueryMode
  },
  options: OpsRequestOptions = {}
): Promise<OpsErrorDistributionResponse> {
  const { data } = await apiClient.get<OpsErrorDistributionResponse>('/admin/ops/dashboard/error-distribution', {
    params,
    signal: options.signal
  })
  return data
}

export async function getOpenAITokenStats(
  params: OpsOpenAITokenStatsParams,
  options: OpsRequestOptions = {}
): Promise<OpsOpenAITokenStatsResponse> {
  const { data } = await apiClient.get<OpsOpenAITokenStatsResponse>('/admin/ops/dashboard/openai-token-stats', {
    params,
    signal: options.signal
  })
  return data
}
