<template>
  <CreateAccountModal
    :show="showCreate"
    :proxies="proxies"
    :groups="groups"
    @close="emit('close-create')"
    @created="emit('created')"
    @models-imported="emit('models-imported', $event)"
  />
  <ModelImportExposureSyncDialog
    :show="syncDialogOpen"
    :models="syncDialogModels"
    :syncing="syncDialogSubmitting"
    @close="emit('close-sync-dialog')"
    @submit="emit('submit-sync-dialog', $event)"
  />
  <EditAccountModal
    :show="showEdit"
    :account="editingAccount"
    :proxies="proxies"
    :groups="groups"
    @close="emit('close-edit')"
    @updated="emit('updated', $event)"
  />
  <ReAuthAccountModal
    :show="showReAuth"
    :account="reAuthAccount"
    @close="emit('close-reauth')"
    @reauthorized="emit('updated', $event)"
  />
  <AccountTestModal
    :show="showTest"
    :account="testingAccount"
    @close="emit('close-test')"
    @blacklist="emit('test-blacklist', $event)"
  />
  <AccountStatsModal
    :show="showStats"
    :account="statsAccount"
    @close="emit('close-stats')"
  />
  <ScheduledTestsPanel
    :show="showSchedulePanel"
    :account-id="scheduleAccount?.id ?? null"
    :model-options="scheduleModelOptions"
    @close="emit('close-schedule')"
  />
  <AccountActionMenu
    :show="menuShow"
    :account="menuAccount"
    :position="menuPosition"
    @close="emit('close-menu')"
    @test="emit('test', $event)"
    @stats="emit('stats', $event)"
    @schedule="emit('schedule', $event)"
    @reauth="emit('reauth', $event)"
    @refresh-token="emit('refresh-token', $event)"
    @set-privacy="emit('set-privacy', $event)"
    @recover-state="emit('recover-state', $event)"
    @reset-quota="emit('reset-quota', $event)"
    @import-models="emit('import-models', $event)"
    @blacklist="emit('blacklist', $event)"
  />
  <SyncFromCrsModal
    :show="showSync"
    @close="emit('close-sync')"
    @synced="emit('reload')"
  />
  <ArchiveAccountsModal
    v-if="showArchiveSelected"
    :show="showArchiveSelected"
    :account-ids="selectedIds"
    :selected-platforms="selectedPlatforms"
    @close="emit('close-archive-selected')"
    @archived="emit('archived', $event)"
  />
  <ArchiveGroupAccountsModal
    v-if="showArchiveGroup"
    :show="showArchiveGroup"
    :source-group="archiveSourceGroup"
    @close="emit('close-archive-group')"
    @archived="emit('group-archived', $event)"
  />
  <ImportDataModal
    :show="showImportData"
    @close="emit('close-import-data')"
    @imported="emit('data-imported')"
  />
  <BulkEditAccountModal
    :show="showBulkEdit"
    :account-ids="selectedIds"
    :selected-platforms="selectedPlatforms"
    :selected-types="selectedTypes"
    :proxies="proxies"
    :groups="groups"
    @close="emit('close-bulk-edit')"
    @updated="emit('bulk-updated')"
  />
  <TempUnschedStatusModal
    :show="showTempUnsched"
    :account="tempUnschedAccount"
    @close="emit('close-temp-unsched')"
    @reset="emit('temp-unsched-reset', $event)"
  />
  <ConfirmDialog
    :show="showDeleteDialog"
    :title="t('admin.accounts.deleteAccount')"
    :message="t('admin.accounts.deleteConfirm', { name: deletingAccount?.name })"
    :confirm-text="t('common.delete')"
    :cancel-text="t('common.cancel')"
    :danger="true"
    @confirm="emit('confirm-delete')"
    @cancel="emit('close-delete')"
  />
  <ConfirmDialog
    :show="showExportDataDialog"
    :title="t('admin.accounts.dataExport')"
    :message="t('admin.accounts.dataExportConfirmMessage')"
    :confirm-text="t('admin.accounts.dataExportConfirm')"
    :cancel-text="t('common.cancel')"
    @confirm="emit('confirm-export')"
    @cancel="emit('close-export')"
  >
    <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
      <input
        v-model="includeProxyOnExport"
        type="checkbox"
        class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
      />
      <span>{{ t('admin.accounts.dataExportIncludeProxies') }}</span>
    </label>
  </ConfirmDialog>
  <ErrorPassthroughRulesModal
    :show="showErrorPassthrough"
    @close="emit('close-error-passthrough')"
  />
  <TLSFingerprintProfilesModal
    :show="showTlsFingerprintProfiles"
    @close="emit('close-tls-fingerprint-profiles')"
  />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { SelectOption } from '@/components/common/Select.vue'
