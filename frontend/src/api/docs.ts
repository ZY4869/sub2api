import { apiClient } from './client'

export interface ApiDocsResponse {
  content: string
}

export async function getAPIDocs(): Promise<ApiDocsResponse> {
  const { data } = await apiClient.get<ApiDocsResponse>('/docs/api')
  return data
}

const docsAPI = {
  getAPIDocs
}

export default docsAPI
