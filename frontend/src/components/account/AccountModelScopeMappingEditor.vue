<template>
  <div>
    <div
      v-if="showActualModelLock"
      class="mb-3 flex items-center justify-between gap-3 rounded-lg border border-purple-200 bg-white/70 px-3 py-3 dark:border-purple-900/40 dark:bg-dark-800/60"
    >
      <div class="min-w-0">
        <div class="text-sm font-medium text-purple-900 dark:text-purple-100">
          {{ t('admin.accounts.actualModelLockLabel') }}
        </div>
        <p class="mt-1 text-xs text-purple-700 dark:text-purple-300">
          {{
            actualModelLocked
              ? t('admin.accounts.actualModelLockHintLocked')
              : t('admin.accounts.actualModelLockHintUnlocked')
          }}
        </p>
      </div>
      <button
        type="button"
        :class="[
          'relative inline-flex h-6 w-11 flex-shrink-0 rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
          actualModelLocked ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
        ]"
        @click="actualModelLocked = !actualModelLocked"
      >
        <span
          :class="[
            'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            actualModelLocked ? 'translate-x-5' : 'translate-x-0'
          ]"
        />
      </button>
    </div>

    <div class="mb-3 rounded-lg bg-purple-50 p-3 dark:bg-purple-900/20">
      <p class="text-xs text-purple-700 dark:text-purple-400">
        {{ t('admin.accounts.mapRequestModels') }}
      </p>
      <p class="mt-1 text-xs text-purple-700 dark:text-purple-400">
        {{ t('admin.accounts.modelMappingWildcardHint') }}
      </p>
      <div class="mt-2 flex flex-wrap gap-2">
        <code class="rounded bg-purple-100 px-2 py-1 text-[11px] text-purple-800 dark:bg-purple-900/30 dark:text-purple-200">
          gpt-5* -> gpt-5.4
        </code>
        <code class="rounded bg-purple-100 px-2 py-1 text-[11px] text-purple-800 dark:bg-purple-900/30 dark:text-purple-200">
          claude-sonnet-4* -> claude-sonnet-4.5
        </code>
        <code class="rounded bg-purple-100 px-2 py-1 text-[11px] text-purple-800 dark:bg-purple-900/30 dark:text-purple-200">
          gemini-3* -> gemini-3.1-flash
        </code>
      </div>
    </div>

    <div
      v-if="showActualModelLock && actualModelLocked && lockedRows.length === 0"
      class="mb-3 rounded-lg border border-dashed border-purple-300 bg-purple-50/70 px-4 py-3 text-sm text-purple-700 dark:border-purple-900/40 dark:bg-purple-950/20 dark:text-purple-300"
    >
      {{ t('admin.accounts.modelMappingSelectionDrivenHint') }}
    </div>

    <div v-if="visibleRows.length > 0" class="mb-3 space-y-2">
      <div
        v-for="(mapping, index) in visibleRows"
        :key="resolveRowKey(mapping, index)"
        class="space-y-1"
      >
        <div class="flex items-center gap-2">
          <input
            v-if="actualModelLocked"
            :value="mapping.from"
            type="text"
            :class="[
              'input flex-1',
              hasWildcardSourceError(mapping.from) ? 'border-red-500 dark:border-red-500' : ''
            ]"
            :placeholder="t('admin.accounts.requestModel')"
            @input="updateLockedAlias(mapping.to, ($event.target as HTMLInputElement).value)"
          />
          <input
            v-else
            v-model="mapping.from"
            type="text"
            :class="[
              'input flex-1',
              hasWildcardSourceError(mapping.from) ? 'border-red-500 dark:border-red-500' : ''
            ]"
            :placeholder="t('admin.accounts.requestModel')"
          />
          <svg class="h-4 w-4 flex-shrink-0 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14 5l7 7m0 0l-7 7m7-7H3" />
          </svg>
          <input
            v-if="actualModelLocked"
            :value="mapping.to"
            type="text"
            :class="[
              'input flex-1 cursor-not-allowed bg-gray-100/90 text-gray-500 dark:bg-dark-900/60 dark:text-gray-400',
              hasWildcardTargetError(mapping.to) ? 'border-red-500 dark:border-red-500' : ''
            ]"
            :placeholder="t('admin.accounts.actualModel')"
            readonly
          />
          <input
            v-else
            v-model="mapping.to"
            type="text"
            :class="[
              'input flex-1',
              hasWildcardTargetError(mapping.to) ? 'border-red-500 dark:border-red-500' : ''
            ]"
            :placeholder="t('admin.accounts.actualModel')"
          />
          <button
            type="button"
            class="rounded-lg p-2 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
            @click="actualModelLocked ? removeSelectedModel(mapping.to) : emit('remove-mapping', index)"
          >
            <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
              />
            </svg>
          </button>
        </div>

        <p v-if="hasWildcardSourceError(mapping.from)" class="text-xs text-red-500">
          {{ t('admin.accounts.wildcardOnlyAtEnd') }}
        </p>
        <p v-if="hasWildcardTargetError(mapping.to)" class="text-xs text-red-500">
          {{ t('admin.accounts.targetNoWildcard') }}
        </p>
      </div>
    </div>

    <button
      v-if="!actualModelLocked"
      type="button"
      class="mb-3 w-full rounded-lg border-2 border-dashed border-gray-300 px-4 py-2 text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-700 dark:border-dark-500 dark:text-gray-400 dark:hover:border-dark-400 dark:hover:text-gray-300"
      @click="emit('add-mapping')"
    >
      + {{ t('admin.accounts.addMapping') }}
    </button>

    <div v-if="!actualModelLocked" class="flex flex-wrap gap-2">
      <button
        v-for="preset in presetMappings"
        :key="`${preset.label}-${preset.from}-${preset.to}`"
        type="button"
        class="rounded-lg px-3 py-1 text-xs transition-colors"
        :class="preset.color"
        @click="emit('add-preset', { from: preset.from, to: preset.to })"
      >
        + {{ preset.label }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelRegistryPreset } from '@/generated/modelRegistry'
