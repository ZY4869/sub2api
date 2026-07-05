<template>
  <div class="flex h-full min-h-0 flex-col bg-white dark:bg-dark-900">
    <!-- Loading State -->
    <div v-if="loading" class="flex flex-1 items-center justify-center py-10">
      <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
    </div>

    <!-- Table Container -->
    <div v-else class="flex min-h-0 flex-1 flex-col">
      <div class="min-h-0 flex-1 overflow-auto border-b border-gray-200 dark:border-dark-700">
        <table class="w-full border-separate border-spacing-0">
          <OpsErrorLogTableHeader
            :hidden-columns="hiddenColumns"
            :sort-by="sortBy"
            :sort-order="sortOrder"
            @sort="(sortBy, sortOrder) => emit('sort', sortBy, sortOrder)"
          />
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-if="rows.length === 0">
              <td :colspan="visibleColumnCount" class="py-12 text-center text-sm text-gray-400 dark:text-dark-500">
                {{ t('admin.ops.errorLog.noErrors') }}
              </td>
            </tr>

            <OpsErrorLogTableRow
              v-for="log in rows"
              :key="log.id"
              :log="log"
              :hidden-columns="hiddenColumns"
              @open-error-detail="emit('openErrorDetail', $event)"
            />
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div class="bg-gray-50/50 dark:bg-dark-800/50">
        <Pagination
          v-if="total > 0"
          :total="total"
          :page="page"
          :page-size="pageSize"
          :page-size-options="[10]"
          @update:page="emit('update:page', $event)"
          @update:pageSize="emit('update:pageSize', $event)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Pagination from '@/components/common/Pagination.vue'
import type { OpsErrorLog } from '@/api/admin/ops'
import OpsErrorLogTableHeader from './OpsErrorLogTableHeader.vue'
import OpsErrorLogTableRow from './OpsErrorLogTableRow.vue'

const { t } = useI18n()

interface Props {
  rows: OpsErrorLog[]
  total: number
  loading: boolean
  page: number
  pageSize: number
  hiddenColumns?: Set<string>
  sortBy?: string
  sortOrder?: 'asc' | 'desc'
}

interface Emits {
  (e: 'openErrorDetail', id: number): void
  (e: 'update:page', value: number): void
  (e: 'update:pageSize', value: number): void
  (e: 'sort', sortBy: string, sortOrder: 'asc' | 'desc'): void
}

const props = withDefaults(defineProps<Props>(), {
  hiddenColumns: () => new Set<string>(),
  sortBy: 'created_at',
  sortOrder: 'desc'
})
const emit = defineEmits<Emits>()

const visibleColumnCount = computed(() => {
  const keys = ['created_at', 'type', 'category', 'platform', 'endpoint', 'model', 'group', 'user', 'api_key', 'status', 'message', 'client_ip', 'user_agent']
  return keys.filter((key) => isColumnVisible(key)).length + 1
})

function isColumnVisible(key: string): boolean {
  return !props.hiddenColumns?.has(key)
}

</script>
