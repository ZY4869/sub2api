export function createEditAccountSubmit(ctx: any) {
  const {
    GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS,
    acceptAIStudioBatchOverflow,
    allowVertexBatchOverflow,
    allowedModels,
    anthropicPassthroughEnabled,
    antigravityModelMappings,
    appStore,
    applyAccountCustomErrorCodesStateToCredentials,
    applyAccountPoolModeStateToCredentials,
    applyDeepSeekModelConcurrencyLimitsExtra,
    applyGoogleBatchArchiveExtra,
    applyInterceptWarmup,
    applyProtocolGatewayClaudeClientMimicExtra,
    applyProtocolGatewayGeminiBatchExtra,
    applyProtocolGatewayOpenAIImageProtocolModeExtra,
    applyProtocolGatewayOpenAIRequestFormatExtra,
    applyTempUnschedConfig,
    autoPauseOnExpired,
    autoRenewEnabled,
    autoRenewPeriod,
    batchArchiveAutoPrefetchEnabled,
    batchArchiveBillingMode,
    batchArchiveDownloadPriceUSD,
    batchArchiveEnabled,
    batchArchiveRetentionDays,
    buildAccountModelScopeExtra,
    buildBaiduDocumentAICredentialsForUpdate,
    buildModelMappingObject,
    buildProbeExtra,
    buildScopedModelMapping,
    claudeCodeMimicEnabled,
    claudeSessionIDMaskingEnabled,
    claudeTLSFingerprintEnabled,
    codexCLIOnlyEnabled,
    currentAccountCredentials,
    customErrorCodesState,
    deepSeekModelConcurrencyLimits,
    defaultBaseUrl,
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
    effectivePlatform,
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
    geminiVertexAccessToken,
    geminiVertexApiKey,
    geminiVertexAuthMode,
    geminiVertexBaseUrl,
    geminiVertexExpiresAtInput,
    geminiVertexLocation,
    geminiVertexProjectId,
    geminiVertexServiceAccountJson,
    interceptWarmupRequests,
    isBaiduDocumentAIAccount,
    isGeminiVertexAccount,
    isOpenAIWSModeEnabled,
    isProtocolGatewayAccount,
    mixedScheduling,
    modelMappings,
    modelRestrictionEnabled,
    modelRestrictionMode,
    normalizeGeminiAIStudioTier,
    openAIImageCompatAllowed,
    openAIImageProtocolMode,
    openaiAPIKeyResponsesWebSocketV2Mode,
    openaiOAuthResponsesWebSocketV2Mode,
    openaiPassthroughEnabled,
    parseDateTimeLocal,
    poolModeState,
    quotaControl,
    resolveGoogleBatchArchiveTargetKind,
    resolveVertexAuthBaseUrl,
    resolveVertexBaseUrl,
    shouldPersistGeminiTierId,
    submitUpdateAccount,
    t,
    props
  } = ctx

return async () => {
  if (!props.account) return
  const accountID = props.account.id
  const runtimePlatform = effectivePlatform.value

  if (form.status !== 'active' && form.status !== 'inactive' && form.status !== 'error') {
    appStore.showError(t('admin.accounts.pleaseSelectStatus'))
    return
  }

  const updatePayload: Record<string, unknown> = { ...form }
  try {
    if (updatePayload.proxy_id === null) {
      updatePayload.proxy_id = 0
    }
    if (form.expires_at === null) {
      updatePayload.expires_at = 0
    }
    const lf = form.load_factor
    if (lf == null || Number.isNaN(lf) || lf <= 0) {
      updatePayload.load_factor = 0
    }
    updatePayload.auto_pause_on_expired = autoPauseOnExpired.value
    updatePayload.auto_renew_enabled = autoRenewEnabled.value
    updatePayload.auto_renew_period = autoRenewPeriod.value

    if (isGeminiVertexAccount.value) {
      const currentCredentials = (props.account.credentials as Record<string, unknown>) || {}
      const newCredentials: Record<string, unknown> = { ...currentCredentials }
      const modelMapping = buildScopedModelMapping('mapping', [], modelMappings.value)
      updatePayload.type = geminiVertexAuthMode.value === 'express_api_key' ? 'apikey' : 'oauth'

      if (geminiVertexAuthMode.value === 'express_api_key') {
        const nextAPIKey = geminiVertexApiKey.value.trim() || String(currentCredentials.api_key || '').trim()
        if (!nextAPIKey) {
          appStore.showError(t('admin.accounts.gemini.vertex.expressApiKeyRequired'))
          return
        }

        newCredentials.gemini_api_variant = GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS
        newCredentials.api_key = nextAPIKey
        newCredentials.base_url = geminiVertexBaseUrl.value.trim() || resolveVertexAuthBaseUrl('express_api_key', '')
        delete newCredentials.oauth_type
        delete newCredentials.vertex_project_id
        delete newCredentials.vertex_location
        delete newCredentials.vertex_service_account_json
        delete newCredentials.access_token
        delete newCredentials.expires_at
        delete newCredentials.project_id
        delete newCredentials.refresh_token
        delete newCredentials.tier_id
      } else {
        if (!geminiVertexProjectId.value.trim()) {
          appStore.showError(t('admin.accounts.gemini.vertex.projectIdRequired'))
          return
        }
        if (!geminiVertexLocation.value.trim()) {
          appStore.showError(t('admin.accounts.gemini.vertex.locationRequired'))
          return
        }

        newCredentials.oauth_type = 'vertex_ai'
        newCredentials.vertex_project_id = geminiVertexProjectId.value.trim()
        newCredentials.vertex_location = geminiVertexLocation.value.trim()

        const serviceAccountJson = geminiVertexServiceAccountJson.value.trim()
        if (serviceAccountJson) {
          newCredentials.vertex_service_account_json = serviceAccountJson
          delete newCredentials.access_token
          delete newCredentials.expires_at
        } else {
          const trimmedToken = geminiVertexAccessToken.value.trim()
          if (trimmedToken) {
            newCredentials.access_token = trimmedToken
          } else if (!currentCredentials.access_token) {
            appStore.showError(t('admin.accounts.gemini.vertex.serviceAccountJsonRequired'))
            return
          }
          const expiresAt = parseDateTimeLocal(geminiVertexExpiresAtInput.value)
          if (expiresAt != null) {
            newCredentials.expires_at = String(expiresAt)
          } else {
            delete newCredentials.expires_at
          }
          delete newCredentials.vertex_service_account_json
        }

        newCredentials.base_url = geminiVertexBaseUrl.value.trim() || resolveVertexBaseUrl(geminiVertexLocation.value)
        delete newCredentials.gemini_api_variant
        delete newCredentials.api_key
        delete newCredentials.project_id
        delete newCredentials.refresh_token
        delete newCredentials.tier_id
      }

      if (modelMapping) {
        newCredentials.model_mapping = modelMapping
      } else {
        delete newCredentials.model_mapping
      }

      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    } else if (isBaiduDocumentAIAccount.value && props.account.type === 'apikey') {
      const newCredentials = buildBaiduDocumentAICredentialsForUpdate()
      if (!newCredentials) {
        return
      }
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }
      updatePayload.credentials = newCredentials
    } else if (props.account.type === 'apikey') {
      const currentCredentials = (props.account.credentials as Record<string, unknown>) || {}
      const newBaseUrl = editBaseUrl.value.trim() || defaultBaseUrl.value
      const shouldApplyModelMapping = !(runtimePlatform === 'openai' && openaiPassthroughEnabled.value)

      const newCredentials: Record<string, unknown> = {
        ...currentCredentials,
        base_url: newBaseUrl
      }
      if (shouldPersistGeminiTierId.value) {
        newCredentials.tier_id = normalizeGeminiAIStudioTier(geminiTierAIStudio.value)
      } else {
        delete newCredentials.tier_id
      }
      if (props.account.platform === 'openrouter') {
        const httpReferer = editOpenRouterHTTPReferer.value.trim()
        const openrouterTitle = editOpenRouterTitle.value.trim()
        if (httpReferer) {
          newCredentials.http_referer = httpReferer
        } else {
          delete newCredentials.http_referer
        }
        if (openrouterTitle) {
          newCredentials.openrouter_title = openrouterTitle
        } else {
          delete newCredentials.openrouter_title
        }
      } else {
        delete newCredentials.http_referer
        delete newCredentials.openrouter_title
      }

      if (editApiKey.value.trim()) {
        newCredentials.api_key = editApiKey.value.trim()
      } else if (currentCredentials.api_key) {
        newCredentials.api_key = currentCredentials.api_key
      } else {
        appStore.showError(t('admin.accounts.apiKeyIsRequired'))
        return
      }

      if (shouldApplyModelMapping && modelRestrictionEnabled.value) {
        const modelMapping = buildScopedModelMapping()
        if (modelMapping) {
          newCredentials.model_mapping = modelMapping
        } else {
          delete newCredentials.model_mapping
        }
      } else {
        delete newCredentials.model_mapping
      }

      applyAccountPoolModeStateToCredentials(newCredentials, poolModeState)
      applyAccountCustomErrorCodesStateToCredentials(newCredentials, customErrorCodesState)

      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    } else if (props.account.type === 'upstream') {
      const currentCredentials = (props.account.credentials as Record<string, unknown>) || {}
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      newCredentials.base_url = editBaseUrl.value.trim()

      if (editApiKey.value.trim()) {
        newCredentials.api_key = editApiKey.value.trim()
      }

      const modelMapping = buildScopedModelMapping()
      if (modelMapping) {
        newCredentials.model_mapping = modelMapping
      } else {
        delete newCredentials.model_mapping
      }

      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    } else {
      const currentCredentials = (props.account.credentials as Record<string, unknown>) || {}
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      const modelMapping = buildScopedModelMapping()
      if (modelMapping) {
        newCredentials.model_mapping = modelMapping
      } else {
        delete newCredentials.model_mapping
      }

      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    }

    if (runtimePlatform === 'openai' && props.account.type === 'oauth') {
      const currentCredentials = (updatePayload.credentials as Record<string, unknown>) ||
        ((props.account.credentials as Record<string, unknown>) || {})
      const newCredentials: Record<string, unknown> = { ...currentCredentials }
      const shouldApplyModelMapping = !openaiPassthroughEnabled.value

      if (shouldApplyModelMapping && modelRestrictionEnabled.value) {
        const modelMapping = buildScopedModelMapping()
        if (modelMapping) {
          newCredentials.model_mapping = modelMapping
        } else {
          delete newCredentials.model_mapping
        }
      } else {
        delete newCredentials.model_mapping
      }

      updatePayload.credentials = newCredentials
    }

    if (runtimePlatform === 'grok' && props.account.type === 'sso') {
      const currentCredentials = (updatePayload.credentials as Record<string, unknown>) ||
        ((props.account.credentials as Record<string, unknown>) || {})
      const newCredentials: Record<string, unknown> = { ...currentCredentials }
      if (editGrokSSOToken.value.trim()) {
        newCredentials.sso_token = editGrokSSOToken.value.trim()
      }
      const modelMapping = buildScopedModelMapping()
      if (modelMapping) {
        newCredentials.model_mapping = modelMapping
      } else {
        delete newCredentials.model_mapping
      }
      updatePayload.credentials = newCredentials
    }

    if (runtimePlatform === 'antigravity') {
      const currentCredentials = (updatePayload.credentials as Record<string, unknown>) ||
        ((props.account.credentials as Record<string, unknown>) || {})
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      delete newCredentials.model_whitelist
      delete newCredentials.model_mapping

      const antigravityModelMapping = buildModelMappingObject(
        'mapping',
        [],
        antigravityModelMappings.value
      )
      if (antigravityModelMapping) {
        newCredentials.model_mapping = antigravityModelMapping
      }

      updatePayload.credentials = newCredentials
    }

    if (runtimePlatform === 'antigravity') {
      const currentExtra = (props.account.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      if (mixedScheduling.value) {
        newExtra.mixed_scheduling = true
      } else {
        delete newExtra.mixed_scheduling
      }
      updatePayload.extra = newExtra
    }

    if (props.account.platform === 'grok') {
      const currentExtra = (updatePayload.extra as Record<string, unknown>) ||
        (props.account.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      if (props.account.type === 'sso') {
        newExtra.grok_tier = editGrokTier.value
      } else {
        delete newExtra.grok_tier
        delete newExtra.grok_capabilities
      }
      updatePayload.extra = newExtra
    }

    if (runtimePlatform === 'anthropic' && (props.account.type === 'oauth' || props.account.type === 'setup-token')) {
      const currentExtra = (props.account.extra as Record<string, unknown>) || {}
      updatePayload.extra = quotaControl.buildExtra(currentExtra)
    }

    if (runtimePlatform === 'anthropic' && props.account.type === 'apikey') {
      const currentExtra = (props.account.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      if (anthropicPassthroughEnabled.value) {
        newExtra.anthropic_passthrough = true
      } else {
        delete newExtra.anthropic_passthrough
      }
      updatePayload.extra = newExtra
    }

    if (runtimePlatform === 'openai' && (props.account.type === 'oauth' || props.account.type === 'apikey')) {
      const currentExtra = (props.account.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      const hadCodexCLIOnlyEnabled = currentExtra.codex_cli_only === true
      if (props.account.type === 'oauth') {
        newExtra.openai_oauth_responses_websockets_v2_mode = openaiOAuthResponsesWebSocketV2Mode.value
        newExtra.openai_oauth_responses_websockets_v2_enabled = isOpenAIWSModeEnabled(openaiOAuthResponsesWebSocketV2Mode.value)
      } else if (props.account.type === 'apikey') {
        newExtra.openai_apikey_responses_websockets_v2_mode = openaiAPIKeyResponsesWebSocketV2Mode.value
        newExtra.openai_apikey_responses_websockets_v2_enabled = isOpenAIWSModeEnabled(openaiAPIKeyResponsesWebSocketV2Mode.value)
      }
      delete newExtra.responses_websockets_v2_enabled
      delete newExtra.openai_ws_enabled
      if (openaiPassthroughEnabled.value) {
        newExtra.openai_passthrough = true
      } else {
        delete newExtra.openai_passthrough
        delete newExtra.openai_oauth_passthrough
      }
      newExtra.image_protocol_mode = openAIImageCompatAllowed.value
        ? openAIImageProtocolMode.value
        : 'native'
      if (props.account.type === 'oauth') {
        newExtra.image_compat_allowed = openAIImageCompatAllowed.value
      } else {
        delete newExtra.image_compat_allowed
      }

      if (props.account.type === 'oauth') {
        if (codexCLIOnlyEnabled.value) {
          newExtra.codex_cli_only = true
        } else if (hadCodexCLIOnlyEnabled) {
          newExtra.codex_cli_only = false
        } else {
          delete newExtra.codex_cli_only
        }
      }

      updatePayload.extra = newExtra
    }

    if (isProtocolGatewayAccount.value) {
      const currentExtra = (updatePayload.extra as Record<string, unknown>) ||
        (props.account.extra as Record<string, unknown>) || {}
      updatePayload.gateway_protocol = gatewayProtocol.value
      updatePayload.extra = applyProtocolGatewayGeminiBatchExtra(
        applyProtocolGatewayOpenAIImageProtocolModeExtra(
          applyProtocolGatewayOpenAIRequestFormatExtra(
            applyProtocolGatewayClaudeClientMimicExtra({
              ...currentExtra,
              gateway_protocol: gatewayProtocol.value,
              gateway_accepted_protocols: [...gatewayAcceptedProtocols.value],
              gateway_client_profiles: [...gatewayClientProfiles.value],
              gateway_client_routes: gatewayClientRoutes.value.map((route: Record<string, unknown>) => ({ ...route })),
              gateway_test_provider: gatewayTestProvider.value || undefined,
              gateway_test_model_id: gatewayTestModelId.value || undefined
            }, {
              platform: props.account.platform,
              type: props.account.type,
              gatewayProtocol: gatewayProtocol.value,
              acceptedProtocols: gatewayAcceptedProtocols.value,
              claudeCodeMimicEnabled: claudeCodeMimicEnabled.value,
              enableTLSFingerprint: claudeTLSFingerprintEnabled.value,
              sessionIDMaskingEnabled: claudeSessionIDMaskingEnabled.value
            }),
            {
              platform: props.account.platform,
              type: props.account.type,
              gatewayProtocol: gatewayProtocol.value,
              acceptedProtocols: gatewayAcceptedProtocols.value,
              gatewayOpenAIRequestFormat: gatewayOpenAIRequestFormat.value
            }
          ),
          {
            platform: props.account.platform,
            type: props.account.type,
            gatewayProtocol: gatewayProtocol.value,
            acceptedProtocols: gatewayAcceptedProtocols.value,
            gatewayOpenAIImageProtocolMode: gatewayOpenAIImageProtocolMode.value
          }
        ),
        {
          platform: props.account.platform,
          type: props.account.type,
          gatewayProtocol: gatewayProtocol.value,
          acceptedProtocols: gatewayAcceptedProtocols.value,
          gatewayBatchEnabled: gatewayBatchEnabled.value
        }
      )
    }

    if (props.account.type !== 'bedrock') {
      const normalizedExtra: Record<string, unknown> = {
        ...(((updatePayload.extra as Record<string, unknown>) ||
          (props.account.extra as Record<string, unknown>) ||
          {}) as Record<string, unknown>)
      }
      normalizedExtra.expiry_probe_extension_days = Math.max(1, expiryProbeExtensionDays.value)
      if (runtimePlatform !== 'openai') {
        delete normalizedExtra.openai_passthrough
        delete normalizedExtra.openai_oauth_passthrough
        delete normalizedExtra.image_protocol_mode
        delete normalizedExtra.image_compat_allowed
        delete normalizedExtra.codex_cli_only
        delete normalizedExtra.openai_oauth_responses_websockets_v2_mode
        delete normalizedExtra.openai_apikey_responses_websockets_v2_mode
        delete normalizedExtra.openai_oauth_responses_websockets_v2_enabled
        delete normalizedExtra.openai_apikey_responses_websockets_v2_enabled
        delete normalizedExtra.responses_websockets_v2_enabled
        delete normalizedExtra.openai_ws_enabled
      } else if (isProtocolGatewayAccount.value) {
        delete normalizedExtra.image_protocol_mode
        delete normalizedExtra.image_compat_allowed
      }
      if (runtimePlatform !== 'anthropic') {
        delete normalizedExtra.anthropic_passthrough
      }
      if (!isProtocolGatewayAccount.value) {
        delete normalizedExtra.gateway_openai_image_protocol_mode
      }
      updatePayload.extra =
        Object.keys(normalizedExtra).length > 0 ? normalizedExtra : undefined
    }

    {
      const currentExtra = (updatePayload.extra as Record<string, unknown>) ||
        (props.account.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      if (editQuotaLimit.value != null && editQuotaLimit.value > 0) {
        newExtra.quota_limit = editQuotaLimit.value
      } else {
        delete newExtra.quota_limit
      }
      if (editQuotaDailyLimit.value != null && editQuotaDailyLimit.value > 0) {
        newExtra.quota_daily_limit = editQuotaDailyLimit.value
      } else {
        delete newExtra.quota_daily_limit
      }
      if (editQuotaWeeklyLimit.value != null && editQuotaWeeklyLimit.value > 0) {
        newExtra.quota_weekly_limit = editQuotaWeeklyLimit.value
      } else {
        delete newExtra.quota_weekly_limit
      }

      if (editQuotaDailyResetMode.value != null) {
        newExtra.quota_daily_reset_mode = editQuotaDailyResetMode.value
      } else {
        delete newExtra.quota_daily_reset_mode
      }
      if (editQuotaDailyResetHour.value != null) {
        newExtra.quota_daily_reset_hour = editQuotaDailyResetHour.value
      } else {
        delete newExtra.quota_daily_reset_hour
      }
      if (editQuotaWeeklyResetMode.value != null) {
        newExtra.quota_weekly_reset_mode = editQuotaWeeklyResetMode.value
      } else {
        delete newExtra.quota_weekly_reset_mode
      }
      if (editQuotaWeeklyResetDay.value != null) {
        newExtra.quota_weekly_reset_day = editQuotaWeeklyResetDay.value
      } else {
        delete newExtra.quota_weekly_reset_day
      }
      if (editQuotaWeeklyResetHour.value != null) {
        newExtra.quota_weekly_reset_hour = editQuotaWeeklyResetHour.value
      } else {
        delete newExtra.quota_weekly_reset_hour
      }
      if (editQuotaResetTimezone.value != null) {
        newExtra.quota_reset_timezone = editQuotaResetTimezone.value
      } else {
        delete newExtra.quota_reset_timezone
      }
      updatePayload.extra = newExtra
    }

    updatePayload.extra = applyGoogleBatchArchiveExtra(
      updatePayload.extra as Record<string, unknown> | undefined,
      resolveGoogleBatchArchiveTargetKind(
        props.account?.platform,
        String(updatePayload.type || props.account?.type || ''),
        ((updatePayload.credentials as Record<string, unknown> | undefined) ||
          currentAccountCredentials.value)
      ),
      {
        enabled: batchArchiveEnabled.value,
        autoPrefetchEnabled: batchArchiveAutoPrefetchEnabled.value,
        retentionDays: batchArchiveRetentionDays.value,
        billingMode: batchArchiveBillingMode.value,
        downloadPriceUSD: batchArchiveDownloadPriceUSD.value,
        allowVertexBatchOverflow: allowVertexBatchOverflow.value,
        acceptAIStudioBatchOverflow: acceptAIStudioBatchOverflow.value
      }
    )

    updatePayload.extra = buildProbeExtra(
      buildAccountModelScopeExtra(
        ((updatePayload.extra as Record<string, unknown>) ||
          (props.account.extra as Record<string, unknown>) ||
          undefined),
        {
          platform: runtimePlatform,
          enabled: runtimePlatform === 'antigravity'
            ? true
            : modelRestrictionEnabled.value &&
              !(runtimePlatform === 'openai' && openaiPassthroughEnabled.value),
          mode: runtimePlatform === 'antigravity' ? 'mapping' : modelRestrictionMode.value,
          allowedModels: allowedModels.value,
          modelMappings: runtimePlatform === 'antigravity'
            ? antigravityModelMappings.value
            : modelMappings.value
        }
      )
    )

    if (props.account.platform === 'grok' && props.account.type !== 'sso') {
      const currentExtra = (updatePayload.extra as Record<string, unknown>) || {}
      const sanitizedExtra: Record<string, unknown> = { ...currentExtra }
      delete sanitizedExtra.grok_tier
      delete sanitizedExtra.grok_capabilities
      updatePayload.extra =
        Object.keys(sanitizedExtra).length > 0 ? sanitizedExtra : undefined
    }

    updatePayload.extra = applyDeepSeekModelConcurrencyLimitsExtra(
      updatePayload.extra as Record<string, unknown> | undefined,
      runtimePlatform,
      deepSeekModelConcurrencyLimits.value
    )

    const canContinue = await ensureMixedChannelConfirmed(async () => {
      await submitUpdateAccount(accountID, updatePayload)
    })
    if (!canContinue) {
      return
    }

    await submitUpdateAccount(accountID, updatePayload)
  } catch (error: any) {
    if (error?.reason === 'ACCOUNT_INVALID_BASE_URL') {
      appStore.showError(t('admin.accounts.invalidBaseUrl'))
      return
    }
    appStore.showError(error.message || t('admin.accounts.failedToUpdate'))
  }
}

}
