import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ImportDataModal from '@/components/admin/account/ImportDataModal.vue'
import { adminAPI } from '@/api/admin'

const showError = vi.fn()
const showSuccess = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      createImportJob: vi.fn(),
      getImportJob: vi.fn(),
      cancelImportJob: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('ImportDataModal', () => {
  beforeEach(() => {
    showError.mockReset()
    showSuccess.mockReset()
    vi.mocked(adminAPI.accounts.createImportJob).mockReset()
    vi.mocked(adminAPI.accounts.getImportJob).mockReset()
    vi.mocked(adminAPI.accounts.cancelImportJob).mockReset()
  })

  it('未选择文件时提示错误', async () => {
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    await wrapper.find('form').trigger('submit')
    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportSelectFile')
  })

  it('无效 JSON 时提示解析失败', async () => {
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const input = wrapper.find('input[type="file"]')
    const file = new File(['invalid json'], 'data.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve('invalid json')
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportParseFailed')
  })

  it('导入成功时创建任务并携带完整任务结果事件', async () => {
    const importResult = {
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0,
      created_accounts: [
        { account_id: 11, name: 'OpenAI 11', platform: 'openai', type: 'apikey' }
      ]
    }
    const importJob = {
      job_id: 'job-1',
      status: 'succeeded',
      progress: { total: 1, processed: 1 },
      result: importResult,
      created_accounts_summary: importResult.created_accounts,
      cancel_requested: false,
      created_at: '2026-06-09T00:00:00Z',
      updated_at: '2026-06-09T00:00:01Z'
    }
    vi.mocked(adminAPI.accounts.createImportJob).mockResolvedValue({ job_id: 'job-1' } as any)
    vi.mocked(adminAPI.accounts.getImportJob).mockResolvedValue(importJob as any)

    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const input = wrapper.find('input[type="file"]')
    const file = new File([
      JSON.stringify({ exported_at: '2026-06-09T00:00:00Z', proxies: [], accounts: [] })
    ], 'data.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve(JSON.stringify({
        exported_at: '2026-06-09T00:00:00Z',
        proxies: [],
        accounts: []
      }))
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(adminAPI.accounts.createImportJob).toHaveBeenCalledWith({
      data: {
        exported_at: '2026-06-09T00:00:00Z',
        proxies: [],
        accounts: []
      },
      skip_default_group_bind: true
    })
    expect(adminAPI.accounts.getImportJob).toHaveBeenCalledWith('job-1')
    expect(wrapper.emitted('imported')).toEqual([[importJob]])
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.dataImportSuccess')
  })
})
