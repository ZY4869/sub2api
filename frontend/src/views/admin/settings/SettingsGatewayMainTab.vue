<template>
        <div class="space-y-6">

        <!-- Overload Cooldown (529) Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.overloadCooldown.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.overloadCooldown.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <div v-if="overloadCooldownLoading" class="flex items-center gap-2 text-gray-500">
              <div class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"></div>
              {{ t('common.loading') }}
            </div>

            <template v-else>
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t('admin.settings.overloadCooldown.enabled')
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.overloadCooldown.enabledHint') }}
                  </p>
                </div>
                <Toggle v-model="overloadCooldownForm.enabled" />
              </div>

              <div
                v-if="overloadCooldownForm.enabled"
                class="space-y-4 border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.overloadCooldown.cooldownMinutes') }}
                  </label>
                  <input
                    v-model.number="overloadCooldownForm.cooldown_minutes"
                    type="number"
                    min="1"
                    max="120"
                    class="input w-32"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.overloadCooldown.cooldownMinutesHint') }}
                  </p>
                </div>
              </div>

              <div class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700">
                <button
                  type="button"
                  @click="saveOverloadCooldownSettings"
                  :disabled="overloadCooldownSaving"
                  class="btn btn-primary btn-sm"
                >
                  <svg
                    v-if="overloadCooldownSaving"
                    class="mr-1 h-4 w-4 animate-spin"
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
                  {{ overloadCooldownSaving ? t('common.saving') : t('common.save') }}
                </button>
              </div>
            </template>
          </div>
        </div>

        <!-- Stream Timeout Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.streamTimeout.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.streamTimeout.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <!-- Loading State -->
            <div v-if="streamTimeoutLoading" class="flex items-center gap-2 text-gray-500">
              <div class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"></div>
              {{ t('common.loading') }}
            </div>

            <template v-else>
              <!-- Enable Stream Timeout -->
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t('admin.settings.streamTimeout.enabled')
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.streamTimeout.enabledHint') }}
                  </p>
                </div>
                <Toggle v-model="streamTimeoutForm.enabled" />
              </div>

              <!-- Settings - Only show when enabled -->
              <div
                v-if="streamTimeoutForm.enabled"
                class="space-y-4 border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <!-- Action -->
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.streamTimeout.action') }}
                  </label>
                  <select v-model="streamTimeoutForm.action" class="input w-64">
                    <option value="temp_unsched">{{ t('admin.settings.streamTimeout.actionTempUnsched') }}</option>
                    <option value="error">{{ t('admin.settings.streamTimeout.actionError') }}</option>
                    <option value="none">{{ t('admin.settings.streamTimeout.actionNone') }}</option>
                  </select>
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.streamTimeout.actionHint') }}
                  </p>
                </div>

                <!-- Temp Unsched Minutes (only show when action is temp_unsched) -->
                <div v-if="streamTimeoutForm.action === 'temp_unsched'">
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.streamTimeout.tempUnschedMinutes') }}
                  </label>
                  <input
                    v-model.number="streamTimeoutForm.temp_unsched_minutes"
                    type="number"
                    min="1"
                    max="60"
                    class="input w-32"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.streamTimeout.tempUnschedMinutesHint') }}
                  </p>
                </div>

                <!-- Threshold Count -->
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.streamTimeout.thresholdCount') }}
                  </label>
                  <input
                    v-model.number="streamTimeoutForm.threshold_count"
                    type="number"
                    min="1"
                    max="10"
                    class="input w-32"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.streamTimeout.thresholdCountHint') }}
                  </p>
                </div>

                <!-- Threshold Window Minutes -->
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.streamTimeout.thresholdWindowMinutes') }}
                  </label>
                  <input
                    v-model.number="streamTimeoutForm.threshold_window_minutes"
                    type="number"
                    min="1"
                    max="60"
                    class="input w-32"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.streamTimeout.thresholdWindowMinutesHint') }}
                  </p>
                </div>
              </div>

              <!-- Save Button -->
              <div class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700">
                <button
                  type="button"
                  @click="saveStreamTimeoutSettings"
                  :disabled="streamTimeoutSaving"
                  class="btn btn-primary btn-sm"
                >
                  <svg
                    v-if="streamTimeoutSaving"
                    class="mr-1 h-4 w-4 animate-spin"
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
                  {{ streamTimeoutSaving ? t('common.saving') : t('common.save') }}
                </button>
              </div>
            </template>
          </div>
        </div>

        <!-- Request Rectifier Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.rectifier.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.rectifier.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <!-- Loading State -->
            <div v-if="rectifierLoading" class="flex items-center gap-2 text-gray-500">
              <div class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"></div>
              {{ t('common.loading') }}
            </div>

            <template v-else>
              <!-- Master Toggle -->
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t('admin.settings.rectifier.enabled')
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.rectifier.enabledHint') }}
                  </p>
                </div>
                <Toggle v-model="rectifierForm.enabled" />
              </div>

              <!-- Sub-toggles (only show when master is enabled) -->
              <div
                v-if="rectifierForm.enabled"
                class="space-y-4 border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <!-- Thinking Signature Rectifier -->
                <div class="flex items-center justify-between">
                  <div>
                    <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{{
                      t('admin.settings.rectifier.thinkingSignature')
                    }}</label>
                    <p class="text-xs text-gray-500 dark:text-gray-400">
                      {{ t('admin.settings.rectifier.thinkingSignatureHint') }}
                    </p>
                  </div>
                  <Toggle v-model="rectifierForm.thinking_signature_enabled" />
                </div>

                <!-- Thinking Budget Rectifier -->
                <div class="flex items-center justify-between">
                  <div>
                    <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{{
                      t('admin.settings.rectifier.thinkingBudget')
                    }}</label>
                    <p class="text-xs text-gray-500 dark:text-gray-400">
                      {{ t('admin.settings.rectifier.thinkingBudgetHint') }}
                    </p>
                  </div>
                  <Toggle v-model="rectifierForm.thinking_budget_enabled" />
                </div>
              </div>

              <!-- Save Button -->
              <div class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700">
                <button
                  type="button"
                  @click="saveRectifierSettings"
                  :disabled="rectifierSaving"
                  class="btn btn-primary btn-sm"
                >
                  <svg
                    v-if="rectifierSaving"
                    class="mr-1 h-4 w-4 animate-spin"
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
                  {{ rectifierSaving ? t('common.saving') : t('common.save') }}
                </button>
              </div>
            </template>
          </div>
        </div>

        <GoogleBatchArchiveSettingsCard />

        <GoogleBatchGCSProfilesManager />

        <OpenAIFastPolicySettingsCard
          v-model="form.openai_fast_policy_settings"
          v-model:enable-injection="form.enable_anthropic_cache_ttl_1h_injection"
        />

        <!-- Beta Policy Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.betaPolicy.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.betaPolicy.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <!-- Loading State -->
            <div v-if="betaPolicyLoading" class="flex items-center gap-2 text-gray-500">
              <div class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"></div>
              {{ t('common.loading') }}
            </div>

            <template v-else>
              <!-- Rule Cards -->
              <div
                v-for="rule in betaPolicyForm.rules"
                :key="rule.beta_token"
                class="rounded-lg border border-gray-200 p-4 dark:border-dark-600"
              >
                <div class="mb-3 flex items-center gap-2">
                  <span class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ getBetaDisplayName(rule.beta_token) }}
                  </span>
                  <span class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-400">
                    {{ rule.beta_token }}
                  </span>
                </div>

                <div class="grid grid-cols-2 gap-4">
                  <!-- Action -->
                  <div>
                    <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                      {{ t('admin.settings.betaPolicy.action') }}
                    </label>
                    <Select
                      :modelValue="rule.action"
                      @update:modelValue="rule.action = $event as any"
                      :options="betaPolicyActionOptions"
                    />
                  </div>

                  <!-- Scope -->
                  <div>
                    <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                      {{ t('admin.settings.betaPolicy.scope') }}
                    </label>
                    <Select
                      :modelValue="rule.scope"
                      @update:modelValue="rule.scope = $event as any"
                      :options="betaPolicyScopeOptions"
                    />
                  </div>
                </div>

                <!-- Error Message (only when action=block) -->
                <div v-if="rule.action === 'block'" class="mt-3">
                  <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                    {{ t('admin.settings.betaPolicy.errorMessage') }}
                  </label>
                  <input
                    v-model="rule.error_message"
                    type="text"
                    class="input"
                    :placeholder="t('admin.settings.betaPolicy.errorMessagePlaceholder')"
                  />
                  <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">
                    {{ t('admin.settings.betaPolicy.errorMessageHint') }}
                  </p>
                </div>
              </div>

              <!-- Save Button -->
              <div class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700">
                <button
                  type="button"
                  @click="saveBetaPolicySettings"
                  :disabled="betaPolicySaving"
                  class="btn btn-primary btn-sm"
                >
                  <svg
                    v-if="betaPolicySaving"
                    class="mr-1 h-4 w-4 animate-spin"
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
                  {{ betaPolicySaving ? t('common.saving') : t('common.save') }}
                </button>
              </div>
            </template>
          </div>
        </div>

        </div><!-- /Tab: Gateway -->
</template>

<script setup lang="ts">
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import GoogleBatchArchiveSettingsCard from '@/components/settings/GoogleBatchArchiveSettingsCard.vue'
import GoogleBatchGCSProfilesManager from '@/components/settings/GoogleBatchGCSProfilesManager.vue'
import OpenAIFastPolicySettingsCard from '@/components/settings/OpenAIFastPolicySettingsCard.vue'
const props = defineProps<{ ctx: any }>()
const {
  t,
  form,
  overloadCooldownLoading,
  overloadCooldownForm,
  overloadCooldownSaving,
  saveOverloadCooldownSettings,
  streamTimeoutLoading,
  streamTimeoutForm,
  streamTimeoutSaving,
  saveStreamTimeoutSettings,
  rectifierLoading,
  rectifierForm,
  rectifierSaving,
  saveRectifierSettings,
  betaPolicyLoading,
  betaPolicyForm,
  betaPolicyActionOptions,
  betaPolicyScopeOptions,
  betaPolicySaving,
  getBetaDisplayName,
  saveBetaPolicySettings,
} = props.ctx
</script>

