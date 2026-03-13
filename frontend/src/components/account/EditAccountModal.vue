<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.editAccount')"
    width="normal"
    @close="handleClose"
  >
    <form
      v-if="account"
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

      <!-- API Key fields (only for apikey type) -->
      <div v-if="account.type === 'apikey'" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.accounts.baseUrl') }}</label>
          <input
            v-model="editBaseUrl"
            type="text"
            class="input"
            :placeholder="
              account.platform === 'openai' || account.platform === 'sora'
                ? 'https://api.openai.com'
                : account.platform === 'gemini'
                  ? 'https://generativelanguage.googleapis.com'
                  : account.platform === 'antigravity'
                    ? 'https://cloudcode-pa.googleapis.com'
                    : 'https://api.anthropic.com'
            "
          />
          <p class="input-hint">{{ baseUrlHint }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.apiKey') }}</label>
          <input
            v-model="editApiKey"
            type="password"
            class="input font-mono"
            :placeholder="
              account.platform === 'openai' || account.platform === 'sora'
                ? 'sk-proj-...'
                : account.platform === 'gemini'
                  ? 'AIza...'
                  : account.platform === 'antigravity'
                    ? 'sk-...'
                    : 'sk-ant-...'
            "
          />
          <p class="input-hint">{{ t('admin.accounts.leaveEmptyToKeep') }}</p>
        </div>

        <!-- Model Restriction Section (娑撳秹鈧倻鏁ゆ禍?Antigravity) -->
        <AccountModelRestrictionEditor
          v-if="account.platform !== 'antigravity'"
          :disabled="isOpenAIModelRestrictionDisabled"
          :platform="account?.platform || 'anthropic'"
          :mode="modelRestrictionMode"
          :allowed-models="allowedModels"
          :model-mappings="modelMappings"
          :preset-mappings="presetMappings"
          :get-mapping-key="getModelMappingKey"
          @update:mode="modelRestrictionMode = $event"
          @update:allowedModels="allowedModels = $event"
          @add-mapping="addModelMapping"
          @remove-mapping="removeModelMapping"
          @add-preset="addPresetMapping($event.from, $event.to)"
        />

        <!-- Pool Mode Section -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.accounts.poolMode') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.poolModeHint') }}
              </p>
            </div>
            <button
              type="button"
              @click="poolModeEnabled = !poolModeEnabled"
              :class="[
                'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                poolModeEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  poolModeEnabled ? 'translate-x-5' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          <div v-if="poolModeEnabled" class="rounded-lg bg-blue-50 p-3 dark:bg-blue-900/20">
            <p class="text-xs text-blue-700 dark:text-blue-400">
              <Icon name="exclamationCircle" size="sm" class="mr-1 inline" :stroke-width="2" />
              {{ t('admin.accounts.poolModeInfo') }}
            </p>
          </div>
          <div v-if="poolModeEnabled" class="mt-3">
            <label class="input-label">{{ t('admin.accounts.poolModeRetryCount') }}</label>
            <input
              v-model.number="poolModeRetryCount"
              type="number"
              min="0"
              :max="MAX_POOL_MODE_RETRY_COUNT"
              step="1"
              class="input"
            />
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{
                t('admin.accounts.poolModeRetryCountHint', {
                  default: DEFAULT_POOL_MODE_RETRY_COUNT,
                  max: MAX_POOL_MODE_RETRY_COUNT
                })
              }}
            </p>
          </div>
        </div>

        <!-- Custom Error Codes Section -->
        <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.accounts.customErrorCodes') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.customErrorCodesHint') }}
              </p>
            </div>
            <button
              type="button"
              @click="customErrorCodesEnabled = !customErrorCodesEnabled"
              :class="[
                'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                customErrorCodesEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  customErrorCodesEnabled ? 'translate-x-5' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="customErrorCodesEnabled" class="space-y-3">
            <div class="rounded-lg bg-amber-50 p-3 dark:bg-amber-900/20">
              <p class="text-xs text-amber-700 dark:text-amber-400">
                <Icon name="exclamationTriangle" size="sm" class="mr-1 inline" :stroke-width="2" />
                {{ t('admin.accounts.customErrorCodesWarning') }}
              </p>
            </div>

            <!-- Error Code Buttons -->
            <div class="flex flex-wrap gap-2">
              <button
                v-for="code in commonErrorCodes"
                :key="code.value"
                type="button"
                @click="toggleErrorCode(code.value)"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm font-medium transition-colors',
                  selectedErrorCodes.includes(code.value)
                    ? 'bg-red-100 text-red-700 ring-1 ring-red-500 dark:bg-red-900/30 dark:text-red-400'
                    : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                ]"
              >
                {{ code.value }} {{ code.label }}
              </button>
            </div>

            <!-- Manual input -->
            <div class="flex items-center gap-2">
              <input
                v-model.number="customErrorCodeInput"
                type="number"
                min="100"
                max="599"
                class="input flex-1"
                :placeholder="t('admin.accounts.enterErrorCode')"
                @keyup.enter="addCustomErrorCode"
              />
              <button type="button" @click="addCustomErrorCode" class="btn btn-secondary px-3">
                <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M12 4v16m8-8H4"
                  />
                </svg>
              </button>
            </div>

            <!-- Selected codes summary -->
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="code in selectedErrorCodes.sort((a, b) => a - b)"
                :key="code"
                class="inline-flex items-center gap-1 rounded-full bg-red-100 px-2.5 py-0.5 text-sm font-medium text-red-700 dark:bg-red-900/30 dark:text-red-400"
              >
                {{ code }}
                <button
                  type="button"
                  @click="removeErrorCode(code)"
                  class="hover:text-red-900 dark:hover:text-red-300"
                >
                  <Icon name="x" size="sm" :stroke-width="2" />
                </button>
              </span>
              <span v-if="selectedErrorCodes.length === 0" class="text-xs text-gray-400">
                {{ t('admin.accounts.noneSelectedUsesDefault') }}
              </span>
            </div>
          </div>
        </div>

      </div>

      <!-- OpenAI OAuth Model Mapping (OAuth 缁鐎峰▽鈩冩箒 apikey 鐎圭懓娅掗敍宀勬付鐟曚胶瀚粩瀣畱濡€崇€烽弰鐘茬殸閸栧搫鐓? -->
      <AccountModelRestrictionEditor
        v-if="account.platform === 'openai' && account.type === 'oauth'"
        :disabled="isOpenAIModelRestrictionDisabled"
        :platform="account?.platform || 'anthropic'"
        :mode="modelRestrictionMode"
        :allowed-models="allowedModels"
        :model-mappings="modelMappings"
        :preset-mappings="presetMappings"
        :get-mapping-key="getModelMappingKey"
        variant="simple"
        @update:mode="modelRestrictionMode = $event"
        @update:allowedModels="allowedModels = $event"
        @add-mapping="addModelMapping"
        @remove-mapping="removeModelMapping"
        @add-preset="addPresetMapping($event.from, $event.to)"
      />

      <!-- Upstream fields (only for upstream type) -->
      <div v-if="account.type === 'upstream'" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.accounts.upstream.baseUrl') }}</label>
          <input
            v-model="editBaseUrl"
            type="text"
            class="input"
            placeholder="https://cloudcode-pa.googleapis.com"
          />
          <p class="input-hint">{{ t('admin.accounts.upstream.baseUrlHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.upstream.apiKey') }}</label>
          <input
            v-model="editApiKey"
            type="password"
            class="input font-mono"
            placeholder="sk-..."
          />
          <p class="input-hint">{{ t('admin.accounts.leaveEmptyToKeep') }}</p>
        </div>
      </div>

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
        v-if="account?.platform === 'anthropic' || account?.platform === 'antigravity'"
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

      <div>
        <label class="input-label">{{ t('admin.accounts.proxy') }}</label>
        <ProxySelector v-model="form.proxy_id" :proxies="proxies" />
      </div>

      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <div>
          <label class="input-label">{{ t('admin.accounts.concurrency') }}</label>
          <input v-model.number="form.concurrency" type="number" min="1" class="input"
            @input="form.concurrency = Math.max(1, form.concurrency || 1)" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.loadFactor') }}</label>
          <input v-model.number="form.load_factor" type="number" min="1"
            class="input" :placeholder="String(form.concurrency || 1)"
            @input="form.load_factor = (form.load_factor &amp;&amp; form.load_factor >= 1) ? form.load_factor : null" />
          <p class="input-hint">{{ t('admin.accounts.loadFactorHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.priority') }}</label>
          <input
            v-model.number="form.priority"
            type="number"
            min="1"
            class="input"
            data-tour="account-form-priority"
          />
          <p class="input-hint">{{ t('admin.accounts.priorityHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.billingRateMultiplier') }}</label>
          <input v-model.number="form.rate_multiplier" type="number" min="0" step="0.001" class="input" />
          <p class="input-hint">{{ t('admin.accounts.billingRateMultiplierHint') }}</p>
        </div>
      </div>
      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <label class="input-label">{{ t('admin.accounts.expiresAt') }}</label>
        <input v-model="expiresAtInput" type="datetime-local" class="input" />
        <p class="input-hint">{{ t('admin.accounts.expiresAtHint') }}</p>
      </div>

      <!-- OpenAI 閼奉亜濮╅柅蹇庣炊瀵偓閸忕绱橭Auth/API Key閿?-->
      <div
        v-if="account?.platform === 'openai' && (account?.type === 'oauth' || account?.type === 'apikey')"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.accounts.openai.oauthPassthrough') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.openai.oauthPassthroughDesc') }}
            </p>
          </div>
          <button
            type="button"
            @click="openaiPassthroughEnabled = !openaiPassthroughEnabled"
            :class="[
              'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
              openaiPassthroughEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
            ]"
          >
            <span
              :class="[
                'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                openaiPassthroughEnabled ? 'translate-x-5' : 'translate-x-0'
              ]"
            />
          </button>
        </div>
      </div>

      <!-- OpenAI WS Mode 娑撳鈧緤绱檕ff/ctx_pool/passthrough閿?-->
      <div
        v-if="account?.platform === 'openai' && (account?.type === 'oauth' || account?.type === 'apikey')"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.accounts.openai.wsMode') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.openai.wsModeDesc') }}
            </p>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t(openAIWSModeConcurrencyHintKey) }}
            </p>
          </div>
          <div class="w-52">
            <Select v-model="openaiResponsesWebSocketV2Mode" :options="openAIWSModeOptions" />
          </div>
        </div>
      </div>

      <!-- Anthropic API Key 閼奉亜濮╅柅蹇庣炊瀵偓閸?-->
      <div
        v-if="account?.platform === 'anthropic' && account?.type === 'apikey'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.accounts.anthropic.apiKeyPassthrough') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.anthropic.apiKeyPassthroughDesc') }}
            </p>
          </div>
          <button
            type="button"
            @click="anthropicPassthroughEnabled = !anthropicPassthroughEnabled"
            :class="[
              'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
              anthropicPassthroughEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
            ]"
          >
            <span
              :class="[
                'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                anthropicPassthroughEnabled ? 'translate-x-5' : 'translate-x-0'
              ]"
            />
          </button>
        </div>
      </div>

      <!-- API Key 鐠愶箑褰块柊宥夘杺闂勬劕鍩?-->
      <div v-if="account?.type === 'apikey'" class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4">
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
          @update:totalLimit="editQuotaLimit = $event"
          @update:dailyLimit="editQuotaDailyLimit = $event"
          @update:weeklyLimit="editQuotaWeeklyLimit = $event"
        />
      </div>

      <!-- OpenAI OAuth Codex 鐎规ɑ鏌熺€广垺鍩涚粩顖炴閸掕泛绱戦崗?-->
      <div
        v-if="account?.platform === 'openai' && account?.type === 'oauth'"
        class="border-t border-gray-200 pt-4 dark:border-dark-600"
      >
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{ t('admin.accounts.openai.codexCLIOnly') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.openai.codexCLIOnlyDesc') }}
            </p>
          </div>
          <button
            type="button"
            @click="codexCLIOnlyEnabled = !codexCLIOnlyEnabled"
            :class="[
              'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
              codexCLIOnlyEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
            ]"
          >
            <span
              :class="[
                'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                codexCLIOnlyEnabled ? 'translate-x-5' : 'translate-x-0'
              ]"
            />
          </button>
        </div>
      </div>

      <div>
        <div class="flex items-center justify-between">
          <div>
            <label class="input-label mb-0">{{
              t('admin.accounts.autoPauseOnExpired')
            }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.autoPauseOnExpiredDesc') }}
            </p>
          </div>
          <button
            type="button"
            @click="autoPauseOnExpired = !autoPauseOnExpired"
            :class="[
              'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
              autoPauseOnExpired ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
            ]"
          >
            <span
              :class="[
                'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                autoPauseOnExpired ? 'translate-x-5' : 'translate-x-0'
              ]"
            />
          </button>
        </div>
      </div>

      <!-- Quota Control Section (Anthropic OAuth/SetupToken only) -->
      <div
        v-if="account?.platform === 'anthropic' && (account?.type === 'oauth' || account?.type === 'setup-token')"
        class="border-t border-gray-200 pt-4 dark:border-dark-600 space-y-4"
      >
        <div class="mb-3">
          <h3 class="input-label mb-0 text-base font-semibold">{{ t('admin.accounts.quotaControl.title') }}</h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.quotaControl.hint') }}
          </p>
        </div>

        <!-- Window Cost Limit -->
        <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.accounts.quotaControl.windowCost.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.quotaControl.windowCost.hint') }}
              </p>
            </div>
            <button
              type="button"
              @click="windowCostEnabled = !windowCostEnabled"
              :class="[
                'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                windowCostEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  windowCostEnabled ? 'translate-x-5' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="windowCostEnabled" class="grid grid-cols-2 gap-4">
            <div>
              <label class="input-label">{{ t('admin.accounts.quotaControl.windowCost.limit') }}</label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 dark:text-gray-400">$</span>
                <input
                  v-model.number="windowCostLimit"
                  type="number"
                  min="0"
                  step="1"
                  class="input pl-7"
                  :placeholder="t('admin.accounts.quotaControl.windowCost.limitPlaceholder')"
                />
              </div>
              <p class="input-hint">{{ t('admin.accounts.quotaControl.windowCost.limitHint') }}</p>
            </div>
            <div>
              <label class="input-label">{{ t('admin.accounts.quotaControl.windowCost.stickyReserve') }}</label>
              <div class="relative">
                <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 dark:text-gray-400">$</span>
                <input
                  v-model.number="windowCostStickyReserve"
                  type="number"
                  min="0"
                  step="1"
                  class="input pl-7"
                  :placeholder="t('admin.accounts.quotaControl.windowCost.stickyReservePlaceholder')"
                />
              </div>
              <p class="input-hint">{{ t('admin.accounts.quotaControl.windowCost.stickyReserveHint') }}</p>
            </div>
          </div>
        </div>

        <!-- Session Limit -->
        <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.accounts.quotaControl.sessionLimit.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.quotaControl.sessionLimit.hint') }}
              </p>
            </div>
            <button
              type="button"
              @click="sessionLimitEnabled = !sessionLimitEnabled"
              :class="[
                'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                sessionLimitEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  sessionLimitEnabled ? 'translate-x-5' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="sessionLimitEnabled" class="grid grid-cols-2 gap-4">
            <div>
              <label class="input-label">{{ t('admin.accounts.quotaControl.sessionLimit.maxSessions') }}</label>
              <input
                v-model.number="maxSessions"
                type="number"
                min="1"
                step="1"
                class="input"
                :placeholder="t('admin.accounts.quotaControl.sessionLimit.maxSessionsPlaceholder')"
              />
              <p class="input-hint">{{ t('admin.accounts.quotaControl.sessionLimit.maxSessionsHint') }}</p>
            </div>
            <div>
              <label class="input-label">{{ t('admin.accounts.quotaControl.sessionLimit.idleTimeout') }}</label>
              <div class="relative">
                <input
                  v-model.number="sessionIdleTimeout"
                  type="number"
                  min="1"
                  step="1"
                  class="input pr-12"
                  :placeholder="t('admin.accounts.quotaControl.sessionLimit.idleTimeoutPlaceholder')"
                />
                <span class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 dark:text-gray-400">{{ t('common.minutes') }}</span>
              </div>
              <p class="input-hint">{{ t('admin.accounts.quotaControl.sessionLimit.idleTimeoutHint') }}</p>
            </div>
          </div>
        </div>

        <!-- RPM Limit -->
        <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
          <div class="mb-3 flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.accounts.quotaControl.rpmLimit.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.quotaControl.rpmLimit.hint') }}
              </p>
            </div>
            <button
              type="button"
              @click="rpmLimitEnabled = !rpmLimitEnabled"
              :class="[
                'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                rpmLimitEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  rpmLimitEnabled ? 'translate-x-5' : 'translate-x-0'
                ]"
              />
            </button>
          </div>

          <div v-if="rpmLimitEnabled" class="space-y-4">
            <div>
              <label class="input-label">{{ t('admin.accounts.quotaControl.rpmLimit.baseRpm') }}</label>
              <input
                v-model.number="baseRpm"
                type="number"
                min="1"
                max="1000"
                step="1"
                class="input"
                :placeholder="t('admin.accounts.quotaControl.rpmLimit.baseRpmPlaceholder')"
              />
              <p class="input-hint">{{ t('admin.accounts.quotaControl.rpmLimit.baseRpmHint') }}</p>
            </div>

            <div>
              <label class="input-label">{{ t('admin.accounts.quotaControl.rpmLimit.strategy') }}</label>
              <div class="flex gap-2">
                <button
                  type="button"
                  @click="rpmStrategy = 'tiered'"
                  :class="[
                    'flex-1 rounded-lg px-3 py-2 text-sm font-medium transition-all',
                    rpmStrategy === 'tiered'
                      ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                      : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                  ]"
                >
                  <div class="text-center">
                    <div>{{ t('admin.accounts.quotaControl.rpmLimit.strategyTiered') }}</div>
                    <div class="mt-0.5 text-[10px] opacity-70">{{ t('admin.accounts.quotaControl.rpmLimit.strategyTieredHint') }}</div>
                  </div>
                </button>
                <button
                  type="button"
                  @click="rpmStrategy = 'sticky_exempt'"
                  :class="[
                    'flex-1 rounded-lg px-3 py-2 text-sm font-medium transition-all',
                    rpmStrategy === 'sticky_exempt'
                      ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                      : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
                  ]"
                >
                  <div class="text-center">
                    <div>{{ t('admin.accounts.quotaControl.rpmLimit.strategyStickyExempt') }}</div>
                    <div class="mt-0.5 text-[10px] opacity-70">{{ t('admin.accounts.quotaControl.rpmLimit.strategyStickyExemptHint') }}</div>
                  </div>
                </button>
              </div>
            </div>

            <div v-if="rpmStrategy === 'tiered'">
              <label class="input-label">{{ t('admin.accounts.quotaControl.rpmLimit.stickyBuffer') }}</label>
              <input
                v-model.number="rpmStickyBuffer"
                type="number"
                min="1"
                step="1"
                class="input"
                :placeholder="t('admin.accounts.quotaControl.rpmLimit.stickyBufferPlaceholder')"
              />
              <p class="input-hint">{{ t('admin.accounts.quotaControl.rpmLimit.stickyBufferHint') }}</p>
            </div>

          </div>

          <!-- 閻劍鍩涘☉鍫熶紖闂勬劙鈧喐膩瀵骏绱欓悪顒傜彌娴?RPM 瀵偓閸忕绱濇慨瀣矒閸欘垵顫嗛敍?-->
          <div class="mt-4">
            <label class="input-label">{{ t('admin.accounts.quotaControl.rpmLimit.userMsgQueue') }}</label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400 mb-2">
              {{ t('admin.accounts.quotaControl.rpmLimit.userMsgQueueHint') }}
            </p>
            <div class="flex space-x-2">
              <button type="button" v-for="opt in umqModeOptions" :key="opt.value"
                @click="userMsgQueueMode = opt.value"
                :class="[
                  'px-3 py-1.5 text-sm rounded-md border transition-colors',
                  userMsgQueueMode === opt.value
                    ? 'bg-primary-600 text-white border-primary-600'
                    : 'bg-white dark:bg-dark-700 text-gray-700 dark:text-gray-300 border-gray-300 dark:border-dark-500 hover:bg-gray-50 dark:hover:bg-dark-600'
                ]">
                {{ opt.label }}
              </button>
            </div>
          </div>
        </div>

        <!-- TLS Fingerprint -->
        <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
          <div class="flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.accounts.quotaControl.tlsFingerprint.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.quotaControl.tlsFingerprint.hint') }}
              </p>
            </div>
            <button
              type="button"
              @click="tlsFingerprintEnabled = !tlsFingerprintEnabled"
              :class="[
                'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                tlsFingerprintEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  tlsFingerprintEnabled ? 'translate-x-5' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
        </div>

        <!-- Session ID Masking -->
        <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
          <div class="flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.accounts.quotaControl.sessionIdMasking.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.quotaControl.sessionIdMasking.hint') }}
              </p>
            </div>
            <button
              type="button"
              @click="sessionIdMaskingEnabled = !sessionIdMaskingEnabled"
              :class="[
                'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                sessionIdMaskingEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  sessionIdMaskingEnabled ? 'translate-x-5' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
        </div>

        <!-- Cache TTL Override -->
        <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
          <div class="flex items-center justify-between">
            <div>
              <label class="input-label mb-0">{{ t('admin.accounts.quotaControl.cacheTTLOverride.label') }}</label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.quotaControl.cacheTTLOverride.hint') }}
              </p>
            </div>
            <button
              type="button"
              @click="cacheTTLOverrideEnabled = !cacheTTLOverrideEnabled"
              :class="[
                'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                cacheTTLOverrideEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
              ]"
            >
              <span
                :class="[
                  'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                  cacheTTLOverrideEnabled ? 'translate-x-5' : 'translate-x-0'
                ]"
              />
            </button>
          </div>
          <div v-if="cacheTTLOverrideEnabled" class="mt-3">
            <label class="input-label text-xs">{{ t('admin.accounts.quotaControl.cacheTTLOverride.target') }}</label>
            <select
              v-model="cacheTTLOverrideTarget"
              class="mt-1 block w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500 dark:border-dark-500 dark:bg-dark-700 dark:text-white"
            >
              <option value="5m">5m</option>
              <option value="1h">1h</option>
            </select>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.quotaControl.cacheTTLOverride.targetHint') }}
            </p>
          </div>
        </div>
      </div>

      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div>
          <label class="input-label">{{ t('common.status') }}</label>
          <Select v-model="form.status" :options="statusOptions" />
        </div>

        <!-- Mixed Scheduling (only for antigravity accounts, read-only in edit mode) -->
        <div v-if="account?.platform === 'antigravity'" class="flex items-center gap-2">
          <label class="flex cursor-not-allowed items-center gap-2 opacity-60">
            <input
              type="checkbox"
              v-model="mixedScheduling"
              disabled
              class="h-4 w-4 cursor-not-allowed rounded border-gray-300 text-primary-500 focus:ring-primary-500 dark:border-dark-500"
            />
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.mixedScheduling') }}
            </span>
          </label>
          <div class="group relative">
            <span
              class="inline-flex h-4 w-4 cursor-help items-center justify-center rounded-full bg-gray-200 text-xs text-gray-500 hover:bg-gray-300 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500"
            >
              ?
            </span>
            <!-- Tooltip閿涘牆鎮滄稉瀣▔缁€娲缉閸忓秷顫﹀鍦崶鐟佷礁澹€閿?-->
            <div
              class="pointer-events-none absolute left-0 top-full z-[100] mt-1.5 w-72 rounded bg-gray-900 px-3 py-2 text-xs text-white opacity-0 transition-opacity group-hover:opacity-100 dark:bg-gray-700"
            >
              {{ t('admin.accounts.mixedSchedulingTooltip') }}
              <div
                class="absolute bottom-full left-3 border-4 border-transparent border-b-gray-900 dark:border-b-gray-700"
              ></div>
            </div>
          </div>
        </div>
      </div>

      <!-- Group Selection - 娴犲懏鐖ｉ崙鍡樐佸蹇旀▔缁€?-->
      <GroupSelector
        v-if="!authStore.isSimpleMode"
        v-model="form.group_ids"
        :groups="groups"
        :platform="account?.platform"
        :mixed-scheduling="mixedScheduling"
        data-tour="account-form-groups"
      />

    </form>

    <template #footer>
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
import { useAccountMixedChannelRisk } from '@/composables/useAccountMixedChannelRisk'
import { useAccountTempUnschedRules } from '@/composables/useAccountTempUnschedRules'
import type { Account, Proxy, AdminGroup } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import ProxySelector from '@/components/common/ProxySelector.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import AccountAntigravityModelMappingEditor from '@/components/account/AccountAntigravityModelMappingEditor.vue'
import AccountMixedChannelWarningDialog from '@/components/account/AccountMixedChannelWarningDialog.vue'
import AccountModelRestrictionEditor from '@/components/account/AccountModelRestrictionEditor.vue'
import AccountTempUnschedRulesEditor from '@/components/account/AccountTempUnschedRulesEditor.vue'
import QuotaLimitCard from '@/components/account/QuotaLimitCard.vue'
import { applyInterceptWarmup } from '@/components/account/credentialsBuilder'
import { formatDateTimeLocalInput, parseDateTimeLocalInput } from '@/utils/format'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import {
  DEFAULT_POOL_MODE_RETRY_COUNT,
  MAX_POOL_MODE_RETRY_COUNT,
  normalizePoolModeRetryCount,
  type ModelMapping
} from '@/utils/accountFormShared'
import {
  // OPENAI_WS_MODE_CTX_POOL,
  OPENAI_WS_MODE_OFF,
  OPENAI_WS_MODE_PASSTHROUGH,
  isOpenAIWSModeEnabled,
  resolveOpenAIWSModeConcurrencyHintKey,
  type OpenAIWSMode,
  resolveOpenAIWSModeFromExtra
} from '@/utils/openaiWsMode'
import {
  getPresetMappingsByPlatform,
  commonErrorCodes,
  buildModelMappingObject
} from '@/composables/useModelWhitelist'

