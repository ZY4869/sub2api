import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ImageBatchStorageSettingsCard from '../ImageBatchStorageSettingsCard.vue'

const mockState = vi.hoisted(() => ({
  getImageBatchStorageSettings: vi.fn(),
  updateImageBatchStorageSettings: vi.fn(),
  testImageBatchStorageSettings: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api', () => ({
  adminAPI: {
    settings: {
      getImageBatchStorageSettings: mockState.getImageBatchStorageSettings,
      updateImageBatchStorageSettings: mockState.updateImageBatchStorageSettings,
      testImageBatchStorageSettings: mockState.testImageBatchStorageSettings
    }
  }
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: mockState.showError,
    showSuccess: mockState.showSuccess
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

function mountCard() {
  return mount(ImageBatchStorageSettingsCard, {
    global: {
      stubs: {
        Select: {
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template:
            '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>'
        },
        Toggle: {
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template:
            '<input type="checkbox" :checked="modelValue" @change="$emit(\'update:modelValue\', $event.target.checked)" />'
        }
      }
    }
  })
}

describe('ImageBatchStorageSettingsCard', () => {
  beforeEach(() => {
    mockState.getImageBatchStorageSettings.mockReset()
    mockState.updateImageBatchStorageSettings.mockReset()
    mockState.testImageBatchStorageSettings.mockReset()
    mockState.showError.mockReset()
    mockState.showSuccess.mockReset()
  })

  it('keeps stored secrets blank and submits a new secret for S3-compatible storage', async () => {
    mockState.getImageBatchStorageSettings.mockResolvedValue({
      backend: 's3',
      endpoint: 'https://s3.example.com',
      bucket: 'images',
      prefix: 'image-batches',
      region: 'auto',
      force_path_style: false,
      access_key_id: 'AKIA_TEST',
      secret_access_key_configured: true
    })
    mockState.updateImageBatchStorageSettings.mockImplementation(async (payload) => ({
      ...payload,
      secret_access_key: '',
      secret_access_key_configured: true
    }))
    mockState.testImageBatchStorageSettings.mockResolvedValue({ ok: true })

    const wrapper = mountCard()
    await flushPromises()

    expect(wrapper.find('input[type="password"]').element.value).toBe('')

    await wrapper.find('input[type="password"]').setValue('new-secret')
    await wrapper.find('input[type="checkbox"]').setValue(true)
    const buttons = wrapper.findAll('button')
    await buttons.find((button) => button.text() === 'admin.settings.imageBatchStorage.test')!.trigger('click')

    expect(mockState.testImageBatchStorageSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        backend: 's3',
        secret_access_key: 'new-secret',
        force_path_style: true
      })
    )
    expect(mockState.showSuccess).toHaveBeenCalledWith('admin.settings.imageBatchStorage.testSuccess')

    await wrapper.find('button.btn-primary').trigger('click')
    expect(mockState.updateImageBatchStorageSettings).toHaveBeenCalledWith(
      expect.objectContaining({
        endpoint: 'https://s3.example.com',
        bucket: 'images',
        access_key_id: 'AKIA_TEST',
        secret_access_key: 'new-secret'
      })
    )
  })
})
