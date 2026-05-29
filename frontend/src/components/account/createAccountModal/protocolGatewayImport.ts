import { adminAPI } from '@/api/admin'
import type { AccountModelImportResult } from '@/api/admin/accounts'
import type { Account } from '@/types'
import { invalidateModelRegistry } from '@/stores/modelRegistry'
import { extractSyncableRegistryModels } from '@/utils/accountModelImport'

interface SyncProtocolGatewaySelectedModelsOptions {
  createdAccount: Account
  selectedModels: Array<{ id: string }>
  emitModelsImported: (result: AccountModelImportResult) => void
  invalidateModelInventory: () => void
  showPartialWarning: (failed: number) => void
  showFailedWarning: (message?: string) => void
}

export async function syncProtocolGatewaySelectedModels({
  createdAccount,
  selectedModels,
  emitModelsImported,
  invalidateModelInventory,
  showPartialWarning,
  showFailedWarning
}: SyncProtocolGatewaySelectedModelsOptions) {
  if (!selectedModels.length) {
    return
  }
  try {
    const result = await adminAPI.accounts.importModels(createdAccount.id, {
      trigger: 'create',
      models: selectedModels.map((model) => model.id)
    })
    emitModelsImported(result)
    const syncableModels = extractSyncableRegistryModels(result)
    if (syncableModels.length > 0) {
      await adminAPI.modelRegistry.syncModelRegistryExposures({
        models: syncableModels,
        exposures: ['runtime', 'test', 'whitelist'],
        mode: 'add'
      })
      invalidateModelRegistry()
      invalidateModelInventory()
    }
    const failedCount = result.failed_models?.length || 0
    if (failedCount > 0) {
      showPartialWarning(failedCount)
    }
  } catch (error: any) {
    console.error('Failed to sync selected protocol gateway models:', error)
    showFailedWarning(error?.message)
  }
}
