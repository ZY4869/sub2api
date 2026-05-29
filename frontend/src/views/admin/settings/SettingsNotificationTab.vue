<template>
        <div class="space-y-6">
        <div class="card">
          <div
            class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-700"
          >
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('admin.settings.telegram.title') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.telegram.description') }}
              </p>
            </div>
            <button
              type="button"
              @click="testTelegramConnection"
              :disabled="testingTelegram"
              class="btn btn-secondary btn-sm"
            >
              <svg v-if="testingTelegram" class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
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
                testingTelegram
                  ? t('admin.settings.telegram.testing')
                  : t('admin.settings.telegram.testConnection')
              }}
            </button>
          </div>
          <div class="space-y-6 p-6">
            <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.telegram.botToken') }}
                </label>
                <input
                  v-model="form.telegram_bot_token"
                  type="password"
                  class="input"
                  :placeholder="
                    form.telegram_bot_token_configured
                      ? form.telegram_bot_token_masked || t('admin.settings.telegram.botTokenPlaceholder')
                      : t('admin.settings.telegram.botTokenPlaceholder')
                  "
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{
                    form.telegram_bot_token_configured
                      ? t('admin.settings.telegram.botTokenConfiguredHint')
                      : t('admin.settings.telegram.botTokenHint')
                  }}
                </p>
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.telegram.chatId') }}
                </label>
                <input
                  v-model="form.telegram_chat_id"
                  type="text"
                  class="input"
                  :placeholder="t('admin.settings.telegram.chatIdPlaceholder')"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.telegram.chatIdHint') }}
                </p>
              </div>
            </div>
          </div>
        </div>
        </div><!-- /Tab: Notification -->
</template>

<script setup lang="ts">

const props = defineProps<{ ctx: any }>()
const {
  t,
  form,
  testingTelegram,
  testTelegramConnection,
} = props.ctx
</script>

