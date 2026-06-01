import type { Account, GatewayClientProfile, GatewayProtocol } from '@/types'

export function useEditAccountModalWatchers(ctx: any) {
  const {
    BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL,
    DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE,
    DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT,
    DEFAULT_POOL_MODE_RETRY_COUNT,
    GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS,
    OPENAI_WS_MODE_OFF,
    acceptAIStudioBatchOverflow,
    actualModelLocked,
    allowVertexBatchOverflow,
    allowedModels,
    anthropicPassthroughEnabled,
    antigravityModelMappings,
    antigravityModelRestrictionMode,
    antigravityWhitelistModels,
    applyModelRestrictionFromRecord,
    autoPauseOnExpired,
    autoRenewEnabled,
    autoRenewPeriod,
    baiduDocumentAIAccessToken,
    baiduDocumentAIAsyncBaseUrl,
    baiduDocumentAIDirectApiUrlsText,
    batchArchiveAutoPrefetchEnabled,
    batchArchiveBillingMode,
    batchArchiveDownloadPriceUSD,
    batchArchiveEnabled,
    batchArchiveRetentionDays,
    claudeCodeMimicEnabled,
    claudeSessionIDMaskingEnabled,
    claudeTLSFingerprintEnabled,
    codexCLIOnlyEnabled,
    createDefaultDeepSeekModelConcurrencyLimitDraft,
    createStaticProbeModels,
    customErrorCodesState,
    deepSeekModelConcurrencyLimits,
    defaultGoogleBatchArchiveState,
    deriveConfiguredAccountModelIds,
    editApiKey,
    editBaseUrl,
    editGrokSSOToken,
    editGrokTier,
    editOpenRouterHTTPReferer,
    editOpenRouterTitle,
    editQuotaDailyLimit,
    editQuotaDailyResetHour,
    editQuotaDailyResetMode,
    editQuotaLimit,
    editQuotaResetTimezone,
    editQuotaWeeklyLimit,
    editQuotaWeeklyResetDay,
    editQuotaWeeklyResetHour,
    editQuotaWeeklyResetMode,
    ensureModelRegistryFresh,
    expiryProbeExtensionDays,
    form,
    formatDateTimeLocal,
    gatewayAcceptedProtocols,
    gatewayBatchEnabled,
    gatewayClientProfiles,
    gatewayClientRoutes,
    gatewayOpenAIImageProtocolMode,
    gatewayOpenAIRequestFormat,
    gatewayProtocol,
    gatewayTestModelId,
    gatewayTestProvider,
    geminiOAuthType,
    geminiTierAIStudio,
    geminiVertexAccessToken,
    geminiVertexApiKey,
    geminiVertexAuthMode,
    geminiVertexBaseUrl,
    geminiVertexExpiresAtInput,
    geminiVertexLocation,
    geminiVertexProjectId,
    geminiVertexServiceAccountJson,
    grokDefaultModelMappingForTier,
    interceptWarmupRequests,
    isBaiduDocumentAIPlatform,
    isGeminiVertexAI,
    isInitializingGatewayProtocol,
    isProtocolGatewayAccount,
    loadAccountCustomErrorCodesStateFromCredentials,
    loadAccountPoolModeStateFromCredentials,
    loadModelScopeFromExtra,
    loadTempUnschedRules,
    manualModels,
    mixedScheduling,
    modelMappings,
    modelProbeSnapshot,
    modelRestrictionEnabled,
    modelRestrictionMode,
    normalizeGatewayAcceptedProtocols,
    normalizeGatewayBatchEnabled,
    normalizeGatewayClientProfile,
    normalizeGatewayClientRoutes,
    normalizeGeminiAIStudioTier,
    normalizeGeminiOAuthType,
    normalizeGrokTier,
    openAIImageCompatAllowed,
    openAIImageProtocolMode,
    openaiAPIKeyResponsesWebSocketV2Mode,
    openaiOAuthResponsesWebSocketV2Mode,
    openaiPassthroughEnabled,
    poolModeState,
    protocolGatewayProbeModels,
    quotaControl,
    readAccountManualModelsFromExtra,
    readAccountModelProbeSnapshot,
    readAccountResolvedUpstreamDraft,
    readDeepSeekModelConcurrencyLimitDraft,
    readGoogleBatchArchiveFormState,
    resetAccountCustomErrorCodesState,
    resetAccountPoolModeState,
    resetMixedChannelRisk,
    resetProtocolGatewayClaudeMimicState,
    resetTempUnschedRules,
    resolveAccountApiKeyDefaultBaseUrl,
    resolveAccountGatewayOpenAIImageProtocolMode,
    resolveAccountGatewayOpenAIRequestFormat,
    resolveAccountGatewayProtocol,
    resolveEffectiveAccountPlatform,
    resolveOpenAIImageProtocolState,
    resolveOpenAIWSModeFromExtra,
    resolvedUpstream,
    showProtocolGatewayBatchEditor,
    showProtocolGatewayClaudeMimicEditor,
    showProtocolGatewayOpenAIRequestFormatEditor,
    stringifyBaiduDocumentAIDirectApiUrls,
    watch,
    props
  } = ctx

// Watchers
watch(
  () => [props.show, props.account] as const,
  ([show, newAccount]: readonly [boolean, Account | null]) => {
    if (show && newAccount) {
      void ensureModelRegistryFresh()
      isInitializingGatewayProtocol.value = true
      actualModelLocked.value = true
      resetMixedChannelRisk()
      gatewayProtocol.value = resolveAccountGatewayProtocol(newAccount) || 'openai'
      gatewayAcceptedProtocols.value = normalizeGatewayAcceptedProtocols(
        gatewayProtocol.value,
        newAccount.extra?.gateway_accepted_protocols
      )
      const rawGatewayClientProfiles = newAccount.extra?.gateway_client_profiles
      gatewayClientProfiles.value = (Array.isArray(rawGatewayClientProfiles)
        ? rawGatewayClientProfiles
        : []
      )
        .map((value) => normalizeGatewayClientProfile(value))
        .filter((value): value is GatewayClientProfile => Boolean(value))
      gatewayClientRoutes.value = normalizeGatewayClientRoutes(newAccount.extra?.gateway_client_routes)
      gatewayTestProvider.value = String(newAccount.extra?.gateway_test_provider || '').trim().toLowerCase()
      gatewayTestModelId.value = String(newAccount.extra?.gateway_test_model_id || '').trim()
      gatewayOpenAIRequestFormat.value = resolveAccountGatewayOpenAIRequestFormat(newAccount)
      gatewayOpenAIImageProtocolMode.value = resolveAccountGatewayOpenAIImageProtocolMode(newAccount)
      gatewayBatchEnabled.value = normalizeGatewayBatchEnabled(
        newAccount.gateway_batch_enabled ?? newAccount.extra?.gateway_batch_enabled
      )
      claudeCodeMimicEnabled.value =
        newAccount.claude_code_mimic_enabled === true || newAccount.extra?.claude_code_mimic_enabled === true
      claudeTLSFingerprintEnabled.value = newAccount.enable_tls_fingerprint === true
      claudeSessionIDMaskingEnabled.value = newAccount.session_id_masking_enabled === true
      protocolGatewayProbeModels.value = []
      const runtimePlatform = resolveEffectiveAccountPlatform(
        newAccount.platform,
        resolveAccountGatewayProtocol(newAccount) || gatewayProtocol.value
      )
      form.name = newAccount.name
      form.notes = newAccount.notes || ''
      form.proxy_id = newAccount.proxy_id
      form.concurrency = newAccount.concurrency
      form.load_factor = newAccount.load_factor ?? null
      form.priority = newAccount.priority
      form.rate_multiplier = newAccount.rate_multiplier ?? 1
      form.status = (newAccount.status === 'active' || newAccount.status === 'inactive' || newAccount.status === 'error')
        ? newAccount.status
        : 'active'
      form.group_ids = newAccount.group_ids || []
      form.expires_at = newAccount.expires_at ?? null

      // Load intercept warmup requests setting (applies to all account types)
      const credentials = newAccount.credentials as Record<string, unknown> | undefined
      interceptWarmupRequests.value = credentials?.intercept_warmup_requests === true
      autoPauseOnExpired.value = newAccount.auto_pause_on_expired === true
      autoRenewEnabled.value = newAccount.auto_renew_enabled === true
      autoRenewPeriod.value =
        newAccount.auto_renew_period === 'quarter' || newAccount.auto_renew_period === 'year'
          ? newAccount.auto_renew_period
          : 'month'

      // Load mixed scheduling setting (only for antigravity accounts)
      const extra = newAccount.extra as Record<string, unknown> | undefined
      deepSeekModelConcurrencyLimits.value = readDeepSeekModelConcurrencyLimitDraft(extra)
      expiryProbeExtensionDays.value = Math.max(
        1,
        Number.parseInt(String(extra?.expiry_probe_extension_days || ''), 10) || 1
      )
      manualModels.value = readAccountManualModelsFromExtra(extra, isProtocolGatewayAccount.value)
      modelProbeSnapshot.value = readAccountModelProbeSnapshot(extra)
      resolvedUpstream.value = readAccountResolvedUpstreamDraft(extra)
      if (!resolvedUpstream.value?.upstream_probed_at && modelProbeSnapshot.value?.updated_at) {
        resolvedUpstream.value = {
          ...(resolvedUpstream.value || {}),
          upstream_probed_at: modelProbeSnapshot.value.updated_at
        }
      }
      mixedScheduling.value = extra?.mixed_scheduling === true
      editGrokSSOToken.value = ''
      editGrokTier.value = normalizeGrokTier(extra?.grok_tier)

      // Load OpenAI passthrough toggle (OpenAI OAuth/API Key)
      openaiPassthroughEnabled.value = false
      openAIImageProtocolMode.value = 'native'
      openAIImageCompatAllowed.value = true
      openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      codexCLIOnlyEnabled.value = false
      anthropicPassthroughEnabled.value = false
      if (runtimePlatform === 'openai' && (newAccount.type === 'oauth' || newAccount.type === 'apikey')) {
        const openAIImageState = resolveOpenAIImageProtocolState({
          accountCategory: newAccount.type === 'oauth' ? 'oauth-based' : 'apikey',
          planType: String(credentials?.plan_type || ''),
          storedMode: String(extra?.image_protocol_mode || ''),
          storedCompatAllowed: extra?.image_compat_allowed
        })
        openaiPassthroughEnabled.value = extra?.openai_passthrough === true || extra?.openai_oauth_passthrough === true
        openAIImageProtocolMode.value = openAIImageState.mode
        openAIImageCompatAllowed.value = openAIImageState.compatAllowed
        openaiOAuthResponsesWebSocketV2Mode.value = resolveOpenAIWSModeFromExtra(extra, {
          modeKey: 'openai_oauth_responses_websockets_v2_mode',
          enabledKey: 'openai_oauth_responses_websockets_v2_enabled',
          fallbackEnabledKeys: ['responses_websockets_v2_enabled', 'openai_ws_enabled'],
          defaultMode: OPENAI_WS_MODE_OFF
        })
        openaiAPIKeyResponsesWebSocketV2Mode.value = resolveOpenAIWSModeFromExtra(extra, {
          modeKey: 'openai_apikey_responses_websockets_v2_mode',
          enabledKey: 'openai_apikey_responses_websockets_v2_enabled',
          fallbackEnabledKeys: ['responses_websockets_v2_enabled', 'openai_ws_enabled'],
          defaultMode: OPENAI_WS_MODE_OFF
        })
        if (newAccount.type === 'oauth') {
          codexCLIOnlyEnabled.value = extra?.codex_cli_only === true
        }
      }
      if (runtimePlatform === 'anthropic' && newAccount.type === 'apikey') {
        anthropicPassthroughEnabled.value = extra?.anthropic_passthrough === true
      }

      const quotaVal = Number(extra?.quota_limit)
      editQuotaLimit.value = Number.isFinite(quotaVal) && quotaVal > 0 ? quotaVal : null
      const dailyVal = Number(extra?.quota_daily_limit)
      editQuotaDailyLimit.value = Number.isFinite(dailyVal) && dailyVal > 0 ? dailyVal : null
      const weeklyVal = Number(extra?.quota_weekly_limit)
      editQuotaWeeklyLimit.value = Number.isFinite(weeklyVal) && weeklyVal > 0 ? weeklyVal : null

      const dailyMode = extra?.quota_daily_reset_mode
      editQuotaDailyResetMode.value = dailyMode === 'fixed' || dailyMode === 'rolling' ? dailyMode : null
      const dailyHour = Number(extra?.quota_daily_reset_hour)
      editQuotaDailyResetHour.value = Number.isFinite(dailyHour) ? dailyHour : null

      const weeklyMode = extra?.quota_weekly_reset_mode
      editQuotaWeeklyResetMode.value = weeklyMode === 'fixed' || weeklyMode === 'rolling' ? weeklyMode : null
      const weeklyDay = Number(extra?.quota_weekly_reset_day)
      editQuotaWeeklyResetDay.value = Number.isFinite(weeklyDay) ? weeklyDay : null
      const weeklyHour = Number(extra?.quota_weekly_reset_hour)
      editQuotaWeeklyResetHour.value = Number.isFinite(weeklyHour) ? weeklyHour : null

      const resetTz = extra?.quota_reset_timezone
      editQuotaResetTimezone.value = typeof resetTz === 'string' && resetTz.trim() ? resetTz : null
      const batchArchiveState = readGoogleBatchArchiveFormState({
        batch_archive_enabled:
          newAccount.batch_archive_enabled ?? extra?.batch_archive_enabled,
        batch_archive_auto_prefetch_enabled:
          newAccount.batch_archive_auto_prefetch_enabled ??
          extra?.batch_archive_auto_prefetch_enabled,
        batch_archive_retention_days:
          newAccount.batch_archive_retention_days ??
          extra?.batch_archive_retention_days,
        batch_archive_billing_mode:
          newAccount.batch_archive_billing_mode ??
          extra?.batch_archive_billing_mode,
        batch_archive_download_price_usd:
          newAccount.batch_archive_download_price_usd ??
          extra?.batch_archive_download_price_usd,
        allow_vertex_batch_overflow:
          newAccount.allow_vertex_batch_overflow ??
          extra?.allow_vertex_batch_overflow,
        accept_aistudio_batch_overflow:
          newAccount.accept_aistudio_batch_overflow ??
          extra?.accept_aistudio_batch_overflow
      })
      batchArchiveEnabled.value = batchArchiveState.enabled
      batchArchiveAutoPrefetchEnabled.value = batchArchiveState.autoPrefetchEnabled
      batchArchiveRetentionDays.value = batchArchiveState.retentionDays
      batchArchiveBillingMode.value = batchArchiveState.billingMode
      batchArchiveDownloadPriceUSD.value = batchArchiveState.downloadPriceUSD
      allowVertexBatchOverflow.value = batchArchiveState.allowVertexBatchOverflow
      acceptAIStudioBatchOverflow.value = batchArchiveState.acceptAIStudioBatchOverflow
      if (runtimePlatform === 'gemini' && newAccount.type === 'apikey') {
        geminiTierAIStudio.value = normalizeGeminiAIStudioTier(credentials?.tier_id)
      } else {
        geminiTierAIStudio.value = 'aistudio_free'
      }
      if (runtimePlatform === 'gemini' && newAccount.type === 'oauth') {
        geminiOAuthType.value = normalizeGeminiOAuthType(credentials?.oauth_type)
        if (isGeminiVertexAI(geminiOAuthType.value)) {
          geminiVertexAuthMode.value = 'service_account'
          geminiVertexProjectId.value = String(credentials?.vertex_project_id || '').trim()
          geminiVertexLocation.value = String(credentials?.vertex_location || '').trim()
          geminiVertexServiceAccountJson.value = String(credentials?.vertex_service_account_json || '').trim()
          geminiVertexApiKey.value = ''
          geminiVertexAccessToken.value = ''
          geminiVertexBaseUrl.value = String(credentials?.base_url || '').trim()
          const rawExpiresAt = Number.parseInt(String(credentials?.expires_at || ''), 10)
          geminiVertexExpiresAtInput.value = Number.isFinite(rawExpiresAt)
            ? formatDateTimeLocal(rawExpiresAt)
            : ''
        } else {
          geminiVertexAuthMode.value = 'service_account'
          geminiVertexProjectId.value = ''
          geminiVertexLocation.value = ''
          geminiVertexServiceAccountJson.value = ''
          geminiVertexApiKey.value = ''
          geminiVertexAccessToken.value = ''
          geminiVertexExpiresAtInput.value = ''
          geminiVertexBaseUrl.value = ''
        }
      } else if (
        runtimePlatform === 'gemini' &&
        newAccount.type === 'apikey' &&
        String(credentials?.gemini_api_variant || '').trim().toLowerCase() === GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS
      ) {
        geminiOAuthType.value = 'vertex_ai'
        geminiVertexAuthMode.value = 'express_api_key'
        geminiVertexProjectId.value = ''
        geminiVertexLocation.value = ''
        geminiVertexServiceAccountJson.value = ''
        geminiVertexApiKey.value = ''
        geminiVertexAccessToken.value = ''
        geminiVertexExpiresAtInput.value = ''
        geminiVertexBaseUrl.value = String(credentials?.base_url || '').trim()
      } else {
        geminiOAuthType.value = 'code_assist'
        geminiVertexAuthMode.value = 'service_account'
        geminiVertexProjectId.value = ''
        geminiVertexLocation.value = ''
        geminiVertexServiceAccountJson.value = ''
        geminiVertexApiKey.value = ''
        geminiVertexAccessToken.value = ''
        geminiVertexExpiresAtInput.value = ''
        geminiVertexBaseUrl.value = ''
      }

      if (newAccount.platform === 'antigravity') {
        const credentials = newAccount.credentials as Record<string, unknown> | undefined

        antigravityModelRestrictionMode.value = 'mapping'
        antigravityWhitelistModels.value = []

        const rawAgMapping = credentials?.model_mapping as Record<string, string> | undefined
        if (rawAgMapping && typeof rawAgMapping === 'object') {
          const entries = Object.entries(rawAgMapping)
          antigravityModelMappings.value = entries.map(([from, to]) => ({ from, to }))
        } else {
          const rawWhitelist = credentials?.model_whitelist
          if (Array.isArray(rawWhitelist) && rawWhitelist.length > 0) {
            antigravityModelMappings.value = rawWhitelist
              .map((v) => String(v).trim())
              .filter((v) => v.length > 0)
              .map((m) => ({ from: m, to: m }))
          } else {
            antigravityModelMappings.value = []
          }
        }
      } else {
        antigravityModelRestrictionMode.value = 'mapping'
        antigravityWhitelistModels.value = []
        antigravityModelMappings.value = []
      }
      quotaControl.loadFromAccount(newAccount)

      loadTempUnschedRules(credentials)

      // Initialize API Key fields for apikey type
        if (newAccount.type === 'apikey' && newAccount.credentials) {
          const credentials = newAccount.credentials as Record<string, unknown>
          if (isBaiduDocumentAIPlatform(newAccount.platform)) {
          const loadedFromScope = loadModelScopeFromExtra(extra)
          if (!loadedFromScope) {
            applyModelRestrictionFromRecord(undefined)
          }
          baiduDocumentAIAccessToken.value =
            String(credentials.async_bearer_token || '').trim() ||
            String(credentials.direct_token || '').trim()
          baiduDocumentAIAsyncBaseUrl.value =
            String(credentials.async_base_url || '').trim() ||
            BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL
          baiduDocumentAIDirectApiUrlsText.value = stringifyBaiduDocumentAIDirectApiUrls(
            credentials.direct_api_urls
          )
          resetAccountPoolModeState(poolModeState, DEFAULT_POOL_MODE_RETRY_COUNT)
          resetAccountCustomErrorCodesState(customErrorCodesState)
          editBaseUrl.value = ''
        } else {
          const platformDefaultUrl = resolveAccountApiKeyDefaultBaseUrl(newAccount.platform, gatewayProtocol.value)
          editBaseUrl.value = (credentials.base_url as string) || platformDefaultUrl
          if (newAccount.platform === 'openrouter') {
            editOpenRouterHTTPReferer.value = String(credentials.http_referer || '').trim()
            editOpenRouterTitle.value = String(credentials.openrouter_title || '').trim()
          } else {
            editOpenRouterHTTPReferer.value = ''
            editOpenRouterTitle.value = ''
          }

          const loadedFromScope = loadModelScopeFromExtra(extra)
          if (!loadedFromScope) {
            applyModelRestrictionFromRecord(credentials.model_mapping)
          }

          loadAccountPoolModeStateFromCredentials(poolModeState, credentials, DEFAULT_POOL_MODE_RETRY_COUNT)
          loadAccountCustomErrorCodesStateFromCredentials(customErrorCodesState, credentials)
        }
      } else if (newAccount.type === 'sso' && newAccount.platform === 'grok' && newAccount.credentials) {
        const credentials = newAccount.credentials as Record<string, unknown>
        editOpenRouterHTTPReferer.value = ''
        editOpenRouterTitle.value = ''
        editBaseUrl.value = resolveAccountApiKeyDefaultBaseUrl(newAccount.platform, gatewayProtocol.value)
        applyModelRestrictionFromRecord(
          credentials.model_mapping || grokDefaultModelMappingForTier(editGrokTier.value)
        )
        resetAccountPoolModeState(poolModeState, DEFAULT_POOL_MODE_RETRY_COUNT)
        resetAccountCustomErrorCodesState(customErrorCodesState)
      } else if (newAccount.type === 'upstream' && newAccount.credentials) {
        const credentials = newAccount.credentials as Record<string, unknown>
        editOpenRouterHTTPReferer.value = ''
        editOpenRouterTitle.value = ''
        editBaseUrl.value = (credentials.base_url as string) || ''
        const loadedFromScope = loadModelScopeFromExtra(extra)
        if (!loadedFromScope) {
          applyModelRestrictionFromRecord(credentials.model_mapping)
        }
        resetAccountPoolModeState(poolModeState, DEFAULT_POOL_MODE_RETRY_COUNT)
        resetAccountCustomErrorCodesState(customErrorCodesState)
        } else {
          editOpenRouterHTTPReferer.value = ''
          editOpenRouterTitle.value = ''
          const platformDefaultUrl = resolveAccountApiKeyDefaultBaseUrl(newAccount.platform, gatewayProtocol.value)
          editBaseUrl.value = platformDefaultUrl

          const loadedFromScope = loadModelScopeFromExtra(extra)

          // Backward-compatible: some legacy OpenAI OAuth accounts may store model mappings in credentials.
          if (!loadedFromScope && runtimePlatform === 'openai' && newAccount.credentials) {
            const oauthCredentials = newAccount.credentials as Record<string, unknown>
            applyModelRestrictionFromRecord(oauthCredentials.model_mapping)
          } else if (!loadedFromScope) {
            applyModelRestrictionFromRecord(undefined)
          }
          resetAccountPoolModeState(poolModeState, DEFAULT_POOL_MODE_RETRY_COUNT)
          resetAccountCustomErrorCodesState(customErrorCodesState)
        }
      const initialProbeModelIDs =
        modelProbeSnapshot.value?.models && modelProbeSnapshot.value.models.length > 0
          ? [...modelProbeSnapshot.value.models]
          : deriveConfiguredAccountModelIds(extra, credentials)
      if (initialProbeModelIDs.length > 0) {
        protocolGatewayProbeModels.value = createStaticProbeModels(initialProbeModelIDs)
      }
      editApiKey.value = ''
      isInitializingGatewayProtocol.value = false
    } else {
      isInitializingGatewayProtocol.value = false
      actualModelLocked.value = true
      resetMixedChannelRisk()
      resetTempUnschedRules()
      quotaControl.reset()
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
      protocolGatewayProbeModels.value = []
      manualModels.value = []
      modelProbeSnapshot.value = null
      resolvedUpstream.value = null
      batchArchiveEnabled.value = defaultGoogleBatchArchiveState.enabled
      batchArchiveAutoPrefetchEnabled.value = defaultGoogleBatchArchiveState.autoPrefetchEnabled
      batchArchiveRetentionDays.value = defaultGoogleBatchArchiveState.retentionDays
      batchArchiveBillingMode.value = defaultGoogleBatchArchiveState.billingMode
      batchArchiveDownloadPriceUSD.value = defaultGoogleBatchArchiveState.downloadPriceUSD
      allowVertexBatchOverflow.value = defaultGoogleBatchArchiveState.allowVertexBatchOverflow
      acceptAIStudioBatchOverflow.value = defaultGoogleBatchArchiveState.acceptAIStudioBatchOverflow
      modelMappings.value = []
      allowedModels.value = []
      modelRestrictionEnabled.value = true
      modelRestrictionMode.value = 'whitelist'
      openAIImageProtocolMode.value = 'native'
      openAIImageCompatAllowed.value = true
      autoRenewEnabled.value = false
      autoRenewPeriod.value = 'month'
      geminiOAuthType.value = 'code_assist'
      geminiVertexAuthMode.value = 'service_account'
      geminiVertexProjectId.value = ''
      geminiVertexLocation.value = ''
      geminiVertexServiceAccountJson.value = ''
      geminiVertexApiKey.value = ''
      geminiVertexAccessToken.value = ''
      geminiVertexExpiresAtInput.value = ''
      geminiVertexBaseUrl.value = ''
      baiduDocumentAIAccessToken.value = ''
      baiduDocumentAIAsyncBaseUrl.value = BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL
      baiduDocumentAIDirectApiUrlsText.value = ''
    }
  },
  { immediate: true }
)

watch(
  gatewayProtocol,
  (newProtocol: GatewayProtocol, oldProtocol: GatewayProtocol) => {
    if (!isProtocolGatewayAccount.value || newProtocol === oldProtocol || isInitializingGatewayProtocol.value) {
      return
    }
    gatewayAcceptedProtocols.value = normalizeGatewayAcceptedProtocols(
      newProtocol,
      gatewayAcceptedProtocols.value
    )
    gatewayClientProfiles.value = []
    gatewayClientRoutes.value = []
    gatewayTestProvider.value = ''
    gatewayTestModelId.value = ''
    gatewayOpenAIRequestFormat.value = DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT
    gatewayOpenAIImageProtocolMode.value = DEFAULT_GATEWAY_OPENAI_IMAGE_PROTOCOL_MODE
    gatewayBatchEnabled.value = false
    protocolGatewayProbeModels.value = []
    allowedModels.value = []
    modelMappings.value = []
    editBaseUrl.value = resolveAccountApiKeyDefaultBaseUrl(
      props.account?.platform || 'protocol_gateway',
      newProtocol
    )
  }
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

}
