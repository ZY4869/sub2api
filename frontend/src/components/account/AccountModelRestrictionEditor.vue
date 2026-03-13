<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <label class="input-label">{{ t('admin.accounts.modelRestriction') }}</label>

    <div
      v-if="disabled"
      class="mb-3 rounded-lg bg-amber-50 p-3 dark:bg-amber-900/20"
    >
      <p class="text-xs text-amber-700 dark:text-amber-400">
        {{ t('admin.accounts.openai.modelRestrictionDisabledByPassthrough') }}
      </p>
    </div>

    <template v-else>
      <div class="mb-4 flex gap-2">
        <button
          type="button"
          @click="emit('update:mode', 'whitelist')"
          :class="[
            'flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            mode === 'whitelist'
              ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
          ]"
        >
          <svg
            v-if="showModeIcons"
            class="mr-1.5 inline h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          {{ t('admin.accounts.modelWhitelist') }}
        </button>
        <button
          type="button"
          @click="emit('update:mode', 'mapping')"
          :class="[
            'flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            mode === 'mapping'
              ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
          ]"
        >
          <svg
            v-if="showModeIcons"
            class="mr-1.5 inline h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
            />
          </svg>
          {{ t('admin.accounts.modelMapping') }}
        </button>
      </div>

      <div v-if="mode === 'whitelist'">
        <ModelWhitelistSelector v-model="selectedAllowedModels" :platform="platform" />
        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.selectedModels', { count: allowedModels.length }) }}
          <span v-if="allowedModels.length === 0">
            {{ t('admin.accounts.supportsAllModels') }}
          </span>
        </p>
      </div>

      <div v-else>
        <div class="mb-3 rounded-lg bg-purple-50 p-3 dark:bg-purple-900/20">
          <p class="text-xs text-purple-700 dark:text-purple-400">
            <svg
              v-if="showModeIcons"
              class="mr-1 inline h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
            {{ t('admin.accounts.mapRequestModels') }}
          </p>
        </div>

        <div v-if="modelMappings.length > 0" class="mb-3 space-y-2">
          <div
            v-for="(mapping, index) in modelMappings"
            :key="getMappingKey(mapping)"
            class="flex items-center gap-2"
          >
            <input
              v-model="mapping.from"
              type="text"
              class="input flex-1"
              :placeholder="t('admin.accounts.requestModel')"
            />
            <svg
              class="h-4 w-4 flex-shrink-0 text-gray-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M14 5l7 7m0 0l-7 7m7-7H3"
              />
            </svg>
            <input
              v-model="mapping.to"
              type="text"
              class="input flex-1"
              :placeholder="t('admin.accounts.actualModel')"
            />
            <button
              type="button"
              @click="emit('remove-mapping', index)"
              class="rounded-lg p-2 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
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
        </div>

        <button
          type="button"
          @click="emit('add-mapping')"
          class="mb-3 w-full rounded-lg border-2 border-dashed border-gray-300 px-4 py-2 text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-700 dark:border-dark-500 dark:text-gray-400 dark:hover:border-dark-400 dark:hover:text-gray-300"
        >
          <svg
            v-if="showModeIcons"
            class="mr-1 inline h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M12 4v16m8-8H4"
            />
          </svg>
          <template v-else>+ </template>
          {{ t('admin.accounts.addMapping') }}
        </button>

        <div class="flex flex-wrap gap-2">
          <button
            v-for="preset in presetMappings"
            :key="`${preset.label}-${preset.from}-${preset.to}`"
            type="button"
            @click="emit('add-preset', { from: preset.from, to: preset.to })"
            :class="['rounded-lg px-3 py-1 text-xs transition-colors', preset.color]"
          >
            + {{ preset.label }}
          </button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, toRefs } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelRegistryPreset } from '@/generated/modelRegistry'
import ModelWhitelistSelector from '@/components/account/ModelWhitelistSelector.vue'
import type { ModelMapping } from '@/utils/accountFormShared'

interface Props {
  disabled?: boolean
  platform: string
  mode: 'whitelist' | 'mapping'
  allowedModels: string[]
  modelMappings: ModelMapping[]
  presetMappings: ModelRegistryPreset[]
  getMappingKey: (mapping: ModelMapping) => string
  variant?: 'default' | 'simple'
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  variant: 'default'
})

const emit = defineEmits<{
  'update:mode': [value: 'whitelist' | 'mapping']
  'update:allowedModels': [value: string[]]
  'add-mapping': []
  'remove-mapping': [index: number]
  'add-preset': [payload: { from: string; to: string }]
}>()

const { t } = useI18n()
const { allowedModels, variant } = toRefs(props)

const selectedAllowedModels = computed({
  get: () => props.allowedModels,
  set: (value: string[]) => emit('update:allowedModels', value)
})

const showModeIcons = computed(() => variant.value === 'default')
</script>
