<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.codexImport.title')"
    width="wide"
    close-on-click-outside
    @close="handleClose"
  >
    <form id="codex-session-import-form" class="space-y-4" @submit.prevent="handleSubmit">
      <textarea
        v-model="content"
        class="input-field min-h-44 font-mono text-xs"
        :placeholder="t('admin.accounts.codexImport.contentPlaceholder')"
        data-codex-import-content="true"
      />

      <div class="grid gap-3 md:grid-cols-2">
        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.codexImport.name') }}</span>
          <input v-model.trim="name" class="input-field" type="text" />
        </label>
        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.codexImport.proxy') }}</span>
          <select v-model="proxyId" class="input-field">
            <option value="">{{ t('common.none') }}</option>
            <option v-for="proxy in proxies" :key="proxy.id" :value="String(proxy.id)">
              {{ proxy.name || `#${proxy.id}` }}
            </option>
          </select>
        </label>
        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.codexImport.concurrency') }}</span>
          <input v-model="concurrency" class="input-field" min="0" placeholder="3" type="number" />
        </label>
        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.codexImport.priority') }}</span>
          <input v-model="priority" class="input-field" min="0" placeholder="50" type="number" />
        </label>
        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.codexImport.rateMultiplier') }}</span>
          <input v-model="rateMultiplier" class="input-field" min="0" step="0.01" type="number" />
        </label>
        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.codexImport.loadFactor') }}</span>
          <input v-model="loadFactor" class="input-field" min="0" max="10000" type="number" />
        </label>
        <label class="space-y-1">
          <span class="input-label">{{ t('admin.accounts.codexImport.expiresAt') }}</span>
          <input v-model="expiresAtLocal" class="input-field" type="datetime-local" />
        </label>
      </div>

      <div v-if="groups.length" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
        <div class="mb-2 text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.codexImport.groups') }}
        </div>
        <div class="grid max-h-36 gap-2 overflow-auto sm:grid-cols-2">
          <label
            v-for="group in groups"
            :key="group.id"
            class="flex items-center gap-2 text-sm text-gray-700 dark:text-dark-300"
          >
            <input v-model="selectedGroupIds" type="checkbox" class="h-4 w-4" :value="group.id" />
            <span class="truncate">{{ group.name }}</span>
          </label>
        </div>
      </div>

      <div class="grid gap-2 text-sm text-gray-700 dark:text-dark-300 sm:grid-cols-2">
        <label class="flex items-center gap-2">
          <input v-model="updateExisting" type="checkbox" class="h-4 w-4" />
          <span>{{ t('admin.accounts.codexImport.updateExisting') }}</span>
        </label>
        <label class="flex items-center gap-2">
          <input v-model="autoPauseOnExpired" type="checkbox" class="h-4 w-4" />
          <span>{{ t('admin.accounts.codexImport.autoPauseOnExpired') }}</span>
        </label>
        <label class="flex items-center gap-2">
          <input v-model="skipDefaultGroupBind" type="checkbox" class="h-4 w-4" />
          <span>{{ t('admin.accounts.codexImport.skipDefaultGroupBind') }}</span>
        </label>
        <label class="flex items-center gap-2">
          <input v-model="confirmMixedChannelRisk" type="checkbox" class="h-4 w-4" />
          <span>{{ t('admin.accounts.codexImport.confirmMixedChannelRisk') }}</span>
        </label>
      </div>

      <CodexSessionImportResultPanel v-if="result" :result="result" />
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" type="button" :disabled="submitting" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          class="btn btn-primary"
          type="submit"
          form="codex-session-import-form"
          :disabled="submitting || !content.trim()"
          data-codex-import-submit="true"
        >
          {{ submitting ? t('admin.accounts.codexImport.importing') : t('admin.accounts.codexImport.submit') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import type { CodexSessionImportResult } from '@/api/admin/accounts'
import type { AdminGroup, Proxy } from '@/types'
import CodexSessionImportResultPanel from './CodexSessionImportResultPanel.vue'
import { useCodexSessionImportModal } from './useCodexSessionImportModal'

const props = defineProps<{
  show: boolean
  proxies: Proxy[]
  groups: AdminGroup[]
}>()

const emit = defineEmits<{
  close: []
  imported: [result: CodexSessionImportResult]
}>()

const { t } = useI18n()
const {
  content,
  name,
  proxyId,
  concurrency,
  priority,
  rateMultiplier,
  loadFactor,
  expiresAtLocal,
  selectedGroupIds,
  updateExisting,
  autoPauseOnExpired,
  skipDefaultGroupBind,
  confirmMixedChannelRisk,
  submitting,
  result,
  handleClose,
  handleSubmit
} = useCodexSessionImportModal({
  show: toRef(props, 'show'),
  onClose: () => emit('close'),
  onImported: (result) => emit('imported', result)
})
</script>
