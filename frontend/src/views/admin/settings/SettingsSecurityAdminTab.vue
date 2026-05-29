<template>
        <!-- Tab: Security — Admin API Key -->
        <div class="space-y-6">
        <!-- Admin API Key Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.adminApiKey.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.adminApiKey.description') }}
            </p>
          </div>
          <div class="space-y-4 p-6">
            <!-- Security Warning -->
            <div
              class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20"
            >
              <div class="flex items-start">
                <Icon
                  name="exclamationTriangle"
                  size="md"
                  class="mt-0.5 flex-shrink-0 text-amber-500"
                />
                <p class="ml-3 text-sm text-amber-700 dark:text-amber-300">
                  {{ t('admin.settings.adminApiKey.securityWarning') }}
                </p>
              </div>
            </div>

            <!-- Loading State -->
            <div v-if="adminApiKeyLoading" class="flex items-center gap-2 text-gray-500">
              <div class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"></div>
              {{ t('common.loading') }}
            </div>

            <!-- No Key Configured -->
            <div v-else-if="!adminApiKeyExists" class="flex items-center justify-between">
              <span class="text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.adminApiKey.notConfigured') }}
              </span>
              <button
                type="button"
                @click="createAdminApiKey"
                :disabled="adminApiKeyOperating"
                class="btn btn-primary btn-sm"
              >
                <svg
                  v-if="adminApiKeyOperating"
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
                {{
                  adminApiKeyOperating
                    ? t('admin.settings.adminApiKey.creating')
                    : t('admin.settings.adminApiKey.create')
                }}
              </button>
            </div>

            <!-- Key Exists -->
            <div v-else class="space-y-4">
              <div class="flex items-center justify-between">
                <div>
                  <label class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.adminApiKey.currentKey') }}
                  </label>
                  <code
                    class="rounded bg-gray-100 px-2 py-1 font-mono text-sm text-gray-900 dark:bg-dark-700 dark:text-gray-100"
                  >
                    {{ adminApiKeyMasked }}
                  </code>
                </div>
                <div class="flex gap-2">
                  <button
                    type="button"
                    @click="regenerateAdminApiKey"
                    :disabled="adminApiKeyOperating"
                    class="btn btn-secondary btn-sm"
                  >
                    {{
                      adminApiKeyOperating
                        ? t('admin.settings.adminApiKey.regenerating')
                        : t('admin.settings.adminApiKey.regenerate')
                    }}
                  </button>
                  <button
                    type="button"
                    @click="deleteAdminApiKey"
                    :disabled="adminApiKeyOperating"
                    class="btn btn-secondary btn-sm text-red-600 hover:text-red-700 dark:text-red-400"
                  >
                    {{ t('admin.settings.adminApiKey.delete') }}
                  </button>
                </div>
              </div>

              <!-- Newly Generated Key Display -->
              <div
                v-if="newAdminApiKey"
                class="space-y-3 rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-900/20"
              >
                <p class="text-sm font-medium text-green-700 dark:text-green-300">
                  {{ t('admin.settings.adminApiKey.keyWarning') }}
                </p>
                <div class="flex items-center gap-2">
                  <code
                    class="flex-1 select-all break-all rounded border border-green-300 bg-white px-3 py-2 font-mono text-sm dark:border-green-700 dark:bg-dark-800"
                  >
                    {{ newAdminApiKey }}
                  </code>
                  <button
                    type="button"
                    @click="copyNewKey"
                    class="btn btn-primary btn-sm flex-shrink-0"
                  >
                    {{ t('admin.settings.adminApiKey.copyKey') }}
                  </button>
                </div>
                <p class="text-xs text-green-600 dark:text-green-400">
                  {{ t('admin.settings.adminApiKey.usage') }}
                </p>
              </div>
            </div>
          </div>
        </div>
        </div><!-- /Tab: Security — Admin API Key -->
</template>

<script setup lang="ts">
import Icon from '@/components/icons/Icon.vue'
const props = defineProps<{ ctx: any }>()
const {
  t,
  adminApiKeyLoading,
  adminApiKeyExists,
  adminApiKeyOperating,
  adminApiKeyMasked,
  newAdminApiKey,
  createAdminApiKey,
  regenerateAdminApiKey,
  deleteAdminApiKey,
  copyNewKey,
} = props.ctx
</script>

