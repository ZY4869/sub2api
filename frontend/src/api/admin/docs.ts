import { apiClient } from '../client'

export interface AdminApiDocsResponse {
  effective_content: string
  default_content: string
  has_override: boolean
}

export async function getAPIDocs(): Promise<AdminApiDocsResponse> {
  const { data } = await apiClient.get<AdminApiDocsResponse>('/admin/docs/api')
  return data
}

export async function updateAPIDocs(content: string): Promise<AdminApiDocsResponse> {
  const { data } = await apiClient.put<AdminApiDocsResponse>('/admin/docs/api', { content })
  return data
}

export async function clearAPIDocsOverride(): Promise<AdminApiDocsResponse> {
  const { data } = await apiClient.delete<AdminApiDocsResponse>('/admin/docs/api/override')
  return data
}

const docsAPI = {
  getAPIDocs,
  updateAPIDocs,
  clearAPIDocsOverride
}

export default docsAPI
