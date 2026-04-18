import { apiClient } from '../client'

export interface AdminApiDocsResponse {
  effective_content: string
  default_content: string
  has_override: boolean
}

function buildPageParams(pageId?: string) {
  const normalizedPageId = String(pageId || '').trim()
  return normalizedPageId ? { page_id: normalizedPageId } : undefined
}

export async function getAPIDocs(pageId?: string): Promise<AdminApiDocsResponse> {
  const { data } = await apiClient.get<AdminApiDocsResponse>('/admin/docs/api', {
    params: buildPageParams(pageId)
  })
  return data
}

export async function updateAPIDocs(content: string, pageId?: string): Promise<AdminApiDocsResponse> {
  const { data } = await apiClient.put<AdminApiDocsResponse>(
    '/admin/docs/api',
    { content },
    { params: buildPageParams(pageId) }
  )
  return data
}

export async function clearAPIDocsOverride(pageId?: string): Promise<AdminApiDocsResponse> {
  const { data } = await apiClient.delete<AdminApiDocsResponse>('/admin/docs/api/override', {
    params: buildPageParams(pageId)
  })
  return data
}

const docsAPI = {
  getAPIDocs,
  updateAPIDocs,
  clearAPIDocsOverride
}

export default docsAPI
