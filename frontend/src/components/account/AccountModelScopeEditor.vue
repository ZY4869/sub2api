<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <label class="input-label">{{ t('admin.accounts.modelRestriction') }}</label>

    <div v-if="disabled" class="mb-3 rounded-lg bg-amber-50 p-3 dark:bg-amber-900/20">
      <p class="text-xs text-amber-700 dark:text-amber-400">
        {{ t('admin.accounts.openai.modelRestrictionDisabledByPassthrough') }}
      </p>
    </div>

    <template v-else>
      <div class="mb-4 flex gap-2">
        <button
          type="button"
          @click="emit('update:mode', 'whitelist')"
          :class="mode === 'whitelist' ? activeModeClass('primary') : inactiveModeClass"
          class="flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all"
        >
          {{ t('admin.accounts.modelWhitelist') }}
        </button>
        <button
          type="button"
          @click="emit('update:mode', 'mapping')"
          :class="mode === 'mapping' ? activeModeClass('purple') : inactiveModeClass"
          class="flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all"
        >
          {{ t('admin.accounts.modelMapping') }}
        </button>
      </div>

      <AccountModelScopeWhitelistEditor
        v-if="mode === 'whitelist'"
        :platform="platform"
        :allowed-models="allowedModels"
        @update:allowedModels="emit('update:allowedModels', $event)"
      />

      <AccountModelScopeMappingEditor
        v-else
        v-model:actual-model-locked="actualModelLocked"
        :model-mappings="modelMappings"
        :preset-mappings="presetMappings"
        :get-mapping-key="getMappingKey"
        :show-actual-model-lock="showActualModelLock"
        @add-mapping="emit('add-mapping')"
        @remove-mapping="emit('remove-mapping', $event)"
        @add-preset="emit('add-preset', $event)"
      />
    </template>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ModelRegistryPreset } from '@/generated/modelRegistry'
import type { ModelMapping } from '@/utils/accountFormShared'
import AccountModelScopeWhitelistEditor from './AccountModelScopeWhitelistEditor.vue'
import AccountModelScopeMappingEditor from './AccountModelScopeMappingEditor.vue'

interface Props {
  disabled?: boolean
  platform: string
  mode: 'whitelist' | 'mapping'
  allowedModels: string[]
  modelMappings: ModelMapping[]
  presetMappings: ModelRegistryPreset[]
  getMappingKey: (mapping: ModelMapping) => string
  showActualModelLock?: boolean
}

withDefaults(defineProps<Props>(), {
  disabled: false,
  showActualModelLock: false
})
const emit = defineEmits<{
  'update:mode': [value: 'whitelist' | 'mapping']
  'update:allowedModels': [value: string[]]
  'add-mapping': []
  'remove-mapping': [index: number]
  'add-preset': [payload: { from: string; to: string }]
}>()
const actualModelLocked = defineModel<boolean>('actualModelLocked', { default: true })

const { t } = useI18n()

const inactiveModeClass = 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'

function activeModeClass(color: 'primary' | 'purple') {
  return color === 'primary'
    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
    : 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
}
</script>
