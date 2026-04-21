import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getModelRegistryMock } = vi.hoisted(() => ({
  getModelRegistryMock: vi.fn()
}))

vi.mock('@/api/meta', () => ({
  metaAPI: {
    getModelRegistry: getModelRegistryMock
  }
}))

import { generatedModelRegistrySnapshot } from '@/generated/modelRegistry'
import {
  ensureModelRegistryFresh,
  getModelRegistrySnapshot,
  invalidateModelRegistry,
  resetModelRegistryStoreForTests
} from '../modelRegistry'

describe('modelRegistry store', () => {
  beforeEach(() => {
    getModelRegistryMock.mockReset()
    resetModelRegistryStoreForTests()
  })

  it('does not fetch when reading the snapshot only', () => {
    const snapshot = getModelRegistrySnapshot()

    expect(snapshot).toEqual(generatedModelRegistrySnapshot)
    expect(getModelRegistryMock).not.toHaveBeenCalled()
  })

  it('fetches only when ensureModelRegistryFresh is called explicitly', async () => {
    getModelRegistryMock.mockResolvedValueOnce({
      notModified: false,
      etag: 'fresh-etag',
      data: {
        ...generatedModelRegistrySnapshot,
        etag: 'fresh-etag',
        updated_at: '2026-04-21T00:00:00Z'
      }
    })

    const snapshot = await ensureModelRegistryFresh()

    expect(getModelRegistryMock).toHaveBeenCalledTimes(1)
    expect(snapshot.etag).toBe('fresh-etag')
  })

  it('keeps invalidateModelRegistry side-effect free until the next explicit ensure', async () => {
    getModelRegistryMock.mockResolvedValueOnce({
      notModified: true,
      etag: generatedModelRegistrySnapshot.etag || null,
      data: null
    })
    await ensureModelRegistryFresh()
    expect(getModelRegistryMock).toHaveBeenCalledTimes(1)

    getModelRegistryMock.mockClear()
    invalidateModelRegistry()

    getModelRegistrySnapshot()
    expect(getModelRegistryMock).not.toHaveBeenCalled()

    getModelRegistryMock.mockResolvedValueOnce({
      notModified: true,
      etag: generatedModelRegistrySnapshot.etag || null,
      data: null
    })
    await ensureModelRegistryFresh()
    expect(getModelRegistryMock).toHaveBeenCalledTimes(1)
  })
})
