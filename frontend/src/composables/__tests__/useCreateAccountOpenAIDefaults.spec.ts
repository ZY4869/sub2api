import { computed, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import { useCreateAccountOpenAIExchange } from '../useCreateAccountOpenAIExchange'
import { useCreateAccountOpenAIRefreshTokenValidation } from '../useCreateAccountOpenAIRefreshTokenValidation'

const { createMock } = vi.hoisted(() => ({
  createMock: vi.fn(),
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

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      create: createMock,
    },
  },
}))

function createBaseOptions() {
  return {
    form: {
      platform: 'openai',
      name: 'OpenAI OAuth',
      notes: '',
      proxy_id: null,
      concurrency: 1,
      load_factor: null,
      priority: 1,
      rate_multiplier: 1,
      group_ids: [],
      expires_at: null,
    },
    autoPauseOnExpired: ref(true),
    isOpenAIModelRestrictionDisabled: computed(() => false),
    modelRestrictionEnabled: ref(true),
    modelRestrictionMode: ref<'whitelist' | 'mapping'>('whitelist'),
    allowedModels: ref<string[]>(['gpt-5.2', 'gpt-5.4', 'gpt-5.4-mini']),
    modelMappings: ref([]),
    hasCustomizedOpenAIDefaults: ref(false),
    buildAccountExtra: (base?: Record<string, unknown>) => base,
    afterCreateImportModels: vi.fn().mockResolvedValue(undefined),
    emitCreated: vi.fn(),
    onClose: vi.fn(),
  }
}

