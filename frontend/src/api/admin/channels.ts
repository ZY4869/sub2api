import { apiClient } from '../client'
import type { FetchOptions, PaginatedResponse } from '@/types'

export type BillingMode = 'token' | 'per_request' | 'image'
export type BillingModelSource = 'channel_mapped' | 'requested' | 'upstream'

export interface PricingInterval {
  id?: number
  pricing_id?: number
  min_tokens: number
  max_tokens: number | null
  tier_label: string
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_read_price: number | null
  per_request_price: number | null
  sort_order: number
}

export interface ChannelModelPricing {
  id?: number
  channel_id?: number
  platform: string
  models: string[]
  billing_mode: BillingMode
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_read_price: number | null
  image_output_price: number | null
  per_request_price: number | null
  intervals: PricingInterval[]
  sort_order?: number
}

export interface Channel {
  id: number
  name: string
  description?: string
  status: 'active' | 'disabled'
  restrict_models: boolean
  billing_model_source: BillingModelSource
  group_ids: number[]
  model_mapping: Record<string, Record<string, string>>
  model_pricing: ChannelModelPricing[]
  created_at: string
  updated_at: string
}

export interface CreateChannelRequest {
  name: string
  description?: string
  status?: 'active' | 'disabled'
  restrict_models?: boolean
  billing_model_source?: BillingModelSource
  group_ids: number[]
  model_mapping: Record<string, Record<string, string>>
  model_pricing: ChannelModelPricing[]
}

export interface UpdateChannelRequest {
  name?: string
  description?: string
  status?: 'active' | 'disabled'
  restrict_models?: boolean
  billing_model_source?: BillingModelSource
  group_ids?: number[]
  model_mapping?: Record<string, Record<string, string>>
  model_pricing?: ChannelModelPricing[]
}

export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    status?: 'active' | 'disabled'
    search?: string
  },
  options?: FetchOptions
): Promise<PaginatedResponse<Channel>> {
  const { data } = await apiClient.get<PaginatedResponse<Channel>>('/admin/channels', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    },
    signal: options?.signal
  })
  return data
}

export async function getById(id: number): Promise<Channel> {
  const { data } = await apiClient.get<Channel>(`/admin/channels/${id}`)
  return data
}

export async function create(payload: CreateChannelRequest): Promise<Channel> {
  const { data } = await apiClient.post<Channel>('/admin/channels', payload)
  return data
}

export async function update(id: number, payload: UpdateChannelRequest): Promise<Channel> {
  const { data } = await apiClient.put<Channel>(`/admin/channels/${id}`, payload)
  return data
}

export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/channels/${id}`)
}

const channelsAPI = {
  list,
  getById,
  create,
  update,
  remove
}

export default channelsAPI
