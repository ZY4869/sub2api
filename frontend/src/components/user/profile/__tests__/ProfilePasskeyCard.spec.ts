import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ProfilePasskeyCard from '../ProfilePasskeyCard.vue'

const mocks = vi.hoisted(() => ({
  listPasskeys: vi.fn(),
  registerPasskey: vi.fn(),
  renamePasskey: vi.fn(),
  deletePasskey: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/passkeys', () => ({
  passkeyAPI: {
    listPasskeys: mocks.listPasskeys,
    registerPasskey: mocks.registerPasskey,
    renamePasskey: mocks.renamePasskey,
    deletePasskey: mocks.deletePasskey
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showSuccess: mocks.showSuccess,
    showError: mocks.showError
  })
}))

vi.mock('@/utils/format', () => ({
  formatDate: (value: string) => value
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span />'
  }
}))

describe('ProfilePasskeyCard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.listPasskeys.mockResolvedValue([])
  })

  it('hides and skips loading when passkeys are disabled', async () => {
    const wrapper = mount(ProfilePasskeyCard, {
      props: { enabled: false }
    })

    await flushPromises()

    expect(wrapper.find('.card').exists()).toBe(false)
    expect(mocks.listPasskeys).not.toHaveBeenCalled()
  })

  it('loads registered passkeys when enabled', async () => {
    mocks.listPasskeys.mockResolvedValue([
      {
        id: 7,
        name: 'Laptop',
        credential_id: 'credential',
        created_at: '2026-08-07T00:00:00Z',
        updated_at: '2026-08-07T00:00:00Z'
      }
    ])

    const wrapper = mount(ProfilePasskeyCard, {
      props: { enabled: true }
    })
    await flushPromises()

    expect(mocks.listPasskeys).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('Laptop')
  })

  it('shows an error toast when registration fails', async () => {
    mocks.registerPasskey.mockRejectedValue(new Error('PASSKEYS_DISABLED'))
    const wrapper = mount(ProfilePasskeyCard, {
      props: { enabled: true }
    })
    await flushPromises()

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('Work laptop')
    await inputs[1].setValue('current-password')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(mocks.registerPasskey).toHaveBeenCalledWith({
      password: 'current-password',
      name: 'Work laptop'
    })
    expect(mocks.showError).toHaveBeenCalledWith('PASSKEYS_DISABLED')
  })
})
