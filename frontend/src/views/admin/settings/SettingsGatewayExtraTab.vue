<template>
        <div class="space-y-6">
        <!-- Claude Code Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.claudeCode.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.claudeCode.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <div class="flex items-center justify-between gap-4">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.claudeCode.allowCodexPlugin') }}
                </label>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.claudeCode.allowCodexPluginHint') }}
                </p>
              </div>
              <Toggle v-model="form.openai_allow_claude_code_codex_plugin" />
            </div>
            <div
              v-if="form.openai_allow_claude_code_codex_plugin"
              class="rounded-lg border border-gray-100 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/60"
            >
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.claudeCode.allowedClients') }}
              </p>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.claudeCode.allowedClientsHint') }}
              </p>
              <div class="mt-3 space-y-2">
                <label
                  v-for="client in openAIAllowedCodexClientOptions"
                  :key="client.value"
                  class="flex items-start gap-3 rounded-md border border-gray-100 bg-white px-3 py-2 dark:border-dark-700 dark:bg-dark-900"
                >
                  <input
                    v-model="form.openai_allowed_codex_clients"
                    type="checkbox"
                    class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-800"
                    :value="client.value"
                  />
                  <span>
                    <span class="block text-sm font-medium text-gray-800 dark:text-gray-200">
                      {{ t(client.labelKey) }}
                    </span>
                    <span class="block text-xs text-gray-500 dark:text-gray-400">
                      {{ t(client.descriptionKey) }}
                    </span>
                  </span>
                </label>
              </div>
            </div>
            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.claudeCode.minVersion') }}
              </label>
              <input
                v-model="form.min_claude_code_version"
                type="text"
                class="input max-w-xs font-mono text-sm"
                :placeholder="t('admin.settings.claudeCode.minVersionPlaceholder')"
              />
              <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.claudeCode.minVersionHint') }}
              </p>
            </div>
            <div>
              <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                {{ t('admin.settings.claudeCode.maxVersion') }}
              </label>
              <input
                v-model="form.max_claude_code_version"
                type="text"
                class="input max-w-xs font-mono text-sm"
                :placeholder="t('admin.settings.claudeCode.maxVersionPlaceholder')"
              />
              <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.claudeCode.maxVersionHint') }}
              </p>
            </div>
            <div class="rounded-lg border border-gray-100 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/60">
              <div class="flex items-center justify-between gap-4">
                <div>
                  <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.claudeCode.oauthPromptBlocks') }}
                  </label>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.claudeCode.oauthPromptBlocksHint') }}
                  </p>
                </div>
                <Toggle v-model="form.claude_oauth_system_prompt_blocks_enabled" />
              </div>
              <div
                v-if="form.claude_oauth_system_prompt_blocks_enabled"
                class="mt-4"
              >
                <textarea
                  v-model="form.claude_oauth_system_prompt_blocks"
                  rows="5"
                  class="input font-mono text-sm"
                  :placeholder="t('admin.settings.claudeCode.oauthPromptBlocksPlaceholder')"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.claudeCode.oauthPromptBlocksPriorityHint') }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <!-- OpenAI alpha/search Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.openaiAlphaSearch.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.openaiAlphaSearch.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <div class="flex items-center justify-between gap-4">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.openaiAlphaSearch.enabled') }}
                </label>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.openaiAlphaSearch.enabledHint') }}
                </p>
              </div>
              <Toggle v-model="form.openai_alpha_search_enabled" />
            </div>
          </div>
        </div>

        <!-- Gateway Scheduling Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.scheduling.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.scheduling.description') }}
            </p>
          </div>
          <div class="space-y-5 p-6">
            <div class="flex items-center justify-between">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.scheduling.allowUngroupedKey') }}
                </label>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.scheduling.allowUngroupedKeyHint') }}
                </p>
              </div>
              <label class="toggle">
                <input v-model="form.allow_ungrouped_key_scheduling" type="checkbox" />
                <span class="toggle-slider"></span>
              </label>
            </div>

            <div class="space-y-4 border-t border-gray-100 pt-5 dark:border-dark-700">
              <div class="flex items-center justify-between gap-4">
                <div>
                  <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.scheduling.openaiAdvancedEnabled') }}
                  </label>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.scheduling.openaiAdvancedEnabledHint') }}
                  </p>
                </div>
                <Toggle v-model="form.openai_advanced_scheduler_enabled" />
              </div>

              <div
                v-if="form.openai_advanced_scheduler_enabled"
                class="space-y-4 rounded-lg border border-gray-100 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/60"
              >
                <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                  <div class="flex items-center justify-between gap-4 rounded-md bg-white px-3 py-2 dark:bg-dark-900">
                    <div>
                      <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t('admin.settings.scheduling.stickyWeighted') }}
                      </label>
                      <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                        {{ t('admin.settings.scheduling.stickyWeightedHint') }}
                      </p>
                    </div>
                    <Toggle v-model="form.openai_advanced_scheduler_sticky_weighted_enabled" />
                  </div>
                  <div class="flex items-center justify-between gap-4 rounded-md bg-white px-3 py-2 dark:bg-dark-900">
                    <div>
                      <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t('admin.settings.scheduling.subscriptionPriority') }}
                      </label>
                      <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                        {{ t('admin.settings.scheduling.subscriptionPriorityHint') }}
                      </p>
                    </div>
                    <Toggle v-model="form.openai_advanced_scheduler_subscription_priority_enabled" />
                  </div>
                </div>

                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.scheduling.lbTopK') }}
                  </label>
                  <input
                    v-model.trim="form.openai_advanced_scheduler_lb_top_k"
                    type="number"
                    min="1"
                    class="input w-36 font-mono text-sm"
                    :placeholder="form.openai_advanced_scheduler_effective_lb_top_k || '7'"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.scheduling.lbTopKHint') }}
                  </p>
                </div>

                <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
                  <div
                    v-for="field in openAIAdvancedSchedulerWeightFields"
                    :key="field.key"
                  >
                    <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                      {{ t(field.labelKey) }}
                    </label>
                    <input
                      v-model.trim="form[field.key]"
                      type="number"
                      min="0"
                      step="0.1"
                      class="input font-mono text-sm"
                      :placeholder="String(form[field.effectiveKey] || '')"
                    />
                  </div>
                </div>
                <p class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.scheduling.weightsHint') }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.maintenance.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.maintenance.description') }}
            </p>
          </div>
          <div class="space-y-4 p-6">
            <div class="flex items-center justify-between gap-4">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.maintenance.enabled') }}
                </label>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.maintenance.enabledHint') }}
                </p>
              </div>
              <Toggle v-model="form.maintenance_mode_enabled" />
            </div>
            <div class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-xs text-amber-800 dark:border-amber-700/40 dark:bg-amber-950/30 dark:text-amber-200">
              {{ t('admin.settings.maintenance.scopeHint') }}
            </div>
          </div>
        </div>

        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.adminCompliance.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.adminCompliance.description') }}
            </p>
          </div>
          <div class="space-y-4 p-6">
            <div class="flex items-center justify-between gap-4">
              <div>
                <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.adminCompliance.enabled') }}
                </label>
                <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.adminCompliance.enabledHint') }}
                </p>
              </div>
              <Toggle v-model="form.admin_compliance_enabled" />
            </div>
            <div class="rounded-xl border border-sky-200 bg-sky-50 px-4 py-3 text-xs text-sky-800 dark:border-sky-700/40 dark:bg-sky-950/30 dark:text-sky-200">
              {{ t('admin.settings.adminCompliance.scopeHint') }}
            </div>
          </div>
        </div>
        </div><!-- /Tab: Gateway — Claude Code, Scheduling -->
