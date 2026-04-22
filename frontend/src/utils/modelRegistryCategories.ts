import type { ModelRegistryDetail } from '@/api/admin/modelRegistry'

export type ModelRegistryCategory = 'text' | 'image' | 'video' | 'audio' | 'other'

export const modelRegistryCategoryOrder: ModelRegistryCategory[] = ['text', 'image', 'video', 'audio', 'other']

export function getModelRegistryCategory(model: Pick<ModelRegistryDetail, 'capabilities' | 'modalities'>): ModelRegistryCategory {
  const capabilities = (model.capabilities || []).map((value) => String(value || '').trim().toLowerCase())
  const modalities = (model.modalities || []).map((value) => String(value || '').trim().toLowerCase())

  if (capabilities.includes('video_generation') || capabilities.includes('video_understanding')) {
    return 'video'
  }
  if (capabilities.includes('audio_generation') || capabilities.includes('audio_understanding')) {
    return 'audio'
  }
  if (capabilities.includes('image_generation') || capabilities.includes('image_generation_tool') || capabilities.includes('vision') || modalities.includes('image')) {
    return 'image'
  }
  if (capabilities.includes('text')) {
    return 'text'
  }
  return 'other'
}

export function groupModelRegistryModels(models: ModelRegistryDetail[]) {
  const groups = new Map<ModelRegistryCategory, ModelRegistryDetail[]>()
  for (const category of modelRegistryCategoryOrder) {
    groups.set(category, [])
  }
  for (const model of models) {
    groups.get(getModelRegistryCategory(model))?.push(model)
  }
  return modelRegistryCategoryOrder
    .map((category) => ({
      category,
      items: groups.get(category) || []
    }))
    .filter((group) => group.items.length > 0)
}
