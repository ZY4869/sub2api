import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountAntigravityModelMappingEditor from '../AccountAntigravityModelMappingEditor.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountAntigravityModelMappingEditor', () => {
  it('renders wildcard validation and emits mapping actions', async () => {
    const wrapper = mount(AccountAntigravityModelMappingEditor, {
      props: {
        modelMappings: [{ from: 'claude*test', to: 'claude-*' }],
        presetMappings: [
          {
            label: 'Preset A',
            from: 'claude-*',
            to: 'claude-sonnet-4.5',
            color: 'bg-gray-100 text-gray-700'
          }
        ],
        getMappingKey: () => 'antigravity-1'
      }
    })

    expect(wrapper.text()).toContain('admin.accounts.wildcardOnlyAtEnd')
    expect(wrapper.text()).toContain('admin.accounts.targetNoWildcard')

    const buttons = wrapper.findAll('button')
    await buttons[0]?.trigger('click')
    expect(wrapper.emitted('remove-mapping')?.[0]).toEqual([0])

    await buttons[1]?.trigger('click')
    expect(wrapper.emitted('add-mapping')?.[0]).toEqual([])

    await buttons[2]?.trigger('click')
    expect(wrapper.emitted('add-preset')?.[0]?.[0]).toEqual({
      from: 'claude-*',
      to: 'claude-sonnet-4.5'
    })
  })
})
