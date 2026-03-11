import { apiClient } from './client'

export interface ExchangeRateInfo {
  base: string
  quote: string
  rate: number
  date: string
  updated_at: string
  cached: boolean
}

export async function getUSDCNYExchangeRate(): Promise<ExchangeRateInfo> {
  const { data } = await apiClient.get<ExchangeRateInfo>('/meta/exchange-rate/usd-cny')
  return data
}

export const metaAPI = {
  getUSDCNYExchangeRate
}

export default metaAPI
