import axios from 'axios'

export type ImageBatchJobStatus =
  | 'created'
  | 'uploading'
  | 'submitted'
  | 'running'
  | 'indexing'
  | 'settling'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | 'output_deleted'

export type ImageBatchItemStatus = 'pending' | 'success' | 'failed' | 'cancelled'

export interface ImageBatchJob {
  id: string
  object: string
  status: ImageBatchJobStatus
  provider: string
  display_model_id: string
  size?: string
  counts: { items: number; succeeded: number; failed: number; cancelled: number }
  friendly_error?: string
  error_id?: string
  created_at: string
  updated_at: string
  submitted_at?: string
  completed_at?: string
  cancelled_at?: string
}

export interface ImageBatchItem {
  custom_id: string
  status: ImageBatchItemStatus
  output_count: number
  friendly_error?: string
  error_id?: string
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface ImageBatchModel {
  id: string
  object: string
  display_name?: string
  provider: string
}

export interface ImageBatchSubmitItem {
  custom_id: string
  prompt: string
  n: number
  size?: string
  metadata?: Record<string, unknown>
}

export interface ImageBatchSubmitRequest {
  provider: string
  model: string
  size?: string
  items: ImageBatchSubmitItem[]
  metadata?: Record<string, unknown>
}

const gatewayEnv = import.meta.env as unknown as Record<string, string | undefined>

const gatewayClient = axios.create({
  baseURL: gatewayEnv.VITE_GATEWAY_BASE_URL || '',
  timeout: 120000,
  headers: { 'Content-Type': 'application/json' },
})

const authHeaders = (apiKey: string, extra?: Record<string, string>) => ({
  'x-api-key': apiKey,
  ...extra,
})

export async function listImageBatchModels(apiKey: string): Promise<ImageBatchModel[]> {
  const { data } = await gatewayClient.get<{ data: ImageBatchModel[] }>(
    '/v1/images/batches/models',
    { headers: authHeaders(apiKey) },
  )
  return data.data || []
}

export async function listImageBatches(apiKey: string, limit = 50): Promise<ImageBatchJob[]> {
  const { data } = await gatewayClient.get<{ data: ImageBatchJob[] }>('/v1/images/batches', {
    headers: authHeaders(apiKey),
    params: { limit },
  })
  return data.data || []
}

export async function submitImageBatch(
  apiKey: string,
  request: ImageBatchSubmitRequest,
  idempotencyKey: string,
): Promise<ImageBatchJob> {
  const { data } = await gatewayClient.post<ImageBatchJob>('/v1/images/batches', request, {
    headers: authHeaders(apiKey, { 'Idempotency-Key': idempotencyKey }),
  })
  return data
}

export async function getImageBatch(apiKey: string, id: string): Promise<ImageBatchJob> {
  const { data } = await gatewayClient.get<ImageBatchJob>(`/v1/images/batches/${id}`, {
    headers: authHeaders(apiKey),
  })
  return data
}

export async function listImageBatchItems(apiKey: string, id: string): Promise<ImageBatchItem[]> {
  const { data } = await gatewayClient.get<{ data: ImageBatchItem[] }>(
    `/v1/images/batches/${id}/items`,
    { headers: authHeaders(apiKey), params: { limit: 1000 } },
  )
  return data.data || []
}

export async function cancelImageBatch(apiKey: string, id: string): Promise<ImageBatchJob> {
  const { data } = await gatewayClient.post<ImageBatchJob>(
    `/v1/images/batches/${id}/cancel`,
    {},
    { headers: authHeaders(apiKey) },
  )
  return data
}

export async function deleteImageBatch(apiKey: string, id: string): Promise<void> {
  await gatewayClient.delete(`/v1/images/batches/${id}`, { headers: authHeaders(apiKey) })
}

export async function deleteImageBatchOutputs(apiKey: string, id: string): Promise<void> {
  await gatewayClient.delete(`/v1/images/batches/${id}/outputs`, { headers: authHeaders(apiKey) })
}

export async function downloadImageBatch(apiKey: string, id: string): Promise<Blob> {
  const { data } = await gatewayClient.get(`/v1/images/batches/${id}/download`, {
    headers: authHeaders(apiKey),
    responseType: 'blob',
  })
  return data
}

export async function downloadImageBatchItem(
  apiKey: string,
  id: string,
  customID: string,
): Promise<Blob> {
  const { data } = await gatewayClient.get(
    `/v1/images/batches/${id}/items/${encodeURIComponent(customID)}/content`,
    { headers: authHeaders(apiKey), responseType: 'blob' },
  )
  return data
}
