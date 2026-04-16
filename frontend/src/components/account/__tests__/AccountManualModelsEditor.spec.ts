import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountManualModelsEditor from '../AccountManualModelsEditor.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('AccountManualModelsEditor', () => {
  it('emits provider changes for manual model rows', async () => {
    const wrapper = mount(AccountManualModelsEditor, {
      props: {
        rows: [
          {
            model_id: 'custom-model',
            request_alias: 'Custom Alias'
          }
        ]
      }
    })

    const selects = wrapper.findAll('select')
    expect(selects).toHaveLength(1)

    await selects[0].setValue('grok')

    expect(wrapper.emitted('update:rows')?.at(-1)?.[0]).toEqual([
      {
        model_id: 'custom-model',
        request_alias: 'Custom Alias',
        provider: 'grok'
      }
    ])
  })
})
