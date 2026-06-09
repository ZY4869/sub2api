import { mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import ImportAccountGroupBindingModal from '../ImportAccountGroupBindingModal.vue'
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
      bindImportJobGroups: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, number>) =>
        params?.count ? `${key}:${params.count}` : key
    })
  }
})

const accounts = [
  { account_id: 1, name: 'openai-oauth-1', platform: 'openai', type: 'oauth' },
  { account_id: 2, name: 'openai-oauth-2', platform: 'openai', type: 'oauth' },
  { account_id: 3, name: 'openai-key-1', platform: 'openai', type: 'apikey' }
]

const groups = [
  { id: 10, name: 'OpenAI A', platform: 'openai' },
  { id: 11, name: 'OpenAI B', platform: 'openai' }
]

function mountModal() {
  return mount(ImportAccountGroupBindingModal, {
    props: {
      show: true,
      jobId: 'job-1',
      accounts: accounts as any,
      groups: groups as any
    },
    global: {
      stubs: {
        BaseDialog: {
          props: ['show', 'title'],
          emits: ['close'],
          template: '<div v-if="show"><h1>{{ title }}</h1><slot /><slot name="footer" /></div>'
        },
        GroupSelector: {
          props: ['modelValue', 'groups', 'platform'],
          emits: ['update:modelValue'],
          template: `
            <div class="group-selector-stub" :data-platform="platform">
              <div class="selected-groups">{{ modelValue.join(',') }}</div>
              <button class="select-group-10" @click="$emit('update:modelValue', [10])" />
              <button class="select-group-11" @click="$emit('update:modelValue', [11])" />
              <button class="clear-groups" @click="$emit('update:modelValue', [])" />
            </div>
          `
        }
      }
    }
  })
}

describe('ImportAccountGroupBindingModal', () => {
  beforeEach(() => {
    showError.mockReset()
    showSuccess.mockReset()
    vi.mocked(adminAPI.accounts.bindImportJobGroups).mockReset().mockResolvedValue({
      success: 3,
      failed: 0,
      bound_count: 3,
      skipped: 0
    } as any)
  })

  it('groups imported accounts by platform and type, then submits task group binding sections', async () => {
    const wrapper = mountModal()
    const selectors = wrapper.findAll('.group-selector-stub')

    expect(selectors).toHaveLength(2)
    expect(wrapper.text()).toContain('admin.accounts.importGroupBinding.sectionCount:2')
    expect(wrapper.text()).toContain('admin.accounts.importGroupBinding.sectionCount:1')

    await selectors[0].get('.select-group-10').trigger('click')
    await selectors[1].get('.select-group-11').trigger('click')
    await wrapper.find('form').trigger('submit')

    expect(adminAPI.accounts.bindImportJobGroups).toHaveBeenCalledTimes(1)
    expect(adminAPI.accounts.bindImportJobGroups).toHaveBeenCalledWith('job-1', {
      sections: [
        { platform: 'openai', type: 'apikey', group_ids: [10] },
        { platform: 'openai', type: 'oauth', group_ids: [11] }
      ]
    })
    expect(showSuccess).toHaveBeenCalledWith('admin.accounts.importGroupBinding.success:3')
    expect(wrapper.emitted('updated')).toEqual([[]])
    expect(wrapper.emitted('close')).toEqual([[]])
  })

  it('skips sections with no selected groups', async () => {
    const wrapper = mountModal()
    const selectors = wrapper.findAll('.group-selector-stub')

    await selectors[1].get('.select-group-11').trigger('click')
    await wrapper.find('form').trigger('submit')

    expect(adminAPI.accounts.bindImportJobGroups).toHaveBeenCalledTimes(1)
    expect(adminAPI.accounts.bindImportJobGroups).toHaveBeenCalledWith('job-1', {
      sections: [
        { platform: 'openai', type: 'oauth', group_ids: [11] }
      ]
    })
  })

  it('requires at least one selected group before submitting', async () => {
    const wrapper = mountModal()

    await wrapper.find('form').trigger('submit')

    expect(adminAPI.accounts.bindImportJobGroups).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith('admin.accounts.importGroupBinding.noGroupsSelected')
  })
})
