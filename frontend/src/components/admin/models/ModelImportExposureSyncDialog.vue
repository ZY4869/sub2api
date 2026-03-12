<template>
  <BaseDialog
    :show="show"
    :title="t(`${i18nBaseKey}.modelImportSyncTitle`)"
    width="wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <div class="space-y-4">
      <p class="text-sm text-gray-600 dark:text-gray-300">
        {{ t(`${i18nBaseKey}.modelImportSyncDescription`, { count: models.length }) }}
      </p>

      <div class="space-y-2">
        <p class="input-label">{{ t(`${i18nBaseKey}.modelImportSyncTargetsLabel`) }}</p>
        <div class="grid gap-2 sm:grid-cols-2">
          <label
            v-for="target in targetOptions"
            :key="target.value"
            class="flex items-center gap-3 rounded-xl border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700 shadow-sm dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200"
          >
            <input
              v-model="selectedTargets"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-primary-600"
              :value="target.value"
            />
            <span>{{ target.label }}</span>
          </label>
        </div>
      </div>

      <div class="space-y-2">
        <div class="flex items-center justify-between gap-3">
          <p class="input-label">{{ t(`${i18nBaseKey}.modelImportSyncModelsLabel`) }}</p>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ models.length }}</span>
        </div>
        <div class="max-h-64 overflow-y-auto rounded-2xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900/40">
          <div class="flex flex-wrap gap-2">
            <span
              v-for="model in visibleModels"
              :key="model"
              class="inline-flex rounded-full bg-white px-2.5 py-1 text-xs font-medium text-gray-700 shadow-sm dark:bg-dark-800 dark:text-gray-200"
            >
              {{ model }}
            </span>
          </div>
          <p v-if="hiddenModelCount > 0" class="mt-3 text-xs text-gray-500 dark:text-gray-400">
            {{ t(`${i18nBaseKey}.modelImportSyncMoreModels`, { count: hiddenModelCount }) }}
          </p>
        </div>
      </div>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="syncing" @click="emit('close')">
        {{ t('common.cancel') }}
      </button>
      <button type="button" class="btn btn-primary" :disabled="syncing || selectedTargets.length === 0" @click="handleSubmit">
        {{ syncing ? t('common.loading') : t(`${i18nBaseKey}.modelImportSyncConfirm`) }}
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { ModelRegistryExposureTarget } from '@/api/admin/modelRegistry'
import { MODEL_REGISTRY_EXPOSURE_OPTIONS } from '@/utils/modelRegistryMeta'

const props = withDefaults(defineProps<{
  show: boolean
  models: string[]
  syncing?: boolean
  i18nBaseKey?: string
}>(), {
  syncing: false,
  i18nBaseKey: 'admin.accounts'
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'submit', exposures: ModelRegistryExposureTarget[]): void
}>()

const { t } = useI18n()
const i18nBaseKey = computed(() => props.i18nBaseKey)
const selectedTargets = ref<ModelRegistryExposureTarget[]>(['whitelist', 'use_key', 'test', 'runtime'])

const targetOptions = computed(() =>
  MODEL_REGISTRY_EXPOSURE_OPTIONS.map((item) => ({
    value: item.value,
    label: item.description
  }))
)

const visibleModels = computed(() => props.models.slice(0, 18))
const hiddenModelCount = computed(() => Math.max(0, props.models.length - visibleModels.value.length))

watch(
  () => props.show,
  (show) => {
    if (show) {
      selectedTargets.value = ['whitelist', 'use_key', 'test', 'runtime']
    }
  },
  { immediate: true }
)

function handleSubmit() {
  emit('submit', [...selectedTargets.value])
}
</script>
