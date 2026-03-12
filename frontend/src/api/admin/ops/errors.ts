import { apiClient } from '../../client'
import type { PaginatedResponse } from '@/types'
import type { OpsErrorDetail, OpsErrorLogsResponse, OpsErrorListQueryParams, OpsRequestDetailsParams, OpsRequestDetailsResponse, OpsRetryAttempt, OpsRetryRequest, OpsRetryResult } from './types'

export async function listErrorLogs(params: OpsErrorListQueryParams): Promise<OpsErrorLogsResponse> {
  const { data } = await apiClient.get<OpsErrorLogsResponse>('/admin/ops/errors', { params })
  return data
}

export async function getErrorLogDetail(id: number): Promise<OpsErrorDetail> {
  const { data } = await apiClient.get<OpsErrorDetail>(`/admin/ops/errors/${id}`)
  return data
}

export async function retryErrorRequest(id: number, req: OpsRetryRequest): Promise<OpsRetryResult> {
  const { data } = await apiClient.post<OpsRetryResult>(`/admin/ops/errors/${id}/retry`, req)
  return data
}

export async function listRetryAttempts(errorId: number, limit = 50): Promise<OpsRetryAttempt[]> {
  const { data } = await apiClient.get<OpsRetryAttempt[]>(`/admin/ops/errors/${errorId}/retries`, { params: { limit } })
  return data
}

export async function updateErrorResolved(errorId: number, resolved: boolean): Promise<void> {
  await apiClient.put(`/admin/ops/errors/${errorId}/resolve`, { resolved })
}

// New split endpoints
export async function listRequestErrors(params: OpsErrorListQueryParams): Promise<OpsErrorLogsResponse> {
  const { data } = await apiClient.get<OpsErrorLogsResponse>('/admin/ops/request-errors', { params })
  return data
}

export async function listUpstreamErrors(params: OpsErrorListQueryParams): Promise<OpsErrorLogsResponse> {
  const { data } = await apiClient.get<OpsErrorLogsResponse>('/admin/ops/upstream-errors', { params })
  return data
}

export async function getRequestErrorDetail(id: number): Promise<OpsErrorDetail> {
  const { data } = await apiClient.get<OpsErrorDetail>(`/admin/ops/request-errors/${id}`)
  return data
}

export async function getUpstreamErrorDetail(id: number): Promise<OpsErrorDetail> {
  const { data } = await apiClient.get<OpsErrorDetail>(`/admin/ops/upstream-errors/${id}`)
  return data
}

export async function retryRequestErrorClient(id: number): Promise<OpsRetryResult> {
  const { data } = await apiClient.post<OpsRetryResult>(`/admin/ops/request-errors/${id}/retry-client`, {})
  return data
}

export async function retryRequestErrorUpstreamEvent(id: number, idx: number): Promise<OpsRetryResult> {
  const { data } = await apiClient.post<OpsRetryResult>(`/admin/ops/request-errors/${id}/upstream-errors/${idx}/retry`, {})
  return data
}

export async function retryUpstreamError(id: number): Promise<OpsRetryResult> {
  const { data } = await apiClient.post<OpsRetryResult>(`/admin/ops/upstream-errors/${id}/retry`, {})
  return data
}

export async function updateRequestErrorResolved(errorId: number, resolved: boolean): Promise<void> {
  await apiClient.put(`/admin/ops/request-errors/${errorId}/resolve`, { resolved })
}

export async function updateUpstreamErrorResolved(errorId: number, resolved: boolean): Promise<void> {
  await apiClient.put(`/admin/ops/upstream-errors/${errorId}/resolve`, { resolved })
}

export async function listRequestErrorUpstreamErrors(
  id: number,
  params: OpsErrorListQueryParams = {},
  options: { include_detail?: boolean } = {}
): Promise<PaginatedResponse<OpsErrorDetail>> {
  const query: Record<string, any> = { ...params }
  if (options.include_detail) query.include_detail = '1'
  const { data } = await apiClient.get<PaginatedResponse<OpsErrorDetail>>(`/admin/ops/request-errors/${id}/upstream-errors`, { params: query })
  return data
}

export async function listRequestDetails(params: OpsRequestDetailsParams): Promise<OpsRequestDetailsResponse> {
  const { data } = await apiClient.get<OpsRequestDetailsResponse>('/admin/ops/requests', { params })
  return data
}
