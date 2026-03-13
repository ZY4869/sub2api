<template>
  <BaseDialog
    :show="show"
    :title="t('admin.models.available.activateDialog.title')"
    width="wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <div class="space-y-4">
      <p class="text-sm text-gray-600 dark:text-gray-300">
        {{ t('admin.models.available.activateDialog.description') }}
      </p>

      <input
        v-model.trim="search"
        type="text"
        class="input"
        :placeholder="t('admin.models.available.activateDialog.searchPlaceholder')"
      />

      <div class="max-h-[28rem] space-y-2 overflow-y-auto">
        <label
          v-for="item in filteredItems"
          :key="item.id"
          class="flex cursor-pointer items-start gap-3 rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm shadow-sm transition hover:border-primary-300 dark:border-dark-700 dark:bg-dark-800"
        >
          <input
            v-model="selected"
            type="checkbox"
            class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600"
            :value="item.id"
          />
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <span class="font-medium text-gray-900 dark:text-white">{{ item.id }}</span>
              <span class="inline-flex rounded-full bg-sky-100 px-2 py-0.5 text-xs font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300">
                {{ item.provider || '-' }}
              </span>
            </div>
            <p v-if="item.display_name" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ item.display_name }}</p>
          </div>
        </label>

        <EmptyState
          v-if="filteredItems.length === 0"
          :title="t('admin.models.available.activateDialog.emptyTitle')"
          :description="t('admin.models.available.activateDialog.emptyDescription')"
        />
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-primary" :disabled="submitting || selected.length === 0" @click="emit('submit', selected)">
          {{ t('admin.models.available.activateDialog.confirm') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import type { ModelRegistryDetail } from '@/api/admin/modelRegistry'

const props = withDefaults(defineProps<{
  show: boolean
  items: ModelRegistryDetail[]
  submitting?: boolean
}>(), {
  submitting: false
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'submit', modelIds: string[]): void
}>()

const { t } = useI18n()
const search = ref('')
const selected = ref<string[]>([])

watch(
  () => props.show,
  (show) => {
    if (!show) {
      search.value = ''
      selected.value = []
    }
  }
)

const filteredItems = computed(() => {
  const query = search.value.toLowerCase()
  if (!query) {
    return props.items
  }
  return props.items.filter((item) =>
    [item.id, item.display_name, item.provider].some((value) => String(value || '').toLowerCase().includes(query))
  )
})
</script>
