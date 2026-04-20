<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.editAccount')"
    width="account-wide"
    @close="handleClose"
  >
    <div
      v-if="loading"
      class="flex min-h-[12rem] items-center justify-center rounded-lg border border-dashed border-gray-200 bg-gray-50/60 px-6 py-10 text-sm text-gray-500 dark:border-dark-600 dark:bg-dark-700/30 dark:text-gray-300"
    >
      {{ t('common.loading') }}
    </div>
    <form
      v-else-if="account"
      id="edit-account-form"
      @submit.prevent="handleSubmit"
      class="space-y-5"
    >
      <div>
        <label class="input-label">{{ t('common.name') }}</label>
        <input v-model="form.name" type="text" required class="input" data-tour="edit-account-form-name" />
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

      <AccountBaiduDocumentAICredentialsEditor
        v-if="isBaiduDocumentAIAccount"
        v-model:async-bearer-token="baiduDocumentAIAsyncBearerToken"
        v-model:async-base-url="baiduDocumentAIAsyncBaseUrl"
        v-model:direct-token="baiduDocumentAIDirectToken"
        v-model:direct-api-urls-text="baiduDocumentAIDirectApiUrlsText"
        mode="edit"
      />

      <div
        v-if="isGrokSSOAccount"
        class="space-y-4 rounded-lg border border-slate-200 bg-slate-50/60 p-4 dark:border-slate-700 dark:bg-slate-900/30"
      >
        <div v-if="isGrokSSOAccount">
          <label class="input-label">{{ t('admin.accounts.grokToken') }}</label>
          <textarea
            v-model="editGrokSSOToken"
            rows="4"
            class="input"
            :placeholder="t('admin.accounts.leaveEmptyToKeep')"
          />
          <p class="input-hint">{{ t('admin.accounts.grokTokenHint') }}</p>
        </div>

        <div>
          <label class="input-label">{{ t('admin.accounts.grokTier') }}</label>
          <select v-model="editGrokTier" class="input">
            <option value="basic">{{ t('admin.accounts.grokTierBasic') }}</option>
            <option value="super">{{ t('admin.accounts.grokTierSuper') }}</option>
            <option value="heavy">{{ t('admin.accounts.grokTierHeavy') }}</option>
          </select>
          <p class="input-hint">{{ t('admin.accounts.grokTierHint') }}</p>
        </div>

        <div
          v-if="isGrokSSOAccount"
          class="space-y-3 rounded-lg border border-slate-200 bg-white/80 p-3 dark:border-slate-700 dark:bg-slate-950/40"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <div class="text-sm font-semibold text-slate-900 dark:text-slate-100">
                {{ t('admin.accounts.grokDerivedMappingTitle') }}
              </div>
              <p class="text-xs leading-5 text-slate-500 dark:text-slate-400">
                {{ t('admin.accounts.grokDerivedMappingHint') }}
              </p>
            </div>
            <button type="button" class="btn btn-secondary btn-sm" @click="applyDefaultGrokCapabilityMapping">
              {{ t('admin.accounts.grokApplyCapabilityMapping') }}
            </button>
          </div>
          <div class="flex flex-wrap gap-2">
            <span
              v-for="model in grokCapabilityModels"
              :key="model"
              class="rounded-full bg-slate-100 px-2.5 py-1 text-[11px] font-medium text-slate-700 dark:bg-slate-800 dark:text-slate-200"
            >
              {{ model }}
            </span>
          </div>
        </div>
      </div>

      <!-- API Key fields (only for apikey type) -->
      <div v-if="showCommonApiKeySection" class="space-y-4">
        <div v-if="isProtocolGatewayAccount">
          <label class="input-label">{{ t('admin.accounts.protocolGateway.protocolLabel') }}</label>
          <Select
            v-model="gatewayProtocol"
            :options="gatewayProtocolOptions"
            value-key="value"
            label-key="label"
          >
            <template #selected="{ option }">
              <PlatformLabel
                v-if="isGatewayProtocolOption(option)"
                :platform="option.iconPlatform"
                :label="option.label"
                :description="option.requestFormatsText"
              />
            </template>

            <template #option="{ option }">
              <PlatformLabel
                v-if="isGatewayProtocolOption(option)"
                :platform="option.iconPlatform"
                :label="option.label"
                :description="option.requestFormatsText"
              />
            </template>
          </Select>
          <p class="input-hint">{{ t('admin.accounts.protocolGateway.protocolHint') }}</p>
        </div>

        <AccountApiKeyBasicSettingsEditor
          v-model:base-url="editBaseUrl"
          v-model:api-key="editApiKey"
          v-model:actual-model-locked="actualModelLocked"
          v-model:model-scope-mode="modelRestrictionMode"
          v-model:allowed-models="allowedModels"
          v-model:gemini-tier-ai-studio="geminiTierAIStudio"
          :platform="account.platform"
          :gateway-protocol="gatewayProtocol"
          :effective-platform="effectivePlatform"
          mode="edit"
          :model-scope-disabled="isOpenAIModelRestrictionDisabled"
          :skip-model-scope-editor="isProtocolGatewayAccount"
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
        />

        <AccountProtocolGatewayBatchEditor
          v-if="showProtocolGatewayBatchEditor"
          v-model:enabled="gatewayBatchEnabled"
          :request-formats="protocolGatewayBatchRequestFormats"
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

      <AccountGeminiVertexCredentialsEditor
        v-if="isGeminiVertexAccount"
        v-model:auth-mode="geminiVertexAuthMode"
        v-model:project-id="geminiVertexProjectId"
        v-model:location="geminiVertexLocation"
        v-model:service-account-json="geminiVertexServiceAccountJson"
        v-model:api-key="geminiVertexApiKey"
        v-model:legacy-access-token="geminiVertexAccessToken"
        v-model:legacy-expires-at-input="geminiVertexExpiresAtInput"
        v-model:base-url="geminiVertexBaseUrl"
        mode="edit"
        :legacy-mode="isGeminiVertexLegacyMode"
      />

      <AccountModelScopeEditor
        v-if="showStandaloneModelScopeEditor"
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

      <!-- Upstream fields (only for upstream type) -->
      <div v-if="account.type === 'upstream'" class="space-y-4">
        <AccountUpstreamSettingsEditor
          v-model:base-url="editBaseUrl"
          v-model:api-key="editApiKey"
          mode="edit"
        />
      </div>

      <AccountProtocolGatewayModelProbeEditor
        v-if="showUnifiedProtocolGatewayProbeEditor"
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
        :base-url="editBaseUrl"
        :api-key="resolvedProtocolGatewayApiKey"
        :proxy-id="form.proxy_id"
      />

      <AccountApiKeyModelProbeEditor
        v-else-if="showUnifiedAPIModelProbeEditor"
        v-model:allowed-models="allowedModels"
        v-model:model-mappings="modelMappings"
        v-model:probed-models="protocolGatewayProbeModels"
        v-model:manual-models="manualModels"
        v-model:probe-snapshot="modelProbeSnapshot"
        v-model:resolved-upstream="resolvedUpstream"
        :platform="account.platform"
        :account-type="unifiedProbeAccountType"
        :credentials="unifiedProbeCredentials"
        :extra="probeExtraForEditor"
        :probe-ready="unifiedProbeReady"
        :proxy-id="form.proxy_id"
      />

      <!-- Antigravity model restriction (applies to all antigravity types) -->
      <AccountAntigravityModelMappingEditor
        v-if="account.platform === 'antigravity'"
        :model-mappings="antigravityModelMappings"
        :preset-mappings="antigravityPresetMappings"
        :get-mapping-key="getAntigravityModelMappingKey"
        @add-mapping="addAntigravityModelMapping"
        @remove-mapping="removeAntigravityModelMapping"
        @add-preset="addAntigravityPresetMapping($event.from, $event.to)"
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
        v-if="effectivePlatform === 'anthropic' || account?.platform === 'antigravity'"
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

      <AccountRuntimeSettingsEditor
        v-model:proxy-id="form.proxy_id"
        v-model:concurrency="form.concurrency"
        v-model:load-factor="form.load_factor"
        v-model:priority="form.priority"
        v-model:rate-multiplier="form.rate_multiplier"
        v-model:expires-at-input="expiresAtInput"
        :proxies="proxies"
      />

      <AccountGatewaySettingsEditor
        :show-open-ai-passthrough="effectivePlatform === 'openai' && (account?.type === 'oauth' || account?.type === 'apikey')"
        :open-ai-passthrough-enabled="openaiPassthroughEnabled"
        :show-open-ai-ws-mode="effectivePlatform === 'openai' && (account?.type === 'oauth' || account?.type === 'apikey')"
        :open-ai-ws-mode="openaiResponsesWebSocketV2Mode"
        :open-ai-ws-mode-options="openAIWSModeOptions"
        :open-ai-ws-mode-concurrency-hint-key="openAIWSModeConcurrencyHintKey"
        :show-anthropic-passthrough="effectivePlatform === 'anthropic' && account?.type === 'apikey'"
        :anthropic-passthrough-enabled="anthropicPassthroughEnabled"
        :show-codex-cli-only="effectivePlatform === 'openai' && account?.type === 'oauth'"
        :codex-cli-only-enabled="codexCLIOnlyEnabled"
        @update:open-ai-passthrough-enabled="openaiPassthroughEnabled = $event"
        @update:open-ai-ws-mode="openaiResponsesWebSocketV2Mode = $event"
        @update:anthropic-passthrough-enabled="anthropicPassthroughEnabled = $event"
        @update:codex-cli-only-enabled="codexCLIOnlyEnabled = $event"
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

      <AccountAutoPauseToggle v-model:enabled="autoPauseOnExpired" />

      <AccountQuotaControlEditor
        v-if="effectivePlatform === 'anthropic' && (account?.type === 'oauth' || account?.type === 'setup-token')"
        v-model:state="quotaControlState"
        :umq-mode-options="umqModeOptions"
      />

      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div>
          <label class="input-label">{{ t('common.status') }}</label>
          <Select v-model="form.status" :options="statusOptions" />
        </div>
      </div>

      <AccountGroupSettingsEditor
        v-model:group-ids="form.group_ids"
        v-model:mixed-scheduling="mixedScheduling"
        :groups="groups"
        :platform="effectivePlatform"
        :platforms="effectiveGroupPlatforms"
        :simple-mode="authStore.isSimpleMode"
        :show-mixed-scheduling="account?.platform === 'antigravity'"
        mixed-scheduling-readonly
      />

    </form>

    <template #footer>
      <div v-if="loading" class="flex justify-end">
        <button @click="handleClose" type="button" class="btn btn-secondary">
          {{ t('common.cancel') }}
        </button>
      </div>
      <div v-if="account" class="flex justify-end gap-3">
        <button @click="handleClose" type="button" class="btn btn-secondary">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          form="edit-account-form"
          :disabled="submitting"
          class="btn btn-primary"
          data-tour="account-form-submit"
        >
          <svg
            v-if="submitting"
            class="-ml-1 mr-2 h-4 w-4 animate-spin"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
          {{ submitting ? t('admin.accounts.updating') : t('common.update') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <AccountMixedChannelWarningDialog
    :show="showMixedChannelWarning"
    :message="mixedChannelWarningMessageText"
    @confirm="handleMixedChannelConfirm"
    @cancel="handleMixedChannelCancel"
  />
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { adminAPI } from '@/api/admin'
import { useAnthropicQuotaControl } from '@/composables/useAnthropicQuotaControl'
import { useAccountMixedChannelRisk } from '@/composables/useAccountMixedChannelRisk'
import { useAccountTempUnschedRules } from '@/composables/useAccountTempUnschedRules'
import type { AccountManualModel } from '@/api/admin/accounts'
import type { Account, AccountPlatform, Proxy, AdminGroup, GatewayProtocol, GroupPlatform } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import PlatformLabel from '@/components/common/PlatformLabel.vue'
import AccountApiKeyBasicSettingsEditor from '@/components/account/AccountApiKeyBasicSettingsEditor.vue'
import AccountAntigravityModelMappingEditor from '@/components/account/AccountAntigravityModelMappingEditor.vue'
import AccountApiKeyModelProbeEditor from '@/components/account/AccountApiKeyModelProbeEditor.vue'
import AccountAutoPauseToggle from '@/components/account/AccountAutoPauseToggle.vue'
import AccountBaiduDocumentAICredentialsEditor from '@/components/account/AccountBaiduDocumentAICredentialsEditor.vue'
import AccountCustomErrorCodesEditor from '@/components/account/AccountCustomErrorCodesEditor.vue'
import AccountGatewaySettingsEditor from '@/components/account/AccountGatewaySettingsEditor.vue'
import AccountGoogleBatchArchiveEditor from '@/components/account/AccountGoogleBatchArchiveEditor.vue'
import AccountGeminiVertexCredentialsEditor from '@/components/account/AccountGeminiVertexCredentialsEditor.vue'
import AccountGroupSettingsEditor from '@/components/account/AccountGroupSettingsEditor.vue'
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
import AccountUpstreamSettingsEditor from '@/components/account/AccountUpstreamSettingsEditor.vue'
import QuotaLimitCard from '@/components/account/QuotaLimitCard.vue'
import { applyInterceptWarmup } from '@/components/account/credentialsBuilder'
import { formatDateTimeLocalInput, parseDateTimeLocalInput } from '@/utils/format'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import {
  DEFAULT_POOL_MODE_RETRY_COUNT,
  MAX_POOL_MODE_RETRY_COUNT,
  createDefaultAccountCustomErrorCodesState,
  createDefaultAccountPoolModeState,
  type ModelMapping
} from '@/utils/accountFormShared'
import {
  applyAccountCustomErrorCodesStateToCredentials,
  applyAccountPoolModeStateToCredentials,
  loadAccountCustomErrorCodesStateFromCredentials,
  loadAccountPoolModeStateFromCredentials,
  resetAccountCustomErrorCodesState,
  resetAccountPoolModeState
} from '@/utils/accountApiKeyAdvancedSettingsForm'
import { resolveAccountApiKeyDefaultBaseUrl } from '@/utils/accountApiKeyBasicSettings'
import {
  OPENAI_WS_MODE_CTX_POOL,
  OPENAI_WS_MODE_OFF,
  OPENAI_WS_MODE_PASSTHROUGH,
  isOpenAIWSModeEnabled,
  resolveOpenAIWSModeConcurrencyHintKey,
  type OpenAIWSMode,
  resolveOpenAIWSModeFromExtra
} from '@/utils/openaiWsMode'
import {
  getPresetMappingsByPlatform,
  createCommonErrorCodeOptions,
  buildModelMappingObject
} from '@/composables/useModelWhitelist'
import { buildAccountModelScopeExtra } from '@/utils/accountModelScope'
import type { ProtocolGatewayProbeModel } from '@/api/admin/accounts'
import {
  GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS,
  resolveVertexAuthBaseUrl,
  resolveVertexBaseUrl,
  type VertexAuthMode
} from '@/utils/vertexAi'
import {
  grokDefaultModelIdsForTier,
  grokDefaultModelMappingForTier,
  mappingRecordToRows,
  normalizeGrokTier,
  type GrokTier
} from '@/utils/grokAccount'
import {
  DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT,
  applyProtocolGatewayOpenAIRequestFormatExtra,
  applyProtocolGatewayGeminiBatchExtra,
  applyProtocolGatewayClaudeClientMimicExtra,
  PROTOCOL_GATEWAY_PROTOCOLS,
  isProtocolGatewayPlatform,
  normalizeGatewayBatchEnabled,
  normalizeGatewayAcceptedProtocols,
  normalizeGatewayClientProfile,
  normalizeGatewayClientRoutes,
  resolveAccountGatewayOpenAIRequestFormat,
  resolveProtocolGatewayBatchRequestFormats,
  resolveAccountGatewayProtocol,
  resolveEffectiveAccountPlatform,
  resolveEffectiveAccountPlatforms,
  resolveGatewayProtocolDescriptor,
  supportsProtocolGatewayClaudeClientMimic,
  supportsProtocolGatewayGeminiBatch,
  supportsProtocolGatewayOpenAIRequestFormat
} from '@/utils/accountProtocolGateway'
import {
  normalizeGeminiAIStudioTier,
  normalizeGeminiOAuthType,
  isGeminiVertexAI,
  type GeminiAIStudioTier,
  type GeminiOAuthType
} from '@/utils/geminiAccount'
import {
  applyGoogleBatchArchiveExtra,
  createDefaultGoogleBatchArchiveFormState,
  readGoogleBatchArchiveFormState,
  resolveGoogleBatchArchiveTargetKind,
  type GoogleBatchArchiveBillingMode
} from '@/utils/accountGoogleBatchArchive'
import {
  BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL,
  parseBaiduDocumentAIDirectApiUrlsInput,
  stringifyBaiduDocumentAIDirectApiUrls
} from '@/utils/baiduDocumentAI'
import type {
  GatewayAcceptedProtocol,
  GatewayClientProfile,
  GatewayClientRoute,
  GatewayOpenAIRequestFormat
} from '@/types'
import { formatModelDisplayName } from '@/utils/modelDisplayName'
import {
  deriveConfiguredAccountModelIds,
  mergeAccountModelProbeSnapshotIntoExtra,
  mergeAccountManualModelsIntoExtra,
  mergeResolvedUpstreamDraftIntoExtra,
  readAccountModelProbeSnapshot,
  readAccountManualModelsFromExtra,
  readAccountResolvedUpstreamDraft,
  type AccountModelProbeSnapshotDraft,
  type AccountResolvedUpstreamDraft
} from '@/utils/accountProbeDraft'

interface Props {
  show: boolean
  loading: boolean
  account: Account | null
  proxies: Proxy[]
  groups: AdminGroup[]
}

interface GatewayProtocolOption extends Record<string, unknown> {
  value: GatewayProtocol
  label: string
  requestFormatsText: string
  iconPlatform: AccountPlatform
}

const props = withDefaults(defineProps<Props>(), {
  loading: false
})
const emit = defineEmits<{
  close: []
  updated: [account: Account]
}>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const showFormError = (message: string) => appStore.showError(message)
const showFormInfo = (message: string) => appStore.showInfo(message)

const antigravityPresetMappings = computed(() => getPresetMappingsByPlatform('antigravity'))

// State
const submitting = ref(false)
const gatewayProtocol = ref<GatewayProtocol>('openai')
const editBaseUrl = ref(resolveAccountApiKeyDefaultBaseUrl('anthropic'))
const editApiKey = ref('')
const editGrokSSOToken = ref('')
const editGrokTier = ref<GrokTier>('basic')
const modelMappings = ref<ModelMapping[]>([])
const actualModelLocked = ref(true)
const modelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const allowedModels = ref<string[]>([])
const manualModels = ref<AccountManualModel[]>([])
const modelProbeSnapshot = ref<AccountModelProbeSnapshotDraft | null>(null)
const resolvedUpstream = ref<AccountResolvedUpstreamDraft | null>(null)
const protocolGatewayProbeModels = ref<ProtocolGatewayProbeModel[]>([])
const gatewayAcceptedProtocols = ref<GatewayAcceptedProtocol[]>(['openai'])
const gatewayClientProfiles = ref<GatewayClientProfile[]>([])
const gatewayClientRoutes = ref<GatewayClientRoute[]>([])
const gatewayTestProvider = ref('')
const gatewayTestModelId = ref('')
const gatewayOpenAIRequestFormat = ref<GatewayOpenAIRequestFormat>(DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT)
const gatewayBatchEnabled = ref(false)
const claudeCodeMimicEnabled = ref(false)
const claudeTLSFingerprintEnabled = ref(false)
const claudeSessionIDMaskingEnabled = ref(false)
const poolModeState = reactive(createDefaultAccountPoolModeState(DEFAULT_POOL_MODE_RETRY_COUNT))
const customErrorCodesState = reactive(createDefaultAccountCustomErrorCodesState())
const interceptWarmupRequests = ref(false)
const autoPauseOnExpired = ref(false)
const mixedScheduling = ref(false) // For antigravity accounts: enable mixed scheduling
const antigravityModelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const antigravityWhitelistModels = ref<string[]>([])
const antigravityModelMappings = ref<ModelMapping[]>([])
const getModelMappingKey = createStableObjectKeyResolver<ModelMapping>('edit-model-mapping')
const getAntigravityModelMappingKey = createStableObjectKeyResolver<ModelMapping>('edit-antigravity-model-mapping')
const quotaControl = useAnthropicQuotaControl()
const quotaControlState = quotaControl.state
const umqModeOptions = quotaControl.umqModeOptions

const openaiPassthroughEnabled = ref(false)
const openaiOAuthResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const openaiAPIKeyResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const codexCLIOnlyEnabled = ref(false)
const anthropicPassthroughEnabled = ref(false)
const editQuotaLimit = ref<number | null>(null)
const editQuotaDailyLimit = ref<number | null>(null)
const editQuotaWeeklyLimit = ref<number | null>(null)
const editQuotaDailyResetMode = ref<'rolling' | 'fixed' | null>(null)
const editQuotaDailyResetHour = ref<number | null>(null)
const editQuotaWeeklyResetMode = ref<'rolling' | 'fixed' | null>(null)
const editQuotaWeeklyResetDay = ref<number | null>(null)
const editQuotaWeeklyResetHour = ref<number | null>(null)
const editQuotaResetTimezone = ref<string | null>(null)
const defaultGoogleBatchArchiveState = createDefaultGoogleBatchArchiveFormState()
const batchArchiveEnabled = ref(defaultGoogleBatchArchiveState.enabled)
const batchArchiveAutoPrefetchEnabled = ref(defaultGoogleBatchArchiveState.autoPrefetchEnabled)
const batchArchiveRetentionDays = ref(defaultGoogleBatchArchiveState.retentionDays)
const batchArchiveBillingMode = ref<GoogleBatchArchiveBillingMode>(defaultGoogleBatchArchiveState.billingMode)
const batchArchiveDownloadPriceUSD = ref(defaultGoogleBatchArchiveState.downloadPriceUSD)
const allowVertexBatchOverflow = ref(defaultGoogleBatchArchiveState.allowVertexBatchOverflow)
const acceptAIStudioBatchOverflow = ref(defaultGoogleBatchArchiveState.acceptAIStudioBatchOverflow)
const geminiOAuthType = ref<GeminiOAuthType>('code_assist')
const geminiTierAIStudio = ref<GeminiAIStudioTier>('aistudio_free')
const geminiVertexAuthMode = ref<VertexAuthMode>('service_account')
const geminiVertexProjectId = ref('')
const geminiVertexLocation = ref('')
const geminiVertexServiceAccountJson = ref('')
const geminiVertexApiKey = ref('')
const geminiVertexAccessToken = ref('')
const geminiVertexExpiresAtInput = ref('')
const geminiVertexBaseUrl = ref('')
const baiduDocumentAIAsyncBearerToken = ref('')
const baiduDocumentAIAsyncBaseUrl = ref(BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL)
const baiduDocumentAIDirectToken = ref('')
const baiduDocumentAIDirectApiUrlsText = ref('')
const currentAccountCredentials = computed<Record<string, unknown>>(
  () => ((props.account?.credentials as Record<string, unknown> | undefined) || {})
)
const currentAccountExtra = computed<Record<string, unknown>>(
  () => ((props.account?.extra as Record<string, unknown> | undefined) || {})
)
const apiKeyProbeCredentials = computed<Record<string, unknown>>(() => {
  const credentials: Record<string, unknown> = {
    api_key: editApiKey.value.trim() || String(currentAccountCredentials.value.api_key || '').trim(),
    base_url: editBaseUrl.value.trim() || resolveAccountApiKeyDefaultBaseUrl(props.account?.platform || 'anthropic', gatewayProtocol.value)
  }
  if (shouldPersistGeminiTierId.value) {
    credentials.tier_id =
      normalizeGeminiAIStudioTier(geminiTierAIStudio.value || currentAccountCredentials.value.tier_id) ||
      'aistudio_free'
  }
  return credentials
})
const upstreamProbeCredentials = computed<Record<string, unknown>>(() => ({
  api_key: editApiKey.value.trim() || String(currentAccountCredentials.value.api_key || '').trim(),
  base_url: editBaseUrl.value.trim() || String(currentAccountCredentials.value.base_url || '').trim()
}))
const vertexProbeCredentials = computed<Record<string, unknown>>(() => {
  const baseUrl = geminiVertexBaseUrl.value.trim() || resolveVertexAuthBaseUrl(
    geminiVertexAuthMode.value,
    geminiVertexLocation.value
  )
  if (geminiVertexAuthMode.value === 'express_api_key') {
    return {
      gemini_api_variant: GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS,
      api_key: geminiVertexApiKey.value.trim() || String(currentAccountCredentials.value.api_key || '').trim(),
      base_url: baseUrl
    }
  }
  const credentials: Record<string, unknown> = {
    oauth_type: 'vertex_ai',
    vertex_project_id: geminiVertexProjectId.value.trim(),
    vertex_location: geminiVertexLocation.value.trim(),
    base_url: baseUrl
  }
  const serviceAccountJson =
    geminiVertexServiceAccountJson.value.trim() ||
    String(currentAccountCredentials.value.vertex_service_account_json || '').trim()
  if (serviceAccountJson) {
    credentials.vertex_service_account_json = serviceAccountJson
  }
  const legacyToken = geminiVertexAccessToken.value.trim() || String(currentAccountCredentials.value.access_token || '').trim()
  if (legacyToken) {
    credentials.access_token = legacyToken
  }
  return credentials
})
const isApiKeyProbeReady = computed(() => Boolean(apiKeyProbeCredentials.value.api_key))
const isUpstreamProbeReady = computed(() => Boolean(upstreamProbeCredentials.value.api_key))
const isVertexProbeReady = computed(() => {
  if (geminiVertexAuthMode.value === 'express_api_key') {
    return Boolean(vertexProbeCredentials.value.api_key)
  }
  return Boolean(
    geminiVertexProjectId.value.trim() &&
      geminiVertexLocation.value.trim() &&
      ((vertexProbeCredentials.value.vertex_service_account_json as string | undefined) ||
        (vertexProbeCredentials.value.access_token as string | undefined))
  )
})
const oauthProbeCredentials = computed<Record<string, unknown>>(() => {
  const credentials: Record<string, unknown> = {
    ...currentAccountCredentials.value
  }
  if (isGrokSSOAccount.value) {
    const ssoToken = editGrokSSOToken.value.trim() || String(currentAccountCredentials.value.sso_token || '').trim()
    if (ssoToken) {
      credentials.sso_token = ssoToken
    }
  }
  return credentials
})
const oauthProbeReady = computed(() => {
  if (isGrokSSOAccount.value) {
    return Boolean(String(oauthProbeCredentials.value.sso_token || '').trim())
  }
  return Object.keys(oauthProbeCredentials.value).length > 0
})
const effectivePlatform = computed<GroupPlatform>(() => {
  const platform = resolveEffectiveAccountPlatform(props.account?.platform || 'anthropic', gatewayProtocol.value)
  return platform === 'protocol_gateway' ? 'openai' : platform
})
const effectiveGroupPlatforms = computed<GroupPlatform[] | undefined>(() => {
  if (!isProtocolGatewayPlatform(props.account?.platform)) {
    return undefined
  }
  return resolveEffectiveAccountPlatforms(
    props.account?.platform || 'protocol_gateway',
    gatewayProtocol.value,
    gatewayAcceptedProtocols.value
  ) as GroupPlatform[]
})
const isProtocolGatewayAccount = computed(() =>
  isProtocolGatewayPlatform(props.account?.platform)
)
const isBaiduDocumentAIAccount = computed(() =>
  props.account?.platform === 'baidu_document_ai'
)
const isGrokSSOAccount = computed(() => props.account?.platform === 'grok' && props.account?.type === 'sso')
const isGeminiVertexAccount = computed(() =>
  effectivePlatform.value === 'gemini' &&
  (
    (props.account?.type === 'oauth' && isGeminiVertexAI(geminiOAuthType.value)) ||
    (props.account?.type === 'apikey' && String(currentAccountCredentials.value.gemini_api_variant || '').trim().toLowerCase() === GEMINI_API_KEY_VARIANT_VERTEX_EXPRESS)
  )
)
const isGeminiVertexLegacyMode = computed(() => {
  const credentials = (props.account?.credentials as Record<string, unknown> | undefined) || {}
  return (
    isGeminiVertexAccount.value &&
    geminiVertexAuthMode.value === 'service_account' &&
    !String(credentials.vertex_service_account_json || '').trim() &&
    Boolean(String(credentials.access_token || '').trim())
  )
})
const showCommonApiKeySection = computed(() =>
  props.account?.type === 'apikey' &&
  !isGeminiVertexAccount.value &&
  !isBaiduDocumentAIAccount.value
)
const supportsUnifiedModelEditor = computed(() => {
  if (!props.account) {
    return false
  }
  if (props.account.platform === 'antigravity' || props.account.platform === 'baidu_document_ai') {
    return false
  }
  if (isProtocolGatewayAccount.value || isGeminiVertexAccount.value || props.account.type === 'upstream') {
    return true
  }
  if (props.account.type === 'apikey') {
    return true
  }
  if (props.account.platform === 'grok' && props.account.type === 'sso') {
    return true
  }
  if (props.account.type === 'oauth') {
    return ['openai', 'anthropic', 'gemini', 'copilot', 'kiro'].includes(props.account.platform)
  }
  return props.account.type === 'setup-token' && props.account.platform === 'anthropic'
})
const showUnifiedProtocolGatewayProbeEditor = computed(() =>
  supportsUnifiedModelEditor.value && isProtocolGatewayAccount.value
)
const showUnifiedAPIModelProbeEditor = computed(() =>
  supportsUnifiedModelEditor.value && !isProtocolGatewayAccount.value
)
const showStandaloneModelScopeEditor = computed(() => {
  if (!supportsUnifiedModelEditor.value || effectivePlatform.value === 'antigravity') {
    return false
  }
  if (isProtocolGatewayAccount.value || isGeminiVertexAccount.value || props.account?.type === 'upstream') {
    return true
  }
  return props.account?.type === 'oauth' || props.account?.type === 'setup-token' || props.account?.type === 'sso'
})
const unifiedProbeAccountType = computed(() => {
  if (isGeminiVertexAccount.value) {
    return geminiVertexAuthMode.value === 'express_api_key' ? 'apikey' : 'oauth'
  }
  return String(props.account?.type || 'apikey')
})
const unifiedProbeCredentials = computed<Record<string, unknown>>(() => {
  if (isGeminiVertexAccount.value) {
    return vertexProbeCredentials.value
  }
  if (props.account?.type === 'upstream') {
    return upstreamProbeCredentials.value
  }
  if (props.account?.type === 'apikey') {
    return apiKeyProbeCredentials.value
  }
  return oauthProbeCredentials.value
})
const unifiedProbeReady = computed(() => {
  if (isGeminiVertexAccount.value) {
    return isVertexProbeReady.value
  }
  if (props.account?.type === 'upstream') {
    return isUpstreamProbeReady.value
  }
  if (props.account?.type === 'apikey') {
    return isApiKeyProbeReady.value
  }
  return oauthProbeReady.value
})
const showQuotaLimitSection = computed(() => Boolean(props.account))
const shouldPersistGeminiTierId = computed(() =>
  props.account?.platform === 'gemini' &&
  props.account?.type === 'apikey' &&
  !isGeminiVertexAccount.value
)
const showGeminiAIStudioBatchArchiveEditor = computed(() =>
  resolveGoogleBatchArchiveTargetKind(
    props.account?.platform,
    props.account?.type,
    currentAccountCredentials.value,
  ) === 'ai_studio'
)
const showGeminiVertexBatchArchiveEditor = computed(() =>
  resolveGoogleBatchArchiveTargetKind(
    props.account?.platform,
    props.account?.type,
    currentAccountCredentials.value,
  ) === 'vertex'
)
const grokCapabilityModels = computed(() => grokDefaultModelIdsForTier(editGrokTier.value))
const showProtocolGatewayClaudeMimicEditor = computed(() =>
  supportsProtocolGatewayClaudeClientMimic({
    platform: props.account?.platform,
    type: props.account?.type,
    gatewayProtocol: gatewayProtocol.value,
    acceptedProtocols: gatewayAcceptedProtocols.value
  })
)
const showProtocolGatewayBatchEditor = computed(() =>
  supportsProtocolGatewayGeminiBatch({
    platform: props.account?.platform,
    type: props.account?.type,
    gatewayProtocol: gatewayProtocol.value,
    acceptedProtocols: gatewayAcceptedProtocols.value
  })
)
const showProtocolGatewayOpenAIRequestFormatEditor = computed(() =>
  supportsProtocolGatewayOpenAIRequestFormat({
    platform: props.account?.platform,
    type: props.account?.type,
    gatewayProtocol: gatewayProtocol.value,
    acceptedProtocols: gatewayAcceptedProtocols.value
  })
)
const protocolGatewayBatchRequestFormats = computed(() =>
  resolveProtocolGatewayBatchRequestFormats({
    gatewayProtocol: gatewayProtocol.value,
    acceptedProtocols: gatewayAcceptedProtocols.value
  })
)
const resolvedProtocolGatewayApiKey = computed(() => {
  if (editApiKey.value.trim()) {
    return editApiKey.value.trim()
  }
  const currentCredentials = (props.account?.credentials as Record<string, unknown>) || {}
  return typeof currentCredentials.api_key === 'string' ? currentCredentials.api_key : ''
})
const gatewayProtocolOptions = computed<GatewayProtocolOption[]>(() =>
  PROTOCOL_GATEWAY_PROTOCOLS.map((id) => {
    const descriptor = resolveGatewayProtocolDescriptor(id)
    return {
      value: id,
      label: descriptor?.displayName || id,
      requestFormatsText: (descriptor?.requestFormats || []).join(', '),
      iconPlatform: descriptor?.targetGroupPlatform || 'protocol_gateway'
    }
  })
)

function isGatewayProtocolOption(option: unknown): option is GatewayProtocolOption {
  return typeof option === 'object' && option !== null && 'value' in option && 'label' in option
}
const openAIWSModeOptions = computed(() => [
  { value: OPENAI_WS_MODE_OFF, label: t('admin.accounts.openai.wsModeOff') },
  { value: OPENAI_WS_MODE_CTX_POOL, label: t('admin.accounts.openai.wsModeCtxPool') },
  { value: OPENAI_WS_MODE_PASSTHROUGH, label: t('admin.accounts.openai.wsModePassthrough') }
])
const openaiResponsesWebSocketV2Mode = computed({
  get: () => {
    if (effectivePlatform.value === 'openai' && props.account?.type === 'apikey') {
      return openaiAPIKeyResponsesWebSocketV2Mode.value
    }
    return openaiOAuthResponsesWebSocketV2Mode.value
  },
  set: (mode: OpenAIWSMode) => {
    if (effectivePlatform.value === 'openai' && props.account?.type === 'apikey') {
      openaiAPIKeyResponsesWebSocketV2Mode.value = mode
      return
    }
    openaiOAuthResponsesWebSocketV2Mode.value = mode
  }
})
const openAIWSModeConcurrencyHintKey = computed(() =>
  resolveOpenAIWSModeConcurrencyHintKey(openaiResponsesWebSocketV2Mode.value)
)
const isOpenAIModelRestrictionDisabled = computed(() =>
  effectivePlatform.value === 'openai' && openaiPassthroughEnabled.value
)

// Computed: current preset mappings based on platform
const presetMappings = computed(() => getPresetMappingsByPlatform(effectivePlatform.value))

// Computed: default base URL based on platform
const defaultBaseUrl = computed(() => {
  return resolveAccountApiKeyDefaultBaseUrl(props.account?.platform || 'anthropic', gatewayProtocol.value)
})
const commonErrorCodeOptions = computed(() => createCommonErrorCodeOptions(t))

function buildBaiduDocumentAICredentialsForUpdate(): Record<string, unknown> | null {
  const currentCredentials = (props.account?.credentials as Record<string, unknown> | undefined) || {}
  let directAPIURLs: Record<string, string> = {}
  try {
    directAPIURLs = parseBaiduDocumentAIDirectApiUrlsInput(
      baiduDocumentAIDirectApiUrlsText.value
    )
  } catch {
    appStore.showError(t('admin.accounts.baiduDocumentAI.directApiUrlsInvalid'))
    return null
  }

  const newCredentials: Record<string, unknown> = {
    ...currentCredentials,
    async_base_url:
      baiduDocumentAIAsyncBaseUrl.value.trim() ||
      String(currentCredentials.async_base_url || '').trim() ||
      BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL
  }
  delete newCredentials.base_url
  delete newCredentials.api_key
  delete newCredentials.tier_id
  delete newCredentials.model_mapping
  delete newCredentials.model_whitelist

  if (baiduDocumentAIAsyncBearerToken.value.trim()) {
    newCredentials.async_bearer_token = baiduDocumentAIAsyncBearerToken.value.trim()
  }
  if (baiduDocumentAIDirectToken.value.trim()) {
    newCredentials.direct_token = baiduDocumentAIDirectToken.value.trim()
  }
  if (Object.keys(directAPIURLs).length > 0) {
    newCredentials.direct_api_urls = directAPIURLs
  } else {
    delete newCredentials.direct_api_urls
  }

  const asyncBearerToken = String(newCredentials.async_bearer_token || '').trim()
  const directToken = String(newCredentials.direct_token || '').trim()
  if (!asyncBearerToken && !directToken) {
    appStore.showError(t('admin.accounts.baiduDocumentAI.tokenRequired'))
    return null
  }

  return newCredentials
}

function loadModelScopeFromExtra(extra?: Record<string, unknown>): boolean {
  const raw = extra?.model_scope_v2
  if (!raw || typeof raw !== 'object') {
    return false
  }
  const scope = raw as Record<string, unknown>
  const rawManualRows = scope.manual_mapping_rows
  if (Array.isArray(rawManualRows)) {
    const entries = rawManualRows
      .map((item) => {
        const row = item as Record<string, unknown>
        return {
          from: String(row?.from || '').trim(),
          to: String(row?.to || '').trim()
        }
      })
      .filter((row) => row.from.length > 0 && row.to.length > 0)
    if (entries.length > 0) {
      modelRestrictionMode.value = 'mapping'
      const selectedModels = [...new Set(entries.map(({ to }) => to))]
      allowedModels.value = selectedModels
      modelMappings.value = entries.filter(({ from, to }) => from !== to)
      if (isProtocolGatewayAccount.value) {
        protocolGatewayProbeModels.value = createStaticProbeModels(selectedModels)
      }
      return true
    }
  }

  const rawManualMappings = scope.manual_mappings
  if (rawManualMappings && typeof rawManualMappings === 'object') {
    const entries = Object.entries(rawManualMappings as Record<string, unknown>)
      .map(([from, to]) => ({ from: String(from || '').trim(), to: String(to || '').trim() }))
      .filter((row) => row.from.length > 0 && row.to.length > 0)
    if (entries.length > 0) {
      modelRestrictionMode.value = 'mapping'
      const selectedModels = [...new Set(entries.map(({ to }) => to))]
      allowedModels.value = selectedModels
      modelMappings.value = entries.filter(({ from, to }) => from !== to)
      if (isProtocolGatewayAccount.value) {
        protocolGatewayProbeModels.value = createStaticProbeModels(selectedModels)
      }
      return true
    }
  }

  const rawModelsByProvider = scope.supported_models_by_provider
  if (rawModelsByProvider && typeof rawModelsByProvider === 'object') {
    const values: string[] = []
    for (const models of Object.values(rawModelsByProvider as Record<string, unknown>)) {
      if (!Array.isArray(models)) continue
      values.push(...models.map((v) => String(v || '').trim()).filter((v) => v.length > 0))
    }
    const unique = [...new Set(values)].sort()
    if (unique.length > 0) {
      if (isProtocolGatewayAccount.value) {
        modelRestrictionMode.value = 'mapping'
        allowedModels.value = unique
        modelMappings.value = []
        protocolGatewayProbeModels.value = createStaticProbeModels(unique)
      } else {
        modelRestrictionMode.value = 'whitelist'
        allowedModels.value = unique
        modelMappings.value = []
      }
      return true
    }
  }

  return false
}

function applyModelRestrictionFromRecord(value: unknown) {
  const entries = Object.entries(value && typeof value === 'object' ? value as Record<string, unknown> : {})
    .map(([from, to]) => ({ from: String(from || '').trim(), to: String(to || '').trim() }))
    .filter((row) => row.from.length > 0 && row.to.length > 0)

  if (entries.length === 0) {
    modelRestrictionMode.value = 'whitelist'
    allowedModels.value = []
    modelMappings.value = []
    if (isProtocolGatewayAccount.value) {
      protocolGatewayProbeModels.value = []
    }
    return
  }

  if (isProtocolGatewayAccount.value) {
    modelRestrictionMode.value = 'mapping'
    const selectedModels = [...new Set(entries.map(({ to }) => to))]
    modelMappings.value = entries.filter(({ from, to }) => from !== to)
    allowedModels.value = selectedModels
    protocolGatewayProbeModels.value = createStaticProbeModels(selectedModels)
    return
  }

  const isWhitelistMode = entries.every(({ from, to }) => from === to)
  if (isWhitelistMode) {
    modelRestrictionMode.value = 'whitelist'
    allowedModels.value = entries.map(({ from }) => from)
    modelMappings.value = []
    return
  }

  modelRestrictionMode.value = 'mapping'
  allowedModels.value = [...new Set(entries.map(({ to }) => to))]
  modelMappings.value = entries.filter(({ from, to }) => from !== to)
}

function applyDefaultGrokCapabilityMapping() {
  modelRestrictionMode.value = 'mapping'
  modelMappings.value = mappingRecordToRows(grokDefaultModelMappingForTier(editGrokTier.value))
  allowedModels.value = []
}

function createStaticProbeModels(modelIds: string[]): ProtocolGatewayProbeModel[] {
  return modelIds.map((modelId) => ({
    id: modelId,
    display_name: formatModelDisplayName(modelId) || modelId,
    registry_state: 'existing',
    registry_model_id: modelId
  }))
}


const form = reactive({
  name: '',
  notes: '',
  proxy_id: null as number | null,
  concurrency: 1,
  load_factor: null as number | null,
  priority: 1,
  rate_multiplier: 1,
  status: 'active' as 'active' | 'inactive' | 'error',
  group_ids: [] as number[],
  expires_at: null as number | null
})

const {
  enabled: tempUnschedEnabled,
  rules: tempUnschedRules,
  presets: tempUnschedPresets,
  getRuleKey: getTempUnschedRuleKey,
  addRule: addTempUnschedRule,
  removeRule: removeTempUnschedRule,
  moveRule: moveTempUnschedRule,
  applyToCredentials: applyTempUnschedConfig,
  loadFromCredentials: loadTempUnschedRules,
  reset: resetTempUnschedRules
} = useAccountTempUnschedRules({
  keyPrefix: 'edit',
  invalidMessage: () => t('admin.accounts.tempUnschedulable.rulesInvalid'),
  showError: showFormError,
  t: (key) => t(key)
})

const {
  showWarning: showMixedChannelWarning,
  warningMessageText: mixedChannelWarningMessageText,
  openDialog: openMixedChannelDialog,
  withConfirmFlag,
  ensureConfirmed: ensureMixedChannelConfirmed,
  handleConfirm: handleMixedChannelConfirm,
  handleCancel: handleMixedChannelCancel,
  reset: resetMixedChannelRisk,
  requiresCheck: requiresMixedChannelCheck
} = useAccountMixedChannelRisk({
  currentPlatform: () => effectivePlatform.value,
  buildCheckPayload: () => {
    if (!props.account) {
      return null
    }
    return {
      platform: props.account.platform,
      gateway_protocol: isProtocolGatewayPlatform(props.account.platform) ? gatewayProtocol.value : undefined,
      group_ids: form.group_ids,
      account_id: props.account.id
    }
  },
  buildWarningText: (details) => t('admin.accounts.mixedChannelWarning', { ...details }),
  fallbackMessage: () => t('admin.accounts.failedToUpdate'),
  showError: showFormError
})

const statusOptions = computed(() => {
  const options = [
    { value: 'active', label: t('common.active') },
    { value: 'inactive', label: t('common.inactive') }
  ]
  if (form.status === 'error') {
    options.push({ value: 'error', label: t('admin.accounts.status.error') })
  }
  return options
})

const expiresAtInput = computed({
  get: () => formatDateTimeLocal(form.expires_at),
  set: (value: string) => {
    form.expires_at = parseDateTimeLocal(value)
  }
})

const resetProtocolGatewayClaudeMimicState = () => {
  claudeCodeMimicEnabled.value = false
  claudeTLSFingerprintEnabled.value = false
  claudeSessionIDMaskingEnabled.value = false
}

// Watchers
watch(
  () => [props.show, props.account] as const,
  ([show, newAccount]) => {
    if (show && newAccount) {
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

      // Load mixed scheduling setting (only for antigravity accounts)
      const extra = newAccount.extra as Record<string, unknown> | undefined
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
      openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      codexCLIOnlyEnabled.value = false
      anthropicPassthroughEnabled.value = false
      if (runtimePlatform === 'openai' && (newAccount.type === 'oauth' || newAccount.type === 'apikey')) {
        openaiPassthroughEnabled.value = extra?.openai_passthrough === true || extra?.openai_oauth_passthrough === true
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
        if (newAccount.platform === 'baidu_document_ai') {
          baiduDocumentAIAsyncBearerToken.value = ''
          baiduDocumentAIAsyncBaseUrl.value =
            String(credentials.async_base_url || '').trim() ||
            BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL
          baiduDocumentAIDirectToken.value = ''
          baiduDocumentAIDirectApiUrlsText.value = stringifyBaiduDocumentAIDirectApiUrls(
            credentials.direct_api_urls
          )
          modelRestrictionMode.value = 'whitelist'
          modelMappings.value = []
          allowedModels.value = []
          protocolGatewayProbeModels.value = []
          resetAccountPoolModeState(poolModeState, DEFAULT_POOL_MODE_RETRY_COUNT)
          resetAccountCustomErrorCodesState(customErrorCodesState)
          editBaseUrl.value = ''
        } else {
          const platformDefaultUrl = resolveAccountApiKeyDefaultBaseUrl(newAccount.platform, gatewayProtocol.value)
          editBaseUrl.value = (credentials.base_url as string) || platformDefaultUrl

          const loadedFromScope = loadModelScopeFromExtra(extra)
          if (!loadedFromScope) {
            applyModelRestrictionFromRecord(credentials.model_mapping)
          }

          loadAccountPoolModeStateFromCredentials(poolModeState, credentials, DEFAULT_POOL_MODE_RETRY_COUNT)
          loadAccountCustomErrorCodesStateFromCredentials(customErrorCodesState, credentials)
        }
      } else if (newAccount.type === 'sso' && newAccount.platform === 'grok' && newAccount.credentials) {
        const credentials = newAccount.credentials as Record<string, unknown>
        editBaseUrl.value = resolveAccountApiKeyDefaultBaseUrl(newAccount.platform, gatewayProtocol.value)
        applyModelRestrictionFromRecord(
          credentials.model_mapping || grokDefaultModelMappingForTier(editGrokTier.value)
        )
        resetAccountPoolModeState(poolModeState, DEFAULT_POOL_MODE_RETRY_COUNT)
        resetAccountCustomErrorCodesState(customErrorCodesState)
      } else if (newAccount.type === 'upstream' && newAccount.credentials) {
        const credentials = newAccount.credentials as Record<string, unknown>
        editBaseUrl.value = (credentials.base_url as string) || ''
        const loadedFromScope = loadModelScopeFromExtra(extra)
        if (!loadedFromScope) {
          applyModelRestrictionFromRecord(credentials.model_mapping)
        }
        resetAccountPoolModeState(poolModeState, DEFAULT_POOL_MODE_RETRY_COUNT)
        resetAccountCustomErrorCodesState(customErrorCodesState)
      } else {
        const platformDefaultUrl = resolveAccountApiKeyDefaultBaseUrl(newAccount.platform, gatewayProtocol.value)
        editBaseUrl.value = platformDefaultUrl

        const loadedFromScope = loadModelScopeFromExtra(extra)

        // Backward-compatible: some legacy OpenAI OAuth accounts may store model mappings in credentials.
        if (!loadedFromScope && runtimePlatform === 'openai' && newAccount.credentials) {
          const oauthCredentials = newAccount.credentials as Record<string, unknown>
          applyModelRestrictionFromRecord(oauthCredentials.model_mapping)
        } else if (!loadedFromScope) {
          modelRestrictionMode.value = 'whitelist'
          modelMappings.value = []
          allowedModels.value = []
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
    } else {
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
      gatewayBatchEnabled.value = false
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
      geminiOAuthType.value = 'code_assist'
      geminiVertexAuthMode.value = 'service_account'
      geminiVertexProjectId.value = ''
      geminiVertexLocation.value = ''
      geminiVertexServiceAccountJson.value = ''
      geminiVertexApiKey.value = ''
      geminiVertexAccessToken.value = ''
      geminiVertexExpiresAtInput.value = ''
      geminiVertexBaseUrl.value = ''
      baiduDocumentAIAsyncBearerToken.value = ''
      baiduDocumentAIAsyncBaseUrl.value = BAIDU_DOCUMENT_AI_DEFAULT_ASYNC_BASE_URL
      baiduDocumentAIDirectToken.value = ''
      baiduDocumentAIDirectApiUrlsText.value = ''
    }
  },
  { immediate: true }
)

watch(
  gatewayProtocol,
  (newProtocol, oldProtocol) => {
    if (!isProtocolGatewayAccount.value || newProtocol === oldProtocol) {
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
  (supported) => {
    if (!supported) {
      gatewayOpenAIRequestFormat.value = DEFAULT_GATEWAY_OPENAI_REQUEST_FORMAT
    }
  }
)

watch(
  showProtocolGatewayBatchEditor,
  (supported) => {
    if (!supported) {
      gatewayBatchEnabled.value = false
    }
  }
)

watch(
  [showProtocolGatewayClaudeMimicEditor, claudeCodeMimicEnabled],
  ([supported, enabled]) => {
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

// Model mapping helpers
const addModelMapping = () => {
  modelMappings.value.push({ from: '', to: '' })
}

const removeModelMapping = (index: number) => {
  modelMappings.value.splice(index, 1)
}

const addPresetMapping = (from: string, to: string) => {
  const exists = modelMappings.value.some((m) => m.from === from)
  if (exists) {
    appStore.showInfo(t('admin.accounts.mappingExists', { model: from }))
    return
  }
  modelMappings.value.push({ from, to })
}

const addAntigravityModelMapping = () => {
  antigravityModelMappings.value.push({ from: '', to: '' })
}

const removeAntigravityModelMapping = (index: number) => {
  antigravityModelMappings.value.splice(index, 1)
}

const addAntigravityPresetMapping = (from: string, to: string) => {
  const exists = antigravityModelMappings.value.some((m) => m.from === from)
  if (exists) {
    appStore.showInfo(t('admin.accounts.mappingExists', { model: from }))
    return
  }
  antigravityModelMappings.value.push({ from, to })
}

const formatDateTimeLocal = formatDateTimeLocalInput
const parseDateTimeLocal = parseDateTimeLocalInput

// Methods
const handleClose = () => {
  resetMixedChannelRisk()
  emit('close')
}

const submitUpdateAccount = async (accountID: number, updatePayload: Record<string, unknown>) => {
  submitting.value = true
  try {
    const updatedAccount = await adminAPI.accounts.update(accountID, withConfirmFlag(updatePayload))
    appStore.showSuccess(t('admin.accounts.accountUpdated'))
    emit('updated', updatedAccount)
    handleClose()
  } catch (error: any) {
    if (
      error.status === 409 &&
      error.error === 'mixed_channel_warning' &&
      requiresMixedChannelCheck.value
    ) {
      openMixedChannelDialog({
        message: error.message,
        onConfirm: async () => submitUpdateAccount(accountID, updatePayload)
      })
      return
    }
    appStore.showError(error.message || t('admin.accounts.failedToUpdate'))
  } finally {
    submitting.value = false
  }
}

const handleSubmit = async () => {
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

    if (isGeminiVertexAccount.value) {
      const currentCredentials = (props.account.credentials as Record<string, unknown>) || {}
      const newCredentials: Record<string, unknown> = { ...currentCredentials }
      const modelMapping = buildModelMappingObject('mapping', [], modelMappings.value)
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
    } else if (props.account.platform === 'baidu_document_ai' && props.account.type === 'apikey') {
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

      if (editApiKey.value.trim()) {
        newCredentials.api_key = editApiKey.value.trim()
      } else if (currentCredentials.api_key) {
        newCredentials.api_key = currentCredentials.api_key
      } else {
        appStore.showError(t('admin.accounts.apiKeyIsRequired'))
        return
      }

      if (shouldApplyModelMapping) {
        const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
        if (modelMapping) {
          newCredentials.model_mapping = modelMapping
        } else {
          delete newCredentials.model_mapping
        }
      } else if (currentCredentials.model_mapping) {
        newCredentials.model_mapping = currentCredentials.model_mapping
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

      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    } else {
      const currentCredentials = (props.account.credentials as Record<string, unknown>) || {}
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

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

      if (shouldApplyModelMapping) {
        const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
        if (modelMapping) {
          newCredentials.model_mapping = modelMapping
        } else {
          delete newCredentials.model_mapping
        }
      } else if (currentCredentials.model_mapping) {
        newCredentials.model_mapping = currentCredentials.model_mapping
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
      const modelMapping = buildModelMappingObject(modelRestrictionMode.value, allowedModels.value, modelMappings.value)
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
        applyProtocolGatewayOpenAIRequestFormatExtra(
          applyProtocolGatewayClaudeClientMimicExtra({
            ...currentExtra,
            gateway_protocol: gatewayProtocol.value,
            gateway_accepted_protocols: [...gatewayAcceptedProtocols.value],
            gateway_client_profiles: [...gatewayClientProfiles.value],
            gateway_client_routes: gatewayClientRoutes.value.map((route) => ({ ...route })),
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
      if (runtimePlatform !== 'openai') {
        delete normalizedExtra.openai_passthrough
        delete normalizedExtra.openai_oauth_passthrough
        delete normalizedExtra.codex_cli_only
        delete normalizedExtra.openai_oauth_responses_websockets_v2_mode
        delete normalizedExtra.openai_apikey_responses_websockets_v2_mode
        delete normalizedExtra.openai_oauth_responses_websockets_v2_enabled
        delete normalizedExtra.openai_apikey_responses_websockets_v2_enabled
        delete normalizedExtra.responses_websockets_v2_enabled
        delete normalizedExtra.openai_ws_enabled
      }
      if (runtimePlatform !== 'anthropic') {
        delete normalizedExtra.anthropic_passthrough
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
      runtimePlatform === 'baidu_document_ai'
        ? (((updatePayload.extra as Record<string, unknown>) ||
            (props.account.extra as Record<string, unknown>) ||
            undefined))
        : buildAccountModelScopeExtra(
          ((updatePayload.extra as Record<string, unknown>) ||
            (props.account.extra as Record<string, unknown>) ||
            undefined),
          {
            platform: runtimePlatform,
            enabled: runtimePlatform === 'antigravity'
              ? true
              : !(runtimePlatform === 'openai' && openaiPassthroughEnabled.value),
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

    const canContinue = await ensureMixedChannelConfirmed(async () => {
      await submitUpdateAccount(accountID, updatePayload)
    })
    if (!canContinue) {
      return
    }

    await submitUpdateAccount(accountID, updatePayload)
  } catch (error: any) {
    appStore.showError(error.message || t('admin.accounts.failedToUpdate'))
  }
}

const probeExtraForEditor = computed(() => buildProbeExtra())

function buildProbeExtra(base?: Record<string, unknown>) {
  return mergeResolvedUpstreamDraftIntoExtra(
    mergeAccountModelProbeSnapshotIntoExtra(
      mergeAccountManualModelsIntoExtra(
        base || currentAccountExtra.value,
        manualModels.value,
        isProtocolGatewayAccount.value
      ),
      modelProbeSnapshot.value
    ),
    resolvedUpstream.value
  )
}
</script>
