<template>
  <ProxyCreateDialog
    :show="showCreateModal"
    v-model:create-mode="createModeModel"
    v-model:create-password-visible="createPasswordVisibleModel"
    v-model:batch-input="batchInputModel"
    :create-form="createForm"
    :protocol-select-options="protocolSelectOptions"
    :batch-parse-result="batchParseResult"
    :submitting="submitting"
    @update:create-form="(value) => emit('update:createForm', value)"
    @close="emit('close-create')"
    @parse-batch="emit('parse-batch')"
    @create="emit('create')"
    @batch-create="emit('batch-create')"
  />

  <ProxyEditDialog
    :show="showEditModal"
    v-model:edit-password-visible="editPasswordVisibleModel"
    :editing-proxy="editingProxy"
    :edit-form="editForm"
    :protocol-select-options="protocolSelectOptions"
    :edit-status-options="editStatusOptions"
    :submitting="submitting"
    @update:edit-form="(value) => emit('update:editForm', value)"
    @close="emit('close-edit')"
    @update="emit('update-proxy')"
    @password-dirty="emit('password-dirty')"
  />

  <ConfirmDialog
    :show="showDeleteDialog"
    :title="t('admin.proxies.deleteProxy')"
    :message="t('admin.proxies.deleteConfirm', { name: deletingProxy?.name })"
    :confirm-text="t('common.delete')"
    :cancel-text="t('common.cancel')"
    :danger="true"
    @confirm="emit('confirm-delete')"
    @cancel="emit('cancel-delete')"
  />

  <ConfirmDialog
    :show="showBatchDeleteDialog"
    :title="t('admin.proxies.batchDelete')"
    :message="t('admin.proxies.batchDeleteConfirm', { count: selectedCount })"
    :confirm-text="t('common.delete')"
    :cancel-text="t('common.cancel')"
    :danger="true"
    @confirm="emit('confirm-batch-delete')"
    @cancel="emit('cancel-batch-delete')"
  />

  <ConfirmDialog
    :show="showExportDataDialog"
    :title="t('admin.proxies.dataExport')"
    :message="t('admin.proxies.dataExportConfirmMessage')"
    :confirm-text="t('admin.proxies.dataExportConfirm')"
    :cancel-text="t('common.cancel')"
    @confirm="emit('export-data')"
    @cancel="emit('cancel-export')"
  />

  <ImportDataModal
    :show="showImportData"
    @close="emit('close-import')"
    @imported="emit('imported')"
  />

  <ProxyQualityReportDialog
    :show="showQualityReportDialog"
    :quality-report-proxy="qualityReportProxy"
    :quality-report="qualityReport"
    :locale="locale"
    :quality-status-class="qualityStatusClass"
    :quality-status-label="qualityStatusLabel"
    :quality-target-label="qualityTargetLabel"
    @close="emit('close-quality-report')"
  />

  <ProxyAccountsDialog
    :show="showAccountsModal"
    :accounts-proxy="accountsProxy"
    :proxy-accounts="proxyAccounts"
    :accounts-loading="accountsLoading"
    @close="emit('close-accounts')"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Proxy, ProxyAccountSummary, ProxyProtocol, ProxyQualityCheckResult } from '@/types'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import ImportDataModal from '@/components/admin/proxy/ImportDataModal.vue'
import ProxyAccountsDialog from './ProxyAccountsDialog.vue'
import ProxyCreateDialog from './ProxyCreateDialog.vue'
import ProxyEditDialog from './ProxyEditDialog.vue'
import ProxyQualityReportDialog from './ProxyQualityReportDialog.vue'

interface ProxyForm {
  name: string
  protocol: ProxyProtocol
  host: string
  port: number
  username: string
  password: string
}

interface ProxyEditForm extends ProxyForm {
  status: 'active' | 'inactive'
}

interface BatchParseResult {
  total: number
  valid: number
  invalid: number
  duplicate: number
}

const props = defineProps<{
  showCreateModal: boolean
  createMode: 'standard' | 'batch'
  createPasswordVisible: boolean
  batchInput: string
  createForm: ProxyForm
  protocolSelectOptions: Array<{ value: string; label: string }>
  batchParseResult: BatchParseResult
  submitting: boolean
  showEditModal: boolean
  editPasswordVisible: boolean
  editingProxy: Proxy | null
  editForm: ProxyEditForm
  editStatusOptions: Array<{ value: string; label: string }>
  showDeleteDialog: boolean
  deletingProxy: Proxy | null
  showBatchDeleteDialog: boolean
  selectedCount: number
  showExportDataDialog: boolean
  showImportData: boolean
  showQualityReportDialog: boolean
  qualityReportProxy: Proxy | null
  qualityReport: ProxyQualityCheckResult | null
  locale: string
  qualityStatusClass: (status: string) => string
  qualityStatusLabel: (status: string) => string
  qualityTargetLabel: (target: string) => string
  showAccountsModal: boolean
  accountsProxy: Proxy | null
  proxyAccounts: ProxyAccountSummary[]
  accountsLoading: boolean
}>()

const emit = defineEmits<{
  'update:createMode': [mode: 'standard' | 'batch']
  'update:createPasswordVisible': [value: boolean]
  'update:batchInput': [value: string]
  'update:editPasswordVisible': [value: boolean]
  'update:createForm': [value: ProxyForm]
  'update:editForm': [value: ProxyEditForm]
  'close-create': []
  'parse-batch': []
  create: []
  'batch-create': []
  'close-edit': []
  'update-proxy': []
  'password-dirty': []
  'confirm-delete': []
  'cancel-delete': []
  'confirm-batch-delete': []
  'cancel-batch-delete': []
  'export-data': []
  'cancel-export': []
  'close-import': []
  imported: []
  'close-quality-report': []
  'close-accounts': []
}>()

const { t } = useI18n()

const createModeModel = computed({
  get: () => props.createMode,
  set: (value: 'standard' | 'batch') => emit('update:createMode', value)
})

const createPasswordVisibleModel = computed({
  get: () => props.createPasswordVisible,
  set: (value: boolean) => emit('update:createPasswordVisible', value)
})

const batchInputModel = computed({
  get: () => props.batchInput,
  set: (value: string) => emit('update:batchInput', value)
})

const editPasswordVisibleModel = computed({
  get: () => props.editPasswordVisible,
  set: (value: boolean) => emit('update:editPasswordVisible', value)
})
</script>