import type { ModelMapping } from '@/utils/accountFormShared'
import { isValidWildcardPattern } from '@/composables/useModelWhitelist'

interface Props {
  allowedModels: string[]
  modelMappings: ModelMapping[]
  presetMappings: ModelRegistryPreset[]
  getMappingKey: (mapping: ModelMapping) => string
  showActualModelLock?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showActualModelLock: false
})

const emit = defineEmits<{
  'add-mapping': []
  'remove-mapping': [index: number]
  'add-preset': [payload: { from: string; to: string }]
  'update:modelMappings': [value: ModelMapping[]]
  'update:allowedModels': [value: string[]]
}>()
const actualModelLocked = defineModel<boolean>('actualModelLocked', { default: true })

const { t } = useI18n()

const normalizeMappingRows = (rows: ModelMapping[]) =>
  rows.map((row) => ({
    from: row.from.trim(),
    to: row.to.trim()
  }))

const explicitMappings = computed(() =>
  normalizeMappingRows(props.modelMappings).filter((row) => Boolean(row.from) && Boolean(row.to) && row.from !== row.to)
)

const lockedRows = computed(() =>
  props.allowedModels
    .map((modelId) => modelId.trim())
    .filter(Boolean)
    .map((modelId) => ({
      from: explicitMappings.value.find((row) => row.to === modelId)?.from ?? modelId,
      to: modelId
    }))
)

const visibleRows = computed(() => (actualModelLocked.value ? lockedRows.value : props.modelMappings))

const hasWildcardSourceError = (value: string) => Boolean(value.trim()) && !isValidWildcardPattern(value.trim())
const hasWildcardTargetError = (value: string) => value.includes('*')

function resolveRowKey(mapping: ModelMapping, index: number) {
  return actualModelLocked.value ? `selected-${mapping.to || index}` : props.getMappingKey(mapping)
}

function updateLockedAlias(targetModel: string, value: string) {
  const nextAlias = value.trim()
  const nextMappings = explicitMappings.value.filter((row) => row.to !== targetModel)
  if (nextAlias && nextAlias !== targetModel) {
    nextMappings.push({ from: nextAlias, to: targetModel })
  }
  emit('update:modelMappings', nextMappings)
}

function removeSelectedModel(targetModel: string) {
  emit(
    'update:allowedModels',
    props.allowedModels.map((item) => item.trim()).filter((item) => item && item !== targetModel)
  )
  emit(
    'update:modelMappings',
    explicitMappings.value.filter((row) => row.to !== targetModel)
  )
}
</script>