interface Props {
  show: boolean
  account: Account | null
  proxies: Proxy[]
  groups: AdminGroup[]
}

const props = defineProps<Props>()
const emit = defineEmits<{
  close: []
  updated: [account: Account]
}>()

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

// Platform-specific hint for Base URL
const baseUrlHint = computed(() => {
  if (!props.account) return t('admin.accounts.baseUrlHint')
  if (props.account.platform === 'openai') return t('admin.accounts.openai.baseUrlHint')
  if (props.account.platform === 'gemini') return t('admin.accounts.gemini.baseUrlHint')
  return t('admin.accounts.baseUrlHint')
})

const antigravityPresetMappings = computed(() => getPresetMappingsByPlatform('antigravity'))

// State
const submitting = ref(false)
const editBaseUrl = ref('https://api.anthropic.com')
const editApiKey = ref('')
const modelMappings = ref<ModelMapping[]>([])
const modelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const allowedModels = ref<string[]>([])
const poolModeEnabled = ref(false)
const poolModeRetryCount = ref(DEFAULT_POOL_MODE_RETRY_COUNT)
const customErrorCodesEnabled = ref(false)
const selectedErrorCodes = ref<number[]>([])
const customErrorCodeInput = ref<number | null>(null)
const interceptWarmupRequests = ref(false)
const autoPauseOnExpired = ref(false)
const mixedScheduling = ref(false) // For antigravity accounts: enable mixed scheduling
const antigravityModelRestrictionMode = ref<'whitelist' | 'mapping'>('whitelist')
const antigravityWhitelistModels = ref<string[]>([])
const antigravityModelMappings = ref<ModelMapping[]>([])
const getModelMappingKey = createStableObjectKeyResolver<ModelMapping>('edit-model-mapping')
const getAntigravityModelMappingKey = createStableObjectKeyResolver<ModelMapping>('edit-antigravity-model-mapping')

