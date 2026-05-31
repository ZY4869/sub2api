import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import UserCreateModal from '../UserCreateModal.vue'
import UserEditModal from '../UserEditModal.vue'

const mocks = vi.hoisted(() => ({
  createUser: vi.fn(),
  updateUser: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      create: mocks.createUser,
      update: mocks.updateUser,
    },
    userAttributes: {
      updateUserAttributeValues: vi.fn(),
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: mocks.showError,
    showSuccess: mocks.showSuccess,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn(),
  }),
}))

const BaseDialogStub = {
  props: ['show', 'title'],
  template: `
    <section v-if="show">
      <slot />
      <slot name="footer" />
    </section>
  `,
}

describe('user api key model binding mode modals', () => {
  beforeEach(() => {
    mocks.createUser.mockReset()
    mocks.updateUser.mockReset()
    mocks.showError.mockReset()
    mocks.showSuccess.mockReset()
  })

  it('submits api_key_model_binding_mode when creating a user', async () => {
    mocks.createUser.mockResolvedValue({})
    const wrapper = mount(UserCreateModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          TimeAccessPolicyEditor: true,
          Icon: { template: '<span />' },
        },
      },
    })

    const inputs = wrapper.findAll('input')
    await inputs[0].setValue('new@example.com')
    await inputs[1].setValue('strong-pass')
    await wrapper.find('select').setValue('group_allowed')
    await wrapper.find('form').trigger('submit.prevent')

    expect(mocks.createUser).toHaveBeenCalledWith(
      expect.objectContaining({
        email: 'new@example.com',
        api_key_model_binding_mode: 'group_allowed',
      }),
    )
  })

  it('submits api_key_model_binding_mode when editing a user', async () => {
    mocks.updateUser.mockResolvedValue({})
    const wrapper = mount(UserEditModal, {
      props: {
        show: true,
        user: {
          id: 7,
          email: 'old@example.com',
          username: 'old',
          notes: '',
          role: 'user',
          status: 'active',
          concurrency: 1,
          api_key_model_binding_mode: 'model_required',
        },
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          TimeAccessPolicyEditor: true,
          UserAttributeForm: { template: '<div />' },
          Icon: { template: '<span />' },
        },
      },
    })

    await wrapper.find('select').setValue('group_allowed')
    await wrapper.find('form').trigger('submit.prevent')

    expect(mocks.updateUser).toHaveBeenCalledWith(
      7,
      expect.objectContaining({
        api_key_model_binding_mode: 'group_allowed',
      }),
    )
  })
})
