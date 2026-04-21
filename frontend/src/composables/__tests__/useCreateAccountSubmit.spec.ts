import { ref, computed } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useCreateAccountSubmit } from '../useCreateAccountSubmit'

const { createMock, showSuccess, showError } = vi.hoisted(() => ({
  createMock: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      create: createMock
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showError
  })
}))

vi.mock('@/stores/modelRegistry', () => ({
  ensureModelRegistryFresh: vi.fn().mockResolvedValue({
    etag: 'test-etag',
    updated_at: '2026-04-08T00:00:00Z',
    models: [],
    presets: []
  }),
  getModelRegistrySnapshot: vi.fn(() => ({
    etag: 'test-etag',
    updated_at: '2026-04-08T00:00:00Z',
    models: [],
    presets: []
  })),
  invalidateModelRegistry: vi.fn()
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

describe('useCreateAccountSubmit', () => {
  beforeEach(() => {
    createMock.mockReset()
    showSuccess.mockReset()
    showError.mockReset()
  })

  it('preserves protocol gateway default provider/model fields when creating an account', async () => {
    createMock.mockResolvedValueOnce({ id: 1, name: 'Gateway' })

    const composable = useCreateAccountSubmit({
      withConfirmFlag: (payload) => payload,
      ensureMixedChannelConfirmed: async () => true,
      requiresMixedChannelCheck: ref(false),
      openMixedChannelDialog: vi.fn(),
      isOpenAIModelRestrictionDisabled: computed(() => true),
      modelRestrictionEnabled: ref(false),
      modelRestrictionMode: ref<'whitelist' | 'mapping'>('mapping'),
      allowedModels: ref([]),
      modelMappings: ref([]),
      antigravityModelMappings: ref([]),
      applyTempUnschedConfig: () => true,
      form: {
        name: 'Gateway Account',
        notes: '',
        proxy_id: null,
        concurrency: 1,
        load_factor: null,
        priority: 1,
        rate_multiplier: 1,
        group_ids: [],
        expires_at: null
      },
      autoPauseOnExpired: ref(false),
      editQuotaLimit: ref(null),
      editQuotaDailyLimit: ref(null),
      editQuotaWeeklyLimit: ref(null),
      editQuotaDailyResetMode: ref(null),
      editQuotaDailyResetHour: ref(null),
      editQuotaWeeklyResetMode: ref(null),
      editQuotaWeeklyResetDay: ref(null),
      editQuotaWeeklyResetHour: ref(null),
      editQuotaResetTimezone: ref(null),
      batchArchiveEnabled: ref(false),
      batchArchiveAutoPrefetchEnabled: ref(false),
      batchArchiveRetentionDays: ref(7),
      batchArchiveBillingMode: ref<'log_only' | 'archive_charge'>('log_only'),
      batchArchiveDownloadPriceUSD: ref(0),
      allowVertexBatchOverflow: ref(false),
      acceptAIStudioBatchOverflow: ref(false),
      afterCreateImportModels: vi.fn().mockResolvedValue(undefined),
      emitCreated: vi.fn(),
      onClose: vi.fn()
    })

    await composable.createAccountAndFinish(
      'protocol_gateway',
      'apikey',
      {
        api_key: 'gateway-key',
        base_url: 'https://gateway.example.com'
      },
      {
        gateway_test_provider: 'anthropic',
        gateway_test_model_id: 'claude-sonnet-4.5'
      },
      'mixed'
    )

    expect(createMock).toHaveBeenCalledTimes(1)
    expect(createMock.mock.calls[0]?.[0]).toMatchObject({
      platform: 'protocol_gateway',
      gateway_protocol: 'mixed',
      extra: {
        gateway_protocol: 'mixed',
        gateway_test_provider: 'anthropic',
        gateway_test_model_id: 'claude-sonnet-4.5'
      }
    })
  })
})
