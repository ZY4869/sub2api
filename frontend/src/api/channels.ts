/**
 * Available Channels API endpoints (user-facing)
 */

import { apiClient } from './client'

export interface AvailableChannelGroup {
  id: number
  name: string
  platform: string
  subscription_type: string
  rate_multiplier: number
  is_exclusive: boolean
}

export interface AvailableChannelPricingInterval {
  min_tokens: number
  max_tokens: number | null
  tier_label?: string
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_read_price: number | null
  per_request_price: number | null
}

export interface AvailableChannelSupportedModelPricing {
  billing_mode: string
  input_price: number | null
  output_price: number | null
  cache_write_price: number | null
  cache_read_price: number | null
  image_output_price: number | null
  per_request_price: number | null
  intervals: AvailableChannelPricingInterval[]
}

export interface AvailableChannelSupportedModel {
  name: string
  platform: string
  pricing: AvailableChannelSupportedModelPricing | null
}

export interface AvailableChannelPlatformSection {
  platform: string
  groups: AvailableChannelGroup[]
  supported_models: AvailableChannelSupportedModel[]
}

export interface AvailableChannel {
  name: string
  description: string
  platforms: AvailableChannelPlatformSection[]
}

export async function getAvailableChannels(): Promise<AvailableChannel[]> {
  const { data } = await apiClient.get<AvailableChannel[]>('/channels/available')
  return data
}

export const channelsAPI = {
  getAvailableChannels
}

export default channelsAPI

