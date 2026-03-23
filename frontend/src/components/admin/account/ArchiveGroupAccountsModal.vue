<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.bulkActions.archiveCurrentGroupTitle')"
    width="narrow"
    @close="handleClose"
  >
    <form id="archive-group-accounts-form" class="space-y-4" @submit.prevent="handleSubmit">
      <div class="rounded-lg bg-amber-50 p-4 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
        {{ t('admin.accounts.bulkActions.archiveCurrentGroupDescription', { group: sourceGroup?.name || '-' }) }}
      </div>

      <div class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.accounts.bulkActions.archivePlatform', { platform: platformLabel }) }}
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.bulkActions.archiveGroupName') }}</label>
        <input
          v-model.trim="groupName"
          type="text"
          class="input"
          :placeholder="t('admin.accounts.bulkActions.archiveGroupNamePlaceholder')"
        />
        <p class="input-hint">{{ t('admin.accounts.bulkActions.archiveGroupHint') }}</p>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="submitting" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button type="submit" form="archive-group-accounts-form" class="btn btn-primary" :disabled="submitting || !sourceGroup">
          {{ submitting ? t('admin.accounts.bulkActions.archiving') : t('admin.accounts.bulkActions.archiveConfirm') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AdminGroup, ArchiveGroupAccountsResult } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'

const STORAGE_PREFIX = 'accounts-batch-archive-group:'

const props = defineProps<{
  show: boolean
  sourceGroup: AdminGroup | null
}>()

const emit = defineEmits<{
  close: []
  archived: [result: ArchiveGroupAccountsResult]
}>()

const { t } = useI18n()
const appStore = useAppStore()

const groupName = ref('')
const submitting = ref(false)

const storageKey = computed(() => {
  const platform = props.sourceGroup?.platform
  return platform ? `${STORAGE_PREFIX}${platform}` : ''
})

const platformLabel = computed(() =>
  props.sourceGroup ? t(`admin.accounts.platforms.${props.sourceGroup.platform}`) : '-'
)

const loadStoredGroupName = () => {
  if (!storageKey.value) return ''
  try {
    return localStorage.getItem(storageKey.value) || ''
  } catch {
    return ''
  }
}

const persistGroupName = (value: string) => {
  if (!storageKey.value) return
  try {
    const trimmed = value.trim()
    if (trimmed) {
      localStorage.setItem(storageKey.value, trimmed)
    } else {
      localStorage.removeItem(storageKey.value)
    }
  } catch {
    // Ignore localStorage failures.
  }
}

const resetForm = () => {
  groupName.value = loadStoredGroupName()
}

const handleClose = () => {
  if (submitting.value) return
  emit('close')
}

const handleSubmit = async () => {
  if (!props.sourceGroup) {
    appStore.showError(t('admin.accounts.bulkActions.archiveCurrentGroupDisabled'))
    return
  }
  if (!groupName.value.trim()) {
    appStore.showError(t('admin.accounts.bulkActions.archiveGroupHint'))
    return
  }

  submitting.value = true
  try {
    const result = await adminAPI.accounts.archiveGroupAccounts({
      source_group_id: props.sourceGroup.id,
      group_name: groupName.value.trim()
    })
    emit('archived', result)
    if (result.failed_count > 0 && result.archived_count > 0) {
      appStore.showWarning(
        t('admin.accounts.bulkActions.partialSuccess', {
          success: result.archived_count,
          failed: result.failed_count
        })
      )
    } else if (result.failed_count > 0) {
      appStore.showError(
        t('admin.accounts.bulkActions.partialSuccess', {
          success: result.archived_count,
          failed: result.failed_count
        })
      )
    } else {
      appStore.showSuccess(
        t('admin.accounts.bulkActions.archiveCurrentGroupSuccess', {
          group: result.source_group_name,
          count: result.archived_count,
          archive: result.archive_group_name
        })
      )
    }
    emit('close')
  } catch (error: any) {
    appStore.showError(error?.message || t('common.error'))
  } finally {
    submitting.value = false
  }
}

watch(
  () => props.show,
  (open) => {
    if (open) {
      resetForm()
    }
  },
  { immediate: true }
)

watch(storageKey, () => {
  if (props.show) {
    groupName.value = loadStoredGroupName()
  }
})

watch(groupName, (value) => {
  persistGroupName(value)
})
</script>
