/**
 * Public model plaza API.
 */

import { apiClient } from './client'
import type { AvailableChannelSupportedModelPricing } from './channels'

export type ModelPlazaModelPricing = AvailableChannelSupportedModelPricing

export interface ModelPlazaModel {
  display_model_id: string
  platform: string
  pricing: ModelPlazaModelPricing | null
  official_pricing: ModelPlazaModelPricing | null
}

export interface ModelPlazaGroup {
  id: number
  name: string
  description: string
  platform: string
  subscription_type: string
  rate_multiplier: number
  peak_rate_enabled: boolean
  peak_start: string
  peak_end: string
  peak_rate_multiplier: number
  is_exclusive: boolean
  image_rate_independent: boolean
  image_rate_multiplier: number
  image_price_1k: number | null
  image_price_2k: number | null
  image_price_4k: number | null
  web_search_price_per_call: number | null
  models: ModelPlazaModel[]
}

export interface ModelPlazaResponse {
  description: string
  groups: ModelPlazaGroup[]
}

export async function getModelPlaza(): Promise<ModelPlazaResponse> {
  const { data } = await apiClient.get<ModelPlazaResponse>('/model-plaza')
  return data
}

export const modelPlazaAPI = {
  getModelPlaza
}

export default modelPlazaAPI
