import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import SettingsView from '../SettingsView.vue'

const mocks = vi.hoisted(() => ({
  replace: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  fetchPublicSettings: vi.fn(),
  adminSettingsFetch: vi.fn(),
  getSettings: vi.fn(),
  updateSettings: vi.fn(),
  getAllGroups: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: { tab: 'users' } }),
  useRouter: () => ({ replace: mocks.replace }),
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

vi.mock('@/api', () => ({
  adminAPI: {
    settings: {
      getSettings: mocks.getSettings,
      updateSettings: mocks.updateSettings,
    },
    groups: {
      getAll: mocks.getAllGroups,
    },
  },
  userAPI: {
    updateProfile: vi.fn(),
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: mocks.showError,
    showSuccess: mocks.showSuccess,
    fetchPublicSettings: mocks.fetchPublicSettings,
  }),
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({
    fetch: mocks.adminSettingsFetch,
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: { global_realtime_countdown_enabled: false },
    setCurrentUser: vi.fn(),
  }),
}))

vi.mock('../settings/useAdminApiKeySettings', () => ({
  useAdminApiKeySettings: () => ({
    adminApiKeyLoading: false,
    adminApiKeyExists: false,
    adminApiKeyMasked: '',
    adminApiKeyOperating: false,
    newAdminApiKey: '',
    loadAdminApiKey: vi.fn(),
    createAdminApiKey: vi.fn(),
    regenerateAdminApiKey: vi.fn(),
    deleteAdminApiKey: vi.fn(),
    copyNewKey: vi.fn(),
  }),
}))

vi.mock('../settings/useGatewaySettingsControls', () => ({
  useGatewaySettingsControls: () => ({
    overloadCooldownLoading: false,
    overloadCooldownSaving: false,
    overloadCooldownForm: { enabled: true, cooldown_minutes: 10 },
    streamTimeoutLoading: false,
    streamTimeoutSaving: false,
    streamTimeoutForm: {
      enabled: true,
      action: 'temp_unsched',
      temp_unsched_minutes: 5,
      threshold_count: 3,
      threshold_window_minutes: 10,
    },
    rectifierLoading: false,
    rectifierSaving: false,
    rectifierForm: {
      enabled: true,
      thinking_signature_enabled: true,
      thinking_budget_enabled: true,
    },
    betaPolicyLoading: false,
    betaPolicySaving: false,
    betaPolicyForm: { rules: [] },
    betaPolicyActionOptions: [],
    betaPolicyScopeOptions: [],
    getBetaDisplayName: (token: string) => token,
    loadOverloadCooldownSettings: vi.fn(),
    saveOverloadCooldownSettings: vi.fn(),
    loadStreamTimeoutSettings: vi.fn(),
    saveStreamTimeoutSettings: vi.fn(),
    loadRectifierSettings: vi.fn(),
    saveRectifierSettings: vi.fn(),
    loadBetaPolicySettings: vi.fn(),
    saveBetaPolicySettings: vi.fn(),
  }),
}))

vi.mock('../settings/useSettingsEmailServices', () => ({
  useSettingsEmailServices: () => ({
    testingSmtp: false,
    testingTelegram: false,
    sendingTestEmail: false,
    testEmailAddress: '',
    emailTemplatesLoading: false,
    emailTemplateSaving: false,
    emailTemplateTesting: false,
    emailTemplateDefinitions: [],
    selectedEmailTemplateKey: '',
    selectedEmailTemplateLocale: 'zh',
    emailTemplateDraft: { subject: '', body: '', enabled: true },
    emailTemplateOptions: [],
    emailTemplateLocaleOptions: [],
    selectedEmailTemplate: null,
    syncEmailTemplateDraft: vi.fn(),
    testSmtpConnection: vi.fn(),
    testTelegramConnection: vi.fn(),
    sendTestEmail: vi.fn(),
    loadEmailTemplates: vi.fn(),
    saveEmailTemplate: vi.fn(),
    resetEmailTemplateDraft: vi.fn(),
    sendTemplateTestEmail: vi.fn(),
  }),
}))

const emptyTabStub = {
  template: '<div />',
}

function makeSettings(overrides: Record<string, unknown> = {}) {
  return {
    registration_enabled: true,
    email_verify_enabled: false,
    registration_email_suffix_whitelist: [],
    promo_code_enabled: true,
    invitation_code_enabled: false,
    password_reset_enabled: false,
    frontend_url: '',
    totp_enabled: false,
    totp_encryption_key_configured: false,
    default_balance: 0,
    default_concurrency: 1,
    default_subscriptions: [],
    default_api_key_model_binding_mode: 'group_allowed',
    content_moderation_cyber_categories: [],
    custom_menu_items: [],
    login_agreement_documents: [],
    payment_allowed_currencies: ['USD'],
    payment_subscription_plans: [],
    openai_allowed_codex_clients: [],
    openai_fast_policy_settings: { rules: [] },
    content_moderation_keywords: [],
    content_moderation_model_filter: { type: 'all', models: [] },
    content_moderation_category_thresholds: {},
    ...overrides,
  }
}

function mountView() {
  return mount(SettingsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        Icon: true,
        SettingsSecurityAdminTab: emptyTabStub,
        SettingsGatewayMainTab: emptyTabStub,
        SettingsSecurityAuthTab: emptyTabStub,
        SettingsGatewayExtraTab: emptyTabStub,
        SettingsGeneralTab: emptyTabStub,
        SettingsNotificationTab: emptyTabStub,
        SettingsEmailTab: emptyTabStub,
        Select: emptyTabStub,
        Toggle: emptyTabStub,
        GroupBadge: emptyTabStub,
        GroupOptionItem: emptyTabStub,
      },
    },
  })
}

describe('SettingsView default API key model binding mode', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.getSettings.mockResolvedValue(makeSettings({
      default_api_key_model_binding_mode: 'group_allowed',
    }))
    mocks.updateSettings.mockImplementation(async (payload) =>
      makeSettings(payload),
    )
    mocks.getAllGroups.mockResolvedValue([])
    mocks.fetchPublicSettings.mockResolvedValue(undefined)
    mocks.adminSettingsFetch.mockResolvedValue(undefined)
  })

  it('loads, switches, saves, and reflects the default API key mode', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('admin.settings.defaults.defaultApiKeyModeGroup')

    const publicModelButton = wrapper
      .findAll('button')
      .find((button) =>
        button.text().includes('admin.settings.defaults.defaultApiKeyModePublicModel'),
      )
    expect(publicModelButton).toBeTruthy()
    await publicModelButton!.trigger('click')
    await nextTick()

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(mocks.updateSettings).toHaveBeenCalledTimes(1)
    expect(mocks.updateSettings.mock.calls[0][0]).toEqual(
      expect.objectContaining({
        default_api_key_model_binding_mode: 'model_required',
      }),
    )
    expect(mocks.showSuccess).toHaveBeenCalledWith('admin.settings.settingsSaved')

    const groupButton = wrapper
      .findAll('button')
      .find((button) =>
        button.text().includes('admin.settings.defaults.defaultApiKeyModeGroup'),
      )
    expect(groupButton?.classes().join(' ')).not.toContain('border-primary-500')
    expect(publicModelButton!.classes().join(' ')).toContain('border-primary-500')
  })
})
