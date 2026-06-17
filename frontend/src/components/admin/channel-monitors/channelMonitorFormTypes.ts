import type {
  ChannelMonitorBodyOverrideMode,
  ChannelMonitorModelProbeStrategy,
  ChannelMonitorOpenAIAPIMode,
  ChannelMonitorProbeMode,
  ChannelMonitorRequestProtocol
} from '@/api/admin/channelMonitors'

export interface ChannelMonitorFormState {
  name: string
  provider: string
  probe_mode: ChannelMonitorProbeMode
  request_protocol: ChannelMonitorRequestProtocol
  endpoint: string
  interval_seconds: number
  jitter_seconds: number
  enabled: boolean
  account_ids: number[]
  primary_model_id: string
  template_id: number | null
  model_probe_strategy: ChannelMonitorModelProbeStrategy
  test_prompt_template: string
  body_override_mode: ChannelMonitorBodyOverrideMode
  openai_api_mode: ChannelMonitorOpenAIAPIMode
  save_as_template: boolean
  template_name: string
}

export interface MonitorModelSelectOption {
  value: string
  label: string
  provider?: string
  provider_label?: string
  display_name?: string
  source_protocol?: string
  disabled?: boolean
  [key: string]: unknown
}
