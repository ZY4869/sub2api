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
          <div class="p-6">
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
            <div class="mt-4">
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
        </div><!-- /Tab: Gateway — Claude Code, Scheduling -->
</template>

<script setup lang="ts">
import Toggle from '@/components/common/Toggle.vue'
const props = defineProps<{ ctx: any }>()
const {
  t,
  form,
} = props.ctx
</script>

