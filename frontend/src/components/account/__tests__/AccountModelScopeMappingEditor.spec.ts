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
      allowedModels: ['gpt-5.4'],
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

  it('renders selection-driven rows from allowed models even without explicit mappings', () => {
    const wrapper = mount(AccountModelScopeMappingEditor, {
      props: {
        allowedModels: ['gpt-5.4'],
        modelMappings: [],
        presetMappings: [],
        getMappingKey: ({ from, to }: { from: string; to: string }) => `${from}:${to}`,
        showActualModelLock: true
      }
    })

    const inputs = wrapper.findAll('input')
    expect(inputs).toHaveLength(2)
    expect((inputs[0].element as HTMLInputElement).value).toBe('gpt-5.4')
    expect((inputs[1].element as HTMLInputElement).value).toBe('gpt-5.4')
  })

  it('clears explicit aliases when a locked alias is reset back to the target model', async () => {
    const wrapper = createWrapper()

    await wrapper.get('input[placeholder="admin.accounts.requestModel"]').setValue('gpt-5.4')

    expect(wrapper.emitted('update:modelMappings')?.at(-1)).toEqual([[]])
  })

  it('removes the selected target model and any explicit alias when deleting a locked row', async () => {
    const wrapper = createWrapper()

    await wrapper.findAll('button[type="button"]').at(-1)?.trigger('click')

    expect(wrapper.emitted('update:allowedModels')?.at(-1)).toEqual([[]])
    expect(wrapper.emitted('update:modelMappings')?.at(-1)).toEqual([[]])
  })

  it('unlocks actual models and restores manual controls when toggled off', async () => {
    const wrapper = createWrapper()

    await wrapper.get('button[type="button"]').trigger('click')

    expect(wrapper.text()).toContain('admin.accounts.actualModelLockHintUnlocked')
    expect(wrapper.get('input[placeholder="admin.accounts.actualModel"]').attributes('readonly')).toBeUndefined()
    expect(wrapper.text()).toContain('admin.accounts.addMapping')
    expect(wrapper.text()).toContain('Preset')
  })

  it('keeps manual add and preset controls in unlocked mode', async () => {
    const wrapper = createWrapper()

    await wrapper.get('button[type="button"]').trigger('click')
    await wrapper.get('button[class*="border-dashed"]').trigger('click')
    await wrapper.get('button[class*="bg-slate-100"]').trigger('click')

    expect(wrapper.emitted('add-mapping')).toHaveLength(1)
    expect(wrapper.emitted('add-preset')?.at(-1)).toEqual([{ from: 'preset', to: 'gpt-5.4' }])
  })

  it('shows the selection-driven hint when locked with no mapping rows', () => {
    const wrapper = mount(AccountModelScopeMappingEditor, {
      props: {
        allowedModels: [],
        modelMappings: [],
        presetMappings: [],
        getMappingKey: () => 'mapping-1',
        showActualModelLock: true
      }
    })

    expect(wrapper.text()).toContain('admin.accounts.modelMappingSelectionDrivenHint')
  })
})
