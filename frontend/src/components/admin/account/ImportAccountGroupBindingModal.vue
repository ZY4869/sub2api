<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.importGroupBinding.title')"
    width="wide"
    @close="handleClose"
  >
    <form id="import-account-group-binding-form" class="space-y-4" @submit.prevent="handleSubmit">
      <div class="rounded-lg bg-blue-50 p-4 text-sm text-blue-700 dark:bg-blue-900/20 dark:text-blue-300">
        {{ t('admin.accounts.importGroupBinding.description', { count: accounts.length }) }}
      </div>

      <div class="space-y-3">
        <div
          v-for="section in sections"
          :key="section.key"
          class="rounded-xl border border-gray-200 p-4 dark:border-dark-700"
        >
          <div class="mb-3 flex items-start justify-between gap-3">
            <div>
              <div class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ section.title }}
              </div>
              <div class="text-xs text-gray-500 dark:text-dark-400">
                {{ t('admin.accounts.importGroupBinding.sectionCount', { count: section.accounts.length }) }}
              </div>
            </div>
            <button
              type="button"
              class="text-xs font-medium text-gray-500 hover:text-gray-700 dark:text-dark-300 dark:hover:text-white"
              @click="clearSection(section.key)"
            >
              {{ t('common.clear') }}
            </button>
          </div>
          <GroupSelector
            :model-value="sectionSelections[section.key] || []"
            :groups="groups"
            :platform="section.platform"
            @update:model-value="setSectionGroups(section.key, $event)"
          />
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="submitting" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="import-account-group-binding-form"
          class="btn btn-primary"
          :disabled="submitting"
        >
          {{
            submitting
              ? t('admin.accounts.importGroupBinding.submitting')
              : t('admin.accounts.importGroupBinding.submit')
          }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { getPlatformEnglishName } from '@/utils/platformBranding'
import type {
  AdminDataImportCreatedAccount,
  AdminGroup,
  AccountPlatform,
  AccountType,
  GroupPlatform
} from '@/types'

const props = defineProps<{
  show: boolean
  jobId: string
  accounts: AdminDataImportCreatedAccount[]
  groups: AdminGroup[]
}>()

const emit = defineEmits<{
  close: []
  updated: []
}>()

interface ImportBindingSection {
  key: string
  platform: GroupPlatform
  type: AccountType
  title: string
  accounts: AdminDataImportCreatedAccount[]
}

const { t } = useI18n()
const appStore = useAppStore()
const submitting = ref(false)
const sectionSelections = reactive<Record<string, number[]>>({})

const sections = computed<ImportBindingSection[]>(() => {
  const byKey = new Map<string, ImportBindingSection>()
  for (const account of props.accounts) {
    const platform = account.platform as AccountPlatform
    const type = account.type as AccountType
    const key = `${platform}:${type}`
    const existing = byKey.get(key)
    if (existing) {
      existing.accounts.push(account)
      continue
    }
    byKey.set(key, {
      key,
      platform: platform as GroupPlatform,
      type,
      title: `${getPlatformEnglishName(platform)} / ${formatAccountType(type)}`,
      accounts: [account]
    })
  }
  return Array.from(byKey.values()).sort((left, right) => left.title.localeCompare(right.title))
})

const formatAccountType = (type: AccountType) => {
  if (type === 'oauth') return t('ui.platformType.oauth')
  if (type === 'setup-token') return t('ui.platformType.token')
  if (type === 'apikey') return t('ui.platformType.key')
  if (type === 'sso') return t('ui.platformType.sso')
  if (type === 'bedrock') return t('ui.platformType.aws')
  return type
}

watch(
  () => props.show,
  (open) => {
    if (!open) return
    for (const key of Object.keys(sectionSelections)) {
      delete sectionSelections[key]
    }
    for (const section of sections.value) {
      sectionSelections[section.key] = []
    }
  }
)

const setSectionGroups = (key: string, groupIds: number[]) => {
  sectionSelections[key] = groupIds
}

const clearSection = (key: string) => {
  sectionSelections[key] = []
}

const handleClose = () => {
  if (submitting.value) return
  emit('close')
}

const handleSubmit = async () => {
  const payloads = sections.value
    .map((section) => ({
      section,
      groupIds: sectionSelections[section.key] || []
    }))
    .filter((item) => item.groupIds.length > 0)

  if (payloads.length === 0) {
    appStore.showError(t('admin.accounts.importGroupBinding.noGroupsSelected'))
    return
  }
  if (!props.jobId) {
    appStore.showError(t('admin.accounts.importGroupBinding.missingJob'))
    return
  }

  submitting.value = true
  try {
    const result = await adminAPI.accounts.bindImportJobGroups(props.jobId, {
      sections: payloads.map((item) => ({
        platform: item.section.platform,
        type: item.section.type,
        group_ids: item.groupIds
      }))
    })
    const success = result.success || result.bound_count || 0
    const failed = result.failed || 0

    if (failed > 0) {
      appStore.showError(t('admin.accounts.importGroupBinding.partialSuccess', { success, failed }))
    } else {
      appStore.showSuccess(t('admin.accounts.importGroupBinding.success', { count: success }))
    }
    emit('updated')
    emit('close')
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.accounts.importGroupBinding.failed'))
  } finally {
    submitting.value = false
  }
}
</script>
