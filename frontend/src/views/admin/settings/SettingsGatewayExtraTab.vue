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
          <div class="p-6">
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
</script>

