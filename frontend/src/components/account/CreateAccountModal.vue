<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.createAccount')"
    width="account-wide"
    @close="handleClose"
  >
    <div class="min-w-0 overflow-x-hidden">
      <!-- Step Indicator for OAuth accounts -->
      <div v-if="isOAuthFlow" class="mb-6 flex justify-center">
        <div class="flex w-full flex-col items-center gap-3 sm:w-auto sm:flex-row sm:gap-4">
          <div class="flex min-w-0 items-center justify-center">
            <div
              :class="[
                'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold',
                step >= 1 ? 'bg-primary-500 text-white' : 'bg-gray-200 text-gray-500 dark:bg-dark-600'
              ]"
            >
              1
            </div>
            <span class="ml-2 min-w-0 break-words text-sm font-medium text-gray-700 dark:text-gray-300">{{
              t('admin.accounts.oauth.authMethod')
            }}</span>
          </div>
          <div class="hidden h-0.5 w-8 bg-gray-300 dark:bg-dark-600 sm:block" />
          <div class="flex min-w-0 items-center justify-center">
            <div
              :class="[
                'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold',
                step >= 2 ? 'bg-primary-500 text-white' : 'bg-gray-200 text-gray-500 dark:bg-dark-600'
              ]"
            >
              2
            </div>
            <span class="ml-2 min-w-0 break-words text-sm font-medium text-gray-700 dark:text-gray-300">{{
              oauthStepTitle
            }}</span>
          </div>
          <template v-if="showOAuthFinalizeStep">
            <div class="hidden h-0.5 w-8 bg-gray-300 dark:bg-dark-600 sm:block" />
            <div class="flex min-w-0 items-center justify-center">
              <div
                :class="[
                  'flex h-8 w-8 items-center justify-center rounded-full text-sm font-semibold',
                  step >= 3 ? 'bg-primary-500 text-white' : 'bg-gray-200 text-gray-500 dark:bg-dark-600'
                ]"
              >
                3
              </div>
              <span class="ml-2 min-w-0 break-words text-sm font-medium text-gray-700 dark:text-gray-300">{{
                t('admin.accounts.probeFinalize.stepTitle')
              }}</span>
            </div>
          </template>
        </div>
      </div>

      <!-- Step 1: Basic Info -->
      <form
        v-if="step === 1 || step === 3"
        id="create-account-form"
        @submit.prevent="handleSubmit"
        class="min-w-0 space-y-5"
      >
        <div
          v-if="step === 3 && showOAuthFinalizeStep"
          class="rounded-2xl border border-emerald-200 bg-emerald-50/80 px-4 py-3 text-sm text-emerald-800 dark:border-emerald-900/50 dark:bg-emerald-950/20 dark:text-emerald-200"
        >
          {{ t('admin.accounts.probeFinalize.readyHint') }}
        </div>

        <div>
          <label class="input-label">{{ t('admin.accounts.accountName') }}</label>
          <input
            v-model="form.name"
            type="text"
            required
            class="input"
            :placeholder="t('admin.accounts.enterAccountName')"
            data-tour="account-form-name"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.notes') }}</label>
          <textarea
            v-model="form.notes"
            rows="3"
            class="input"
            :placeholder="t('admin.accounts.notesPlaceholder')"
          ></textarea>
          <p class="input-hint">{{ t('admin.accounts.notesHint') }}</p>
        </div>

        <AccountCreatePlatformSelector v-model:platform="form.platform" />

        <div
          v-if="isBaiduDocumentAISelected"
          class="rounded-2xl border border-rose-200 bg-rose-50/70 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/20 dark:text-rose-200"
          data-testid="baidu-document-ai-selected-hint"
        >
          <div class="font-medium">{{ t('admin.accounts.baiduDocumentAI.selectedHintTitle') }}</div>
          <div class="mt-1 text-xs leading-5 opacity-90">
            {{ t('admin.accounts.baiduDocumentAI.selectedHintBody') }}
          </div>
        </div>

        <AccountCreatePlatformTypeEditor
          v-model:platform="form.platform"
          v-model:account-category="accountCategory"
          v-model:gateway-protocol="gatewayProtocol"
          v-model:add-method="addMethod"
          v-model:antigravity-account-type="antigravityAccountType"
          v-model:gemini-o-auth-type="geminiOAuthType"
          v-model:show-advanced="showAdvancedOAuth"
          v-model:gemini-tier-google-one="geminiTierGoogleOne"
          v-model:gemini-tier-gcp="geminiTierGcp"
          v-model:gemini-tier-ai-studio="geminiTierAIStudio"
          v-model:upstream-base-url="upstreamBaseUrl"
          v-model:upstream-api-key="upstreamApiKey"
          :ai-studio-o-auth-enabled="geminiAIStudioOAuthEnabled"
          :api-key-help-link="geminiHelpLinks.apiKey"
          :gcp-project-help-link="geminiHelpLinks.gcpProject"
          @open-gemini-help="showGeminiHelpDialog = true"
        />

        <AccountTierSelector
          v-if="accountCategory === 'oauth-based' && (form.platform === 'openai' || form.platform === 'anthropic')"
          v-model:tier="accountTier"
          :platform="form.platform"
          @apply-capacity="applyAccountTierCapacity"
        />

        <AccountBaiduDocumentAICredentialsEditor
          v-if="isBaiduDocumentAISelected"
          v-model:access-token="baiduDocumentAIAccessToken"
          v-model:async-base-url="baiduDocumentAIAsyncBaseUrl"
          v-model:direct-api-urls-text="baiduDocumentAIDirectApiUrlsText"
          mode="create"
        />

        <AccountGeminiVertexCredentialsEditor
          v-if="form.platform === 'gemini' && accountCategory === 'vertex_ai'"
          v-model:auth-mode="geminiVertexAuthMode"
          v-model:project-id="geminiVertexProjectId"
          v-model:location="geminiVertexLocation"
          v-model:service-account-json="geminiVertexServiceAccountJson"
          v-model:api-key="geminiVertexApiKey"
          v-model:legacy-access-token="geminiVertexAccessToken"
          v-model:legacy-expires-at-input="geminiVertexExpiresAtInput"
          v-model:base-url="geminiVertexBaseUrl"
          mode="create"
        />

        <AccountApiKeyModelProbeEditor
          v-if="form.platform === 'gemini' && accountCategory === 'vertex_ai'"
          v-model:allowed-models="allowedModels"
          v-model:model-mappings="modelMappings"
          v-model:probed-models="protocolGatewayProbeModels"
          v-model:manual-models="manualModels"
          v-model:probe-snapshot="modelProbeSnapshot"
          v-model:resolved-upstream="resolvedUpstream"
          platform="gemini"
          :account-type="geminiVertexAuthMode === 'express_api_key' ? 'apikey' : 'oauth'"
          :credentials="vertexProbeCredentials"
          :extra="probeExtraForEditor"
          :probe-ready="isVertexProbeReady"
          :proxy-id="form.proxy_id"
        />

        <AccountApiKeyModelProbeEditor
          v-if="form.type === 'upstream'"
          v-model:allowed-models="allowedModels"
          v-model:model-mappings="modelMappings"
          v-model:probed-models="protocolGatewayProbeModels"
          v-model:manual-models="manualModels"
          v-model:probe-snapshot="modelProbeSnapshot"
          v-model:resolved-upstream="resolvedUpstream"
          :platform="form.platform"
          account-type="upstream"
          :credentials="upstreamProbeCredentials"
          :extra="probeExtraForEditor"
          :probe-ready="isUpstreamProbeReady"
          :proxy-id="form.proxy_id"
        />

        <div
          v-if="form.platform === 'grok'"
          class="space-y-4 rounded-lg border border-slate-200 bg-slate-50/60 p-4 dark:border-slate-700 dark:bg-slate-900/30"
        >
          <AccountGrokOAuthPanel
            v-if="form.type === 'oauth'"
            ref="grokOAuthRef"
            :proxy-id="form.proxy_id"
            :submit-label="t('common.create')"
            :submitting="submitting"
            @submit="handleCreateGrokOAuthAccount"
          />

          <div v-if="form.type === 'sso'">
            <label class="input-label">{{ t('admin.accounts.grokToken') }}</label>
            <textarea
              v-model="grokSSOToken"
              rows="4"
              class="input"
              :placeholder="t('admin.accounts.grokTokenPlaceholder')"
            />
            <p class="input-hint">{{ t('admin.accounts.grokTokenHint') }}</p>
          </div>

          <div v-if="form.type === 'sso'">
            <label class="input-label">{{ t('admin.accounts.grokTier') }}</label>
            <select v-model="grokTier" class="input">
              <option value="basic">{{ t('admin.accounts.grokTierBasic') }}</option>
              <option value="super">{{ t('admin.accounts.grokTierSuper') }}</option>
              <option value="heavy">{{ t('admin.accounts.grokTierHeavy') }}</option>
            </select>
            <p class="input-hint">{{ t('admin.accounts.grokTierHint') }}</p>
          </div>

          <AccountGrokImportPanel
            :show="show"
            @imported="handleGrokImportCompleted"
          />
        </div>

      <!-- Antigravity model restriction (applies to OAuth + Upstream) -->
      <AccountAntigravityModelMappingEditor
        v-if="form.platform === 'antigravity'"
        :model-mappings="antigravityModelMappings"
        :preset-mappings="antigravityPresetMappings"
        :get-mapping-key="getAntigravityModelMappingKey"
        @add-mapping="addAntigravityModelMapping"
        @remove-mapping="removeAntigravityModelMapping"
        @add-preset="addAntigravityPresetMapping($event.from, $event.to)"
      />

      <!-- API Key input (only for apikey type, excluding Antigravity which has its own fields) -->
      <div v-if="showCommonApiKeySection" class="space-y-4">
        <AccountApiKeyBasicSettingsEditor
          v-model:base-url="apiKeyBaseUrl"
          v-model:api-key="apiKeyValue"
          v-model:model-scope-enabled="modelRestrictionEnabled"
          v-model:actual-model-locked="actualModelLocked"
          v-model:model-scope-mode="modelRestrictionMode"
          v-model:allowed-models="allowedModels"
          v-model:gemini-tier-ai-studio="geminiTierAIStudio"
          :platform="form.platform"
          :gateway-protocol="gatewayProtocol"
          :effective-platform="effectivePlatform"
          mode="create"
          :model-scope-disabled="isOpenAIModelRestrictionDisabled"
          :skip-model-scope-editor="!showApiKeyModelScopeEditor"
          :model-mappings="modelMappings"
          :preset-mappings="presetMappings"
          :get-mapping-key="getModelMappingKey"
          :show-gemini-tier="shouldPersistGeminiTierId"
          :show-actual-model-lock="true"
          @update:modelMappings="modelMappings = $event"
          @add-mapping="addModelMapping"
          @remove-mapping="removeModelMapping"
          @add-preset="addPresetMapping($event.from, $event.to)"
        />

        <AccountProtocolGatewayClaudeMimicEditor
          v-if="showProtocolGatewayClaudeMimicEditor"
          v-model:enabled="claudeCodeMimicEnabled"
          v-model:tls-fingerprint-enabled="claudeTLSFingerprintEnabled"
          v-model:session-id-masking-enabled="claudeSessionIDMaskingEnabled"
        />

        <AccountProtocolGatewayOpenAIRequestFormatEditor
          v-if="showProtocolGatewayOpenAIRequestFormatEditor"
          v-model:value="gatewayOpenAIRequestFormat"
          v-model:image-protocol-mode="gatewayOpenAIImageProtocolMode"
        />

        <AccountDeepSeekConcurrencyLimitsEditor
          v-if="showDeepSeekConcurrencyEditor"
          v-model:limits="deepSeekModelConcurrencyLimits"
        />

        <AccountOpenRouterSettingsEditor
          v-if="form.platform === 'openrouter'"
          v-model:http-referer="openRouterHTTPReferer"
          v-model:openrouter-title="openRouterTitle"
        />

        <AccountProtocolGatewayBatchEditor
          v-if="showProtocolGatewayBatchEditor"
          v-model:enabled="gatewayBatchEnabled"
          :request-formats="protocolGatewayBatchRequestFormats"
        />

        <AccountProtocolGatewayModelProbeEditor
          v-if="form.platform === 'protocol_gateway'"
          v-model:allowed-models="allowedModels"
          v-model:model-mappings="modelMappings"
          v-model:probed-models="protocolGatewayProbeModels"
          v-model:manual-models="manualModels"
          v-model:probe-snapshot="modelProbeSnapshot"
          v-model:resolved-upstream="resolvedUpstream"
          v-model:accepted-protocols="gatewayAcceptedProtocols"
          v-model:client-profiles="gatewayClientProfiles"
          v-model:client-routes="gatewayClientRoutes"
          v-model:gateway-test-provider="gatewayTestProvider"
          v-model:gateway-test-model-id="gatewayTestModelId"
          :gateway-protocol="gatewayProtocol"
          :base-url="apiKeyBaseUrl"
          :api-key="apiKeyValue"
          :proxy-id="form.proxy_id"
        />

        <AccountApiKeyModelProbeEditor
          v-if="form.platform !== 'protocol_gateway'"
          v-model:allowed-models="allowedModels"
          v-model:model-mappings="modelMappings"
          v-model:probed-models="protocolGatewayProbeModels"
          v-model:manual-models="manualModels"
          v-model:probe-snapshot="modelProbeSnapshot"
          v-model:resolved-upstream="resolvedUpstream"
          :platform="form.platform"
          account-type="apikey"
          :credentials="apiKeyProbeCredentials"
          :extra="probeExtraForEditor"
          :probe-ready="isApiKeyProbeReady"
          :proxy-id="form.proxy_id"
        />

        <AccountPoolModeEditor
          :state="poolModeState"
          :default-retry-count="DEFAULT_POOL_MODE_RETRY_COUNT"
          :max-retry-count="MAX_POOL_MODE_RETRY_COUNT"
        />

        <AccountCustomErrorCodesEditor
          :state="customErrorCodesState"
          :error-code-options="commonErrorCodeOptions"
          :show-error="showFormError"
          :show-info="showFormInfo"
        />

      </div>

      <AccountApiKeyModelProbeEditor
        v-if="showOAuthFinalizeProbeEditor"
        v-model:allowed-models="allowedModels"
        v-model:model-mappings="modelMappings"
        v-model:probed-models="protocolGatewayProbeModels"
        v-model:manual-models="manualModels"
        v-model:probe-snapshot="modelProbeSnapshot"
        v-model:resolved-upstream="resolvedUpstream"
        :platform="form.platform"
        account-type="oauth"
        :credentials="oauthDraftCredentials"
        :extra="buildProbeExtra(oauthDraftExtra)"
        :probe-ready="oauthDraftProbeReady"
        :proxy-id="form.proxy_id"
      />

      <div v-if="showQuotaLimitSection" class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4">
        <div class="mb-3">
          <h3 class="input-label mb-0 text-base font-semibold">{{ t('admin.accounts.quotaLimit') }}</h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.quotaLimitHint') }}
          </p>
        </div>
        <QuotaLimitCard
          :totalLimit="editQuotaLimit"
          :dailyLimit="editQuotaDailyLimit"
          :weeklyLimit="editQuotaWeeklyLimit"
          :dailyResetMode="editQuotaDailyResetMode"
          :dailyResetHour="editQuotaDailyResetHour"
          :weeklyResetMode="editQuotaWeeklyResetMode"
          :weeklyResetDay="editQuotaWeeklyResetDay"
          :weeklyResetHour="editQuotaWeeklyResetHour"
          :resetTimezone="editQuotaResetTimezone"
          @update:totalLimit="editQuotaLimit = $event"
          @update:dailyLimit="editQuotaDailyLimit = $event"
          @update:weeklyLimit="editQuotaWeeklyLimit = $event"
          @update:dailyResetMode="editQuotaDailyResetMode = $event"
          @update:dailyResetHour="editQuotaDailyResetHour = $event"
          @update:weeklyResetMode="editQuotaWeeklyResetMode = $event"
          @update:weeklyResetDay="editQuotaWeeklyResetDay = $event"
          @update:weeklyResetHour="editQuotaWeeklyResetHour = $event"
          @update:resetTimezone="editQuotaResetTimezone = $event"
        />
      </div>

      <AccountGoogleBatchArchiveEditor
        v-if="showGeminiAIStudioBatchArchiveEditor"
        mode="ai_studio"
        :archive-enabled="batchArchiveEnabled"
        :auto-prefetch-enabled="batchArchiveAutoPrefetchEnabled"
        :retention-days="batchArchiveRetentionDays"
        :billing-mode="batchArchiveBillingMode"
        :download-price-usd="batchArchiveDownloadPriceUSD"
        :allow-vertex-batch-overflow="allowVertexBatchOverflow"
        @update:archive-enabled="batchArchiveEnabled = $event"
        @update:auto-prefetch-enabled="batchArchiveAutoPrefetchEnabled = $event"
        @update:retention-days="batchArchiveRetentionDays = $event"
        @update:billing-mode="batchArchiveBillingMode = $event"
        @update:download-price-usd="batchArchiveDownloadPriceUSD = $event"
        @update:allow-vertex-batch-overflow="allowVertexBatchOverflow = $event"
      />

      <AccountGoogleBatchArchiveEditor
        v-if="showGeminiVertexBatchArchiveEditor"
        mode="vertex"
        :archive-enabled="batchArchiveEnabled"
        :retention-days="batchArchiveRetentionDays"
        :billing-mode="batchArchiveBillingMode"
        :download-price-usd="batchArchiveDownloadPriceUSD"
        :accept-ai-studio-batch-overflow="acceptAIStudioBatchOverflow"
        @update:archive-enabled="batchArchiveEnabled = $event"
        @update:retention-days="batchArchiveRetentionDays = $event"
        @update:billing-mode="batchArchiveBillingMode = $event"
        @update:download-price-usd="batchArchiveDownloadPriceUSD = $event"
        @update:accept-ai-studio-batch-overflow="acceptAIStudioBatchOverflow = $event"
      />

      <AccountModelScopeEditor
        v-if="showStandaloneModelScopeEditor"
        v-model:enabled="modelRestrictionEnabled"
        v-model:actual-model-locked="actualModelLocked"
        :disabled="isOpenAIModelRestrictionDisabled"
        :platform="effectivePlatform"
        :mode="modelRestrictionMode"
        :allowed-models="allowedModels"
        :model-mappings="modelMappings"
        :preset-mappings="presetMappings"
        :get-mapping-key="getModelMappingKey"
        :show-actual-model-lock="true"
        @update:mode="modelRestrictionMode = $event"
        @update:allowedModels="allowedModels = $event"
        @update:modelMappings="modelMappings = $event"
        @add-mapping="addModelMapping"
        @remove-mapping="removeModelMapping"
        @add-preset="addPresetMapping($event.from, $event.to)"
      />

      <AccountTempUnschedRulesEditor
        :enabled="tempUnschedEnabled"
        :rules="tempUnschedRules"
        :presets="tempUnschedPresets"
        :get-rule-key="getTempUnschedRuleKey"
        @update:enabled="tempUnschedEnabled = $event"
        @add-rule="addTempUnschedRule"
        @remove-rule="removeTempUnschedRule"
        @move-rule="moveTempUnschedRule($event.index, $event.direction)"
      />

      <!-- Intercept Warmup Requests (Anthropic/Antigravity) -->
      <div
        v-if="effectivePlatform === 'anthropic' || form.platform === 'antigravity'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{
              t('admin.accounts.interceptWarmupRequests')
            }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.interceptWarmupRequestsDesc') }}
            </p>
          </div>
          <button
            type="button"
            @click="interceptWarmupRequests = !interceptWarmupRequests"
            :class="[
              'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
              interceptWarmupRequests ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
            ]"
          >
            <span
              :class="[
                'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                interceptWarmupRequests ? 'translate-x-5' : 'translate-x-0'
              ]"
            />
          </button>
        </div>
      </div>

      <AccountQuotaControlEditor
        v-if="effectivePlatform === 'anthropic' && accountCategory === 'oauth-based'"
        v-model:state="quotaControlState"
        :umq-mode-options="umqModeOptions"
      />

      <AccountRuntimeSettingsEditor
        v-model:proxy-id="form.proxy_id"
        v-model:concurrency="form.concurrency"
        v-model:load-factor="form.load_factor"
        v-model:priority="form.priority"
        v-model:rate-multiplier="form.rate_multiplier"
        v-model:expires-at-input="expiresAtInput"
        v-model:expiry-probe-extension-days="expiryProbeExtensionDays"
        v-model:auto-renew-enabled="autoRenewEnabled"
        v-model:auto-renew-period="autoRenewPeriod"
        :proxies="proxies"
      />

      <AccountGatewaySettingsEditor
        :show-open-ai-passthrough="effectivePlatform === 'openai'"
        :open-ai-passthrough-enabled="openaiPassthroughEnabled"
        :show-open-ai-image-protocol-mode="form.platform === 'openai' && (accountCategory === 'oauth-based' || accountCategory === 'apikey')"
        :open-ai-image-protocol-mode="openAIImageProtocolMode"
        :open-ai-image-protocol-compat-allowed="openAIImageCompatAllowed"
        :show-open-ai-image-protocol-compat-toggle="accountCategory === 'oauth-based'"
        :show-open-ai-ws-mode="effectivePlatform === 'openai' && (accountCategory === 'oauth-based' || accountCategory === 'apikey')"
        :open-ai-ws-mode="openaiResponsesWebSocketV2Mode"
        :open-ai-ws-mode-options="openAIWSModeOptions"
        :open-ai-ws-mode-concurrency-hint-key="openAIWSModeConcurrencyHintKey"
        :show-anthropic-passthrough="effectivePlatform === 'anthropic' && accountCategory === 'apikey'"
        :anthropic-passthrough-enabled="anthropicPassthroughEnabled"
        :show-codex-cli-only="effectivePlatform === 'openai' && accountCategory === 'oauth-based'"
        :codex-cli-only-enabled="codexCLIOnlyEnabled"
        @update:open-ai-passthrough-enabled="openaiPassthroughEnabled = $event"
        @update:open-ai-image-protocol-mode="handleOpenAIImageProtocolModeChange"
        @update:open-ai-image-protocol-compat-allowed="openAIImageCompatAllowed = $event"
        @update:open-ai-ws-mode="openaiResponsesWebSocketV2Mode = $event"
        @update:anthropic-passthrough-enabled="anthropicPassthroughEnabled = $event"
        @update:codex-cli-only-enabled="codexCLIOnlyEnabled = $event"
      />

      <AccountAutoPauseToggle v-model:enabled="autoPauseOnExpired" />

      <AccountGroupSettingsEditor
        v-model:group-ids="form.group_ids"
        v-model:mixed-scheduling="mixedScheduling"
        :groups="groups"
        :platform="effectivePlatform"
        :platforms="effectiveGroupPlatforms"
        :simple-mode="authStore.isSimpleMode"
        :show-mixed-scheduling="form.platform === 'antigravity'"
      />

      </form>

      <AccountKiroAuthPanel
        v-else-if="form.platform === 'kiro'"
        ref="kiroAuthRef"
        :proxy-id="form.proxy_id"
        :submit-label="t('common.create')"
        :submitting="submitting"
        @submit="handleCreateKiroAccount"
      />

      <AccountCreateOAuthStep
        v-else
        ref="oauthFlowRef"
        :add-method="form.platform === 'anthropic' ? addMethod : 'oauth'"
        :auth-url="currentAuthUrl"
        :session-id="currentSessionId"
        :loading="currentOAuthLoading"
        :error="currentOAuthError"
        :show-help="form.platform === 'anthropic'"
        :show-proxy-warning="form.platform !== 'openai' && !!form.proxy_id"
        :allow-multiple="form.platform === 'anthropic'"
        :show-cookie-option="form.platform === 'anthropic'"
        :show-refresh-token-option="form.platform === 'openai' || form.platform === 'antigravity'"
        :show-refresh-token-submit-button="false"
        :platform="form.platform"
        :show-project-id="geminiOAuthType === 'code_assist'"
        @generate-url="handleGenerateUrl"
        @cookie-auth="handleCookieAuth"
        @validate-refresh-token="handleValidateRefreshToken"
        @update-input-method="handleOAuthInputMethodUpdate"
        @update-auth-code="oauthInputDraft.authCode = $event"
        @update-oauth-state="oauthInputDraft.oauthState = $event"
        @update-project-id="oauthInputDraft.projectId = $event"
        @update-session-key="oauthInputDraft.sessionKey = $event"
        @update-refresh-token="oauthInputDraft.refreshToken = $event"
      />
    </div>

    <template #footer>
      <AccountCreateFooterActions
        v-model:auto-import-models="autoImportModels"
        :step="step"
        :submitting="submitting"
        :is-o-auth-flow="isOAuthFlow"
        :is-manual-input-method="isManualInputMethod"
        :current-o-auth-loading="currentOAuthLoading"
        :can-exchange-code="canExchangeCode"
        :show-complete-auth-action="showCompleteAuthAction"
        :can-complete-auth="canCompleteAuth"
        @close="handleClose"
        @back="goBackToBasicInfo"
        @complete-auth="handleCompleteAuth"
      />
    </template>
  </BaseDialog>

  <AccountGeminiHelpDialog :show="showGeminiHelpDialog" @close="showGeminiHelpDialog = false" />

  <AccountMixedChannelWarningDialog
    :show="showMixedChannelWarning"
    :message="mixedChannelWarningMessageText"
    @confirm="handleMixedChannelConfirm"
    @cancel="handleMixedChannelCancel"
  />
