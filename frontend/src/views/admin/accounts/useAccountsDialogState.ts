import { computed, ref } from 'vue'
import type { Ref } from 'vue'
import type { Account, AccountPlatform, AccountType, SelectOption } from '@/types'
import type {
  AccountModelDiagnosticsResponse,
  BulkUpdateAccountsFilters
} from '@/api/admin/accounts'

export function useAccountsDialogState(
  accounts: Ref<Account[]>,
  isSelected: (id: number) => boolean
) {
  const showCreate = ref(false)
  const showArchiveSelected = ref(false)
  const showEdit = ref(false)
  const editLoading = ref(false)
  const showSync = ref(false)
  const showImportData = ref(false)
  const showExportDataDialog = ref(false)
  const includeProxyOnExport = ref(true)
  const showBulkEdit = ref(false)
  const bulkEditFilters = ref<BulkUpdateAccountsFilters | null>(null)
  const bulkEditFiltersTotal = ref<number | null>(null)
  const showTempUnsched = ref(false)
  const showDeleteDialog = ref(false)
  const showReAuth = ref(false)
  const showTest = ref(false)
  const showBatchTest = ref(false)
  const showStats = ref(false)
  const showModelDiagnostics = ref(false)
  const showErrorPassthrough = ref(false)
  const showTLSFingerprintProfiles = ref(false)
  const showDaily5HTriggerSettings = ref(false)

  const edAcc = ref<Account | null>(null)
  const tempUnschedAcc = ref<Account | null>(null)
  const deletingAcc = ref<Account | null>(null)
  const reAuthAcc = ref<Account | null>(null)
  const testingAcc = ref<Account | null>(null)
  const batchTestAccounts = ref<Account[]>([])
  const batchTestDefaultTestMode = ref<'real_forward' | 'health_check'>('health_check')
  const batchTestDefaultModelStrategy = ref<'auto' | 'specified'>('auto')
  const statsAcc = ref<Account | null>(null)
  const diagnosticsAccount = ref<Account | null>(null)
  const diagnosticsResult = ref<AccountModelDiagnosticsResponse | null>(null)
  const diagnosticsLoading = ref(false)
  const showSchedulePanel = ref(false)
  const scheduleAcc = ref<Account | null>(null)
  const scheduleModelOptions = ref<SelectOption[]>([])
  const activeEditRequestToken = ref(0)
  const activeEditAbortController = ref<AbortController | null>(null)
  const togglingSchedulable = ref<number | null>(null)
  const exportingData = ref(false)
  const usageManualRefreshToken = ref(0)
  const usageRefreshing = ref(false)
  const daily5HTriggerSettingsLoading = ref(false)
  const daily5HTriggerSettingsSaving = ref(false)
  const archivedPanelRefreshToken = ref(0)

  const selPlatforms = computed<AccountPlatform[]>(() => [
    ...new Set(accounts.value.filter((account) => isSelected(account.id)).map((account) => account.platform))
  ])
  const selTypes = computed<AccountType[]>(() => [
    ...new Set(accounts.value.filter((account) => isSelected(account.id)).map((account) => account.type))
  ])
  const bulkEditSelectedPlatforms = computed<AccountPlatform[]>(() => {
    const platform = String(bulkEditFilters.value?.platform || '').trim()
    return platform ? [platform as AccountPlatform] : selPlatforms.value
  })
  const bulkEditSelectedTypes = computed<AccountType[]>(() => {
    const type = String(bulkEditFilters.value?.type || '').trim()
    return type ? [type as AccountType] : selTypes.value
  })

  return {
    showCreate, showArchiveSelected, showEdit, editLoading, showSync,
    showImportData, showExportDataDialog, includeProxyOnExport, showBulkEdit,
    bulkEditFilters, bulkEditFiltersTotal, showTempUnsched, showDeleteDialog,
    showReAuth, showTest, showBatchTest, showStats, showModelDiagnostics,
    showErrorPassthrough, showTLSFingerprintProfiles, showDaily5HTriggerSettings,
    edAcc, tempUnschedAcc, deletingAcc, reAuthAcc, testingAcc,
    batchTestAccounts, batchTestDefaultTestMode, batchTestDefaultModelStrategy,
    statsAcc, diagnosticsAccount, diagnosticsResult, diagnosticsLoading,
    showSchedulePanel, scheduleAcc, scheduleModelOptions, activeEditRequestToken,
    activeEditAbortController,
    togglingSchedulable, exportingData, usageManualRefreshToken, usageRefreshing,
    daily5HTriggerSettingsLoading, daily5HTriggerSettingsSaving,
    archivedPanelRefreshToken, selPlatforms, selTypes, bulkEditSelectedPlatforms,
    bulkEditSelectedTypes
  }
}
