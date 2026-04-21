<template>
  <div class="space-y-3">
    <div class="space-y-1.5">
      <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ t('admin.accounts.testModelInputModeLabel') }}
      </label>
      <div class="grid gap-2 sm:grid-cols-2">
        <button
          type="button"
          data-test="model-input-mode-catalog"
          :disabled="loadingModels || disabled"
          :class="buttonClass('catalog')"
          @click="emit('update:modelInputMode', 'catalog')"
        >
          <div class="text-sm font-semibold">
            {{ t('admin.accounts.testModelInputModes.catalog') }}
          </div>
          <p class="mt-1 text-xs leading-5 opacity-80">
            {{ t('admin.accounts.testModelInputModes.catalogHint') }}
          </p>
        </button>
        <button
          type="button"
          data-test="model-input-mode-manual"
          :disabled="disabled"
          :class="buttonClass('manual')"
          @click="emit('update:modelInputMode', 'manual')"
        >
          <div class="text-sm font-semibold">
            {{ t('admin.accounts.testModelInputModes.manual') }}
          </div>
          <p class="mt-1 text-xs leading-5 opacity-80">
            {{ t('admin.accounts.testModelInputModes.manualHint') }}
          </p>
        </button>
      </div>
    </div>

    <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
      {{
        modelInputMode === 'manual'
          ? t('admin.accounts.probeFinalize.manualModelId')
          : t('admin.accounts.selectTestModel')
      }}
    </label>

    <div
      v-if="modelInputMode === 'catalog' && (hasTextModels || hasImageModels)"
      class="flex flex-wrap items-center gap-2"
    >
      <button
        v-if="hasTextModels"
        type="button"
        class="rounded-full px-3 py-1.5 text-xs font-semibold transition"
        :class="quickFilterClass('text')"
        @click="activeQuickFilter = 'text'"
      >
        {{ t('admin.accounts.testModelQuickFilters.text') }}
      </button>
      <button
        v-if="hasImageModels"
        type="button"
        class="rounded-full px-3 py-1.5 text-xs font-semibold transition"
        :class="quickFilterClass('image')"
        @click="activeQuickFilter = 'image'"
      >
        {{ t('admin.accounts.testModelQuickFilters.image') }}
      </button>
      <button
        type="button"
        class="rounded-full px-3 py-1.5 text-xs font-semibold transition"
        :class="quickFilterClass('all')"
        @click="activeQuickFilter = 'all'"
      >
        {{ t('admin.accounts.testModelQuickFilters.all') }}
      </button>
    </div>

    <Select
      v-if="modelInputMode === 'catalog'"
      :model-value="selectedModelKey"
      :options="filteredAvailableModelOptions"
      :disabled="loadingModels || disabled"
      searchable
      value-key="key"
      label-key="display_name"
      :placeholder="loadingModels ? `${t('common.loading')}...` : t('admin.accounts.selectTestModel')"
      @update:model-value="emit('update:selectedModelKey', String($event || ''))"
    >
      <template #selected="{ option }">
        <div v-if="option" class="min-w-0">
          <div class="flex min-w-0 items-center gap-2">
            <ModelIcon
              :model="displayModelIconModel(option)"
              :provider="displayModelProvider(option)"
              :display-name="displayModelTitle(option)"
              size="16px"
            />
            <span class="truncate font-medium text-gray-900 dark:text-white">
              {{ displayModelTitle(option) }}
            </span>
          </div>
          <div class="mt-1 flex min-w-0 flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
            <span class="truncate" data-test="model-display-id">{{ displayModelIdentifier(option) }}</span>
            <span
              v-if="displayModelTargetRelation(option)"
              data-test="model-target-relation"
              class="truncate text-gray-500 dark:text-gray-400"
            >
              {{ displayModelTargetRelation(option) }}
            </span>
            <span
              v-if="availabilityBadgeLabel(option)"
              data-test="model-availability-badge"
              :class="stateBadgeClass('availability', option.availability_state)"
            >
              {{ availabilityBadgeLabel(option) }}
            </span>
            <span
              v-if="staleBadgeLabel(option)"
              data-test="model-stale-badge"
              :class="stateBadgeClass('stale', option.stale_state)"
            >
              {{ staleBadgeLabel(option) }}
            </span>
            <span
              v-if="option.source_protocol"
              class="inline-flex rounded-full bg-sky-100 px-2 py-0.5 text-[11px] font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300"
            >
              {{ protocolSourceLabel(option.source_protocol) }}
            </span>
            <span
              v-if="isDeprecatedModel(option)"
              class="inline-flex rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300"
            >
              {{ t('admin.models.registry.lifecycleLabels.deprecated') }}
            </span>
          </div>
        </div>
        <span v-else>
          {{ loadingModels ? `${t('common.loading')}...` : t('admin.accounts.selectTestModel') }}
        </span>
      </template>

      <template #option="{ option, selected }">
        <div class="flex min-w-0 flex-1 items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex min-w-0 items-center gap-2">
              <ModelIcon
                :model="displayModelIconModel(option)"
                :provider="displayModelProvider(option)"
                :display-name="displayModelTitle(option)"
                size="16px"
              />
              <span class="truncate font-medium text-gray-900 dark:text-white">
                {{ displayModelTitle(option) }}
              </span>
            </div>
            <div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
              <span class="truncate" data-test="model-display-id">{{ displayModelIdentifier(option) }}</span>
              <span
                v-if="displayModelTargetRelation(option)"
                data-test="model-target-relation"
                class="truncate text-gray-500 dark:text-gray-400"
              >
                {{ displayModelTargetRelation(option) }}
              </span>
              <span
                v-if="availabilityBadgeLabel(option)"
                data-test="model-availability-badge"
                :class="stateBadgeClass('availability', option.availability_state)"
              >
                {{ availabilityBadgeLabel(option) }}
              </span>
              <span
                v-if="staleBadgeLabel(option)"
                data-test="model-stale-badge"
                :class="stateBadgeClass('stale', option.stale_state)"
              >
                {{ staleBadgeLabel(option) }}
              </span>
              <span
                v-if="option.source_protocol"
                class="inline-flex rounded-full bg-sky-100 px-2 py-0.5 text-[11px] font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300"
              >
                {{ protocolSourceLabel(option.source_protocol) }}
              </span>
              <span
                v-if="isDeprecatedModel(option)"
                class="inline-flex rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300"
              >
                {{ t('admin.models.registry.lifecycleLabels.deprecated') }}
              </span>
            </div>
            <div
              v-if="option.replaced_by"
              class="truncate text-[11px] text-amber-600 dark:text-amber-300"
            >
              {{ t('admin.models.registry.replacedByHint', { model: option.replaced_by }) }}
            </div>
          </div>
          <Icon
            v-if="selected"
            name="check"
            size="sm"
            class="mt-0.5 shrink-0 text-primary-500"
            :stroke-width="2"
          />
        </div>
      </template>
    </Select>

    <p
      v-if="modelInputMode === 'catalog' && !loadingModels && filteredAvailableModelOptions.length === 0 && resolvedEmptyHint"
      class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
    >
      {{ resolvedEmptyHint }}
    </p>

    <div v-else-if="modelInputMode === 'manual'" class="grid gap-3 md:grid-cols-2">
      <label class="space-y-1.5" :class="showManualSourceProtocolField ? 'md:col-span-1' : 'md:col-span-2'">
        <input
          :value="manualModelId"
          data-test="manual-model-id"
          type="text"
          class="input"
          :disabled="disabled"
          :placeholder="t('admin.accounts.probeFinalize.manualModelIdPlaceholder')"
          @input="emit('update:manualModelId', ($event.target as HTMLInputElement).value)"
        />
      </label>

      <label v-if="showManualSourceProtocolField" class="space-y-1.5">
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ t('admin.accounts.probeFinalize.manualSourceProtocol') }}
        </span>
        <select
          :value="manualSourceProtocol"
          data-test="manual-source-protocol"
          class="input"
          :disabled="disabled"
          @change="emit('update:manualSourceProtocol', ($event.target as HTMLSelectElement).value as ManualSourceProtocol)"
        >
          <option value="">{{ t('admin.accounts.probeFinalize.manualSourceProtocolAuto') }}</option>
          <option value="openai">OpenAI</option>
          <option value="anthropic">Anthropic</option>
          <option value="gemini">Gemini</option>
        </select>
      </label>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { Icon } from '@/components/icons'