import type {
  AccountModelImportResult,
  BlacklistFeedbackPayload
} from '@/api/admin/accounts'
import type { ModelRegistryExposureTarget } from '@/api/admin/modelRegistry'
import type {
  Account,
  AccountPlatform,
  AccountType,
  AdminGroup,
  ArchiveGroupAccountsResult,
  BatchArchiveAccountsResult,
  Proxy
} from '@/types'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import {
  BulkEditAccountModal,
  CreateAccountModal,
  EditAccountModal,
  SyncFromCrsModal,
  TempUnschedStatusModal
} from '@/components/account'
import ErrorPassthroughRulesModal from '@/components/admin/ErrorPassthroughRulesModal.vue'
import TLSFingerprintProfilesModal from '@/components/admin/TLSFingerprintProfilesModal.vue'
import ModelImportExposureSyncDialog from '@/components/admin/models/ModelImportExposureSyncDialog.vue'
import AccountActionMenu from './AccountActionMenu.vue'
import ArchiveAccountsModal from './ArchiveAccountsModal.vue'
import ArchiveGroupAccountsModal from './ArchiveGroupAccountsModal.vue'
import ImportDataModal from './ImportDataModal.vue'
import ReAuthAccountModal from './ReAuthAccountModal.vue'
import AccountTestModal from './AccountTestModal.vue'
import AccountStatsModal from './AccountStatsModal.vue'
import ScheduledTestsPanel from './ScheduledTestsPanel.vue'

defineProps<{
  showCreate: boolean
  showArchiveSelected: boolean
  showArchiveGroup: boolean
  showEdit: boolean
  showSync: boolean
  showImportData: boolean
  showExportDataDialog: boolean
  showBulkEdit: boolean
  showTempUnsched: boolean
  showDeleteDialog: boolean
  showReAuth: boolean
  showTest: boolean
  showStats: boolean
  showErrorPassthrough: boolean
  showTlsFingerprintProfiles: boolean
  showSchedulePanel: boolean
  proxies: Proxy[]
  groups: AdminGroup[]
  selectedIds: number[]
  selectedPlatforms: AccountPlatform[]
  selectedTypes: AccountType[]
  archiveSourceGroup: AdminGroup | null
  editingAccount: Account | null
  tempUnschedAccount: Account | null
  deletingAccount: Account | null
  reAuthAccount: Account | null
  testingAccount: Account | null
  statsAccount: Account | null
  scheduleAccount: Account | null
  scheduleModelOptions: SelectOption[]
  syncDialogOpen: boolean
  syncDialogModels: string[]
  syncDialogSubmitting: boolean
  menuShow: boolean
  menuAccount: Account | null
  menuPosition: { top: number; left: number } | null
}>()

const includeProxyOnExport = defineModel<boolean>('includeProxyOnExport', { required: true })

const emit = defineEmits<{
  'close-create': []
  created: []
  'models-imported': [result: AccountModelImportResult]
  'close-archive-selected': []
  archived: [result: BatchArchiveAccountsResult]
  'close-archive-group': []
  'group-archived': [result: ArchiveGroupAccountsResult]
  'close-sync-dialog': []
  'submit-sync-dialog': [exposures: ModelRegistryExposureTarget[]]
  'close-edit': []
  updated: [account: Account]
  'close-reauth': []
  'close-test': []
  'close-stats': []
  'close-schedule': []
  'close-menu': []
  test: [account: Account]
  stats: [account: Account]
  schedule: [account: Account]
  reauth: [account: Account]
  'refresh-token': [account: Account]
  'set-privacy': [account: Account]
  'recover-state': [account: Account]
  'reset-quota': [account: Account]
  'import-models': [account: Account]
  blacklist: [account: Account]
  'test-blacklist': [payload: { account: Account; source: 'test_modal'; feedback?: BlacklistFeedbackPayload }]
  'close-sync': []
  reload: []
  'close-import-data': []
  'data-imported': []
  'close-bulk-edit': []
  'bulk-updated': []
  'close-temp-unsched': []
  'temp-unsched-reset': [account: Account]
  'confirm-delete': []
  'close-delete': []
  'confirm-export': []
  'close-export': []
  'close-error-passthrough': []
  'close-tls-fingerprint-profiles': []
}>()

const { t } = useI18n()
</script>
