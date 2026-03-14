<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import type { ModelMapping } from '@/utils/accountFormShared'

interface ModelOption {
  value: string
  label: string
}

interface PresetMapping {
  label: string
  from: string
  to: string
  color: string
}

defineProps<{
  models: ModelOption[]
  presets: PresetMapping[]
}>()

const enabled = defineModel<boolean>('enabled', { required: true })
const mode = defineModel<'whitelist' | 'mapping'>('mode', { required: true })
const allowedModels = defineModel<string[]>('allowedModels', { required: true })
const modelMappings = defineModel<ModelMapping[]>('modelMappings', { required: true })

const { t } = useI18n()
const appStore = useAppStore()

const addModelMapping = () => {
  modelMappings.value.push({ from: '', to: '' })
}

const removeModelMapping = (index: number) => {
  modelMappings.value.splice(index, 1)
}

const addPresetMapping = (from: string, to: string) => {
  const exists = modelMappings.value.some((mapping) => mapping.from === from)
  if (exists) {
    appStore.showInfo(t('admin.accounts.mappingExists', { model: from }))
    return
  }
  modelMappings.value.push({ from, to })
}
</script>

<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="mb-3 flex items-center justify-between">
      <label
        id="bulk-edit-model-restriction-label"
        class="input-label mb-0"
        for="bulk-edit-model-restriction-enabled"
      >
        {{ t('admin.accounts.modelRestriction') }}
      </label>
      <input
        v-model="enabled"
        id="bulk-edit-model-restriction-enabled"
        type="checkbox"
        aria-controls="bulk-edit-model-restriction-body"
        class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
      />
    </div>

    <div
      id="bulk-edit-model-restriction-body"
      :class="!enabled && 'pointer-events-none opacity-50'"
      role="group"
      aria-labelledby="bulk-edit-model-restriction-label"
    >
      <div class="mb-4 flex gap-2">
        <button
          type="button"
          :class="[
            'flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            mode === 'whitelist'
              ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
          ]"
          @click="mode = 'whitelist'"
        >
          <svg
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
          :class="[
            'flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all',
            mode === 'mapping'
              ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
          ]"
          @click="mode = 'mapping'"
        >
          <svg
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
        <div class="mb-3 rounded-lg bg-blue-50 p-3 dark:bg-blue-900/20">
          <p class="text-xs text-blue-700 dark:text-blue-400">
            <svg
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
            {{ t('admin.accounts.selectAllowedModels') }}
          </p>
        </div>

        <div class="mb-3 grid grid-cols-2 gap-2">
          <label
            v-for="model in models"
            :key="model.value"
            class="flex cursor-pointer items-center rounded-lg border p-3 transition-all hover:bg-gray-50 dark:border-dark-600 dark:hover:bg-dark-700"
            :class="
              allowedModels.includes(model.value)
                ? 'border-primary-500 bg-primary-50 dark:bg-primary-900/20'
                : 'border-gray-200'
            "
          >
            <input
              v-model="allowedModels"
              type="checkbox"
              :value="model.value"
              class="mr-2 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            />
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ model.label }}</span>
          </label>
        </div>

        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.selectedModels', { count: allowedModels.length }) }}
          <span v-if="allowedModels.length === 0">{{ t('admin.accounts.supportsAllModels') }}</span>
        </p>
      </div>

      <div v-else>
        <div class="mb-3 rounded-lg bg-purple-50 p-3 dark:bg-purple-900/20">
          <p class="text-xs text-purple-700 dark:text-purple-400">
            <svg
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
            :key="`${mapping.from}-${mapping.to}-${index}`"
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
              class="rounded-lg p-2 text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
              @click="removeModelMapping(index)"
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
          class="mb-3 w-full rounded-lg border-2 border-dashed border-gray-300 px-4 py-2 text-gray-600 transition-colors hover:border-gray-400 hover:text-gray-700 dark:border-dark-500 dark:text-gray-400 dark:hover:border-dark-400 dark:hover:text-gray-300"
          @click="addModelMapping"
        >
          <svg
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
          {{ t('admin.accounts.addMapping') }}
        </button>

        <div class="flex flex-wrap gap-2">
          <button
            v-for="preset in presets"
            :key="preset.label"
            type="button"
            :class="['rounded-lg px-3 py-1 text-xs transition-colors', preset.color]"
            @click="addPresetMapping(preset.from, preset.to)"
          >
            + {{ preset.label }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
