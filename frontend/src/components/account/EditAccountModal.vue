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
        v-model:access-token="baiduDocumentAIAccessToken"
        v-model:async-base-url="baiduDocumentAIAsyncBaseUrl"
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
          v-model:model-scope-enabled="modelRestrictionEnabled"
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
          v-model:image-protocol-mode="gatewayOpenAIImageProtocolMode"
        />

        <AccountDeepSeekConcurrencyLimitsEditor
          v-if="showDeepSeekConcurrencyEditor"
          v-model:limits="deepSeekModelConcurrencyLimits"
        />

        <AccountOpenRouterSettingsEditor
          v-if="account.platform === 'openrouter'"
          v-model:http-referer="editOpenRouterHTTPReferer"
          v-model:openrouter-title="editOpenRouterTitle"
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

      <AccountTierSelector
        v-if="showAccountTierSelector"
        v-model:tier="accountTier"
        :platform="account.platform"
        @apply-capacity="applyAccountTierCapacity"
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
        v-model:expiry-probe-extension-days="expiryProbeExtensionDays"
        v-model:auto-renew-enabled="autoRenewEnabled"
        v-model:auto-renew-period="autoRenewPeriod"
        :proxies="proxies"
      />

      <AccountGatewaySettingsEditor
        :show-open-ai-passthrough="effectivePlatform === 'openai' && (account?.type === 'oauth' || account?.type === 'apikey')"
        :open-ai-passthrough-enabled="openaiPassthroughEnabled"
        :show-open-ai-image-protocol-mode="account?.platform === 'openai' && (account?.type === 'oauth' || account?.type === 'apikey')"
        :open-ai-image-protocol-mode="openAIImageProtocolMode"
        :open-ai-image-protocol-compat-allowed="openAIImageCompatAllowed"
        :show-open-ai-image-protocol-compat-toggle="account?.type === 'oauth'"
        :show-open-ai-ws-mode="effectivePlatform === 'openai' && (account?.type === 'oauth' || account?.type === 'apikey')"
        :open-ai-ws-mode="openaiResponsesWebSocketV2Mode"
        :open-ai-ws-mode-options="openAIWSModeOptions"
        :open-ai-ws-mode-concurrency-hint-key="openAIWSModeConcurrencyHintKey"
        :show-anthropic-passthrough="effectivePlatform === 'anthropic' && account?.type === 'apikey'"
        :anthropic-passthrough-enabled="anthropicPassthroughEnabled"
        :show-codex-cli-only="effectivePlatform === 'openai' && account?.type === 'oauth'"
        :codex-cli-only-enabled="codexCLIOnlyEnabled"
        @update:open-ai-passthrough-enabled="openaiPassthroughEnabled = $event"
        @update:open-ai-image-protocol-mode="handleOpenAIImageProtocolModeChange"
        @update:open-ai-image-protocol-compat-allowed="openAIImageCompatAllowed = $event"
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
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import PlatformLabel from '@/components/common/PlatformLabel.vue'
import AccountApiKeyBasicSettingsEditor from '@/components/account/AccountApiKeyBasicSettingsEditor.vue'
import AccountAntigravityModelMappingEditor from '@/components/account/AccountAntigravityModelMappingEditor.vue'
import AccountApiKeyModelProbeEditor from '@/components/account/AccountApiKeyModelProbeEditor.vue'
import AccountAutoPauseToggle from '@/components/account/AccountAutoPauseToggle.vue'
import AccountBaiduDocumentAICredentialsEditor from '@/components/account/AccountBaiduDocumentAICredentialsEditor.vue'
import AccountCustomErrorCodesEditor from '@/components/account/AccountCustomErrorCodesEditor.vue'
import AccountDeepSeekConcurrencyLimitsEditor from '@/components/account/AccountDeepSeekConcurrencyLimitsEditor.vue'
import AccountGatewaySettingsEditor from '@/components/account/AccountGatewaySettingsEditor.vue'
import AccountGoogleBatchArchiveEditor from '@/components/account/AccountGoogleBatchArchiveEditor.vue'
import AccountGeminiVertexCredentialsEditor from '@/components/account/AccountGeminiVertexCredentialsEditor.vue'
import AccountGroupSettingsEditor from '@/components/account/AccountGroupSettingsEditor.vue'
import AccountMixedChannelWarningDialog from '@/components/account/AccountMixedChannelWarningDialog.vue'
import AccountModelScopeEditor from '@/components/account/AccountModelScopeEditor.vue'
import AccountOpenRouterSettingsEditor from '@/components/account/AccountOpenRouterSettingsEditor.vue'
import AccountPoolModeEditor from '@/components/account/AccountPoolModeEditor.vue'
import AccountProtocolGatewayClaudeMimicEditor from '@/components/account/AccountProtocolGatewayClaudeMimicEditor.vue'
import AccountProtocolGatewayBatchEditor from '@/components/account/AccountProtocolGatewayBatchEditor.vue'
import AccountProtocolGatewayOpenAIRequestFormatEditor from '@/components/account/AccountProtocolGatewayOpenAIRequestFormatEditor.vue'
import AccountProtocolGatewayModelProbeEditor from '@/components/account/AccountProtocolGatewayModelProbeEditor.vue'
import AccountQuotaControlEditor from '@/components/account/AccountQuotaControlEditor.vue'
import AccountRuntimeSettingsEditor from '@/components/account/AccountRuntimeSettingsEditor.vue'
import AccountTempUnschedRulesEditor from '@/components/account/AccountTempUnschedRulesEditor.vue'
import AccountTierSelector from '@/components/account/AccountTierSelector.vue'
import AccountUpstreamSettingsEditor from '@/components/account/AccountUpstreamSettingsEditor.vue'
import QuotaLimitCard from '@/components/account/QuotaLimitCard.vue'
import { useEditAccountModal } from './editAccountModal/useEditAccountModal'
import type { EditAccountModalEmits, EditAccountModalProps } from './editAccountModal/types'