import type { AdminAccountModelOption } from '@/types'
import { resolveGatewayProtocolLabel } from '@/utils/accountProtocolGateway'
import { buildAccountTestModelOptionKeyFromModel } from '@/utils/accountTestModelOptions'

type ManualSourceProtocol = 'openai' | 'anthropic' | 'gemini' | ''

type AccountTestModelOption = AdminAccountModelOption & {
  key: string
  description: string
  [key: string]: unknown
}

type AccountTestQuickFilter = 'text' | 'image' | 'all'

const props = withDefaults(defineProps<{
  availableModels: AdminAccountModelOption[]
  loadingModels?: boolean
  disabled?: boolean
  modelInputMode: 'catalog' | 'manual'
  selectedModelKey: string
  manualModelId: string
  manualSourceProtocol?: ManualSourceProtocol
  showManualSourceProtocolField?: boolean
  emptyHint?: string
}>(), {
  loadingModels: false,
  disabled: false,
  manualSourceProtocol: '',
  showManualSourceProtocolField: false,
  emptyHint: ''
})

const emit = defineEmits<{
  (e: 'update:modelInputMode', value: 'catalog' | 'manual'): void
  (e: 'update:selectedModelKey', value: string): void
  (e: 'update:manualModelId', value: string): void
  (e: 'update:manualSourceProtocol', value: ManualSourceProtocol): void
}>()

