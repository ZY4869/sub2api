import { computed, reactive, toRefs, type Ref } from 'vue'
import type { AccountPlatform, AccountType } from '@/types'
import {
  buildBulkEditAccountPayload,
  canBulkEditAccountPreCheck,
  createDefaultBulkEditAccountFormState,
  hasBulkEditAccountFieldEnabled
} from '@/utils/bulkEditAccountForm'
import {
  buildModelMappingObject as buildModelMappingPayload,
  getModelsByPlatform,
  getPresetMappingsByPlatform
} from '@/composables/useModelWhitelist'

const supportedModelPlatforms: AccountPlatform[] = [
  'anthropic',
  'antigravity',
  'openai',
  'gemini'
]

function dedupeByKey<T>(items: T[], getKey: (item: T) => string): T[] {
  const seen = new Set<string>()
  return items.filter((item) => {
    const key = getKey(item)
    if (seen.has(key)) {
      return false
    }
    seen.add(key)
    return true
  })
}

export function useBulkEditAccountForm(options: {
  selectedPlatforms: Ref<AccountPlatform[]>
  selectedTypes: Ref<AccountType[]>
}) {
  const form = reactive(createDefaultBulkEditAccountFormState())

  const visiblePlatforms = computed(() => {
    const platforms = options.selectedPlatforms.value.filter((platform) =>
      supportedModelPlatforms.includes(platform)
    )
    return platforms.length > 0 ? platforms : supportedModelPlatforms
  })

  const allAnthropicOAuthOrSetupToken = computed(() => {
    return (
      options.selectedPlatforms.value.length === 1 &&
      options.selectedPlatforms.value[0] === 'anthropic' &&
      options.selectedTypes.value.every((type) => type === 'oauth' || type === 'setup-token')
    )
  })

  const allModels = computed(() => {
    const modelIds = visiblePlatforms.value.flatMap((platform) =>
      getModelsByPlatform(platform, 'whitelist')
    )
    return dedupeByKey(modelIds, (model) => model).map((model) => ({
      value: model,
      label: model
    }))
  })

  const presetMappings = computed(() => {
    const presets = visiblePlatforms.value.flatMap((platform) =>
      getPresetMappingsByPlatform(platform)
    )
    return dedupeByKey(presets, (preset) => `${preset.from}->${preset.to}`)
  })

  const hasAnyFieldEnabled = computed(() => hasBulkEditAccountFieldEnabled(form))

  const buildUpdatePayload = () => {
    return buildBulkEditAccountPayload(form, () =>
      buildModelMappingPayload(
        form.modelRestrictionMode,
        form.allowedModels,
        form.modelMappings
      )
    )
  }

  const canPreCheck = () => canBulkEditAccountPreCheck(form, options.selectedPlatforms.value)

  const resetFormState = () => {
    Object.assign(form, createDefaultBulkEditAccountFormState())
  }

  return {
    ...toRefs(form),
    allAnthropicOAuthOrSetupToken,
    allModels,
    presetMappings,
    hasAnyFieldEnabled,
    buildUpdatePayload,
    canPreCheck,
    resetFormState
  }
}
