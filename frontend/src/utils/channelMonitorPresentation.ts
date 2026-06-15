import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import type { Component } from 'vue'

export type ChannelMonitorProbeMode = 'direct' | 'account_pool'
export type ChannelMonitorRequestProtocol = 'openai' | 'anthropic' | 'gemini'
export type ChannelMonitorModelProbeStrategy = 'primary_only' | 'all_selected'
export type ChannelMonitorBodyOverrideMode = 'off' | 'merge' | 'replace'
export type ChannelMonitorStatus = 'success' | 'degraded' | 'failure'

export interface SelectOptionItem {
  value: string | number | null
  label: string
  description?: string
  icon?: Component | string
  iconProps?: Record<string, any>
  [key: string]: unknown
}

const PROVIDER_OPTIONS: SelectOptionItem[] = [
  { value: 'openai', label: 'OpenAI', icon: ModelPlatformIcon, iconProps: { platform: 'openai' } },
  { value: 'anthropic', label: 'Anthropic', icon: ModelPlatformIcon, iconProps: { platform: 'anthropic' } },
  { value: 'gemini', label: 'Gemini / Google', icon: ModelPlatformIcon, iconProps: { platform: 'gemini' } },
  { value: 'google', label: 'Google', icon: ModelPlatformIcon, iconProps: { platform: 'gemini' } },
  { value: 'grok', label: 'Grok / xAI', icon: ModelPlatformIcon, iconProps: { platform: 'grok' } },
  { value: 'xai', label: 'xAI', icon: ModelPlatformIcon, iconProps: { platform: 'grok' } },
  { value: 'antigravity', label: 'Antigravity', icon: ModelPlatformIcon, iconProps: { platform: 'antigravity' } },
  { value: 'deepseek', label: 'DeepSeek', icon: ModelPlatformIcon, iconProps: { platform: 'deepseek' } },
  { value: 'openrouter', label: 'OpenRouter', icon: ModelPlatformIcon, iconProps: { platform: 'openrouter' } },
  { value: 'qwen', label: 'Qwen / 阿里', icon: ModelPlatformIcon, iconProps: { platform: 'qwen' } },
  { value: 'alibaba', label: '阿里', icon: ModelPlatformIcon, iconProps: { platform: 'alibaba' } },
  { value: 'doubao', label: 'Doubao / 字节', icon: ModelPlatformIcon, iconProps: { platform: 'doubao' } },
  { value: 'bytedance', label: '字节', icon: ModelPlatformIcon, iconProps: { platform: 'doubao' } },
  { value: 'moonshot', label: 'Moonshot / Kimi', icon: ModelPlatformIcon, iconProps: { platform: 'moonshot' } },
  { value: 'kimi', label: 'Kimi', icon: ModelPlatformIcon, iconProps: { platform: 'moonshot' } },
  { value: 'zhipu', label: 'Zhipu / GLM', icon: ModelPlatformIcon, iconProps: { platform: 'zhipu' } },
  { value: 'mistral', label: 'Mistral', icon: ModelPlatformIcon, iconProps: { platform: 'mistral' } },
  { value: 'cohere', label: 'Cohere', icon: ModelPlatformIcon, iconProps: { platform: 'cohere' } },
  { value: 'perplexity', label: 'Perplexity', icon: ModelPlatformIcon, iconProps: { platform: 'perplexity' } }
]

const REQUEST_PROTOCOL_OPTIONS: SelectOptionItem[] = [
  { value: 'openai', label: 'OpenAI 兼容', description: 'chat/completions 或 responses' },
  { value: 'anthropic', label: 'Anthropic 兼容', description: 'messages' },
  { value: 'gemini', label: 'Gemini 兼容', description: 'v1beta/models' }
]

const PROBE_MODE_OPTIONS: SelectOptionItem[] = [
  { value: 'direct', label: '直连地址' },
  { value: 'account_pool', label: '已有账号' }
]

const MODEL_PROBE_STRATEGY_OPTIONS: SelectOptionItem[] = [
  { value: 'primary_only', label: '仅主模型', description: '最小探测代表账号可用' },
  { value: 'all_selected', label: '全部选中模型', description: '对主模型和附加模型都探测' }
]

const BODY_OVERRIDE_MODE_OPTIONS: SelectOptionItem[] = [
  { value: 'off', label: '关闭' },
  { value: 'merge', label: '合并' },
  { value: 'replace', label: '替换' }
]