</template>

<script setup lang="ts">
import Toggle from '@/components/common/Toggle.vue'
const props = defineProps<{ ctx: any }>()
const {
  t,
  form,
} = props.ctx

const openAIAllowedCodexClientOptions = [
  {
    value: 'claude_code',
    labelKey: 'admin.settings.claudeCode.allowedClientClaudeCodeLabel',
    descriptionKey: 'admin.settings.claudeCode.allowedClientClaudeCode'
  }
]

const openAIAdvancedSchedulerWeightFields = [
  {
    key: 'openai_advanced_scheduler_weight_priority',
    effectiveKey: 'openai_advanced_scheduler_effective_weight_priority',
    labelKey: 'admin.settings.scheduling.weightPriority'
  },
  {
    key: 'openai_advanced_scheduler_weight_load',
    effectiveKey: 'openai_advanced_scheduler_effective_weight_load',
    labelKey: 'admin.settings.scheduling.weightLoad'
  },
  {
    key: 'openai_advanced_scheduler_weight_queue',
    effectiveKey: 'openai_advanced_scheduler_effective_weight_queue',
    labelKey: 'admin.settings.scheduling.weightQueue'
  },
  {
    key: 'openai_advanced_scheduler_weight_error_rate',
    effectiveKey: 'openai_advanced_scheduler_effective_weight_error_rate',
    labelKey: 'admin.settings.scheduling.weightErrorRate'
  },
  {
    key: 'openai_advanced_scheduler_weight_ttft',
    effectiveKey: 'openai_advanced_scheduler_effective_weight_ttft',
    labelKey: 'admin.settings.scheduling.weightTTFT'
  },
  {
    key: 'openai_advanced_scheduler_weight_quota_headroom',
    effectiveKey: 'openai_advanced_scheduler_effective_weight_quota_headroom',
    labelKey: 'admin.settings.scheduling.weightQuotaHeadroom'
  },
  {
    key: 'openai_advanced_scheduler_weight_previous_response',
    effectiveKey: 'openai_advanced_scheduler_effective_weight_previous_response',
    labelKey: 'admin.settings.scheduling.weightPreviousResponse'
  },
  {
    key: 'openai_advanced_scheduler_weight_session_sticky',
    effectiveKey: 'openai_advanced_scheduler_effective_weight_session_sticky',
    labelKey: 'admin.settings.scheduling.weightSessionSticky'
  }
] as const
</script>

