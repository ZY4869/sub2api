import type {
  AccountPlatform,
  AccountType,
  GatewayAcceptedProtocol,
  GatewayProtocol,
  GroupPlatform
} from '@/types'
import type { AddMethod } from '@/composables/useAccountOAuth'
import type { VertexAuthMode } from '@/utils/vertexAi'
import type { AccountCategory } from './accountCategory'

export function useCreateAccountModalWatchers(ctx: any) {
  const {
    BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL,
    DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE,
    DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT,
    OPENAI_WS_MODE_OFF,
    accountCategory,
    accountTier,
    actualModelLocked,
    addMethod,
    allowedModels,
    anthropicPassthroughEnabled,
    antigravityAccountType,
    antigravityModelMappings,
    antigravityOAuth,
    apiKeyBaseUrl,
    applyAccountTierCapacity,
    applyOpenAIImageProtocolDefaults,
    applyOpenAIOAuthPresetModels,
    autoImportModels,
    baiduDocumentAIAccessToken,
    baiduDocumentAIAsyncBaseUrl,
    baiduDocumentAIDirectApiUrlsText,
    claudeCodeMimicEnabled,
    claudeSessionIDMaskingEnabled,
    claudeTLSFingerprintEnabled,
    codexCLIOnlyEnabled,
    createDefaultDeepSeekModelConcurrencyLimitDraft,
    deepSeekModelConcurrencyLimits,
    effectivePlatform,
    ensureModelRegistryFresh,
    form,
    gatewayAcceptedProtocols,
    gatewayBatchEnabled,
    gatewayClientProfiles,
    gatewayClientRoutes,
    gatewayOpenAIImageProtocolMode,
    gatewayOpenAIRequestFormat,
    gatewayProtocol,
    gatewayTestModelId,
    gatewayTestProvider,
    geminiAIStudioOAuthEnabled,
    geminiOAuth,
    geminiOAuthType,
    geminiVertexAccessToken,
    geminiVertexApiKey,
    geminiVertexAuthMode,
    geminiVertexBaseUrl,
    geminiVertexExpiresAtInput,
    geminiVertexLocation,
    geminiVertexProjectId,
    geminiVertexServiceAccountJson,
    grokSSOToken,
    grokOAuthRef,
    grokTier,
    interceptWarmupRequests,
    isBaiduDocumentAIPlatform,
    isBaiduDocumentAISelected,
    kiroAuthRef,
    loadAntigravityDefaultMappings,
    manualModels,
    markOpenAIOAuthDefaultsCustomized,
    modelMappings,
    modelRestrictionEnabled,
    modelRestrictionMode,
    nextTick,
    oauth,
    oauthDraftCredentials,
    oauthDraftExtra,
    openAIImageCompatAllowed,
    openAIImageProtocolMode,
    openAIImageProtocolTouched,
    openRouterHTTPReferer,
    openRouterTitle,
    openaiAPIKeyResponsesWebSocketV2Mode,
    openaiOAuth,
    openaiOAuthResponsesWebSocketV2Mode,
    openaiPassthroughEnabled,
    props,
    protocolGatewayProbeModels,
    resetOAuthInputDraft,
    resetForm,
    resetOpenAIOAuthDefaultSelection,
    resetProtocolGatewayClaudeMimicState,
    resolveAccountApiKeyDefaultBaseUrl,
    resolvedUpstream,
    showProtocolGatewayBatchEditor,
    showProtocolGatewayClaudeMimicEditor,
    showProtocolGatewayOpenAIRequestFormatEditor,
    watch
  } = ctx

// Watchers
watch(
  () => props.show,
  (newVal: boolean) => {
    if (newVal) {
      void ensureModelRegistryFresh()
      resetOpenAIOAuthDefaultSelection()
      modelRestrictionMode.value = form.platform === 'protocol_gateway' ? 'mapping' : 'whitelist'
      modelRestrictionEnabled.value = true
      allowedModels.value = []
      modelMappings.value = []
      protocolGatewayProbeModels.value = []
      manualModels.value = []
      resolvedUpstream.value = null
      oauthDraftCredentials.value = {}
      oauthDraftExtra.value = {}
      gatewayAcceptedProtocols.value = ['openai']
      gatewayClientProfiles.value = []
      gatewayClientRoutes.value = []
      gatewayTestProvider.value = ''
      gatewayTestModelId.value = ''
      gatewayOpenAIRequestFormat.value = DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT
      gatewayOpenAIImageProtocolMode.value = DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE
      gatewayBatchEnabled.value = false
      deepSeekModelConcurrencyLimits.value = createDefaultDeepSeekModelConcurrencyLimitDraft()
      resetProtocolGatewayClaudeMimicState()
      if (form.platform === 'antigravity') {
        loadAntigravityDefaultMappings()
      } else {
        antigravityModelMappings.value = []
      }
      openAIImageProtocolTouched.value = false
      accountTier.value = ctx.defaultAccountTierForPlatform(form.platform)
      applyAccountTierCapacity()
      applyOpenAIImageProtocolDefaults(undefined, true)
      applyOpenAIOAuthPresetModels(undefined, null, true)
    } else {
      resetForm()
    }
  }
)

// Sync form.type based on accountCategory, addMethod, and platform-specific type
watch(
  [accountCategory, addMethod, antigravityAccountType, gatewayProtocol, geminiVertexAuthMode],
  ([category, method, agType, _gatewayProtocol, vertexAuthMode]: [
    AccountCategory,
    AddMethod,
    'oauth' | 'upstream',
    GatewayProtocol,
    VertexAuthMode
  ]) => {
    if (form.platform === 'antigravity' && agType === 'upstream') {
      form.type = 'apikey'
      return
    }
    if (isBaiduDocumentAISelected.value) {
      form.type = 'apikey'
      return
    }
    if (form.platform === 'grok') {
      if (category === 'oauth-based') {
        form.type = 'oauth'
      } else if (category === 'sso') {
        form.type = 'sso'
      } else {
        form.type = 'apikey'
      }
      return
    }
    if (form.platform === 'deepseek') {
      form.type = 'apikey'
      return
    }
    if (form.platform === 'protocol_gateway') {
      form.type = 'apikey'
      return
    }
    if (form.platform === 'gemini' && category === 'vertex_ai') {
      form.type = vertexAuthMode === 'express_api_key' ? 'apikey' : 'oauth'
      return
    }
    if (category === 'oauth-based') {
      form.type = form.platform === 'anthropic' ? method as AccountType : 'oauth'
    } else {
      form.type = 'apikey'
    }
  },
  { immediate: true }
)

// Reset platform-specific settings when platform changes
watch(
  () => form.platform,
  (newPlatform: AccountPlatform, previousPlatform: AccountPlatform) => {
    resetOpenAIOAuthDefaultSelection()
    accountTier.value = ctx.defaultAccountTierForPlatform(newPlatform)
    if (accountTier.value) {
      applyAccountTierCapacity()
    }
    apiKeyBaseUrl.value = resolveAccountApiKeyDefaultBaseUrl(newPlatform, gatewayProtocol.value)
    actualModelLocked.value = true
    modelRestrictionEnabled.value = true
    modelRestrictionMode.value = newPlatform === 'protocol_gateway' ? 'mapping' : 'whitelist'
    allowedModels.value = []
    manualModels.value = []
    resolvedUpstream.value = null
    oauthDraftCredentials.value = {}
    oauthDraftExtra.value = {}
    protocolGatewayProbeModels.value = []
    gatewayClientProfiles.value = []
    gatewayClientRoutes.value = []
    gatewayTestProvider.value = ''
    gatewayTestModelId.value = ''
    gatewayOpenAIRequestFormat.value = DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT
    gatewayOpenAIImageProtocolMode.value = DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE
    gatewayBatchEnabled.value = false
    resetProtocolGatewayClaudeMimicState()
    modelMappings.value = []
    openAIImageProtocolTouched.value = false
    if (newPlatform !== 'anthropic') {
      addMethod.value = 'oauth'
    }
    if (newPlatform !== 'gemini') {
      if (accountCategory.value === 'vertex_ai' || accountCategory.value === 'sso') {
        accountCategory.value = 'oauth-based'
      }
      geminiVertexAuthMode.value = 'service_account'
      geminiVertexProjectId.value = ''
      geminiVertexLocation.value = ''
      geminiVertexServiceAccountJson.value = ''
      geminiVertexApiKey.value = ''
      geminiVertexAccessToken.value = ''
      geminiVertexExpiresAtInput.value = ''
      geminiVertexBaseUrl.value = ''
    }
    if (isBaiduDocumentAISelected.value) {
      accountCategory.value = 'apikey'
      form.type = 'apikey'
      autoImportModels.value = false
      nextTick(() => {
        const editor = document.querySelector('[data-testid="baidu-document-ai-credentials-editor"]') as any
        if (editor && typeof editor.scrollIntoView === 'function') {
          try {
            editor.scrollIntoView({ block: 'start' })
          } catch {
            editor.scrollIntoView()
          }
        }
      })
    } else if (isBaiduDocumentAIPlatform(previousPlatform)) {
      baiduDocumentAIAccessToken.value = ''
      baiduDocumentAIAsyncBaseUrl.value = BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL
      baiduDocumentAIDirectApiUrlsText.value = ''
    }
    if (newPlatform !== 'openrouter') {
      openRouterHTTPReferer.value = ''
      openRouterTitle.value = ''
    }
    if (newPlatform === 'protocol_gateway') {
      accountCategory.value = 'apikey'
      form.type = 'apikey'
      modelRestrictionMode.value = 'mapping'
      gatewayAcceptedProtocols.value = gatewayProtocol.value === 'mixed'
        ? ['openai', 'anthropic', 'gemini']
        : [gatewayProtocol.value as GatewayAcceptedProtocol]
    } else {
      gatewayAcceptedProtocols.value = ['openai']
    }
    if (newPlatform === 'grok') {
      accountCategory.value = 'apikey'
      form.type = 'apikey'
      grokSSOToken.value = ''
      grokOAuthRef.value?.reset?.()
      grokTier.value = 'basic'
    } else {
      grokSSOToken.value = ''
      grokOAuthRef.value?.reset?.()
      grokTier.value = 'basic'
    }
    if (newPlatform === 'deepseek') {
      accountCategory.value = 'apikey'
      form.type = 'apikey'
      autoImportModels.value = false
      deepSeekModelConcurrencyLimits.value = createDefaultDeepSeekModelConcurrencyLimitDraft()
    }
    if (newPlatform === 'openrouter') {
      accountCategory.value = 'apikey'
      form.type = 'apikey'
      autoImportModels.value = false
    }
    if (newPlatform === 'antigravity') {
      loadAntigravityDefaultMappings()
      accountCategory.value = 'oauth-based'
      antigravityAccountType.value = 'oauth'
    } else {
      antigravityModelMappings.value = []
    }
    // Reset Anthropic/Antigravity-specific settings when switching to other platforms
    if (effectivePlatform.value !== 'anthropic' && newPlatform !== 'antigravity') {
      interceptWarmupRequests.value = false
    }
    if (newPlatform === 'kiro') {
      accountCategory.value = 'oauth-based'
      form.type = 'oauth'
    }
    if (effectivePlatform.value !== 'openai') {
      openaiPassthroughEnabled.value = false
      openAIImageProtocolMode.value = 'native'
      openAIImageCompatAllowed.value = true
      openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      codexCLIOnlyEnabled.value = false
    } else {
      applyOpenAIImageProtocolDefaults(undefined, true)
      applyOpenAIOAuthPresetModels(undefined, null, true)
    }
    if (effectivePlatform.value !== 'anthropic') {
      anthropicPassthroughEnabled.value = false
    }
    // Reset OAuth states
    oauth.resetState()
    openaiOAuth.resetState()
    geminiOAuth.resetState()
    antigravityOAuth.resetState()
    resetOAuthInputDraft()
    kiroAuthRef.value?.reset()
  }
)

watch(
  gatewayProtocol,
  (newProtocol: GatewayProtocol, oldProtocol: GatewayProtocol) => {
    if (form.platform !== 'protocol_gateway' || newProtocol === oldProtocol) {
      return
    }
    apiKeyBaseUrl.value = resolveAccountApiKeyDefaultBaseUrl(form.platform, newProtocol)
    allowedModels.value = []
    protocolGatewayProbeModels.value = []
    modelMappings.value = []
    gatewayAcceptedProtocols.value = newProtocol === 'mixed'
      ? ['openai', 'anthropic', 'gemini']
      : [newProtocol as GatewayAcceptedProtocol]
    gatewayClientProfiles.value = []
    gatewayClientRoutes.value = []
    gatewayTestProvider.value = ''
    gatewayTestModelId.value = ''
    gatewayOpenAIRequestFormat.value = DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT
    gatewayOpenAIImageProtocolMode.value = DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE
    gatewayBatchEnabled.value = false
    if (oldProtocol === 'openai' && newProtocol !== 'openai') {
      openaiPassthroughEnabled.value = false
      openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      codexCLIOnlyEnabled.value = false
    }
    if (oldProtocol === 'anthropic' && newProtocol !== 'anthropic') {
      anthropicPassthroughEnabled.value = false
      interceptWarmupRequests.value = false
    }
  }
)

watch(
  accountCategory,
  () => {
    if (form.platform === 'openai') {
      openAIImageProtocolTouched.value = false
      applyOpenAIImageProtocolDefaults(undefined, true)
      if (accountCategory.value === 'oauth-based') {
        accountTier.value = ctx.normalizeAccountTier('openai', accountTier.value) || ctx.defaultAccountTierForPlatform('openai')
        applyAccountTierCapacity()
        applyOpenAIOAuthPresetModels(undefined, null, true)
      } else {
        resetOpenAIOAuthDefaultSelection()
      }
    }
  }
)

watch(
  accountTier,
  () => {
    if (form.platform !== 'openai' && form.platform !== 'anthropic') {
      return
    }
    const normalized = ctx.normalizeAccountTier(form.platform, accountTier.value)
    if (!normalized) {
      accountTier.value = ctx.defaultAccountTierForPlatform(form.platform)
      return
    }
    applyAccountTierCapacity()
    if (form.platform === 'openai' && accountCategory.value === 'oauth-based') {
      applyOpenAIImageProtocolDefaults(undefined, true)
    }
  }
)

watch(
  [allowedModels, modelMappings, modelRestrictionMode],
  () => {
    markOpenAIOAuthDefaultsCustomized()
  },
  { deep: true }
)

watch(
  showProtocolGatewayOpenAIRequestFormatEditor,
  (supported: boolean) => {
    if (!supported) {
      gatewayOpenAIRequestFormat.value = DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT
      gatewayOpenAIImageProtocolMode.value = DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE
    }
  }
)

watch(
  showProtocolGatewayBatchEditor,
  (supported: boolean) => {
    if (!supported) {
      gatewayBatchEnabled.value = false
    }
  }
)

watch(
  [showProtocolGatewayClaudeMimicEditor, claudeCodeMimicEnabled],
  ([supported, enabled]: [boolean, boolean]) => {
    if (!supported) {
      resetProtocolGatewayClaudeMimicState()
      return
    }
    if (!enabled) {
      claudeTLSFingerprintEnabled.value = false
      claudeSessionIDMaskingEnabled.value = false
    }
  }
)

// Gemini AI Studio OAuth availability (requires operator-configured OAuth client)
watch(
  [accountCategory, effectivePlatform],
  ([category, platform]: [AccountCategory, GroupPlatform]) => {
    if (platform === 'openai' && category !== 'oauth-based') {
      codexCLIOnlyEnabled.value = false
    }
    if (platform !== 'anthropic' || category !== 'apikey') {
      anthropicPassthroughEnabled.value = false
    }
  }
)

watch(
  [() => props.show, effectivePlatform, accountCategory],
  async ([show, platform, category]: [boolean, GroupPlatform, AccountCategory]) => {
    if (!show || platform !== 'gemini' || category !== 'oauth-based') {
      geminiAIStudioOAuthEnabled.value = false
      return
    }
    const caps = await geminiOAuth.getCapabilities()
    geminiAIStudioOAuthEnabled.value = !!caps?.ai_studio_oauth_enabled
    if (!geminiAIStudioOAuthEnabled.value && geminiOAuthType.value === 'ai_studio') {
      geminiOAuthType.value = 'code_assist'
    }
  },
  { immediate: true }
)

}
