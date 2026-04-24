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
    modelRestrictionEnabled: ref(false),
    modelRestrictionMode: ref<'whitelist' | 'mapping'>('whitelist'),
    allowedModels: ref<string[]>(['gpt-5.2', 'gpt-5.4', 'gpt-5.4-mini']),
    modelMappings: ref([]),
    buildAccountExtra: (base?: Record<string, unknown>) => base,
    afterCreateImportModels: vi.fn().mockResolvedValue(undefined),
    emitCreated: vi.fn(),
    onClose: vi.fn(),
  }
}

describe('OpenAI create-account defaults', () => {
  it('upgrades the untouched OAuth whitelist to the pro default set', async () => {
    createMock.mockReset()
    createMock.mockResolvedValue({ id: 1, platform: 'openai', type: 'oauth' })

    const base = createBaseOptions()
    const exchangeAuthCode = vi.fn().mockResolvedValue({ plan_type: 'pro' })
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

    expect(base.allowedModels.value).toEqual(['gpt-5.2', 'gpt-5.4', 'gpt-5.4-mini', 'gpt-5.5', 'gpt-5.3-codex-spark'])
    expect(createMock).toHaveBeenCalledTimes(1)
  })

  it('keeps a manually edited OAuth whitelist untouched', async () => {
    createMock.mockReset()
    createMock.mockResolvedValue({ id: 2, platform: 'openai', type: 'oauth' })

    const base = createBaseOptions()
    base.allowedModels.value = ['gpt-5.4']
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

  it('upgrades the untouched refresh-token whitelist to the pro default set', async () => {
    createMock.mockReset()
    createMock.mockResolvedValue({ id: 3, platform: 'openai', type: 'oauth' })

    const base = createBaseOptions()
    const { handleOpenAIValidateRT } = useCreateAccountOpenAIRefreshTokenValidation({
      oauthClient: computed(() => ({
        loading: ref(false),
        error: ref(''),
        validateRefreshToken: vi.fn().mockResolvedValue({ plan_type: 'pro' }),
        buildCredentials: (tokenInfo: any) => ({ plan_type: tokenInfo.plan_type }),
        buildExtraInfo: () => undefined,
      })),
      ...base,
    })

    await handleOpenAIValidateRT('rt_123')

    expect(base.allowedModels.value).toEqual(['gpt-5.2', 'gpt-5.4', 'gpt-5.4-mini', 'gpt-5.5', 'gpt-5.3-codex-spark'])
    expect(createMock).toHaveBeenCalledTimes(1)
  })
})
