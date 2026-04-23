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
        showOpenAiImageProtocolMode: true,
        openAiImageProtocolMode: 'native',
        openAiImageProtocolCompatAllowed: true,
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
            template:
              '<button data-test="select" @click="$emit(\'update:modelValue\', options?.[1]?.value ?? modelValue)">select</button>'
          }
        }
      }
    })

    const buttons = wrapper.findAll('button[type="button"]')
    const selects = wrapper.findAll('[data-test="select"]')

    await selects[0]?.trigger('click')
    expect(wrapper.emitted('update:openAiImageProtocolMode')?.[0]).toEqual(['compat'])

    await buttons[0]?.trigger('click')
    expect(wrapper.emitted('update:openAiPassthroughEnabled')?.[0]).toEqual([true])

    await selects[1]?.trigger('click')
    expect(wrapper.emitted('update:openAiWsMode')?.[0]).toEqual(['passthrough'])

    await buttons[1]?.trigger('click')
    expect(wrapper.emitted('update:anthropicPassthroughEnabled')?.[0]).toEqual([true])

    await buttons[2]?.trigger('click')
    expect(wrapper.emitted('update:codexCliOnlyEnabled')?.[0]).toEqual([true])
  })
})
