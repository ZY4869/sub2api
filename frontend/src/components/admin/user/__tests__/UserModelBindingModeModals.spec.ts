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
  props: ['show', 'title', 'width'],
  template: `
    <section v-if="show" data-testid="base-dialog" :data-width="width">
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
    await wrapper.findAll('select')[1].setValue('group_allowed')
    await wrapper.find('form').trigger('submit.prevent')

    expect(mocks.createUser).toHaveBeenCalledWith(
      expect.objectContaining({
        email: 'new@example.com',
        role: 'user',
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

    await wrapper.findAll('select')[1].setValue('group_allowed')
    await wrapper.find('form').trigger('submit.prevent')

    expect(mocks.updateUser).toHaveBeenCalledWith(
      7,
      expect.objectContaining({
        role: 'user',
        api_key_model_binding_mode: 'group_allowed',
      }),
    )
  })

  it('submits role when creating and editing users', async () => {
    mocks.createUser.mockResolvedValue({})
    mocks.updateUser.mockResolvedValue({})

    const createWrapper = mount(UserCreateModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          TimeAccessPolicyEditor: true,
          Icon: { template: '<span />' },
        },
      },
    })

    const inputs = createWrapper.findAll('input')
    await inputs[0].setValue('new-admin@example.com')
    await inputs[1].setValue('strong-pass')
    await createWrapper.findAll('select')[0].setValue('admin')
    await createWrapper.find('form').trigger('submit.prevent')

    expect(mocks.createUser).toHaveBeenCalledWith(
      expect.objectContaining({
        role: 'admin',
      }),
    )

    const editWrapper = mount(UserEditModal, {
      props: {
        show: true,
        user: {
          id: 8,
          email: 'admin@example.com',
          username: 'admin',
          notes: '',
          role: 'admin',
          status: 'active',
          concurrency: 1,
          admin_free_billing: true,
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

    await editWrapper.findAll('select')[0].setValue('user')
    await editWrapper.find('form').trigger('submit.prevent')

    expect(mocks.updateUser).toHaveBeenCalledWith(
      8,
      expect.objectContaining({
        role: 'user',
        admin_free_billing: false,
      }),
    )
  })

  it('uses a wide dialog for editing user details', () => {
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

    expect(wrapper.get('[data-testid="base-dialog"]').attributes('data-width')).toBe('wide')
  })
})
