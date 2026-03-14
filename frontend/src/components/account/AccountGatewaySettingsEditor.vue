<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import type { OpenAIWSMode } from '@/utils/openaiWsMode'

defineProps<{
  showOpenAiPassthrough: boolean
  openAiPassthroughEnabled: boolean
  showOpenAiWsMode: boolean
  openAiWsMode: OpenAIWSMode
  openAiWsModeOptions: SelectOption[]
  openAiWsModeConcurrencyHintKey: string
  showAnthropicPassthrough: boolean
  anthropicPassthroughEnabled: boolean
  showCodexCliOnly: boolean
  codexCliOnlyEnabled: boolean
}>()

const emit = defineEmits<{
  'update:openAiPassthroughEnabled': [value: boolean]
  'update:openAiWsMode': [value: OpenAIWSMode]
  'update:anthropicPassthroughEnabled': [value: boolean]
  'update:codexCliOnlyEnabled': [value: boolean]
}>()

const { t } = useI18n()
</script>

<template>
  <div
    v-if="showOpenAiPassthrough"
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
        @click="emit('update:openAiPassthroughEnabled', !openAiPassthroughEnabled)"
        :class="[
          'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
          openAiPassthroughEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
        ]"
      >
        <span
          :class="[
            'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            openAiPassthroughEnabled ? 'translate-x-5' : 'translate-x-0'
          ]"
        />
      </button>
    </div>
  </div>

  <div
    v-if="showOpenAiWsMode"
    class="border-t border-gray-200 pt-4 dark:border-dark-600"
  >
    <div class="flex items-center justify-between">
      <div>
        <label class="input-label mb-0">{{ t('admin.accounts.openai.wsMode') }}</label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.openai.wsModeDesc') }}
        </p>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t(openAiWsModeConcurrencyHintKey) }}
        </p>
      </div>
      <div class="w-52">
        <Select
          :model-value="openAiWsMode"
          :options="openAiWsModeOptions"
          @update:model-value="emit('update:openAiWsMode', $event as OpenAIWSMode)"
        />
      </div>
    </div>
  </div>

  <div
    v-if="showAnthropicPassthrough"
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
        @click="emit('update:anthropicPassthroughEnabled', !anthropicPassthroughEnabled)"
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

  <div
    v-if="showCodexCliOnly"
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
        @click="emit('update:codexCliOnlyEnabled', !codexCliOnlyEnabled)"
        :class="[
          'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
          codexCliOnlyEnabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
        ]"
      >
        <span
          :class="[
            'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            codexCliOnlyEnabled ? 'translate-x-5' : 'translate-x-0'
          ]"
        />
      </button>
    </div>
  </div>
</template>
