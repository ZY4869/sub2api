import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  syncModelRegistryExposures: vi.fn(),
  invalidateModelRegistry: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    modelRegistry: {
      syncModelRegistryExposures: mocks.syncModelRegistryExposures
    }
  }
}))

vi.mock('@/stores/modelRegistry', () => ({
  invalidateModelRegistry: mocks.invalidateModelRegistry
}))

import { useModelImportExposureSync } from '../useModelImportExposureSync'

describe('useModelImportExposureSync', () => {
  const t = (key: string, named?: Record<string, unknown>) => {
    if (named?.count != null) {
      return `${key}:${named.count}`
    }
    if (named?.updated != null) {
      return `${key}:${named.updated}/${named.skipped}/${named.failed}`
    }
    if (named?.targets != null) {
      return `${key}:${named.targets}`
    }
    return key
  }

  const appStore = {
    showSuccess: vi.fn(() => 'success'),
    showWarning: vi.fn(() => 'warning'),
    showError: vi.fn(() => 'error')
  }

  const modelInventoryStore = {
    invalidate: vi.fn()
  }

  beforeEach(() => {
    mocks.syncModelRegistryExposures.mockReset()
    mocks.invalidateModelRegistry.mockReset()
    appStore.showSuccess.mockClear()
    appStore.showWarning.mockClear()
    appStore.showError.mockClear()
    modelInventoryStore.invalidate.mockClear()
  })

  it('deduplicates raw models before opening sync dialog', () => {
    const sync = useModelImportExposureSync({
      t,
      appStore,
      modelInventoryStore,
      i18nBaseKey: 'admin.models.registry'
    })

    const opened = sync.openSyncDialogForModels([' model-a ', '', 'model-a', 'model-b'])

    expect(opened).toBe(true)
    expect(sync.syncDialogOpen.value).toBe(true)
    expect(sync.syncDialogModels.value).toEqual(['model-a', 'model-b'])
  })

  it('submits selected models and runs success side effects', async () => {
    const onSynced = vi.fn()
    mocks.syncModelRegistryExposures.mockResolvedValue({
      exposures: ['whitelist', 'runtime'],
      updated_count: 2,
      skipped_count: 0,
      failed_count: 0,
      updated_models: ['model-a', 'model-b']
    })

    const sync = useModelImportExposureSync({
      t,
      appStore,
      modelInventoryStore,
      i18nBaseKey: 'admin.models.registry',
      onSynced
    })

    sync.openSyncDialogForModels(['model-a', 'model-b'])
    await sync.submitSyncDialog(['whitelist', 'runtime'])

    expect(mocks.syncModelRegistryExposures).toHaveBeenCalledWith({
      models: ['model-a', 'model-b'],
      exposures: ['whitelist', 'runtime']
    })
    expect(mocks.invalidateModelRegistry).toHaveBeenCalledTimes(1)
    expect(modelInventoryStore.invalidate).toHaveBeenCalledTimes(1)
    expect(onSynced).toHaveBeenCalledTimes(1)
    expect(appStore.showSuccess).toHaveBeenCalledTimes(1)
    expect(sync.syncDialogOpen.value).toBe(false)
    expect(sync.syncDialogModels.value).toEqual([])
  })
})
