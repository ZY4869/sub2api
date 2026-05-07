import { apiClient } from './client'
import type { CustomPageContent } from '@/types'

export async function getCustomPage(slug: string): Promise<CustomPageContent> {
  const { data } = await apiClient.get<CustomPageContent>(`/pages/${encodeURIComponent(slug)}`)
  return data
}

export const pagesAPI = {
  getCustomPage
}

export default pagesAPI