const STATUS_LABELS: Record<ChannelMonitorStatus, string> = {
  success: '成功',
  degraded: '降级',
  failure: '失败'
}

const BODY_OVERRIDE_LABELS: Record<ChannelMonitorBodyOverrideMode, string> = {
  off: '关闭',
  merge: '合并',
  replace: '替换'
}

const PROBE_MODE_LABELS: Record<ChannelMonitorProbeMode, string> = {
  direct: '直连地址',
  account_pool: '已有账号'
}

const REQUEST_PROTOCOL_LABELS: Record<ChannelMonitorRequestProtocol, string> = {
  openai: 'OpenAI 兼容',
  anthropic: 'Anthropic 兼容',
  gemini: 'Gemini 兼容'
}

const MODEL_PROBE_STRATEGY_LABELS: Record<ChannelMonitorModelProbeStrategy, string> = {
  primary_only: '仅主模型',
  all_selected: '全部选中模型'
}

function normalize(value?: string | null): string {
  return String(value || '').trim().toLowerCase()
}

export function getChannelMonitorProviderOptions(): SelectOptionItem[] {
  return PROVIDER_OPTIONS
}

export function getChannelMonitorRequestProtocolOptions(): SelectOptionItem[] {
  return REQUEST_PROTOCOL_OPTIONS
}

export function getChannelMonitorProbeModeOptions(): SelectOptionItem[] {
  return PROBE_MODE_OPTIONS
}

export function getChannelMonitorModelProbeStrategyOptions(): SelectOptionItem[] {
  return MODEL_PROBE_STRATEGY_OPTIONS
}

export function getChannelMonitorBodyOverrideModeOptions(): SelectOptionItem[] {
  return BODY_OVERRIDE_MODE_OPTIONS
}

export function getChannelMonitorProviderLabel(provider?: string | null): string {
  const normalized = normalize(provider)
  const option = PROVIDER_OPTIONS.find((item) => item.value === normalized)
  return option?.label || provider || '-'
}

export function getChannelMonitorRequestProtocolLabel(protocol?: string | null): string {
  return REQUEST_PROTOCOL_LABELS[normalize(protocol) as ChannelMonitorRequestProtocol] || protocol || '-'
}

export function getChannelMonitorProbeModeLabel(mode?: string | null): string {
  return PROBE_MODE_LABELS[normalize(mode) as ChannelMonitorProbeMode] || mode || '-'
}

export function getChannelMonitorModelProbeStrategyLabel(strategy?: string | null): string {
  return MODEL_PROBE_STRATEGY_LABELS[normalize(strategy) as ChannelMonitorModelProbeStrategy] || strategy || '-'
}

export function getChannelMonitorBodyOverrideModeLabel(mode?: string | null): string {
  return BODY_OVERRIDE_LABELS[normalize(mode) as ChannelMonitorBodyOverrideMode] || mode || '-'
}

export function getChannelMonitorStatusLabel(status?: string | null): string {
  const normalized = normalize(status) as ChannelMonitorStatus
  return STATUS_LABELS[normalized] || status || '-'
}

export function getChannelMonitorStatusClass(status?: string | null): string {
  const normalized = normalize(status)
  if (normalized === 'success') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
  if (normalized === 'degraded') return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300'
  if (normalized === 'failure') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
  return 'bg-gray-100 text-gray-700 dark:bg-gray-900/30 dark:text-gray-400'
}

export function getChannelMonitorProviderIcon(provider?: string | null): Component {
  const option = PROVIDER_OPTIONS.find((item) => item.value === normalize(provider))
  return (option?.icon as Component) || ModelPlatformIcon
}

export function getChannelMonitorProviderIconProps(provider?: string | null): Record<string, any> {
  const option = PROVIDER_OPTIONS.find((item) => item.value === normalize(provider))
  return option?.iconProps || { platform: provider }
}

export function getChannelMonitorModelIconProps(provider?: string | null, modelId?: string | null): Record<string, any> {
  return {
    model: String(modelId || ''),
    provider: String(provider || ''),
    displayName: String(modelId || '')
  }
}

export function getChannelMonitorProviderSelectOptions(): SelectOptionItem[] {
  return PROVIDER_OPTIONS
}
