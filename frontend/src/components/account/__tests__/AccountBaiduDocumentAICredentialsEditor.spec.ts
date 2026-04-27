import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountBaiduDocumentAICredentialsEditor from '../AccountBaiduDocumentAICredentialsEditor.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountBaiduDocumentAICredentialsEditor', () => {
  it('emits create-mode credential updates for all baidu document ai fields', async () => {
    const wrapper = mount(AccountBaiduDocumentAICredentialsEditor, {
      props: {
        mode: 'create',
        asyncBearerToken: '',
        asyncBaseUrl: '',
        directToken: '',
        directApiUrlsText: ''
      }
    })

    await wrapper.get('[data-testid="baidu-document-ai-async-base-url"]').setValue('https://aistudio.baidu.com/async')
    await wrapper.get('[data-testid="baidu-document-ai-async-bearer-token"]').setValue('async-token')
    await wrapper.get('[data-testid="baidu-document-ai-direct-token"]').setValue('direct-token')
    await wrapper.get('[data-testid="baidu-document-ai-direct-api-urls"]').setValue('{"pp-ocrv5-server":"https://direct.baidu.com/ocr"}')

    expect(wrapper.emitted('update:asyncBaseUrl')?.at(-1)).toEqual(['https://aistudio.baidu.com/async'])
    expect(wrapper.emitted('update:asyncBearerToken')?.at(-1)).toEqual(['async-token'])
    expect(wrapper.emitted('update:directToken')?.at(-1)).toEqual(['direct-token'])
    expect(wrapper.emitted('update:directApiUrlsText')?.at(-1)).toEqual(['{"pp-ocrv5-server":"https://direct.baidu.com/ocr"}'])
  })

  it('uses the keep-current placeholder in edit mode', () => {
    const wrapper = mount(AccountBaiduDocumentAICredentialsEditor, {
      props: {
        mode: 'edit',
        asyncBearerToken: '',
        asyncBaseUrl: 'https://aistudio.baidu.com/async',
        directToken: '',
        directApiUrlsText: ''
      }
    })

    const passwordInputs = wrapper.findAll('input[type="password"]')
    expect(passwordInputs).toHaveLength(2)
    expect(passwordInputs[0]?.attributes('placeholder')).toBe('admin.accounts.leaveEmptyToKeep')
    expect(passwordInputs[1]?.attributes('placeholder')).toBe('admin.accounts.leaveEmptyToKeep')
  })

  it('shows the direct API URL JSON placeholder without relying on i18n interpolation', () => {
    const wrapper = mount(AccountBaiduDocumentAICredentialsEditor, {
      props: {
        mode: 'create',
        asyncBearerToken: '',
        asyncBaseUrl: '',
        directToken: '',
        directApiUrlsText: ''
      }
    })

    const directApiUrlsInput = wrapper.get('[data-testid="baidu-document-ai-direct-api-urls"]')

    expect(directApiUrlsInput.attributes('placeholder')).toContain('{')
    expect(directApiUrlsInput.attributes('placeholder')).toContain('"pp-ocrv5-server": "https://..."')
    expect(directApiUrlsInput.attributes('placeholder')?.trim().endsWith('}')).toBe(true)
  })
})