const { t } = useI18n()

type ModelDescriptor = {
  id?: unknown
  canonical_id?: unknown
  target_model_id?: unknown
  display_name?: unknown
  provider?: unknown
  source_protocol?: unknown
  availability_state?: unknown
  stale_state?: unknown
  status?: unknown
  replaced_by?: unknown
  mode?: unknown
}

const getModelStringField = (model: ModelDescriptor | null | undefined, key: keyof ModelDescriptor) =>
  typeof model?.[key] === 'string' ? model[key].trim() : ''

const displayModelIdentifier = (model: ModelDescriptor | null | undefined) => {
  return getModelStringField(model, 'id') ||
    getModelStringField(model, 'target_model_id') ||
    getModelStringField(model, 'canonical_id')
}

const displayModelTitle = (model: ModelDescriptor | null | undefined) =>
  getModelStringField(model, 'display_name') ||
  getModelStringField(model, 'id') ||
  getModelStringField(model, 'target_model_id') ||
  getModelStringField(model, 'canonical_id')

const displayModelIconModel = (model: ModelDescriptor | null | undefined) =>
  getModelStringField(model, 'target_model_id') ||
  getModelStringField(model, 'id') ||
  getModelStringField(model, 'canonical_id') ||
  displayModelTitle(model)

const displayModelProvider = (model: ModelDescriptor | null | undefined) =>
  getModelStringField(model, 'provider')

const displayModelTargetRelation = (model: ModelDescriptor | null | undefined) => {
  const displayID = getModelStringField(model, 'id')
  const targetModelID = getModelStringField(model, 'target_model_id')
  if (!displayID || !targetModelID || displayID === targetModelID) {
    return ''
  }
  return t('admin.accounts.testModelTargetRelation', { target: targetModelID })
}

const availabilityBadgeLabel = (model: ModelDescriptor | null | undefined) => {
  switch (getModelStringField(model, 'availability_state')) {
    case 'verified':
      return t('admin.accounts.testModelAvailability.verified')
    case 'unavailable':
      return t('admin.accounts.testModelAvailability.unavailable')
    case 'unknown':
      return t('admin.accounts.testModelAvailability.unknown')
    default:
      return ''
  }
}

const staleBadgeLabel = (model: ModelDescriptor | null | undefined) => {
  switch (getModelStringField(model, 'stale_state')) {
    case 'fresh':
      return t('admin.accounts.testModelStale.fresh')
    case 'stale':
      return t('admin.accounts.testModelStale.stale')
    case 'unverified':
      return t('admin.accounts.testModelStale.unverified')
    default:
      return ''
  }
}

