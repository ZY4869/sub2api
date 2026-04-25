/**
 * Channel Monitor API endpoints (user-facing)
 */

import { apiClient } from './client'

export type ChannelMonitorStatus = 'success' | 'degraded' | 'failure'

export interface ChannelMonitorTimelineItem {
  status: ChannelMonitorStatus | string
  latency_ms: number
  checked_at: string
}

export interface ChannelMonitorModelLastStatus {
  model_id: string
  status: ChannelMonitorStatus | string
  latency_ms: number
  checked_at?: string
  http_status?: number
}

export interface ChannelMonitorUserListItem {
  id: number
  name: string
  provider: string
  primary_model_id: string
  primary_last?: ChannelMonitorModelLastStatus
  primary_availability_7d?: number
  timeline: ChannelMonitorTimelineItem[]
  additional_last: ChannelMonitorModelLastStatus[]
}

export interface ChannelMonitorUserModelDetail {
  model_id: string
  last?: ChannelMonitorModelLastStatus
  availability_7d?: number
  availability_15d?: number
  availability_30d?: number
}

export interface ChannelMonitorUserDetail {
  id: number
  name: string
  provider: string
  primary_model_id: string
  models: ChannelMonitorUserModelDetail[]
}

export async function getChannelMonitors(): Promise<ChannelMonitorUserListItem[]> {
  const { data } = await apiClient.get<ChannelMonitorUserListItem[]>('/channel-monitors')
  return data
}

export async function getChannelMonitorStatus(id: number): Promise<ChannelMonitorUserDetail> {
  const { data } = await apiClient.get<ChannelMonitorUserDetail>(`/channel-monitors/${id}/status`)
  return data
}

export const channelMonitorsAPI = {
  getChannelMonitors,
  getChannelMonitorStatus
}

export default channelMonitorsAPI

