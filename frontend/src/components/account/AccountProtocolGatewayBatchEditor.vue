<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  enabled: boolean
  requestFormats: string[]
}>()

const emit = defineEmits<{
  'update:enabled': [value: boolean]
}>()

const { t } = useI18n()

const requestFormatsText = computed(() => props.requestFormats.join(', '))
</script>

<template>
  <div class="rounded-lg border border-sky-200 bg-sky-50/70 p-4 dark:border-sky-800/40 dark:bg-sky-900/20">
    <div class="flex items-center justify-between gap-4">
      <div class="min-w-0">
        <label class="input-label mb-0">{{ t('admin.accounts.protocolGateway.batch.toggle') }}</label>
        <p class="mt-1 text-xs text-gray-600 dark:text-gray-300">
          {{ t('admin.accounts.protocolGateway.batch.toggleHint') }}
        </p>
      </div>
      <button
        type="button"
        @click="emit('update:enabled', !enabled)"
        :class="[
          'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
          enabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
        ]"
      >
        <span
          :class="[
            'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            enabled ? 'translate-x-5' : 'translate-x-0'
          ]"
        />
      </button>
    </div>

    <div class="mt-3 space-y-2 text-xs text-sky-900 dark:text-sky-100">
      <p class="font-medium">
        {{ t('admin.accounts.protocolGateway.batch.title') }}
      </p>
      <p>
        {{ t('admin.accounts.protocolGateway.batch.hint') }}
      </p>
      <p v-if="enabled" class="break-words text-sky-800 dark:text-sky-200">
        {{ requestFormatsText }}
      </p>
    </div>
  </div>
</template>
