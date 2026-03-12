import { ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type {
  ModelRegistryExposureTarget,
  SyncModelRegistryExposuresResult
} from '@/api/admin/modelRegistry'
import type { AccountModelImportResult } from '@/api/admin/accounts'
import { invalidateModelRegistry } from '@/stores/modelRegistry'
import { extractSyncableRegistryModels } from '@/utils/accountModelImport'

interface ToastLikeStore {
  showSuccess: (message: string, options?: number | Record<string, unknown>) => string
  showWarning: (message: string, options?: number | Record<string, unknown>) => string
  showError: (message: string, options?: number | Record<string, unknown>) => string
}

interface InventoryLikeStore {
  invalidate: () => void
}

interface UseModelImportExposureSyncOptions {
  t: (key: string, named?: Record<string, unknown>) => string
  appStore: ToastLikeStore
  modelInventoryStore: InventoryLikeStore
  i18nBaseKey?: string
  onSynced?: (result: SyncModelRegistryExposuresResult) => void | Promise<void>
}

function buildI18nKey(baseKey: string, suffix: string): string {
  return `${baseKey}.${suffix}`
}

function normalizeModels(models: readonly string[]): string[] {
  const uniqueModels = new Set<string>()
  for (const model of models) {
    const value = typeof model === 'string' ? model.trim() : ''
    if (!value) {
      continue
    }
    uniqueModels.add(value)
  }
  return Array.from(uniqueModels)
}

function extractSyncErrorMessage(t: UseModelImportExposureSyncOptions['t'], baseKey: string, error: unknown): string {
  const err = (error || {}) as {
    message?: unknown
    response?: {
      data?: {
        detail?: unknown
        message?: unknown
      }
    }
  }
  const detail = typeof err.response?.data?.detail === 'string' ? err.response.data.detail.trim() : ''
  if (detail) {
    return detail
  }
  const responseMessage = typeof err.response?.data?.message === 'string' ? err.response.data.message.trim() : ''
  if (responseMessage) {
    return responseMessage
  }
  return typeof err.message === 'string' && err.message.trim()
    ? err.message.trim()
    : t(buildI18nKey(baseKey, 'modelImportSyncFailed'))
}

function translateExposureTarget(
  t: UseModelImportExposureSyncOptions['t'],
  baseKey: string,
  target: ModelRegistryExposureTarget
): string {
  return t(buildI18nKey(baseKey, `modelImportSyncTargets.${target}`))
}

function showSyncToast(
  t: UseModelImportExposureSyncOptions['t'],
  baseKey: string,
  appStore: ToastLikeStore,
  result: SyncModelRegistryExposuresResult
): void {
  const targets = result.exposures.map((target) => translateExposureTarget(t, baseKey, target)).join(' / ')
  const details: string[] = [
    t(buildI18nKey(baseKey, 'modelImportSyncAppliedTargets'), { targets })
  ]
  for (const item of result.failed_models || []) {
    details.push(`${item.model} - ${item.error}`)
  }
  const message = t(buildI18nKey(baseKey, 'modelImportSyncSummary'), {
    updated: result.updated_count,
    skipped: result.skipped_count,
    failed: result.failed_count
  })
  const options = {
    title: t(buildI18nKey(baseKey, 'modelImportSyncResultTitle')),
    details,
    persistent: result.failed_count > 0 || result.skipped_count > 0
  }
  if (result.failed_count > 0 || result.skipped_count > 0) {
    appStore.showWarning(message, options)
    return
  }
  appStore.showSuccess(message, options)
}

export function useModelImportExposureSync({
  t,
  appStore,
  modelInventoryStore,
  i18nBaseKey = 'admin.accounts',
  onSynced
}: UseModelImportExposureSyncOptions) {
  const syncDialogOpen = ref(false)
  const syncDialogModels = ref<string[]>([])
  const syncDialogSubmitting = ref(false)

  function openSyncDialogForModels(models: readonly string[]): boolean {
    const normalizedModels = normalizeModels(models)
    if (normalizedModels.length === 0) {
      return false
    }
    syncDialogModels.value = normalizedModels
    syncDialogOpen.value = true
    return true
  }

  function handleImportedModels(result: AccountModelImportResult | null | undefined): boolean {
    return openSyncDialogForModels(extractSyncableRegistryModels(result))
  }

  function closeSyncDialog(force = false): void {
    if (syncDialogSubmitting.value && !force) {
      return
    }
    syncDialogOpen.value = false
    syncDialogModels.value = []
  }

  async function submitSyncDialog(exposures: ModelRegistryExposureTarget[]): Promise<void> {
    if (exposures.length === 0) {
      appStore.showWarning(t(buildI18nKey(i18nBaseKey, 'modelImportSyncNoTarget')))
      return
    }
    if (syncDialogModels.value.length === 0) {
      closeSyncDialog()
      return
    }
    syncDialogSubmitting.value = true
    try {
      const result = await adminAPI.modelRegistry.syncModelRegistryExposures({
        models: syncDialogModels.value,
        exposures
      })
      invalidateModelRegistry()
      modelInventoryStore.invalidate()
      await onSynced?.(result)
      showSyncToast(t, i18nBaseKey, appStore, result)
      closeSyncDialog(true)
    } catch (error) {
      console.error('Failed to sync models to pages:', error)
      appStore.showError(extractSyncErrorMessage(t, i18nBaseKey, error))
    } finally {
      syncDialogSubmitting.value = false
    }
  }

  return {
    syncDialogOpen,
    syncDialogModels,
    syncDialogSubmitting,
    openSyncDialogForModels,
    handleImportedModels,
    closeSyncDialog,
    submitSyncDialog
  }
}