// Quota control state (Anthropic OAuth/SetupToken only)
const windowCostEnabled = ref(false)
const windowCostLimit = ref<number | null>(null)
const windowCostStickyReserve = ref<number | null>(null)
const sessionLimitEnabled = ref(false)
const maxSessions = ref<number | null>(null)
const sessionIdleTimeout = ref<number | null>(null)
const rpmLimitEnabled = ref(false)
const baseRpm = ref<number | null>(null)
const rpmStrategy = ref<'tiered' | 'sticky_exempt'>('tiered')
const rpmStickyBuffer = ref<number | null>(null)
const userMsgQueueMode = ref('')
const umqModeOptions = computed(() => [
  { value: '', label: t('admin.accounts.quotaControl.rpmLimit.umqModeOff') },
  { value: 'throttle', label: t('admin.accounts.quotaControl.rpmLimit.umqModeThrottle') },
  { value: 'serialize', label: t('admin.accounts.quotaControl.rpmLimit.umqModeSerialize') },
])
const tlsFingerprintEnabled = ref(false)
const sessionIdMaskingEnabled = ref(false)
const cacheTTLOverrideEnabled = ref(false)
const cacheTTLOverrideTarget = ref<string>('5m')

// OpenAI 閼奉亜濮╅柅蹇庣炊瀵偓閸忕绱橭Auth/API Key閿?
const openaiPassthroughEnabled = ref(false)
const openaiOAuthResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const openaiAPIKeyResponsesWebSocketV2Mode = ref<OpenAIWSMode>(OPENAI_WS_MODE_OFF)
const codexCLIOnlyEnabled = ref(false)
const anthropicPassthroughEnabled = ref(false)
const editQuotaLimit = ref<number | null>(null)
const editQuotaDailyLimit = ref<number | null>(null)
const editQuotaWeeklyLimit = ref<number | null>(null)
const openAIWSModeOptions = computed(() => [
  { value: OPENAI_WS_MODE_OFF, label: t('admin.accounts.openai.wsModeOff') },
  // TODO: ctx_pool 闁銆嶉弳鍌涙闂呮劘妫岄敍灞界窡濞村鐦€瑰本鍨氶崥搴划婢?
  // { value: OPENAI_WS_MODE_CTX_POOL, label: t('admin.accounts.openai.wsModeCtxPool') },
  { value: OPENAI_WS_MODE_PASSTHROUGH, label: t('admin.accounts.openai.wsModePassthrough') }
])
const openaiResponsesWebSocketV2Mode = computed({
  get: () => {
    if (props.account?.type === 'apikey') {
      return openaiAPIKeyResponsesWebSocketV2Mode.value
    }
    return openaiOAuthResponsesWebSocketV2Mode.value
  },
  set: (mode: OpenAIWSMode) => {
    if (props.account?.type === 'apikey') {
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
  props.account?.platform === 'openai' && openaiPassthroughEnabled.value
)

// Computed: current preset mappings based on platform
const presetMappings = computed(() => getPresetMappingsByPlatform(props.account?.platform || 'anthropic'))

// Computed: default base URL based on platform
const defaultBaseUrl = computed(() => {
  if (props.account?.platform === 'openai' || props.account?.platform === 'sora') return 'https://api.openai.com'
  if (props.account?.platform === 'gemini') return 'https://generativelanguage.googleapis.com'
  return 'https://api.anthropic.com'
})


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
  showError: (message) => appStore.showError(message),
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
  currentPlatform: () => props.account?.platform ?? null,
  buildCheckPayload: () => {
    if (!props.account) {
      return null
    }
    return {
      platform: props.account.platform,
      group_ids: form.group_ids,
      account_id: props.account.id
    }
  },
  buildWarningText: (details) => t('admin.accounts.mixedChannelWarning', { ...details }),
  fallbackMessage: () => t('admin.accounts.failedToUpdate'),
  showError: (message) => appStore.showError(message)
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

// Watchers
watch(
  () => props.account,
  (newAccount) => {
    if (newAccount) {
      resetMixedChannelRisk()
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
      mixedScheduling.value = extra?.mixed_scheduling === true

      // Load OpenAI passthrough toggle (OpenAI OAuth/API Key)
      openaiPassthroughEnabled.value = false
      openaiOAuthResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      openaiAPIKeyResponsesWebSocketV2Mode.value = OPENAI_WS_MODE_OFF
      codexCLIOnlyEnabled.value = false
      anthropicPassthroughEnabled.value = false
      if (newAccount.platform === 'openai' && (newAccount.type === 'oauth' || newAccount.type === 'apikey')) {
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
      if (newAccount.platform === 'anthropic' && newAccount.type === 'apikey') {
        anthropicPassthroughEnabled.value = extra?.anthropic_passthrough === true
      }

      // Load quota limit for apikey accounts
      if (newAccount.type === 'apikey') {
        const quotaVal = extra?.quota_limit as number | undefined
        editQuotaLimit.value = (quotaVal && quotaVal > 0) ? quotaVal : null
        const dailyVal = extra?.quota_daily_limit as number | undefined
        editQuotaDailyLimit.value = (dailyVal && dailyVal > 0) ? dailyVal : null
        const weeklyVal = extra?.quota_weekly_limit as number | undefined
        editQuotaWeeklyLimit.value = (weeklyVal && weeklyVal > 0) ? weeklyVal : null
      } else {
        editQuotaLimit.value = null
        editQuotaDailyLimit.value = null
        editQuotaWeeklyLimit.value = null
      }

      // Load antigravity model mapping (Antigravity 閸欘亝鏁幐浣规Ё鐏忓嫭膩瀵?
      if (newAccount.platform === 'antigravity') {
        const credentials = newAccount.credentials as Record<string, unknown> | undefined

        // Antigravity 婵绮撴担璺ㄦ暏閺勭姴鐨犲Ο鈥崇础
        antigravityModelRestrictionMode.value = 'mapping'
        antigravityWhitelistModels.value = []

        // 娴?model_mapping 鐠囪褰囬弰鐘茬殸闁板秶鐤?
        const rawAgMapping = credentials?.model_mapping as Record<string, string> | undefined
        if (rawAgMapping && typeof rawAgMapping === 'object') {
          const entries = Object.entries(rawAgMapping)
          // 閺冪姾顔戦弰顖滄閸氬秴宕熼弽宄扮础(key===value)鏉╂ɑ妲搁惇鐔割劀閻ㄥ嫭妲х亸鍕剁礉闁晫绮烘稉鈧潪顒佸床娑撶儤妲х亸鍕灙鐞?
          antigravityModelMappings.value = entries.map(([from, to]) => ({ from, to }))
        } else {
          // 閸忕厧顔愰弮褎鏆熼幑顕嗙窗娴?model_whitelist 鐠囪褰囬敍宀冩祮閹诡澀璐熼弰鐘茬殸閺嶇厧绱?
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

      // Load quota control settings (Anthropic OAuth/SetupToken only)
      loadQuotaControlSettings(newAccount)

      loadTempUnschedRules(credentials)

      // Initialize API Key fields for apikey type
      if (newAccount.type === 'apikey' && newAccount.credentials) {
        const credentials = newAccount.credentials as Record<string, unknown>
        const platformDefaultUrl =
          newAccount.platform === 'openai' || newAccount.platform === 'sora'
            ? 'https://api.openai.com'
            : newAccount.platform === 'gemini'
              ? 'https://generativelanguage.googleapis.com'
              : 'https://api.anthropic.com'
        editBaseUrl.value = (credentials.base_url as string) || platformDefaultUrl

        // Load model mappings and detect mode
        const existingMappings = credentials.model_mapping as Record<string, string> | undefined
        if (existingMappings && typeof existingMappings === 'object') {
          const entries = Object.entries(existingMappings)

          // Detect if this is whitelist mode (all from === to) or mapping mode
          const isWhitelistMode = entries.length > 0 && entries.every(([from, to]) => from === to)

          if (isWhitelistMode) {
            // Whitelist mode: populate allowedModels
            modelRestrictionMode.value = 'whitelist'
            allowedModels.value = entries.map(([from]) => from)
            modelMappings.value = []
          } else {
            // Mapping mode: populate modelMappings
            modelRestrictionMode.value = 'mapping'
            modelMappings.value = entries.map(([from, to]) => ({ from, to }))
            allowedModels.value = []
          }
        } else {
          // No mappings: default to whitelist mode with empty selection (allow all)
          modelRestrictionMode.value = 'whitelist'
          modelMappings.value = []
          allowedModels.value = []
        }

        // Load pool mode
        poolModeEnabled.value = credentials.pool_mode === true
        poolModeRetryCount.value = normalizePoolModeRetryCount(
          Number(credentials.pool_mode_retry_count ?? DEFAULT_POOL_MODE_RETRY_COUNT)
        )

        // Load custom error codes
        customErrorCodesEnabled.value = credentials.custom_error_codes_enabled === true
        const existingErrorCodes = credentials.custom_error_codes as number[] | undefined
        if (existingErrorCodes && Array.isArray(existingErrorCodes)) {
          selectedErrorCodes.value = [...existingErrorCodes]
        } else {
          selectedErrorCodes.value = []
        }
      } else if (newAccount.type === 'upstream' && newAccount.credentials) {
        const credentials = newAccount.credentials as Record<string, unknown>
        editBaseUrl.value = (credentials.base_url as string) || ''
      } else {
        const platformDefaultUrl =
          newAccount.platform === 'openai' || newAccount.platform === 'sora'
            ? 'https://api.openai.com'
            : newAccount.platform === 'gemini'
              ? 'https://generativelanguage.googleapis.com'
              : 'https://api.anthropic.com'
        editBaseUrl.value = platformDefaultUrl

        // Load model mappings for OpenAI OAuth accounts
        if (newAccount.platform === 'openai' && newAccount.credentials) {
          const oauthCredentials = newAccount.credentials as Record<string, unknown>
          const existingMappings = oauthCredentials.model_mapping as Record<string, string> | undefined
          if (existingMappings && typeof existingMappings === 'object') {
            const entries = Object.entries(existingMappings)
            const isWhitelistMode = entries.length > 0 && entries.every(([from, to]) => from === to)
            if (isWhitelistMode) {
              modelRestrictionMode.value = 'whitelist'
              allowedModels.value = entries.map(([from]) => from)
              modelMappings.value = []
            } else {
              modelRestrictionMode.value = 'mapping'
              modelMappings.value = entries.map(([from, to]) => ({ from, to }))
              allowedModels.value = []
            }
          } else {
            modelRestrictionMode.value = 'whitelist'
            modelMappings.value = []
            allowedModels.value = []
          }
        } else {
          modelRestrictionMode.value = 'whitelist'
          modelMappings.value = []
          allowedModels.value = []
        }
        poolModeEnabled.value = false
        poolModeRetryCount.value = DEFAULT_POOL_MODE_RETRY_COUNT
        customErrorCodesEnabled.value = false
        selectedErrorCodes.value = []
      }
      editApiKey.value = ''
    } else {
      resetMixedChannelRisk()
      resetTempUnschedRules()
    }
  },
  { immediate: true }
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

// Error code toggle helper
const toggleErrorCode = (code: number) => {
  const index = selectedErrorCodes.value.indexOf(code)
  if (index === -1) {
    // Adding code - check for 429/529 warning
    if (code === 429) {
      if (!confirm(t('admin.accounts.customErrorCodes429Warning'))) {
        return
      }
    } else if (code === 529) {
      if (!confirm(t('admin.accounts.customErrorCodes529Warning'))) {
        return
      }
    }
    selectedErrorCodes.value.push(code)
  } else {
    selectedErrorCodes.value.splice(index, 1)
  }
}

// Add custom error code from input
const addCustomErrorCode = () => {
  const code = customErrorCodeInput.value
  if (code === null || code < 100 || code > 599) {
    appStore.showError(t('admin.accounts.invalidErrorCode'))
    return
  }
  if (selectedErrorCodes.value.includes(code)) {
    appStore.showInfo(t('admin.accounts.errorCodeExists'))
    return
  }
  // Check for 429/529 warning
  if (code === 429) {
    if (!confirm(t('admin.accounts.customErrorCodes429Warning'))) {
      return
    }
  } else if (code === 529) {
    if (!confirm(t('admin.accounts.customErrorCodes529Warning'))) {
      return
    }
  }
  selectedErrorCodes.value.push(code)
  customErrorCodeInput.value = null
}

// Remove error code
const removeErrorCode = (code: number) => {
  const index = selectedErrorCodes.value.indexOf(code)
  if (index !== -1) {
    selectedErrorCodes.value.splice(index, 1)
  }
}

// Load quota control settings from account (Anthropic OAuth/SetupToken only)
function loadQuotaControlSettings(account: Account) {
  // Reset all quota control state first
  windowCostEnabled.value = false
  windowCostLimit.value = null
  windowCostStickyReserve.value = null
  sessionLimitEnabled.value = false
  maxSessions.value = null
  sessionIdleTimeout.value = null
  rpmLimitEnabled.value = false
  baseRpm.value = null
  rpmStrategy.value = 'tiered'
  rpmStickyBuffer.value = null
  userMsgQueueMode.value = ''
  tlsFingerprintEnabled.value = false
  sessionIdMaskingEnabled.value = false
  cacheTTLOverrideEnabled.value = false
  cacheTTLOverrideTarget.value = '5m'

  // Only applies to Anthropic OAuth/SetupToken accounts
  if (account.platform !== 'anthropic' || (account.type !== 'oauth' && account.type !== 'setup-token')) {
    return
  }

  // Load from extra field (via backend DTO fields)
  if (account.window_cost_limit != null && account.window_cost_limit > 0) {
    windowCostEnabled.value = true
    windowCostLimit.value = account.window_cost_limit
    windowCostStickyReserve.value = account.window_cost_sticky_reserve ?? 10
  }

  if (account.max_sessions != null && account.max_sessions > 0) {
    sessionLimitEnabled.value = true
    maxSessions.value = account.max_sessions
    sessionIdleTimeout.value = account.session_idle_timeout_minutes ?? 5
  }

  // RPM limit
  if (account.base_rpm != null && account.base_rpm > 0) {
    rpmLimitEnabled.value = true
    baseRpm.value = account.base_rpm
    rpmStrategy.value = (account.rpm_strategy as 'tiered' | 'sticky_exempt') || 'tiered'
    rpmStickyBuffer.value = account.rpm_sticky_buffer ?? null
  }

  // UMQ mode閿涘牏瀚粩瀣╃艾 RPM 閸旂姾娴囬敍宀勬Щ濮濄垻绱潏鎴炴￥ RPM 鐠愶箑褰块弮鏈垫丢婢跺崬鍑￠張澶愬帳缂冾噯绱?
  userMsgQueueMode.value = account.user_msg_queue_mode ?? ''

  // Load TLS fingerprint setting
  if (account.enable_tls_fingerprint === true) {
    tlsFingerprintEnabled.value = true
  }

  // Load session ID masking setting
  if (account.session_id_masking_enabled === true) {
    sessionIdMaskingEnabled.value = true
  }

  // Load cache TTL override setting
  if (account.cache_ttl_override_enabled === true) {
    cacheTTLOverrideEnabled.value = true
    cacheTTLOverrideTarget.value = account.cache_ttl_override_target || '5m'
  }
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

  if (form.status !== 'active' && form.status !== 'inactive' && form.status !== 'error') {
    appStore.showError(t('admin.accounts.pleaseSelectStatus'))
    return
  }

  const updatePayload: Record<string, unknown> = { ...form }
  try {
    // 閸氬海顏張鐔告箿 proxy_id: 0 鐞涖劎銇氬〒鍛存珟娴狅絿鎮婇敍宀冣偓灞肩瑝閺?null
    if (updatePayload.proxy_id === null) {
      updatePayload.proxy_id = 0
    }
    if (form.expires_at === null) {
      updatePayload.expires_at = 0
    }
    // load_factor: 缁屽搫鈧?NaN/0/鐠愮喐鏆?閺冭泛褰傞柅?0閿涘牆鎮楃粩顖滃鐎?<= 0 = 濞撳懘娅庨敍?
    const lf = form.load_factor
    if (lf == null || Number.isNaN(lf) || lf <= 0) {
      updatePayload.load_factor = 0
    }
    updatePayload.auto_pause_on_expired = autoPauseOnExpired.value

    // For apikey type, handle credentials update
    if (props.account.type === 'apikey') {
      const currentCredentials = (props.account.credentials as Record<string, unknown>) || {}
      const newBaseUrl = editBaseUrl.value.trim() || defaultBaseUrl.value
      const shouldApplyModelMapping = !(props.account.platform === 'openai' && openaiPassthroughEnabled.value)

      // Always update credentials for apikey type to handle model mapping changes
      const newCredentials: Record<string, unknown> = {
        ...currentCredentials,
        base_url: newBaseUrl
      }

      // Handle API key
      if (editApiKey.value.trim()) {
        // User provided a new API key
        newCredentials.api_key = editApiKey.value.trim()
      } else if (currentCredentials.api_key) {
        // Preserve existing api_key
        newCredentials.api_key = currentCredentials.api_key
      } else {
        appStore.showError(t('admin.accounts.apiKeyIsRequired'))
        return
      }

      // Add model mapping if configured閿涘湦penAI 瀵偓閸氼垵鍤滈崝銊┾偓蹇庣炊閺冩湹绻氶悾娆戝箛閺堝妲х亸鍕剁礉娑撳秴鍟€缂傛牞绶敍?
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

      // Add pool mode if enabled
      if (poolModeEnabled.value) {
        newCredentials.pool_mode = true
        newCredentials.pool_mode_retry_count = normalizePoolModeRetryCount(poolModeRetryCount.value)
      } else {
        delete newCredentials.pool_mode
        delete newCredentials.pool_mode_retry_count
      }

      // Add custom error codes if enabled
      if (customErrorCodesEnabled.value) {
        newCredentials.custom_error_codes_enabled = true
        newCredentials.custom_error_codes = [...selectedErrorCodes.value]
      } else {
        delete newCredentials.custom_error_codes_enabled
        delete newCredentials.custom_error_codes
      }

      // Add intercept warmup requests setting
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

      // Add intercept warmup requests setting
      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')

      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    } else {
      // For oauth/setup-token types, only update intercept_warmup_requests if changed
      const currentCredentials = (props.account.credentials as Record<string, unknown>) || {}
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      applyInterceptWarmup(newCredentials, interceptWarmupRequests.value, 'edit')
      if (!applyTempUnschedConfig(newCredentials)) {
        return
      }

      updatePayload.credentials = newCredentials
    }

    // OpenAI OAuth: persist model mapping to credentials
    if (props.account.platform === 'openai' && props.account.type === 'oauth') {
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
        // 闁繋绱跺Ο鈥崇础娣囨繄鏆€閻滅増婀侀弰鐘茬殸
        newCredentials.model_mapping = currentCredentials.model_mapping
      }

      updatePayload.credentials = newCredentials
    }

    // Antigravity: persist model mapping to credentials (applies to all antigravity types)
    // Antigravity 閸欘亝鏁幐浣规Ё鐏忓嫭膩瀵?
    if (props.account.platform === 'antigravity') {
      const currentCredentials = (updatePayload.credentials as Record<string, unknown>) ||
        ((props.account.credentials as Record<string, unknown>) || {})
      const newCredentials: Record<string, unknown> = { ...currentCredentials }

      // 缁夊娅庨弮褍鐡у▓?
      delete newCredentials.model_whitelist
      delete newCredentials.model_mapping

      // 閸欘亙濞囬悽銊︽Ё鐏忓嫭膩瀵?
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

    // For antigravity accounts, handle mixed_scheduling in extra
    if (props.account.platform === 'antigravity') {
      const currentExtra = (props.account.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      if (mixedScheduling.value) {
        newExtra.mixed_scheduling = true
      } else {
        delete newExtra.mixed_scheduling
      }
      updatePayload.extra = newExtra
    }

    // For Anthropic OAuth/SetupToken accounts, handle quota control settings in extra
    if (props.account.platform === 'anthropic' && (props.account.type === 'oauth' || props.account.type === 'setup-token')) {
      const currentExtra = (props.account.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }

      // Window cost limit settings
      if (windowCostEnabled.value && windowCostLimit.value != null && windowCostLimit.value > 0) {
        newExtra.window_cost_limit = windowCostLimit.value
        newExtra.window_cost_sticky_reserve = windowCostStickyReserve.value ?? 10
      } else {
        delete newExtra.window_cost_limit
        delete newExtra.window_cost_sticky_reserve
      }

      // Session limit settings
      if (sessionLimitEnabled.value && maxSessions.value != null && maxSessions.value > 0) {
        newExtra.max_sessions = maxSessions.value
        newExtra.session_idle_timeout_minutes = sessionIdleTimeout.value ?? 5
      } else {
        delete newExtra.max_sessions
        delete newExtra.session_idle_timeout_minutes
      }

      // RPM limit settings
      if (rpmLimitEnabled.value) {
        const DEFAULT_BASE_RPM = 15
        newExtra.base_rpm = (baseRpm.value != null && baseRpm.value > 0)
          ? baseRpm.value
          : DEFAULT_BASE_RPM
        newExtra.rpm_strategy = rpmStrategy.value
        if (rpmStickyBuffer.value != null && rpmStickyBuffer.value > 0) {
          newExtra.rpm_sticky_buffer = rpmStickyBuffer.value
        } else {
          delete newExtra.rpm_sticky_buffer
        }
      } else {
        delete newExtra.base_rpm
        delete newExtra.rpm_strategy
        delete newExtra.rpm_sticky_buffer
      }

      // UMQ mode閿涘牏瀚粩瀣╃艾 RPM 娣囨繂鐡ㄩ敍?
      if (userMsgQueueMode.value) {
        newExtra.user_msg_queue_mode = userMsgQueueMode.value
      } else {
        delete newExtra.user_msg_queue_mode
      }
      delete newExtra.user_msg_queue_enabled  // 濞撳懐鎮婇弮褍鐡у▓?

      // TLS fingerprint setting
      if (tlsFingerprintEnabled.value) {
        newExtra.enable_tls_fingerprint = true
      } else {
        delete newExtra.enable_tls_fingerprint
      }

      // Session ID masking setting
      if (sessionIdMaskingEnabled.value) {
        newExtra.session_id_masking_enabled = true
      } else {
        delete newExtra.session_id_masking_enabled
      }

      // Cache TTL override setting
      if (cacheTTLOverrideEnabled.value) {
        newExtra.cache_ttl_override_enabled = true
        newExtra.cache_ttl_override_target = cacheTTLOverrideTarget.value
      } else {
        delete newExtra.cache_ttl_override_enabled
        delete newExtra.cache_ttl_override_target
      }

      updatePayload.extra = newExtra
    }

    // For Anthropic API Key accounts, handle passthrough mode in extra
    if (props.account.platform === 'anthropic' && props.account.type === 'apikey') {
      const currentExtra = (props.account.extra as Record<string, unknown>) || {}
      const newExtra: Record<string, unknown> = { ...currentExtra }
      if (anthropicPassthroughEnabled.value) {
        newExtra.anthropic_passthrough = true
      } else {
        delete newExtra.anthropic_passthrough
      }
      updatePayload.extra = newExtra
    }

    // For OpenAI OAuth/API Key accounts, handle passthrough mode in extra
    if (props.account.platform === 'openai' && (props.account.type === 'oauth' || props.account.type === 'apikey')) {
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
          // 閸忔娊妫撮弮鑸垫▔瀵繐鍟?false閿涘矂浼╅崗?extra 娑撹櫣鈹栫悮顐㈡倵缁旑垰鎷烽悾銉ヮ嚤閼峰瓨妫崐鍏兼￥濞夋洘绔婚梽?
          newExtra.codex_cli_only = false
        } else {
          delete newExtra.codex_cli_only
        }
      }

      updatePayload.extra = newExtra
    }

    // For apikey accounts, handle quota_limit in extra
    if (props.account.type === 'apikey') {
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
      updatePayload.extra = newExtra
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
</script>