</template>

<script setup lang="ts">
import BaseDialog from '@/components/common/BaseDialog.vue'
import AccountApiKeyBasicSettingsEditor from '@/components/account/AccountApiKeyBasicSettingsEditor.vue'
import AccountOpenRouterSettingsEditor from '@/components/account/AccountOpenRouterSettingsEditor.vue'
import AccountAntigravityModelMappingEditor from '@/components/account/AccountAntigravityModelMappingEditor.vue'
import AccountApiKeyModelProbeEditor from '@/components/account/AccountApiKeyModelProbeEditor.vue'
import AccountAutoPauseToggle from '@/components/account/AccountAutoPauseToggle.vue'
import AccountBaiduDocumentAICredentialsEditor from '@/components/account/AccountBaiduDocumentAICredentialsEditor.vue'
import AccountCreateFooterActions from '@/components/account/AccountCreateFooterActions.vue'
import AccountCreateOAuthStep from '@/components/account/AccountCreateOAuthStep.vue'
import AccountCreatePlatformSelector from '@/components/account/AccountCreatePlatformSelector.vue'
import AccountCreatePlatformTypeEditor from '@/components/account/AccountCreatePlatformTypeEditor.vue'
import AccountCustomErrorCodesEditor from '@/components/account/AccountCustomErrorCodesEditor.vue'
import AccountDeepSeekConcurrencyLimitsEditor from '@/components/account/AccountDeepSeekConcurrencyLimitsEditor.vue'
import AccountGatewaySettingsEditor from '@/components/account/AccountGatewaySettingsEditor.vue'
import AccountGoogleBatchArchiveEditor from '@/components/account/AccountGoogleBatchArchiveEditor.vue'
import AccountGeminiHelpDialog from '@/components/account/AccountGeminiHelpDialog.vue'
import AccountGeminiVertexCredentialsEditor from '@/components/account/AccountGeminiVertexCredentialsEditor.vue'
import AccountGrokOAuthPanel from '@/components/account/AccountGrokOAuthPanel.vue'
import AccountGrokImportPanel from '@/components/account/AccountGrokImportPanel.vue'
import AccountGroupSettingsEditor from '@/components/account/AccountGroupSettingsEditor.vue'
import AccountKiroAuthPanel from '@/components/account/AccountKiroAuthPanel.vue'
import AccountMixedChannelWarningDialog from '@/components/account/AccountMixedChannelWarningDialog.vue'
import AccountModelScopeEditor from '@/components/account/AccountModelScopeEditor.vue'
import AccountPoolModeEditor from '@/components/account/AccountPoolModeEditor.vue'
import AccountProtocolGatewayClaudeMimicEditor from '@/components/account/AccountProtocolGatewayClaudeMimicEditor.vue'
import AccountProtocolGatewayBatchEditor from '@/components/account/AccountProtocolGatewayBatchEditor.vue'
import AccountProtocolGatewayOpenAIRequestFormatEditor from '@/components/account/AccountProtocolGatewayOpenAIRequestFormatEditor.vue'
import AccountProtocolGatewayModelProbeEditor from '@/components/account/AccountProtocolGatewayModelProbeEditor.vue'
import AccountQuotaControlEditor from '@/components/account/AccountQuotaControlEditor.vue'
import AccountRuntimeSettingsEditor from '@/components/account/AccountRuntimeSettingsEditor.vue'
import AccountTempUnschedRulesEditor from '@/components/account/AccountTempUnschedRulesEditor.vue'
import AccountTierSelector from '@/components/account/AccountTierSelector.vue'
import QuotaLimitCard from '@/components/account/QuotaLimitCard.vue'
import { useCreateAccountModal } from './createAccountModal/useCreateAccountModal'
import type { CreateAccountModalEmits, CreateAccountModalProps } from './createAccountModal/types'

