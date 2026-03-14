import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountGroupSettingsEditor from '../AccountGroupSettingsEditor.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const groupSelectorStub = {
  name: 'GroupSelector',
  props: ['modelValue', 'groups', 'platform', 'mixedScheduling'],
  emits: ['update:modelValue'],
  template:
    '<button type="button" data-testid="group-selector" @click="$emit(\'update:modelValue\', [1, 2])">groups</button>'
}

describe('AccountGroupSettingsEditor', () => {
  it('updates mixed scheduling and group ids in editable mode', async () => {
    const wrapper = mount(AccountGroupSettingsEditor, {
      props: {
        groups: [],
        platform: 'antigravity',
        simpleMode: false,
        showMixedScheduling: true,
        mixedSchedulingReadonly: false,
        groupIds: [],
        mixedScheduling: false
      },
      global: {
        stubs: {
          GroupSelector: groupSelectorStub
        }
      }
    })

    await wrapper.get('input[type="checkbox"]').setValue(true)
    await wrapper.get('[data-testid="group-selector"]').trigger('click')

    expect(wrapper.emitted('update:mixedScheduling')?.[0]).toEqual([true])
    expect(wrapper.emitted('update:groupIds')?.[0]).toEqual([[1, 2]])
  })

  it('keeps mixed scheduling read-only when configured', () => {
    const wrapper = mount(AccountGroupSettingsEditor, {
      props: {
        groups: [],
        platform: 'antigravity',
        simpleMode: true,
        showMixedScheduling: true,
        mixedSchedulingReadonly: true,
        groupIds: [],
        mixedScheduling: true
      },
      global: {
        stubs: {
          GroupSelector: groupSelectorStub
        }
      }
    })

    expect(wrapper.get('input[type="checkbox"]').attributes('disabled')).toBeDefined()
    expect(wrapper.find('[data-testid="group-selector"]').exists()).toBe(false)
  })
})