const stateBadgeClass = (kind: 'availability' | 'stale', value?: unknown) => {
  const normalized = String(value || '').trim()
  if (kind === 'availability') {
    switch (normalized) {
      case 'verified':
        return 'inline-flex rounded-full bg-emerald-100 px-2 py-0.5 text-[11px] font-medium text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
      case 'unavailable':
        return 'inline-flex rounded-full bg-rose-100 px-2 py-0.5 text-[11px] font-medium text-rose-700 dark:bg-rose-500/15 dark:text-rose-300'
      default:
        return 'inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-medium text-slate-700 dark:bg-slate-500/15 dark:text-slate-300'
    }
  }
  switch (normalized) {
    case 'fresh':
      return 'inline-flex rounded-full bg-cyan-100 px-2 py-0.5 text-[11px] font-medium text-cyan-700 dark:bg-cyan-500/15 dark:text-cyan-300'
    case 'stale':
      return 'inline-flex rounded-full bg-amber-100 px-2 py-0.5 text-[11px] font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'
    default:
      return 'inline-flex rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-medium text-slate-700 dark:bg-slate-500/15 dark:text-slate-300'
  }
}

const activeQuickFilter = ref<AccountTestQuickFilter>('all')

const normalizeTestModelMode = (model: Pick<AdminAccountModelOption, 'mode'> | null | undefined) => {
  const normalized = String(model?.mode || '').trim().toLowerCase()
  if (normalized === 'image') {
    return 'image'
  }
  if (normalized === 'video') {
    return 'video'
  }
  if (normalized === 'embedding') {
    return 'embedding'
  }
  if (normalized === 'other') {
    return 'other'
  }
  return 'text'
}

const availableModelOptions = computed<AccountTestModelOption[]>(() =>
  props.availableModels.map((model) => ({
    ...model,
    key: buildAccountTestModelOptionKeyFromModel(model),
    description: displayModelIdentifier(model)
  }))
)

const hasTextModels = computed(() =>
  props.availableModels.some((model) => normalizeTestModelMode(model) === 'text')
)

const hasImageModels = computed(() =>
  props.availableModels.some((model) => normalizeTestModelMode(model) === 'image')
)

const filteredAvailableModelOptions = computed<AccountTestModelOption[]>(() => {
  if (activeQuickFilter.value === 'all') {
    return availableModelOptions.value
  }
  return availableModelOptions.value.filter(
    (model) => normalizeTestModelMode(model) === activeQuickFilter.value
  )
})

const resolvedEmptyHint = computed(() => {
  if (filteredAvailableModelOptions.value.length > 0) {
    return props.emptyHint
  }
  if (activeQuickFilter.value === 'image') {
    return t('admin.accounts.testModelQuickFilters.imageEmpty')
  }
  if (activeQuickFilter.value === 'text') {
    return t('admin.accounts.testModelQuickFilters.textEmpty')
  }
  return props.emptyHint
})

const protocolSourceLabel = (sourceProtocol?: unknown) =>
  resolveGatewayProtocolLabel(sourceProtocol) || String(sourceProtocol || '').trim()

const isDeprecatedModel = (model: ModelDescriptor | null | undefined) => getModelStringField(model, 'status') === 'deprecated'

const buttonClass = (mode: 'catalog' | 'manual') => [
  'rounded-xl border px-4 py-3 text-left transition-all',
  props.modelInputMode === mode
    ? 'border-primary-500 bg-primary-50 text-primary-700 shadow-sm dark:border-primary-400 dark:bg-primary-500/10 dark:text-primary-200'
    : 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-200 dark:hover:border-primary-500/60',
  props.disabled ? 'cursor-not-allowed opacity-70' : ''
]

const quickFilterClass = (mode: AccountTestQuickFilter) => [
  activeQuickFilter.value === mode
    ? 'bg-primary-500 text-white shadow-sm'
    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-200 dark:hover:bg-dark-600',
  props.disabled || props.loadingModels ? 'cursor-not-allowed opacity-70' : ''
]

watch(
  () => props.availableModels,
  (models) => {
    const hasText = models.some((model) => normalizeTestModelMode(model) === 'text')
    const hasImage = models.some((model) => normalizeTestModelMode(model) === 'image')
    if (hasText) {
      activeQuickFilter.value = 'text'
      return
    }
    if (hasImage) {
      activeQuickFilter.value = 'image'
      return
    }
    activeQuickFilter.value = 'all'
  },
  { immediate: true }
)

watch(
  [filteredAvailableModelOptions, () => props.modelInputMode],
  ([options, modelInputMode]) => {
    if (modelInputMode !== 'catalog') {
      return
    }
    if (options.some((option) => option.key === props.selectedModelKey)) {
      return
    }
    emit('update:selectedModelKey', options[0]?.key || '')
  },
  { immediate: true }
)
</script>
