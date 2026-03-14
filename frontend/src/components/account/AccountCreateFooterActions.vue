<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    step: number
    submitting: boolean
    isOAuthFlow: boolean
    isManualInputMethod: boolean
    currentOAuthLoading: boolean
    canExchangeCode: boolean
    formId?: string
  }>(),
  {
    formId: 'create-account-form'
  }
)

const autoImportModels = defineModel<boolean>('autoImportModels', { required: true })

const emit = defineEmits<{
  close: []
  back: []
  exchangeCode: []
}>()

const { t } = useI18n()

const submitLabel = computed(() => {
  if (props.isOAuthFlow) return t('common.next')
  if (props.submitting) return t('admin.accounts.creating')
  return t('common.create')
})

const exchangeLabel = computed(() =>
  props.currentOAuthLoading ? t('admin.accounts.oauth.verifying') : t('admin.accounts.oauth.completeAuth')
)
</script>

<template>
  <div v-if="step === 1" class="flex flex-wrap items-center justify-between gap-3">
    <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
      <input
        v-model="autoImportModels"
        type="checkbox"
        class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
      />
      <span>{{ t('admin.accounts.autoImportModels') }}</span>
    </label>
    <div class="flex justify-end gap-3">
      <button type="button" class="btn btn-secondary" @click="emit('close')">
        {{ t('common.cancel') }}
      </button>
      <button
        type="submit"
        :form="formId"
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
        {{ submitLabel }}
      </button>
    </div>
  </div>

  <div v-else class="flex flex-wrap items-center justify-between gap-3">
    <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
      <input
        v-model="autoImportModels"
        type="checkbox"
        class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
      />
      <span>{{ t('admin.accounts.autoImportModels') }}</span>
    </label>
    <div class="flex items-center gap-3">
      <button type="button" class="btn btn-secondary" @click="emit('back')">
        {{ t('common.back') }}
      </button>
      <button
        v-if="isManualInputMethod"
        type="button"
        :disabled="!canExchangeCode"
        class="btn btn-primary"
        @click="emit('exchangeCode')"
      >
        <svg
          v-if="currentOAuthLoading"
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
        {{ exchangeLabel }}
      </button>
    </div>
  </div>
</template>
