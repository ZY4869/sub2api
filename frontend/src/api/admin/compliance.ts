import { apiClient } from '../client'

export interface AdminComplianceStatus {
  enabled: boolean
  required: boolean
  document_version: string
  document_hash: string
  acknowledged_at?: string | null
}

export async function getStatus(): Promise<AdminComplianceStatus> {
  const { data } = await apiClient.get<AdminComplianceStatus>('/admin/compliance/status')
  return data
}

export async function acknowledge(): Promise<AdminComplianceStatus> {
  const { data } = await apiClient.post<AdminComplianceStatus>('/admin/compliance/acknowledge')
  return data
}

export const complianceAPI = {
  getStatus,
  acknowledge,
}

export default complianceAPI