const props = defineProps<CreateAccountModalProps>()
const emit = defineEmits<CreateAccountModalEmits>()

const {
  t,
  authStore,
  oauthStepTitle,
  showFormError,
  showFormInfo,
  currentAuthUrl,
  currentSessionId,
  currentOAuthLoading,
  currentOAuthError,
  step,
  autoImportModels,
  accountCategory,
  addMethod,
  gatewayProtocol,
  apiKeyBaseUrl,
  apiKeyValue,
  openRouterHTTPReferer,
  openRouterTitle,
  deepSeekModelConcurrencyLimits,
  grokSSOToken,
  grokTier,
  grokOAuthRef,
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
  actualModelLocked,
  modelRestrictionEnabled,
  modelMappings,
  modelRestrictionMode,
  allowedModels,
  manualModels,
  modelProbeSnapshot,
  resolvedUpstream,
  protocolGatewayProbeModels,
  gatewayAcceptedProtocols,
  gatewayClientProfiles,
  gatewayClientRoutes,
  gatewayTestProvider,
  gatewayTestModelId,
  gatewayOpenAIRequestFormat,
  gatewayBatchEnabled,
  claudeCodeMimicEnabled,
  claudeTLSFingerprintEnabled,
  claudeSessionIDMaskingEnabled,
  poolModeState,
  customErrorCodesState,
  interceptWarmupRequests,
  autoPauseOnExpired,
  autoRenewEnabled,
  autoRenewPeriod,
  expiryProbeExtensionDays,
  openaiPassthroughEnabled,
  openAIImageProtocolMode,
  openAIImageCompatAllowed,
  codexCLIOnlyEnabled,
  anthropicPassthroughEnabled,
  gatewayOpenAIImageProtocolMode,
  mixedScheduling,
  antigravityAccountType,
  upstreamBaseUrl,
  upstreamApiKey,
  geminiVertexAuthMode,
  geminiVertexProjectId,
  geminiVertexLocation,
  geminiVertexServiceAccountJson,
  geminiVertexApiKey,
  geminiVertexAccessToken,
  geminiVertexExpiresAtInput,
  geminiVertexBaseUrl,
  baiduDocumentAIAccessToken,
  baiduDocumentAIAsyncBaseUrl,
  baiduDocumentAIDirectApiUrlsText,
  oauthDraftCredentials,
  oauthDraftExtra,
  apiKeyProbeCredentials,
  upstreamProbeCredentials,
  vertexProbeCredentials,
  isApiKeyProbeReady,
  isUpstreamProbeReady,
  oauthDraftProbeReady,
  isVertexProbeReady,
  showCommonApiKeySection,
  showApiKeyModelScopeEditor,
  showDeepSeekConcurrencyEditor,
  showStandaloneModelScopeEditor,
  showQuotaLimitSection,
  showGeminiAIStudioBatchArchiveEditor,
  showGeminiVertexBatchArchiveEditor,
  showOAuthFinalizeStep,
  showOAuthFinalizeProbeEditor,
  antigravityModelMappings,
  antigravityPresetMappings,
  getModelMappingKey,
  getAntigravityModelMappingKey,
  geminiOAuthType,
  geminiAIStudioOAuthEnabled,
  showAdvancedOAuth,
  showGeminiHelpDialog,
  quotaControlState,
  umqModeOptions,
  geminiTierGoogleOne,
  geminiTierGcp,
  geminiTierAIStudio,
  accountTier,
  effectivePlatform,
  effectiveGroupPlatforms,
  showProtocolGatewayClaudeMimicEditor,
  showProtocolGatewayBatchEditor,
  showProtocolGatewayOpenAIRequestFormatEditor,
  protocolGatewayBatchRequestFormats,
  shouldPersistGeminiTierId,
  openAIWSModeOptions,
  openaiResponsesWebSocketV2Mode,
  openAIWSModeConcurrencyHintKey,
  commonErrorCodeOptions,
  isOpenAIModelRestrictionDisabled,
  geminiHelpLinks,
  presetMappings,
  form,
  isBaiduDocumentAISelected,
  tempUnschedEnabled,
  tempUnschedRules,
  tempUnschedPresets,
  getTempUnschedRuleKey,
  addTempUnschedRule,
  removeTempUnschedRule,
  moveTempUnschedRule,
  showMixedChannelWarning,
  mixedChannelWarningMessageText,
  handleMixedChannelConfirm,
  handleMixedChannelCancel,
  isOAuthFlow,
  isManualInputMethod,
  oauthInputDraft,
  showCompleteAuthAction,
  expiresAtInput,
  canExchangeCode,
  canCompleteAuth,
  handleOAuthInputMethodUpdate,
  handleOpenAIImageProtocolModeChange,
  applyAccountTierCapacity,
  addModelMapping,
  removeModelMapping,
  addPresetMapping,
  addAntigravityModelMapping,
  removeAntigravityModelMapping,
  addAntigravityPresetMapping,
  handleClose,
  submitting,
  handleCookieAuth,
  handleCreateKiroAccount,
  handleCreateGrokOAuthAccount,
  handleGrokImportCompleted,
  goBackToBasicInfo,
  handleGenerateUrl,
  handleValidateRefreshToken,
  handleCompleteAuth,
  probeExtraForEditor,
  buildProbeExtra,
  DEFAULT_POOL_MODE_RETRY_COUNT,
  MAX_POOL_MODE_RETRY_COUNT,
  handleSubmit
} = useCreateAccountModal(props, emit)
</script>
