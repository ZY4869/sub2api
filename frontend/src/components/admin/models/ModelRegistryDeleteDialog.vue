<template>
  <BaseDialog
    :show="show"
    :title="t('admin.models.registry.dialogs.manageVisibility')"
    width="narrow"
    close-on-click-outside
    @close="emit('close')"
  >
    <div v-if="entry" class="space-y-4">
      <div class="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/40">
        <p class="text-sm text-gray-700 dark:text-gray-300">
          <span class="font-medium text-gray-900 dark:text-white">{{ entry.id }}</span>
          <span v-if="entry.display_name" class="ml-2 text-gray-500 dark:text-gray-400">{{ entry.display_name }}</span>
        </p>
        <div class="mt-3 flex flex-wrap gap-2">
          <span class="inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-xs font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300">
            {{ formatSourceLabel(entry.source) }}
          </span>
          <span
            class="inline-flex rounded-full px-2.5 py-1 text-xs font-medium"
            :class="entry.hidden ? 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300' : 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'"
          >
            {{ entry.hidden ? t('admin.models.registry.statusLabels.hidden') : t('admin.models.registry.statusLabels.active') }}
          </span>
          <span
            v-if="entry.tombstoned"
            class="inline-flex rounded-full bg-red-100 px-2.5 py-1 text-xs font-medium text-red-700 dark:bg-red-500/15 dark:text-red-300"
          >
            {{ t('admin.models.registry.statusLabels.tombstoned') }}
          </span>
        </div>
      </div>

      <p class="text-sm text-gray-600 dark:text-gray-300">
        {{ entry.hidden ? t('admin.models.registry.dialogs.restoreDescription') : t('admin.models.registry.dialogs.hideDescription') }}
      </p>
      <p class="rounded-2xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300">
        {{ t('admin.models.registry.dialogs.hardDeleteDescription') }}
      </p>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-secondary" :disabled="saving || !entry" @click="emit('toggle-visibility')">
          {{ entry?.hidden ? t('admin.models.registry.actions.show') : t('admin.models.registry.actions.hide') }}
        </button>
        <button type="button" class="btn btn-danger" :disabled="saving || !entry" @click="emit('hard-delete')">
          {{ t('admin.models.registry.actions.hardDelete') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { ModelRegistryDetail } from '@/api/admin/modelRegistry'

withDefaults(defineProps<{
  show: boolean
  entry?: ModelRegistryDetail | null
  saving?: boolean
}>(), {
  entry: null,
  saving: false
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'toggle-visibility'): void
  (e: 'hard-delete'): void
}>()

const { t } = useI18n()

function formatSourceLabel(source: string) {
  const normalizedSource = source === 'runtime' ? 'manual' : source
  const key = `admin.models.registry.sourceLabels.${normalizedSource}`
  const translated = t(key)
  return translated === key ? normalizedSource : translated
}
</script>
