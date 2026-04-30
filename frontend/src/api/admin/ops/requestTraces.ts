import { apiClient } from '../../client'
import type {
  OpsRequestTraceCleanupRequest,
  OpsRequestTraceCleanupResult,
  OpsRequestOptions,
  OpsRequestTraceDetail,
  OpsRequestTraceFilter,
  OpsRequestTraceListResponse,
  OpsRequestTraceRawDetail,
  OpsRequestSubjectInsights,
  OpsRequestSubjectInsightsParams,
  OpsRequestTraceSummary
} from './types'

export async function listRequestTraces(
  params: OpsRequestTraceFilter,
  options: OpsRequestOptions = {}
): Promise<OpsRequestTraceListResponse> {
  const { data } = await apiClient.get<OpsRequestTraceListResponse>('/admin/ops/request-details', {
    params,
    signal: options.signal
  })
  return data
}

export async function getRequestTraceSummary(
  params: OpsRequestTraceFilter,
  options: OpsRequestOptions = {}
): Promise<OpsRequestTraceSummary> {
  const { data } = await apiClient.get<OpsRequestTraceSummary>('/admin/ops/request-details/summary', {
    params,
    signal: options.signal
  })
  return data
}

export async function getRequestTraceDetail(
  id: number,
  options: OpsRequestOptions = {}
): Promise<OpsRequestTraceDetail> {
  const { data } = await apiClient.get<OpsRequestTraceDetail>(`/admin/ops/request-details/${id}`, {
    signal: options.signal
  })
  return data
}

export async function getRequestTraceRawDetail(
  id: number,
  options: OpsRequestOptions = {}
): Promise<OpsRequestTraceRawDetail> {
  const { data } = await apiClient.get<OpsRequestTraceRawDetail>(`/admin/ops/request-details/${id}/raw`, {
    signal: options.signal
  })
  return data
}

export async function getSubjectInsights(
  params: OpsRequestSubjectInsightsParams,
  options: OpsRequestOptions = {}
): Promise<OpsRequestSubjectInsights> {
  const { data } = await apiClient.get<OpsRequestSubjectInsights>('/admin/ops/request-details/subjects/insights', {
    params,
    signal: options.signal
  })
  return data
}

export async function exportRequestTracesCSV(
  params: OpsRequestTraceFilter,
  includeRaw: boolean,
  options: OpsRequestOptions = {}
): Promise<{ blob: Blob; filename: string }> {
  const response = await apiClient.get<Blob>('/admin/ops/request-details/export.csv', {
    params: {
      ...params,
      include_raw: includeRaw ? '1' : '0'
    },
    responseType: 'blob',
    signal: options.signal
  })

  const header = String(response.headers['content-disposition'] || '')
  const matched = header.match(/filename="?([^"]+)"?/)
  const filename = matched?.[1] || 'request-details.csv'

  return {
    blob: response.data,
    filename
  }
}

export async function cleanupRequestTraces(
  payload: OpsRequestTraceCleanupRequest,
  options: OpsRequestOptions = {}
): Promise<OpsRequestTraceCleanupResult> {
  const { data } = await apiClient.post<OpsRequestTraceCleanupResult>('/admin/ops/request-details/cleanup', payload, {
    signal: options.signal
  })
  return data
}
