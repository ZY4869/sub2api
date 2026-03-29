import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountKiroTokenImportPanel from '../AccountKiroTokenImportPanel.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountKiroTokenImportPanel', () => {
  it('emits imported credentials with membership metadata', async () => {
    const wrapper = mount(AccountKiroTokenImportPanel, {
      props: {
        submitLabel: '导入',
        submitting: false
      }
    })

    await wrapper.find('textarea').setValue(JSON.stringify({
      access_token: 'at',
      refresh_token: 'rt',
      email: 'kiro@example.com',
      provider: 'aws'
    }))
    await wrapper.find('select').setValue('kiro_pro_plus')
    await wrapper.find('input[type="number"]').setValue('2345')
    await wrapper.find('button').trigger('click')

    expect(wrapper.emitted('submit')?.[0]?.[0]).toEqual({
      credentials: {
        access_token: 'at',
        refresh_token: 'rt'
      },
      extra: {
        provider: 'aws',
        source: 'kiro_import',
        email: 'kiro@example.com',
        kiro_member_level: 'kiro_pro_plus',
        kiro_member_credits: 2345
      },
      suggestedName: 'kiro@example.com'
    })
  })
})
