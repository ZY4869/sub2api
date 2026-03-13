import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountTempUnschedRulesEditor from '../AccountTempUnschedRulesEditor.vue'
import type { TempUnschedRuleForm } from '@/utils/accountFormShared'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountTempUnschedRulesEditor', () => {
  it('emits enable toggle and add-rule events', async () => {
    const rules: TempUnschedRuleForm[] = []
    const wrapper = mount(AccountTempUnschedRulesEditor, {
      props: {
        enabled: false,
        rules,
        presets: [
          {
            label: 'Preset A',
            rule: {
              error_code: 429,
              keywords: 'rate limit',
              duration_minutes: 10,
              description: 'preset'
            }
          }
        ],
        getRuleKey: () => 'rule-1'
      },
      global: {
        stubs: {
          Icon: true
        }
      }
    })

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('update:enabled')?.[0]).toEqual([true])

    await wrapper.setProps({ enabled: true })
    const buttons = wrapper.findAll('button')
    await buttons[1]?.trigger('click')
    expect(wrapper.emitted('add-rule')?.[0]?.[0]).toEqual({
      error_code: 429,
      keywords: 'rate limit',
      duration_minutes: 10,
      description: 'preset'
    })
  })
})
