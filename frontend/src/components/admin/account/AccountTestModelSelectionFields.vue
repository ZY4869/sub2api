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

    <Select
      v-if="modelInputMode === 'catalog'"
      :model-value="selectedModelKey"
      :options="availableModelOptions"
      :disabled="loadingModels || disabled"
      searchable
      value-key="key"
      label-key="display_name"
      :placeholder="loadingModels ? `${t('common.loading')}...` : t('admin.accounts.selectTestModel')"
      @update:model-value="emit('update:selectedModelKey', String($event || ''))"
    >
      <template #selected="{ option }">
        <div v-if="option" class="min-w-0">
          <div class="flex items-center gap-2">
            <span class="truncate font-medium text-gray-900 dark:text-white">
              {{ displayModelTitle(option) }}
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
          <div class="truncate text-xs text-gray-500 dark:text-gray-400">
            {{ displayModelIdentifier(option) }}
          </div>
        </div>
        <span v-else>
          {{ loadingModels ? `${t('common.loading')}...` : t('admin.accounts.selectTestModel') }}
        </span>
      </template>

      <template #option="{ option, selected }">
        <div class="flex min-w-0 flex-1 items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span class="truncate font-medium text-gray-900 dark:text-white">
                {{ displayModelTitle(option) }}
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
            <div class="truncate text-xs text-gray-500 dark:text-gray-400">
              {{ displayModelIdentifier(option) }}
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
      v-if="modelInputMode === 'catalog' && !loadingModels && availableModelOptions.length === 0 && emptyHint"
      class="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
    >
      {{ emptyHint }}
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
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import { Icon } from '@/components/icons'
import type { ClaudeModel } from '@/types'
import { resolveGatewayProtocolLabel } from '@/utils/accountProtocolGateway'
import { buildAccountTestModelOptionKeyFromModel } from '@/utils/accountTestModelOptions'
import { buildProviderDisplayName } from '@/utils/providerLabels'

type ManualSourceProtocol = 'openai' | 'anthropic' | 'gemini' | ''

type AccountTestModelOption = ClaudeModel & {
  key: string
  description: string
  [key: string]: unknown
}

const props = withDefaults(defineProps<{
  availableModels: ClaudeModel[]
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
  display_name?: unknown
  provider?: unknown
  provider_label?: unknown
  status?: unknown
}

const getModelStringField = (model: ModelDescriptor | null | undefined, key: keyof ModelDescriptor) =>
  typeof model?.[key] === 'string' ? model[key].trim() : ''

const displayModelIdentifier = (model: ModelDescriptor | null | undefined) => {
  const canonicalID = getModelStringField(model, 'canonical_id')
  if (canonicalID) {
    return canonicalID
  }
  return getModelStringField(model, 'id')
}

const displayModelTitle = (model: ModelDescriptor | null | undefined) =>
  buildProviderDisplayName({
    provider: getModelStringField(model, 'provider'),
    providerLabel: getModelStringField(model, 'provider_label'),
    displayName: getModelStringField(model, 'display_name'),
    fallbackId: getModelStringField(model, 'id')
  })

const availableModelOptions = computed<AccountTestModelOption[]>(() =>
  props.availableModels.map((model) => ({
    ...model,
    key: buildAccountTestModelOptionKeyFromModel(model),
    description: displayModelIdentifier(model)
  }))
)

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
</script>
