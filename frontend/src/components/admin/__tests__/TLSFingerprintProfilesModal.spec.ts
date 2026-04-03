import { defineComponent } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

const { listMock } = vi.hoisted(() => ({
  listMock: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    tlsFingerprintProfiles: {
      list: listMock,
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn()
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn()
  })
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

import TLSFingerprintProfilesModal from '../TLSFingerprintProfilesModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div><slot /><slot name="footer" /></div>'
})

describe('TLSFingerprintProfilesModal', () => {
  it('mounts without triggering a before-initialization error when closed', () => {
    listMock.mockResolvedValue([])

    expect(() => mount(TLSFingerprintProfilesModal, {
      props: {
        show: false
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          ConfirmDialog: true,
          Icon: true
        }
      }
    })).not.toThrow()

    expect(listMock).not.toHaveBeenCalled()
  })

  it('loads profiles immediately when opened', async () => {
    listMock.mockResolvedValue([])

    mount(TLSFingerprintProfilesModal, {
      props: {
        show: true
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          ConfirmDialog: true,
          Icon: true
        }
      }
    })

    await Promise.resolve()
    expect(listMock).toHaveBeenCalledTimes(1)
  })
})
