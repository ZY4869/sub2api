<template>
  <ConfirmDialog
    :show="showDeleteDialog"
    :title="t('admin.groups.deleteGroup')"
    :message="deleteConfirmMessage"
    :confirm-text="t('common.delete')"
    :cancel-text="t('common.cancel')"
    :danger="true"
    @confirm="confirmDelete"
    @cancel="showDeleteDialog = false"
  />

    <BaseDialog
      :show="showSortModal"
      :title="t('admin.groups.sortOrder')"
      width="normal"
      @close="closeSortModal"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.groups.sortOrderHint') }}
        </p>
        <VueDraggable
          v-model="sortableGroups"
          :animation="200"
          class="space-y-2"
        >
          <div
            v-for="group in sortableGroups"
            :key="group.id"
            class="flex cursor-grab items-center gap-3 rounded-lg border border-gray-200 bg-white p-3 transition-shadow hover:shadow-md active:cursor-grabbing dark:border-dark-600 dark:bg-dark-700"
          >
            <div class="text-gray-400">
              <Icon name="menu" size="md" />
            </div>
            <div class="flex-1">
              <div class="font-medium text-gray-900 dark:text-white">{{ group.name }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                <span
                  :class="[
                    'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium',
                    group.platform === 'anthropic'
                      ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
                      : group.platform === 'kiro'
                        ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                      : group.platform === 'openai'
                        ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                        : group.platform === 'grok'
                          ? 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-200'
                        : group.platform === 'deepseek'
                          ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-400'
                        : group.platform === 'antigravity'
                          ? 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
                          : 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                  ]"
                >
                  {{ t('admin.groups.platforms.' + group.platform) }}
                </span>
              </div>
            </div>
            <div class="text-sm text-gray-400">
              #{{ group.id }}
            </div>
          </div>
        </VueDraggable>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3 pt-4">
          <button @click="closeSortModal" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            @click="saveSortOrder"
            :disabled="sortSubmitting"
            class="btn btn-primary"
          >
            <svg
              v-if="sortSubmitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{ sortSubmitting ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <GroupRateMultipliersModal
      :show="showRateMultipliersModal"
      :group="rateMultipliersGroup"
      @close="showRateMultipliersModal = false"
      @success="loadGroups"
    />
</template>

<script setup lang="ts">
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import GroupRateMultipliersModal from '@/components/admin/group/GroupRateMultipliersModal.vue'
import Icon from '@/components/icons/Icon.vue'
import { VueDraggable } from 'vue-draggable-plus'

const props = defineProps<{ ctx: any }>()
const {
  t,
  showDeleteDialog,
  deleteConfirmMessage,
  confirmDelete,
  showSortModal,
  closeSortModal,
  sortableGroups,
  sortSubmitting,
  saveSortOrder,
  showRateMultipliersModal,
  rateMultipliersGroup,
  loadGroups
} = props.ctx
</script>
