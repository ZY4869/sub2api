import { mount } from '@vue/test-utils'
import { reactive } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import AccountPoolModeEditor from '../AccountPoolModeEditor.vue'
import { createDefaultAccountPoolModeState } from '@/utils/accountApiKeyAdvancedSettings'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountPoolModeEditor', () => {
  it('updates pool mode state through toggle and input', async () => {
    const state = reactive(createDefaultAccountPoolModeState(2))
    const wrapper = mount(AccountPoolModeEditor, {
      props: {
        state,
        defaultRetryCount: 2,
        maxRetryCount: 5
      }
    })

    await wrapper.find('button').trigger('click')
    expect(state.enabled).toBe(true)

    await wrapper.find('input').setValue('4')
    expect(state.retryCount).toBe(4)
  })
})
