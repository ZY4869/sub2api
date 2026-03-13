import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountModelRestrictionEditor from '../AccountModelRestrictionEditor.vue'
import type { ModelMapping } from '@/utils/accountFormShared'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const ModelWhitelistSelectorStub = {
  name: 'ModelWhitelistSelector',
  props: ['modelValue', 'platform'],
  template: '<button class="selector" @click="$emit(\'update:modelValue\', [\'gpt-5.4\'])">selector</button>'
}

describe('AccountModelRestrictionEditor', () => {
  it('emits mode and whitelist updates', async () => {
    const wrapper = mount(AccountModelRestrictionEditor, {
      props: {
        platform: 'openai',
        mode: 'whitelist',
        allowedModels: [],
        modelMappings: [],
        presetMappings: [],
        getMappingKey: () => 'mapping-1',
        variant: 'simple'
      },
      global: {
        stubs: {
          ModelWhitelistSelector: ModelWhitelistSelectorStub
        }
      }
    })

    const buttons = wrapper.findAll('button')
    await buttons[1]?.trigger('click')
    expect(wrapper.emitted('update:mode')?.[0]).toEqual(['mapping'])

    await wrapper.find('button.selector').trigger('click')
    expect(wrapper.emitted('update:allowedModels')?.[0]).toEqual([['gpt-5.4']])
  })

  it('emits mapping actions', async () => {
    const mappings: ModelMapping[] = [{ from: 'gpt-4.1', to: 'gpt-5.4' }]
    const wrapper = mount(AccountModelRestrictionEditor, {
      props: {
        platform: 'openai',
        mode: 'mapping',
        allowedModels: [],
        modelMappings: mappings,
        presetMappings: [
          {
            label: 'Preset A',
            from: 'gpt-5',
            to: 'gpt-5.4',
            color: 'bg-gray-100 text-gray-700'
          }
        ],
        getMappingKey: () => 'mapping-1'
      },
      global: {
        stubs: {
          ModelWhitelistSelector: ModelWhitelistSelectorStub
        }
      }
    })

    const buttons = wrapper.findAll('button')
    await buttons[2]?.trigger('click')
    expect(wrapper.emitted('remove-mapping')?.[0]).toEqual([0])

    await buttons[3]?.trigger('click')
    expect(wrapper.emitted('add-mapping')?.[0]).toEqual([])

    await buttons[4]?.trigger('click')
    expect(wrapper.emitted('add-preset')?.[0]?.[0]).toEqual({
      from: 'gpt-5',
      to: 'gpt-5.4'
    })
  })
})
