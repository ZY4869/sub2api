import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountGrokImportPanel from '../AccountGrokImportPanel.vue'

const { previewGrokImportMock, importGrokMock, showErrorMock, showInfoMock, showSuccessMock } = vi.hoisted(() => ({
  previewGrokImportMock: vi.fn(),
  importGrokMock: vi.fn(),
  showErrorMock: vi.fn(),
  showInfoMock: vi.fn(),
  showSuccessMock: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      previewGrokImport: previewGrokImportMock,
      importGrok: importGrokMock
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showInfo: showInfoMock,
    showSuccess: showSuccessMock
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, number | string>) => {
        if (key === 'admin.accounts.grokImport.importCounts') {
          return `Created ${params?.created} · Skipped ${params?.skipped} · Failed ${params?.failed}`
        }
        if (key === 'admin.accounts.grokImport.previewCounts') {
          return `Total ${params?.total} · Ready ${params?.ready} · Skipped ${params?.skipped} · Failed ${params?.failed}`
        }
        if (key === 'admin.accounts.grokImport.detectedKind') {
          return `Detected: ${params?.kind}`
        }
        return key
      }
    })
  }
})

describe('AccountGrokImportPanel', () => {
  beforeEach(() => {
    previewGrokImportMock.mockReset()
    importGrokMock.mockReset()
    showErrorMock.mockReset()
    showInfoMock.mockReset()
    showSuccessMock.mockReset()
  })

  it('runs Grok preview before import and emits imported results', async () => {
    previewGrokImportMock.mockResolvedValue({
      detected_kind: 'sso',
      total: 1,
      items: [
        {
          index: 1,
          name: 'grok-sso-1',
          type: 'sso',
          detected_kind: 'sso',
          credential_masked: 'abcd...wxyz',
          grok_tier: 'basic',
          priority: 50,
          concurrency: 10,
          status: 'ready'
        }
      ]
    })
    importGrokMock.mockResolvedValue({
      detected_kind: 'sso',
      created: 1,
      skipped: 0,
      failed: 0,
      results: [
        {
          index: 1,
          name: 'grok-sso-1',
          type: 'sso',
          status: 'created',
          account_id: 1001
        }
      ]
    })

    const wrapper = mount(AccountGrokImportPanel, {
      props: {
        show: true
      }
    })

    await wrapper.get('textarea').setValue('Bearer test-token')
    await wrapper.get('button.btn-secondary').trigger('click')
    await flushPromises()

    expect(previewGrokImportMock).toHaveBeenCalledWith({
      content: 'Bearer test-token',
      skip_default_group_bind: false
    })
    expect(wrapper.text()).toContain('Detected: sso')
    expect(wrapper.text()).toContain('Total 1 · Ready 1 · Skipped 0 · Failed 0')

    const importButton = wrapper.findAll('button').find((button) => button.text().includes('common.import'))
    expect(importButton).toBeTruthy()

    await importButton!.trigger('click')
    await flushPromises()

    expect(importGrokMock).toHaveBeenCalledWith({
      content: 'Bearer test-token',
      skip_default_group_bind: false
    })
    expect(wrapper.emitted('imported')?.[0]?.[0]).toMatchObject({
      created: 1,
      results: [{ account_id: 1001 }]
    })
    expect(showSuccessMock).toHaveBeenCalledWith('Created 1 · Skipped 0 · Failed 0')
  })
})