describe('OpenAI create-account defaults', () => {
  it('uses the free OAuth whitelist defaults without gpt-image-2', async () => {
    createMock.mockReset()
    createMock.mockResolvedValue({ id: 4, platform: 'openai', type: 'oauth' })

    const base = createBaseOptions()
    const exchangeAuthCode = vi.fn().mockResolvedValue({ plan_type: 'free' })
    const { handleOpenAIExchange } = useCreateAccountOpenAIExchange({
      oauthClient: computed(() => ({
        sessionId: ref('session'),
        oauthState: ref('state'),
        loading: ref(false),
        error: ref(''),
        exchangeAuthCode,
        buildCredentials: (tokenInfo: any) => ({ plan_type: tokenInfo.plan_type }),
        buildExtraInfo: () => undefined,
      })),
      getOAuthState: () => 'state',
      applyTempUnschedConfig: () => true,
      ...base,
    })

    await handleOpenAIExchange('code')

    expect(base.allowedModels.value).toEqual([
      'gpt-5.2',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-5.5',
    ])
    expect(
      createMock.mock.calls[0][0]?.extra?.model_scope_v2?.entries?.map((entry: any) => ({
        display_model_id: entry.display_model_id,
        target_model_id: entry.target_model_id,
      }))
    ).toEqual([
      { display_model_id: 'gpt-5.2', target_model_id: 'gpt-5.2' },
      { display_model_id: 'gpt-5.4', target_model_id: 'gpt-5.4' },
      { display_model_id: 'gpt-5.4-mini', target_model_id: 'gpt-5.4-mini' },
      { display_model_id: 'gpt-5.5', target_model_id: 'gpt-5.5' },
    ])
  })

  it('applies the pro OAuth whitelist defaults and includes Spark for Pro tiers', async () => {
    createMock.mockReset()
    createMock.mockResolvedValue({ id: 1, platform: 'openai', type: 'oauth' })

    const base = createBaseOptions()
    const exchangeAuthCode = vi.fn().mockResolvedValue({ plan_type: 'pro', pro_multiplier: 20 })
    const { handleOpenAIExchange } = useCreateAccountOpenAIExchange({
      oauthClient: computed(() => ({
        sessionId: ref('session'),
        oauthState: ref('state'),
        loading: ref(false),
        error: ref(''),
        exchangeAuthCode,
        buildCredentials: (tokenInfo: any) => ({ plan_type: tokenInfo.plan_type }),
        buildExtraInfo: () => undefined,
      })),
      getOAuthState: () => 'state',
      applyTempUnschedConfig: () => true,
      ...base,
    })

    await handleOpenAIExchange('code')

    expect(base.allowedModels.value).toEqual([
      'gpt-image-2',
      'gpt-5.2',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-5.5',
      'gpt-5.3-codex-spark',
    ])
    expect(createMock).toHaveBeenCalledTimes(1)
    expect(createMock.mock.calls[0][0]?.extra?.model_scope_v2?.policy_mode).toBe('whitelist')
    expect(
      createMock.mock.calls[0][0]?.extra?.model_scope_v2?.entries?.map((entry: any) => ({
        display_model_id: entry.display_model_id,
        target_model_id: entry.target_model_id,
      }))
    ).toEqual([
      { display_model_id: 'gpt-image-2', target_model_id: 'gpt-image-2' },
      { display_model_id: 'gpt-5.2', target_model_id: 'gpt-5.2' },
      { display_model_id: 'gpt-5.4', target_model_id: 'gpt-5.4' },
      { display_model_id: 'gpt-5.4-mini', target_model_id: 'gpt-5.4-mini' },
      { display_model_id: 'gpt-5.5', target_model_id: 'gpt-5.5' },
      { display_model_id: 'gpt-5.3-codex-spark', target_model_id: 'gpt-5.3-codex-spark' },
    ])
  })

  it('keeps a manually edited OAuth whitelist untouched', async () => {
    createMock.mockReset()
    createMock.mockResolvedValue({ id: 2, platform: 'openai', type: 'oauth' })

    const base = createBaseOptions()
    base.allowedModels.value = ['gpt-5.4']
    base.hasCustomizedOpenAIDefaults.value = true
    const { handleOpenAIExchange } = useCreateAccountOpenAIExchange({
      oauthClient: computed(() => ({
        sessionId: ref('session'),
        oauthState: ref('state'),
        loading: ref(false),
        error: ref(''),
        exchangeAuthCode: vi.fn().mockResolvedValue({ plan_type: 'pro' }),
        buildCredentials: (tokenInfo: any) => ({ plan_type: tokenInfo.plan_type }),
        buildExtraInfo: () => undefined,
      })),
      getOAuthState: () => 'state',
      applyTempUnschedConfig: () => true,
      ...base,
    })

    await handleOpenAIExchange('code')

    expect(base.allowedModels.value).toEqual(['gpt-5.4'])
  })

  it('omits Spark for non-Pro refresh token defaults', async () => {
    createMock.mockReset()
    createMock.mockResolvedValue({ id: 3, platform: 'openai', type: 'oauth' })

    const base = createBaseOptions()
    const { handleOpenAIValidateRT } = useCreateAccountOpenAIRefreshTokenValidation({
      oauthClient: computed(() => ({
        loading: ref(false),
        error: ref(''),
        validateRefreshToken: vi.fn().mockResolvedValue({ plan_type: 'plus' }),
        buildCredentials: (tokenInfo: any) => ({ plan_type: tokenInfo.plan_type }),
        buildExtraInfo: () => undefined,
      })),
      ...base,
    })

    await handleOpenAIValidateRT('rt_123')

    expect(base.allowedModels.value).toEqual([
      'gpt-image-2',
      'gpt-5.2',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-5.5',
    ])
    expect(createMock).toHaveBeenCalledTimes(1)
  })

  it('keeps paid OAuth defaults for plus refresh token accounts', async () => {
    createMock.mockReset()
    createMock.mockResolvedValue({ id: 5, platform: 'openai', type: 'oauth' })

    const base = createBaseOptions()
    const { handleOpenAIValidateRT } = useCreateAccountOpenAIRefreshTokenValidation({
      oauthClient: computed(() => ({
        loading: ref(false),
        error: ref(''),
        validateRefreshToken: vi.fn().mockResolvedValue({ plan_type: 'plus' }),
        buildCredentials: (tokenInfo: any) => ({ plan_type: tokenInfo.plan_type }),
        buildExtraInfo: () => undefined,
      })),
      ...base,
    })

    await handleOpenAIValidateRT('rt_456')

    expect(base.allowedModels.value).toEqual([
      'gpt-image-2',
      'gpt-5.2',
      'gpt-5.4',
      'gpt-5.4-mini',
      'gpt-5.5',
    ])
  })
})
