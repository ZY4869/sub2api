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

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportParseFailedFile')
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
      skip_default_group_bind: true,
      account_defaults: {
        openai_tier: 'pro_5x',
        claude_tier: 'pro'
      }
    })
    expect(adminAPI.accounts.getImportJob).toHaveBeenCalledWith('job-1')
    expect(wrapper.emitted('imported')).toEqual([[importJob]])
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.dataImportSuccess')
  })

  it('支持选择多个 JSON 文件并合并账号与代理后创建导入任务', async () => {
    const importResult = {
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 2,
      account_failed: 0,
      created_accounts: [
        { account_id: 11, name: 'OpenAI 11', platform: 'openai', type: 'apikey' }
      ]
    }
    const importJob = {
      job_id: 'job-2',
      status: 'succeeded',
      progress: { total: 2, processed: 2 },
      result: importResult,
      created_accounts_summary: importResult.created_accounts,
      cancel_requested: false,
      created_at: '2026-06-09T00:00:00Z',
      updated_at: '2026-06-09T00:00:01Z'
    }
    vi.mocked(adminAPI.accounts.createImportJob).mockResolvedValue({ job_id: 'job-2' } as any)
    vi.mocked(adminAPI.accounts.getImportJob).mockResolvedValue(importJob as any)

    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const payloadA = {
      type: 'sub2api_admin_export',
      version: 1,
      exported_at: '2026-06-09T00:00:00Z',
      proxies: [{ proxy_key: 'proxy-a' }],
      accounts: [{ name: 'account-a' }]
    }
    const payloadB = {
      exported_at: '2026-06-10T00:00:00Z',
      proxies: [{ proxy_key: 'proxy-b' }],
      accounts: [{ name: 'account-b' }]
    }
    const fileA = new File([JSON.stringify(payloadA)], 'a.json', { type: 'application/json' })
    const fileB = new File([JSON.stringify(payloadB)], 'b.json', { type: 'application/json' })
    Object.defineProperty(fileA, 'text', {
      value: () => Promise.resolve(JSON.stringify(payloadA))
    })
    Object.defineProperty(fileB, 'text', {
      value: () => Promise.resolve(JSON.stringify(payloadB))
    })

    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [fileA, fileB]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(adminAPI.accounts.createImportJob).toHaveBeenCalledWith({
      data: expect.objectContaining({
        type: 'sub2api_admin_export',
        version: 1,
        proxies: [{ proxy_key: 'proxy-a' }, { proxy_key: 'proxy-b' }],
        accounts: [{ name: 'account-a' }, { name: 'account-b' }]
      }),
      skip_default_group_bind: true,
      account_defaults: {
        openai_tier: 'pro_5x',
        claude_tier: 'pro'
      }
    })
  })
})
