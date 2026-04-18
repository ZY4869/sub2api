import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountModelScopeMappingEditor from '../AccountModelScopeMappingEditor.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const createWrapper = () =>
  mount(AccountModelScopeMappingEditor, {
    props: {
      modelMappings: [{ from: 'friendly-model', to: 'gpt-5.4' }],
      presetMappings: [{ label: 'Preset', from: 'preset', to: 'gpt-5.4', color: 'bg-slate-100 text-slate-700' }],
      getMappingKey: ({ from, to }: { from: string; to: string }) => `${from}:${to}`,
      showActualModelLock: true
    }
  })

describe('AccountModelScopeMappingEditor', () => {
  it('locks actual models by default and hides manual mapping controls', () => {
    const wrapper = createWrapper()

    expect(wrapper.text()).toContain('admin.accounts.actualModelLockLabel')
    expect(wrapper.text()).toContain('admin.accounts.actualModelLockHintLocked')
    expect(wrapper.get('input[placeholder="admin.accounts.actualModel"]').attributes('readonly')).toBeDefined()
    expect(wrapper.text()).not.toContain('admin.accounts.addMapping')
    expect(wrapper.find('button').text()).not.toContain('Preset')
  })

  it('unlocks actual models and restores manual controls when toggled off', async () => {
    const wrapper = createWrapper()

    await wrapper.get('button[type="button"]').trigger('click')

    expect(wrapper.text()).toContain('admin.accounts.actualModelLockHintUnlocked')
    expect(wrapper.get('input[placeholder="admin.accounts.actualModel"]').attributes('readonly')).toBeUndefined()
    expect(wrapper.text()).toContain('admin.accounts.addMapping')
    expect(wrapper.text()).toContain('Preset')
  })

  it('shows the selection-driven hint when locked with no mapping rows', () => {
    const wrapper = mount(AccountModelScopeMappingEditor, {
      props: {
        modelMappings: [],
        presetMappings: [],
        getMappingKey: () => 'mapping-1',
        showActualModelLock: true
      }
    })

    expect(wrapper.text()).toContain('admin.accounts.modelMappingSelectionDrivenHint')
  })
})
