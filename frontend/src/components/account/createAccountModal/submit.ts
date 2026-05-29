import type { AccountModelProbeSnapshotDraft } from '@/utils/accountProbeDraft'
import type { ParsedKiroTokenImport } from '@/utils/kiroTokenImport'

export function createCreateAccountSubmit(ctx: any) {
  const {
    BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL,
    GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS,
    acceptAIStudioBatchOverflow,
    accountCategory,
    addMethod,
    allowVertexBatchOverflow,
    allowedModels,
    anthropicPassthroughEnabled,
    antigravityAccountType,
    antigravityModelMappings,
    antigravityModelRestrictionMode,
    antigravityOAuth,
    antigravityWhitelistModels,
    apiKeyBaseUrl,
    apiKeyValue,
    appStore,
    applyAccountCustomErrorCodesStateToCredentials,
    applyAccountPoolModeStateToCredentials,
    applyDeepSeekModelConcurrencyLimitsExtra,
    applyInterceptWarmup,
    applyOpenAIImageProtocolDefaults,
    applyProtocolGatewayClaudeClientMimicExtra,
    applyProtocolGatewayGeminiBatchExtra,
    applyProtocolGatewayOpenAIImageProtocolModeExtra,
    applyProtocolGatewayOpenAIRequestFormatExtra,
    applyTempUnschedConfig,
    autoPauseOnExpired,
    baiduDocumentAIAccessToken,
    baiduDocumentAIAsyncBaseUrl,
    baiduDocumentAIDirectApiUrlsText,
    batchArchiveAutoPrefetchEnabled,
    batchArchiveBillingMode,
    batchArchiveDownloadPriceUSD,
    batchArchiveEnabled,
    batchArchiveRetentionDays,
    buildAnthropicExtra,
    buildLocalAccountModelProbeSnapshot,
    buildModelMappingObject,
    buildOpenAIExtra,
    buildTempUnschedPayload,
    claudeCodeMimicEnabled,
    claudeSessionIDMaskingEnabled,
    claudeTLSFingerprintEnabled,
    codexCLIOnlyEnabled,
    computed,
    createAccountModelProbeSnapshotDraft,
    customErrorCodesState,
    deepSeekModelConcurrencyLimits,
    editQuotaDailyLimit,
    editQuotaDailyResetHour,
    editQuotaDailyResetMode,
    editQuotaLimit,
    editQuotaResetTimezone,
    editQuotaWeeklyLimit,
    editQuotaWeeklyResetDay,
    editQuotaWeeklyResetHour,
    editQuotaWeeklyResetMode,
    effectivePlatform,
    emit,
    ensureMixedChannelConfirmed,
    expiryProbeExtensionDays,
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
    geminiTierAIStudio,
    geminiVertexApiKey,
    geminiVertexAuthMode,
    geminiVertexBaseUrl,
    geminiVertexLocation,
    geminiVertexProjectId,
    geminiVertexServiceAccountJson,
    grokSSOToken,
    grokTier,
    handleClose,
    hasCustomizedOpenAIOAuthDefaults,
    interceptWarmupRequests,
    isBaiduDocumentAISelected,
    isOAuthFlow,
    isOpenAIModelRestrictionDisabled,
    isProtocolGatewayPlatform,
    manualModels,
    maybeImportCreatedAccounts,
    mergeAccountManualModelsIntoExtra,
    mergeAccountModelProbeSnapshotIntoExtra,
    mergeResolvedUpstreamDraftIntoExtra,
    mixedScheduling,
    modelMappings,
    modelProbeSnapshot,
    modelRestrictionEnabled,
    modelRestrictionMode,
    normalizeGeminiAIStudioTier,
    oauth,
    oauthDraftCredentials,
    oauthDraftExtra,
    oauthDraftProbeReady,
    oauthFlowRef,
    openAIImageCompatAllowed,
    openAIImageProtocolMode,
    openMixedChannelDialog,
    openRouterHTTPReferer,
    openRouterTitle,
    openaiAPIKeyResponsesWebSocketV2Mode,
    openaiOAuth,
    openaiOAuthResponsesWebSocketV2Mode,
    openaiPassthroughEnabled,
    parseBaiduDocumentAIDirectApiUrlsInput,
    poolModeState,
    quotaControl,
    requiresMixedChannelCheck,
    resolveAccountApiKeyDefaultBaseUrl,
    resolveVertexAuthBaseUrl,
    resolveVertexBaseUrl,
    resolvedUpstream,
    shouldPersistGeminiTierId,
    showOAuthFinalizeStep,
    step,
    t,
    tempUnschedEnabled,
    toRef,
    upstreamApiKey,
    upstreamBaseUrl,
    useCreateAccountAnthropicCookieAuth,
    useCreateAccountAnthropicExchange,
    useCreateAccountAntigravityHandlers,
    useCreateAccountOpenAIExchange,
    useCreateAccountOpenAIRefreshTokenValidation,
    useCreateAccountSubmit,
    withConfirmFlag
  } = ctx

const resolveConfiguredModelProbeSnapshot = () =>
  buildLocalAccountModelProbeSnapshot({
    current: modelProbeSnapshot.value,
    enabled: effectivePlatform.value === 'antigravity' ? true : modelRestrictionEnabled.value,
    modelRestrictionMode: effectivePlatform.value === 'antigravity'
      ? antigravityModelRestrictionMode.value
      : modelRestrictionMode.value,
    allowedModels: effectivePlatform.value === 'antigravity'
      ? antigravityWhitelistModels.value
      : allowedModels.value,
    modelMappings: effectivePlatform.value === 'antigravity'
      ? antigravityModelMappings.value
      : modelMappings.value,
    source: 'model_scope_preview'
  })

const buildAccountExtra = (base?: Record<string, unknown>) => {
  const openaiExtra = buildOpenAIExtra({
    platform: effectivePlatform.value,
    accountCategory: accountCategory.value,
    base: {
      ...(base || {}),
      expiry_probe_extension_days: expiryProbeExtensionDays.value
    },
    openaiOAuthResponsesWebSocketV2Mode: openaiOAuthResponsesWebSocketV2Mode.value,
    openaiAPIKeyResponsesWebSocketV2Mode: openaiAPIKeyResponsesWebSocketV2Mode.value,
    openaiPassthroughEnabled: openaiPassthroughEnabled.value,
    codexCLIOnlyEnabled: codexCLIOnlyEnabled.value,
    openAIImageProtocolMode: openAIImageProtocolMode.value,
    openAIImageCompatAllowed: openAIImageCompatAllowed.value,
    includeOpenAIImageProtocolMode: !isProtocolGatewayPlatform(form.platform)
  })

  const anthropicExtra = buildAnthropicExtra({
    platform: effectivePlatform.value,
    accountCategory: accountCategory.value,
    base: openaiExtra,
    anthropicPassthroughEnabled: anthropicPassthroughEnabled.value
  })

  const extraWithProtocolGateway = !isProtocolGatewayPlatform(form.platform)
    ? anthropicExtra
    : applyProtocolGatewayGeminiBatchExtra(
      applyProtocolGatewayOpenAIImageProtocolModeExtra(
        applyProtocolGatewayOpenAIRequestFormatExtra(
          applyProtocolGatewayClaudeClientMimicExtra({
            ...(anthropicExtra || {}),
            gateway_protocol: gatewayProtocol.value,
            gateway_accepted_protocols: [...gatewayAcceptedProtocols.value],
            gateway_client_profiles: [...gatewayClientProfiles.value],
            gateway_client_routes: gatewayClientRoutes.value.map((route: Record<string, unknown>) => ({ ...route })),
            gateway_test_provider: gatewayTestProvider.value || undefined,
            gateway_test_model_id: gatewayTestModelId.value || undefined
          }, {
            platform: form.platform,
            type: form.type,
            gatewayProtocol: gatewayProtocol.value,
            acceptedProtocols: gatewayAcceptedProtocols.value,
            claudeCodeMimicEnabled: claudeCodeMimicEnabled.value,
            enableTLSFingerprint: claudeTLSFingerprintEnabled.value,
            sessionIDMaskingEnabled: claudeSessionIDMaskingEnabled.value
          }),
          {
            platform: form.platform,
            type: form.type,
            gatewayProtocol: gatewayProtocol.value,
            acceptedProtocols: gatewayAcceptedProtocols.value,
            gatewayOpenAIRequestFormat: gatewayOpenAIRequestFormat.value
          }
        ),
        {
          platform: form.platform,
          type: form.type,
          gatewayProtocol: gatewayProtocol.value,
          acceptedProtocols: gatewayAcceptedProtocols.value,
          gatewayOpenAIImageProtocolMode: gatewayOpenAIImageProtocolMode.value
        }
      ),
      {
        platform: form.platform,
        type: form.type,
        gatewayProtocol: gatewayProtocol.value,
        acceptedProtocols: gatewayAcceptedProtocols.value,
        gatewayBatchEnabled: gatewayBatchEnabled.value
      }
    )

  const extraWithDeepSeek = applyDeepSeekModelConcurrencyLimitsExtra(
    extraWithProtocolGateway,
    effectivePlatform.value,
    deepSeekModelConcurrencyLimits.value
  )

  return mergeResolvedUpstreamDraftIntoExtra(
    mergeAccountModelProbeSnapshotIntoExtra(
      mergeAccountManualModelsIntoExtra(
        extraWithDeepSeek,
        manualModels.value,
        isProtocolGatewayPlatform(form.platform)
      ),
      resolveConfiguredModelProbeSnapshot()
    ),
    resolvedUpstream.value
  )
}

function buildBaiduDocumentAICredentialsForCreate(): Record<string, unknown> | null {
  const accessToken = baiduDocumentAIAccessToken.value.trim()
  if (!accessToken) {
    appStore.showError(t('admin.accounts.baiduDocumentAI.tokenRequired'))
    return null
  }

  let directAPIURLs: Record<string, string> = {}
  try {
    directAPIURLs = parseBaiduDocumentAIDirectApiUrlsInput(
      baiduDocumentAIDirectApiUrlsText.value
    )
  } catch {
    appStore.showError(t('admin.accounts.baiduDocumentAI.directApiUrlsInvalid'))
    return null
  }

  const credentials: Record<string, unknown> = {
    async_base_url:
      baiduDocumentAIAsyncBaseUrl.value.trim() ||
      BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL
  }
  credentials.async_bearer_token = accessToken
  credentials.direct_token = accessToken
  if (Object.keys(directAPIURLs).length > 0) {
    credentials.direct_api_urls = directAPIURLs
  }
  return credentials
}

const { submitting, createAccountAndFinish } = useCreateAccountSubmit({
  withConfirmFlag,
  ensureMixedChannelConfirmed,
  requiresMixedChannelCheck,
  openMixedChannelDialog,
  isOpenAIModelRestrictionDisabled,
  modelRestrictionEnabled,
  modelRestrictionMode,
  allowedModels,
  modelMappings,
  antigravityModelMappings,
  applyTempUnschedConfig,
  form,
  autoPauseOnExpired,
  expiryProbeExtensionDays,
  editQuotaLimit,
  editQuotaDailyLimit,
  editQuotaWeeklyLimit,
  editQuotaDailyResetMode,
  editQuotaDailyResetHour,
  editQuotaWeeklyResetMode,
  editQuotaWeeklyResetDay,
  editQuotaWeeklyResetHour,
  editQuotaResetTimezone,
  batchArchiveEnabled,
  batchArchiveAutoPrefetchEnabled,
  batchArchiveRetentionDays,
  batchArchiveBillingMode,
  batchArchiveDownloadPriceUSD,
  allowVertexBatchOverflow,
  acceptAIStudioBatchOverflow,
  afterCreateImportModels: maybeImportCreatedAccounts,
  emitCreated: () => emit('created'),
  onClose: handleClose
})

const { handleAnthropicExchange } = useCreateAccountAnthropicExchange({
  oauthClient: oauth,
  platform: toRef(form, 'platform'),
  addMethod,
  proxyId: toRef(form, 'proxy_id'),
  interceptWarmupRequests,
  quotaControl,
  createAccountAndFinish
})

const { handleCookieAuth } = useCreateAccountAnthropicCookieAuth({
  oauthClient: oauth,
  platform: toRef(form, 'platform'),
  addMethod,
  proxyId: toRef(form, 'proxy_id'),
  form,
  autoPauseOnExpired,
  interceptWarmupRequests,
  quotaControl,
  tempUnschedEnabled,
  buildTempUnschedPayload,
  afterCreateImportModels: maybeImportCreatedAccounts,
  emitCreated: () => emit('created'),
  onClose: handleClose
})

const getOpenAIOAuthState = () =>
  (oauthFlowRef.value?.oauthState || openaiOAuth.oauthState.value || '').trim()

const { handleOpenAIExchange } = useCreateAccountOpenAIExchange({
  oauthClient: computed(() => openaiOAuth),
  getOAuthState: getOpenAIOAuthState,
  form,
  autoPauseOnExpired,
  applyTempUnschedConfig,
  isOpenAIModelRestrictionDisabled,
  modelRestrictionEnabled,
  modelRestrictionMode,
  allowedModels,
  modelMappings,
  hasCustomizedOpenAIDefaults: hasCustomizedOpenAIOAuthDefaults,
  buildAccountExtra,
  applyOpenAIImageProtocolDefaults: (planType: string | undefined) => applyOpenAIImageProtocolDefaults(planType),
  afterCreateImportModels: maybeImportCreatedAccounts,
  emitCreated: () => emit('created'),
  onClose: handleClose
})

const { handleOpenAIValidateRT } = useCreateAccountOpenAIRefreshTokenValidation({
  oauthClient: computed(() => openaiOAuth),
  form,
  autoPauseOnExpired,
  isOpenAIModelRestrictionDisabled,
  modelRestrictionEnabled,
  modelRestrictionMode,
  allowedModels,
  modelMappings,
  hasCustomizedOpenAIDefaults: hasCustomizedOpenAIOAuthDefaults,
  buildAccountExtra,
  applyOpenAIImageProtocolDefaults: (planType: string | undefined) => applyOpenAIImageProtocolDefaults(planType),
  afterCreateImportModels: maybeImportCreatedAccounts,
  emitCreated: () => emit('created'),
  onClose: handleClose
})

const getAntigravityOAuthState = () =>
  (oauthFlowRef.value?.oauthState || antigravityOAuth.state.value || '').trim()

const { handleAntigravityValidateRT, handleAntigravityExchange } = useCreateAccountAntigravityHandlers({
  oauthClient: antigravityOAuth,
  getOAuthState: getAntigravityOAuthState,
  withConfirmFlag,
  form,
  autoPauseOnExpired,
  interceptWarmupRequests,
  antigravityModelMappings,
  mixedScheduling,
  createAccountAndFinish,
  afterCreateImportModels: maybeImportCreatedAccounts,
  emitCreated: () => emit('created'),
  onClose: handleClose
})

const handleCreateKiroAccount = async (payload: ParsedKiroTokenImport) => {
  oauthDraftCredentials.value = { ...(payload.credentials || {}) }
  oauthDraftExtra.value = { ...(payload.extra || {}) }
  modelProbeSnapshot.value = createAccountModelProbeSnapshotDraft(
    payload.extra?.model_probe_snapshot as AccountModelProbeSnapshotDraft | null | undefined
  )
  step.value = 3
}


const handleSubmit = async () => {
  // For OAuth-based type, handle OAuth flow (goes to step 2)
  if (isOAuthFlow.value) {
    if (!form.name.trim()) {
      appStore.showError(t('admin.accounts.pleaseEnterAccountName'))
      return
    }
    if (showOAuthFinalizeStep.value && step.value === 3 && oauthDraftProbeReady.value) {
      await createAccountAndFinish(
        form.platform,
        'oauth',
        { ...oauthDraftCredentials.value },
        buildAccountExtra(oauthDraftExtra.value)
      )
      return
    }
    const canContinue = await ensureMixedChannelConfirmed(async () => {
      step.value = 2
    })
    if (!canContinue) {
      return
    }
    step.value = 2
    return
  }

  // For Antigravity upstream type, create directly
  if (form.platform === 'antigravity' && antigravityAccountType.value === 'upstream') {
    if (!form.name.trim()) {
      appStore.showError(t('admin.accounts.pleaseEnterAccountName'))
      return
    }
    if (!upstreamBaseUrl.value.trim()) {
      appStore.showError(t('admin.accounts.upstream.pleaseEnterBaseUrl'))
      return
    }
    if (!upstreamApiKey.value.trim()) {
      appStore.showError(t('admin.accounts.upstream.pleaseEnterApiKey'))
      return
    }

    // Build upstream credentials (and optional model restriction)
    const credentials: Record<string, unknown> = {
      base_url: upstreamBaseUrl.value.trim(),
      api_key: upstreamApiKey.value.trim()
    }

    const antigravityModelMapping = buildModelMappingObject(
      'mapping',
      [],
      antigravityModelMappings.value
    )
    if (antigravityModelMapping) {
      credentials.model_mapping = antigravityModelMapping
    }

    applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')

    const extra = mixedScheduling.value ? { mixed_scheduling: true } : undefined
    await createAccountAndFinish(form.platform, 'apikey', credentials, extra)
    return
  }

  if (form.platform === 'grok' && form.type === 'sso') {
    if (!form.name.trim()) {
      appStore.showError(t('admin.accounts.pleaseEnterAccountName'))
      return
    }
    if (!grokSSOToken.value.trim()) {
      appStore.showError(t('admin.accounts.pleaseEnterGrokToken'))
      return
    }

    const credentials: Record<string, unknown> = {
      sso_token: grokSSOToken.value.trim()
    }
    if (!isOpenAIModelRestrictionDisabled.value && modelRestrictionEnabled.value) {
      const modelMapping = buildModelMappingObject(
        modelRestrictionMode.value,
        allowedModels.value,
        modelMappings.value
      )
      if (modelMapping) {
        credentials.model_mapping = modelMapping
      }
    }

    await createAccountAndFinish(
      form.platform,
      'sso',
      credentials,
      buildAccountExtra({ grok_tier: grokTier.value })
    )
    return
  }

  if (form.platform === 'gemini' && accountCategory.value === 'vertex_ai') {
    if (!form.name.trim()) {
      appStore.showError(t('admin.accounts.pleaseEnterAccountName'))
      return
    }
    if (geminiVertexAuthMode.value === 'express_api_key') {
      if (!geminiVertexApiKey.value.trim()) {
        appStore.showError(t('admin.accounts.gemini.vertex.expressApiKeyRequired'))
        return
      }

      const credentials: Record<string, unknown> = {
        gemini_api_variant: GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS,
        api_key: geminiVertexApiKey.value.trim(),
        base_url: geminiVertexBaseUrl.value.trim() || resolveVertexAuthBaseUrl('express_api_key', '')
      }
      const modelMapping = modelRestrictionEnabled.value
        ? buildModelMappingObject('mapping', [], modelMappings.value)
        : null
      if (modelMapping) {
        credentials.model_mapping = modelMapping
      }

      await createAccountAndFinish('gemini', 'apikey', credentials)
      return
    }
    if (!geminiVertexProjectId.value.trim()) {
      appStore.showError(t('admin.accounts.gemini.vertex.projectIdRequired'))
      return
    }
    if (!geminiVertexLocation.value.trim()) {
      appStore.showError(t('admin.accounts.gemini.vertex.locationRequired'))
      return
    }
    if (!geminiVertexServiceAccountJson.value.trim()) {
      appStore.showError(t('admin.accounts.gemini.vertex.serviceAccountJsonRequired'))
      return
    }

    const credentials: Record<string, unknown> = {
      oauth_type: 'vertex_ai',
      vertex_project_id: geminiVertexProjectId.value.trim(),
      vertex_location: geminiVertexLocation.value.trim(),
      vertex_service_account_json: geminiVertexServiceAccountJson.value.trim()
    }
    credentials.base_url = geminiVertexBaseUrl.value.trim() || resolveVertexBaseUrl(geminiVertexLocation.value)
    const modelMapping = modelRestrictionEnabled.value
      ? buildModelMappingObject('mapping', [], modelMappings.value)
      : null
    if (modelMapping) {
      credentials.model_mapping = modelMapping
    }

    await createAccountAndFinish('gemini', 'oauth', credentials)
    return
  }

  if (isBaiduDocumentAISelected.value) {
    if (!form.name.trim()) {
      appStore.showError(t('admin.accounts.pleaseEnterAccountName'))
      return
    }

    const credentials = buildBaiduDocumentAICredentialsForCreate()
    if (!credentials) {
      return
    }

    await createAccountAndFinish('baidu_document_ai', 'apikey', credentials)
    return
  }

  // For apikey type, create directly
  if (!apiKeyValue.value.trim()) {
    appStore.showError(t('admin.accounts.pleaseEnterApiKey'))
    return
  }

  // Determine default base URL based on platform
  const defaultBaseUrl = resolveAccountApiKeyDefaultBaseUrl(form.platform, gatewayProtocol.value)

  // Build credentials with optional model mapping
  const credentials: Record<string, unknown> = {
    base_url: apiKeyBaseUrl.value.trim() || defaultBaseUrl,
    api_key: apiKeyValue.value.trim()
  }
  if (shouldPersistGeminiTierId.value) {
    credentials.tier_id = normalizeGeminiAIStudioTier(geminiTierAIStudio.value)
  }
  if (form.platform === 'openrouter') {
    if (openRouterHTTPReferer.value.trim()) {
      credentials.http_referer = openRouterHTTPReferer.value.trim()
    }
    if (openRouterTitle.value.trim()) {
      credentials.openrouter_title = openRouterTitle.value.trim()
    }
  }

  if (!isOpenAIModelRestrictionDisabled.value && modelRestrictionEnabled.value) {
    const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
    if (modelMapping) {
      credentials.model_mapping = modelMapping
    }
  }

  applyAccountPoolModeStateToCredentials(credentials, poolModeState)
  applyAccountCustomErrorCodesStateToCredentials(credentials, customErrorCodesState)

  applyInterceptWarmup(credentials, interceptWarmupRequests.value, 'create')
  const extra = buildAccountExtra(
    form.platform === 'grok' && form.type === 'sso'
      ? { grok_tier: grokTier.value }
      : undefined
  )
  await createAccountAndFinish(
    form.platform,
    'apikey',
    credentials,
    extra,
    isProtocolGatewayPlatform(form.platform) ? gatewayProtocol.value : undefined
  )
}


return {
  resolveConfiguredModelProbeSnapshot,
  buildAccountExtra,
  buildBaiduDocumentAICredentialsForCreate,
  submitting,
  createAccountAndFinish,
  handleAnthropicExchange,
  handleCookieAuth,
  getOpenAIOAuthState,
  handleOpenAIExchange,
  handleOpenAIValidateRT,
  getAntigravityOAuthState,
  handleAntigravityValidateRT,
  handleAntigravityExchange,
  handleCreateKiroAccount,
  handleSubmit
}
}
