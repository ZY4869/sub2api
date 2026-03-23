<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.bulkActions.archiveTitle')"
    width="narrow"
    @close="handleClose"
  >
    <form id="archive-accounts-form" class="space-y-4" @submit.prevent="handleSubmit">
      <div class="rounded-lg bg-amber-50 p-4 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-200">
        {{ t('admin.accounts.bulkActions.archiveDescription', { count: accountIds.length }) }}
      </div>

      <div v-if="singlePlatform" class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.accounts.bulkActions.archivePlatform', { platform: platformLabel }) }}
      </div>

      <div
        v-else
        class="rounded-lg bg-red-50 p-4 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-300"
      >
        {{ t('admin.accounts.bulkActions.archiveMixedPlatformDisabled') }}
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
        <button
          type="submit"
          form="archive-accounts-form"
          class="btn btn-primary"
          :disabled="submitting || !singlePlatform"
        >
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
import type { AccountPlatform, BatchArchiveAccountsResult } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'

const STORAGE_PREFIX = 'accounts-batch-archive-group:'

const props = defineProps<{
  show: boolean
  accountIds: number[]
  selectedPlatforms: AccountPlatform[]
}>()

const emit = defineEmits<{
  close: []
  archived: [result: BatchArchiveAccountsResult]
}>()

const { t } = useI18n()
const appStore = useAppStore()

const groupName = ref('')
const submitting = ref(false)

const singlePlatform = computed<AccountPlatform | null>(() =>
  props.selectedPlatforms.length === 1 ? props.selectedPlatforms[0] : null
)

const platformLabel = computed(() =>
  singlePlatform.value ? t(`admin.accounts.platforms.${singlePlatform.value}`) : ''
)

const storageKey = (platform: AccountPlatform) => `${STORAGE_PREFIX}${platform}`

const loadStoredGroupName = (platform: AccountPlatform | null) => {
  if (!platform) return ''
  try {
    return localStorage.getItem(storageKey(platform)) || ''
  } catch {
    return ''
  }
}

const persistGroupName = (platform: AccountPlatform | null, value: string) => {
  if (!platform) return
  try {
    const trimmed = value.trim()
    if (trimmed) {
      localStorage.setItem(storageKey(platform), trimmed)
    } else {
      localStorage.removeItem(storageKey(platform))
    }
  } catch {
    // Ignore localStorage failures.
  }
}

const resetForm = () => {
  groupName.value = loadStoredGroupName(singlePlatform.value)
}

const handleClose = () => {
  if (submitting.value) return
  emit('close')
}

const handleSubmit = async () => {
  if (!singlePlatform.value) {
    appStore.showError(t('admin.accounts.bulkActions.archiveMixedPlatformDisabled'))
    return
  }
  if (props.accountIds.length === 0) {
    appStore.showError(t('admin.accounts.bulkActions.archiveNoAccounts'))
    return
  }
  if (!groupName.value.trim()) {
    appStore.showError(t('admin.accounts.bulkActions.archiveGroupHint'))
    return
  }

  submitting.value = true
  try {
    const result = await adminAPI.accounts.batchArchiveAccounts({
      account_ids: [...props.accountIds],
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
        t('admin.accounts.bulkActions.archiveSuccess', {
          count: result.archived_count,
          group: result.archive_group_name
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

watch(singlePlatform, (platform) => {
  if (props.show) {
    groupName.value = loadStoredGroupName(platform)
  }
})

watch(groupName, (value) => {
  persistGroupName(singlePlatform.value, value)
})
</script>
