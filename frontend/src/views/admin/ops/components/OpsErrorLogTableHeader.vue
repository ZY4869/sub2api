<template>
  <thead class="sticky top-0 z-10 bg-gray-50 dark:bg-dark-800">
    <tr>
      <th v-for="column in visibleColumns" :key="column.key" :class="headerClass">
        <button v-if="column.sortable" type="button" class="font-bold uppercase" @click="emitSort(column.key)">
          {{ column.label }}{{ sortMark(column.key) }}
        </button>
        <template v-else>{{ column.label }}</template>
      </th>
      <th :class="[headerClass, 'text-right']">
        {{ t('admin.ops.errorLog.action') }}
      </th>
    </tr>
  </thead>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { mapErrorSortKey } from '@/utils/errorBadges'

const props = defineProps<{
  hiddenColumns: Set<string>
  sortBy: string
  sortOrder: 'asc' | 'desc'
}>()

const emit = defineEmits<{
  (e: 'sort', sortBy: string, sortOrder: 'asc' | 'desc'): void
}>()

const { t } = useI18n()
const headerClass = 'border-b border-gray-200 px-4 py-2.5 text-left text-[11px] font-bold uppercase tracking-wider text-gray-500 dark:border-dark-700 dark:text-dark-400'

const columns = computed(() => [
  { key: 'created_at', label: t('admin.ops.errorLog.time'), sortable: true },
  { key: 'type', label: t('admin.ops.errorLog.type'), sortable: false },
  { key: 'category', label: t('usage.errors.category'), sortable: false },
  { key: 'platform', label: t('admin.ops.errorLog.platform'), sortable: false },
  { key: 'endpoint', label: t('admin.ops.errorLog.endpoint'), sortable: false },
  { key: 'model', label: t('admin.ops.errorLog.model'), sortable: true },
  { key: 'group', label: t('admin.ops.errorLog.group'), sortable: false },
  { key: 'user', label: t('admin.ops.errorLog.user'), sortable: false },
  { key: 'api_key', label: t('admin.ops.errorLog.apiKey'), sortable: false },
  { key: 'status', label: t('admin.ops.errorLog.status'), sortable: true },
  { key: 'message', label: t('admin.ops.errorLog.message'), sortable: false },
  { key: 'client_ip', label: t('admin.ops.errorLog.ip'), sortable: false },
  { key: 'user_agent', label: t('usage.userAgent'), sortable: false }
])

const visibleColumns = computed(() => columns.value.filter((column) => !props.hiddenColumns.has(column.key)))

function emitSort(key: string) {
  const mapped = mapErrorSortKey(key)
  const nextOrder = props.sortBy === mapped && props.sortOrder === 'asc' ? 'desc' : 'asc'
  emit('sort', mapped, nextOrder)
}

function sortMark(key: string): string {
  if (props.sortBy !== mapErrorSortKey(key)) return ''
  return props.sortOrder === 'asc' ? ' ↑' : ' ↓'
}
</script>
