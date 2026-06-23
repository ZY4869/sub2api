import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { submitApiKeyForm } from '../submit'
import type { ApiKeyFormData } from '../types'

const mocks = vi.hoisted(() => ({
  createWithPayload: vi.fn(),
  update: vi.fn(),
}))

vi.mock('@/api', () => ({
  keysAPI: {
    createWithPayload: mocks.createWithPayload,
    update: mocks.update,
  },
}))

const baseForm = (): ApiKeyFormData => ({
  name: 'windowed',
  group_bindings: [{
    group_id: 10,
    quota: 0,
    model_patterns_text: '',
    selected_models: [],
    model_selection_dirty: false,
  }],
  status: 'active',
  use_custom_key: false,
  custom_key: '',
  enable_ip_restriction: false,
  ip_whitelist: '',
  ip_blacklist: '',
  enable_quota: false,
  quota: null,
  image_only_enabled: false,
  image_count_billing_enabled: false,
  image_max_count: null,
  image_count_weights: { '1K': 1, '2K': 1, '4K': 1 },
  enable_rate_limit: false,
  rate_limit_5h: null,
  rate_limit_1d: null,
  rate_limit_7d: null,
  enable_expiration: false,
  expiration_preset: 'custom',
  expiration_date: '',
  enable_starts_at: false,
  starts_at: '',
  enable_time_access: true,
  time_access_preset: 'custom',
  access_time_policy: {
    enabled: true,
    timezone: 'Asia/Singapore',
    weekly_windows: [
      { days: [1, 2, 3], start: '22:00', end: '02:00' },
    ],
    daily_allowed_minutes: 240,
  },
})

function context(overrides: Partial<{
  formData: ApiKeyFormData
  showEditModal: boolean
  selectedKey: any
  isAdminMode: boolean
}> = {}) {
  return {
    formData: ref(overrides.formData || baseForm()),
    selectedKey: ref(overrides.selectedKey || null),
    showEditModal: ref(!!overrides.showEditModal),
    submitting: ref(false),
    isAdminMode: ref(!!overrides.isAdminMode),
    apiKeyModelSelectionRequired: ref(false),
    customKeyError: ref(''),
    t: (key: string) => key,
    appStore: {
      showError: vi.fn(),
      showSuccess: vi.fn(),
    },
    onboardingStore: {
      isCurrentStep: vi.fn(() => false),
      nextStep: vi.fn(),
    },
    syncImageOnlyGroupBindings: vi.fn(),
    normalizeImageCountWeights: vi.fn((weights) => weights),
    closeModals: vi.fn(),
    loadApiKeys: vi.fn(),
  }
}

describe('submitApiKeyForm time access payloads', () => {
  it('clears image count billing payloads for normal users', async () => {
    mocks.createWithPayload.mockResolvedValueOnce({})
    const formData = baseForm()
    formData.image_only_enabled = true
    formData.image_count_billing_enabled = true
    formData.image_max_count = 100
    formData.image_count_weights = { '1K': 2, '2K': 3, '4K': 4 }

    await submitApiKeyForm(context({ formData }))

    const payload = mocks.createWithPayload.mock.calls[0][0]
    expect(payload).toEqual(expect.objectContaining({
      image_only_enabled: true,
      image_count_billing_enabled: false,
      image_max_count: 0,
    }))
    expect(payload).not.toHaveProperty('image_count_weights')
  })

  it('keeps image count billing payloads for admins', async () => {
    mocks.createWithPayload.mockResolvedValueOnce({})
    const formData = baseForm()
    formData.image_only_enabled = true
    formData.image_count_billing_enabled = true
    formData.image_max_count = 100
    formData.image_count_weights = { '1K': 2, '2K': 3, '4K': 4 }

    await submitApiKeyForm(context({ formData, isAdminMode: true }))

    expect(mocks.createWithPayload).toHaveBeenCalledWith(expect.objectContaining({
      image_only_enabled: true,
      image_count_billing_enabled: true,
      image_max_count: 100,
      image_count_weights: { '1K': 2, '2K': 3, '4K': 4 },
    }))
  })

  it('sends multi-window time policy payloads on create', async () => {
    mocks.createWithPayload.mockResolvedValueOnce({})
    const ctx = context()

    await submitApiKeyForm(ctx)

    expect(mocks.createWithPayload).toHaveBeenCalledWith(expect.objectContaining({
      access_time_policy: expect.objectContaining({
        enabled: true,
        timezone: 'Asia/Singapore',
        weekly_windows: [
          { days: [1, 2, 3], start: '22:00', end: '02:00' },
        ],
        daily_allowed_minutes: 240,
      }),
    }))
  })

  it('clears access time policy when disabled on update', async () => {
    mocks.update.mockResolvedValueOnce({})
    const formData = baseForm()
    formData.enable_time_access = false

    await submitApiKeyForm(context({
      formData,
      showEditModal: true,
      selectedKey: { id: 7 },
    }))

    expect(mocks.update).toHaveBeenCalledWith(7, expect.objectContaining({
      clear_access_time_policy: true,
    }))
    expect(mocks.update.mock.calls[0][1]).not.toHaveProperty('access_time_policy')
  })
})
