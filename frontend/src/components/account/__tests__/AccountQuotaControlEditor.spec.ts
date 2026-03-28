import { flushPromises, mount } from '@vue/test-utils'
import { nextTick, reactive } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountQuotaControlEditor from '../AccountQuotaControlEditor.vue'
import { createDefaultAnthropicQuotaControlState } from '@/utils/accountQuotaControl'

const { listProfilesMock } = vi.hoisted(() => ({
  listProfilesMock: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    tlsFingerprintProfiles: {
      list: listProfilesMock
    }
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountQuotaControlEditor', () => {
  beforeEach(() => {
    listProfilesMock.mockReset()
    listProfilesMock.mockResolvedValue([
      {
        id: 7,
        name: 'Node 24',
        description: 'Default Node.js-like template'
      }
    ])
  })

  it('updates quota state through toggles and inputs', async () => {
    const state = reactive(createDefaultAnthropicQuotaControlState())
    const wrapper = mount(AccountQuotaControlEditor, {
      props: {
        state,
        umqModeOptions: [
          { value: '', label: 'Off' },
          { value: 'throttle', label: 'Throttle' },
          { value: 'serialize', label: 'Serialize' }
        ]
      }
    })

    const buttons = wrapper.findAll('button')
    await buttons[0]?.trigger('click')
    expect(state.windowCostEnabled).toBe(true)

    const serializeButton = buttons.find((button) => button.text() === 'Serialize')
    expect(serializeButton).toBeTruthy()
    await serializeButton?.trigger('click')
    expect(state.userMsgQueueMode).toBe('serialize')

    state.cacheTTLOverrideEnabled = true
    await nextTick()
    await wrapper.find('select').setValue('1h')
    expect(state.cacheTTLOverrideTarget).toBe('1h')
  })

  it('loads TLS fingerprint profiles and writes selected profile id', async () => {
    const state = reactive(createDefaultAnthropicQuotaControlState())
    state.tlsFingerprintEnabled = true

    const wrapper = mount(AccountQuotaControlEditor, {
      props: {
        state,
        umqModeOptions: [
          { value: '', label: 'Off' },
          { value: 'throttle', label: 'Throttle' },
          { value: 'serialize', label: 'Serialize' }
        ]
      }
    })

    await flushPromises()

    const selects = wrapper.findAll('select')
    expect(selects).toHaveLength(1)
    await selects[0]?.setValue('-1')
    expect(state.tlsFingerprintProfileId).toBe(-1)

    await selects[0]?.setValue('7')
    expect(state.tlsFingerprintProfileId).toBe(7)
    expect(wrapper.text()).toContain('Default Node.js-like template')
  })
})
