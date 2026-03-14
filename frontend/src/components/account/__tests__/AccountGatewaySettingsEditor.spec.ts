import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountGatewaySettingsEditor from '../AccountGatewaySettingsEditor.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountGatewaySettingsEditor', () => {
  it('emits gateway setting updates', async () => {
    const wrapper = mount(AccountGatewaySettingsEditor, {
      props: {
        showOpenAiPassthrough: true,
        openAiPassthroughEnabled: false,
        showOpenAiWsMode: true,
        openAiWsMode: 'off',
        openAiWsModeOptions: [
          { value: 'off', label: 'Off' },
          { value: 'passthrough', label: 'Passthrough' }
        ],
        openAiWsModeConcurrencyHintKey: 'admin.accounts.openai.wsModeHint',
        showAnthropicPassthrough: true,
        anthropicPassthroughEnabled: false,
        showCodexCliOnly: true,
        codexCliOnlyEnabled: false
      },
      global: {
        stubs: {
          Select: {
            props: ['modelValue', 'options'],
            template: '<button data-test="ws-mode" @click="$emit(\'update:modelValue\', \'passthrough\')">select</button>'
          }
        }
      }
    })

    const buttons = wrapper.findAll('button')
    await buttons[0]?.trigger('click')
    expect(wrapper.emitted('update:openAiPassthroughEnabled')?.[0]).toEqual([true])

    await wrapper.get('[data-test="ws-mode"]').trigger('click')
    expect(wrapper.emitted('update:openAiWsMode')?.[0]).toEqual(['passthrough'])

    await buttons[2]?.trigger('click')
    expect(wrapper.emitted('update:anthropicPassthroughEnabled')?.[0]).toEqual([true])

    await buttons[3]?.trigger('click')
    expect(wrapper.emitted('update:codexCliOnlyEnabled')?.[0]).toEqual([true])
  })
})