const props = withDefaults(defineProps<EditAccountModalProps>(), {
  loading: false
})
const emit = defineEmits<EditAccountModalEmits>()

const {
  t,
  authStore,
  showFormError,
  showFormInfo,
  antigravityPresetMappings,
  submitting,
  gatewayProtocol,
  editBaseUrl,
  editApiKey,
  editOpenRouterHTTPReferer,
  editOpenRouterTitle,
  deepSeekModelConcurrencyLimits,
  editGrokSSOToken,
  editGrokTier,
  modelMappings,
  actualModelLocked,
  modelRestrictionEnabled,
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
  mixedScheduling,
  antigravityModelMappings,
  getModelMappingKey,
  getAntigravityModelMappingKey,
  quotaControlState,
  umqModeOptions,
  openaiPassthroughEnabled,
  openAIImageProtocolMode,
  openAIImageCompatAllowed,
  codexCLIOnlyEnabled,
  anthropicPassthroughEnabled,
  gatewayOpenAIImageProtocolMode,
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
  geminiTierAIStudio,
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
  effectivePlatform,
  effectiveGroupPlatforms,
  isProtocolGatewayAccount,
  isBaiduDocumentAIAccount,
  isGrokSSOAccount,
  isGeminiVertexAccount,
  isGeminiVertexLegacyMode,
  showCommonApiKeySection,
  showDeepSeekConcurrencyEditor,
  showUnifiedProtocolGatewayProbeEditor,
  showUnifiedAPIModelProbeEditor,
  showStandaloneModelScopeEditor,
  unifiedProbeAccountType,
  unifiedProbeCredentials,
  unifiedProbeReady,
  showQuotaLimitSection,
  showAccountTierSelector,
  shouldPersistGeminiTierId,
  showGeminiAIStudioBatchArchiveEditor,
  showGeminiVertexBatchArchiveEditor,
  grokCapabilityModels,
  showProtocolGatewayClaudeMimicEditor,
  showProtocolGatewayBatchEditor,
  showProtocolGatewayOpenAIRequestFormatEditor,
  protocolGatewayBatchRequestFormats,
  resolvedProtocolGatewayApiKey,
  gatewayProtocolOptions,
  isGatewayProtocolOption,
  openAIWSModeOptions,
  openaiResponsesWebSocketV2Mode,
  openAIWSModeConcurrencyHintKey,
  isOpenAIModelRestrictionDisabled,
  presetMappings,
  commonErrorCodeOptions,
  applyDefaultGrokCapabilityMapping,
  form,
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
  statusOptions,
  accountTier,
  expiresAtInput,
  handleOpenAIImageProtocolModeChange,
  applyAccountTierCapacity,
  addModelMapping,
  removeModelMapping,
  addPresetMapping,
  addAntigravityModelMapping,
  removeAntigravityModelMapping,
  addAntigravityPresetMapping,
  handleClose,
  probeExtraForEditor,
  DEFAULT_POOL_MODE_RETRY_COUNT,
  MAX_POOL_MODE_RETRY_COUNT,
  handleSubmit
} = useEditAccountModal(props, emit)
</script>
