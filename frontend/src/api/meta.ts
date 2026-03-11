import { apiClient } from './client'

export interface ExchangeRateInfo {
  base: string
  quote: string
  rate: number
  date: string
  updated_at: string
  cached: boolean
}

export async function getUSDCNYExchangeRate(force = false): Promise<ExchangeRateInfo> {
  const requestConfig = force ? { params: { force: true } } : undefined
  const { data } = await apiClient.get<ExchangeRateInfo>('/meta/exchange-rate/usd-cny', requestConfig)
  return data
}

export const metaAPI = {
  getUSDCNYExchangeRate
}

export default metaAPI
