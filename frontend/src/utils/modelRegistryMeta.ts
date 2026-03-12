import type { ModelRegistryExposureTarget } from '@/api/admin/modelRegistry'

export const MODEL_REGISTRY_CUSTOM_PROVIDER = '__custom__'

export const MODEL_REGISTRY_PLATFORM_PRESETS = [
  'anthropic',
  'openai',
  'gemini',
  'antigravity',
  'sora'
] as const

export type ModelRegistryCapability =
  | 'text'
  | 'vision'
  | 'image_generation'
  | 'web_search'
  | 'audio_understanding'
  | 'video_understanding'
  | 'audio_generation'
  | 'video_generation'

export interface ModelRegistryCapabilityMeta {
  value: ModelRegistryCapability
  label: string
  icon: 'chatBubble' | 'eye' | 'photo' | 'globe' | 'speakerWave' | 'videoCamera' | 'speakerWaveFilled' | 'film'
}

export interface ModelRegistryExposureMeta {
  value: ModelRegistryExposureTarget
  shortLabel: string
  description: string
}

export const MODEL_REGISTRY_CAPABILITY_OPTIONS: readonly ModelRegistryCapabilityMeta[] = [
  { value: 'text', label: '文本', icon: 'chatBubble' },
  { value: 'vision', label: '识图', icon: 'eye' },
  { value: 'image_generation', label: '生图', icon: 'photo' },
  { value: 'web_search', label: '联网', icon: 'globe' },
  { value: 'audio_understanding', label: '识别音频', icon: 'speakerWave' },
  { value: 'video_understanding', label: '识别视频', icon: 'videoCamera' },
  { value: 'audio_generation', label: '生成音频', icon: 'speakerWaveFilled' },
  { value: 'video_generation', label: '生成视频', icon: 'film' }
] as const

export const MODEL_REGISTRY_CAPABILITY_ORDER = MODEL_REGISTRY_CAPABILITY_OPTIONS.map((item) => item.value)

export const MODEL_REGISTRY_EXPOSURE_OPTIONS: readonly ModelRegistryExposureMeta[] = [
  {
    value: 'whitelist',
    shortLabel: '账号白名单页',
    description: '账号白名单页（创建/编辑账号时显示）'
  },
  {
    value: 'use_key',
    shortLabel: 'Use Key 页面',
    description: 'Use Key 页面（用户创建 Key 时显示）'
  },
  {
    value: 'test',
    shortLabel: '账号测试页',
    description: '账号测试页（测试账号时显示）'
  },
  {
    value: 'runtime',
    shortLabel: '运行时模型列表',
    description: '对外模型列表 / API（程序运行时可见）'
  }
] as const

export function normalizeRegistryToken(value: string): string {
  return value.trim().toLowerCase()
}

export function normalizeProviderOptions(items: Array<{ provider?: string | null }>): string[] {
  return Array.from(
    new Set(
      items
        .map((item) => normalizeRegistryToken(item.provider || ''))
        .filter(Boolean)
    )
  ).sort((left, right) => left.localeCompare(right))
}

export function normalizeRegistryList(value: string): string[] {
  return Array.from(
    new Set(
      value
        .replace(/\r/g, '')
        .split('\n')
        .flatMap((item) => item.split(','))
        .map((item) => normalizeRegistryToken(item))
        .filter(Boolean)
    )
  )
}

export function formatRegistryList(items?: string[] | null): string {
  return Array.isArray(items) ? items.join(', ') : ''
}

export function normalizePlatformList(values: string[]): string[] {
  return Array.from(new Set(values.map(normalizeRegistryToken).filter(Boolean)))
}

export function normalizeCapabilityList(values: string[]): ModelRegistryCapability[] {
  const selected = new Set(values.map(normalizeRegistryToken))
  return MODEL_REGISTRY_CAPABILITY_ORDER.filter((item) => selected.has(item))
}

export function isKnownCapability(value: string): value is ModelRegistryCapability {
  return MODEL_REGISTRY_CAPABILITY_ORDER.includes(value as ModelRegistryCapability)
}

export function getCapabilityMeta(value: string): ModelRegistryCapabilityMeta | undefined {
  return MODEL_REGISTRY_CAPABILITY_OPTIONS.find((item) => item.value === value)
}

export function getExposureMeta(value: string): ModelRegistryExposureMeta | undefined {
  return MODEL_REGISTRY_EXPOSURE_OPTIONS.find((item) => item.value === value)
}

export function describeExposure(value: string): string {
  return getExposureMeta(value)?.description || value
}

export function describeExposureShort(value: string): string {
  return getExposureMeta(value)?.shortLabel || value
}

export function normalizeExposureTargets(values: string[]): ModelRegistryExposureTarget[] {
  const selected = new Set(values.map(normalizeRegistryToken))
  return MODEL_REGISTRY_EXPOSURE_OPTIONS
    .map((item) => item.value)
    .filter((item) => selected.has(item))
}

