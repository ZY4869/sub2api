<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { AccountCustomErrorCodesState } from '@/utils/accountApiKeyAdvancedSettings'

interface ErrorCodeOption {
  value: number
  label: string
}

const props = defineProps<{
  errorCodeOptions: ErrorCodeOption[]
  showError: (message: string) => void
  showInfo: (message: string) => void
}>()

const state = defineModel<AccountCustomErrorCodesState>('state', { required: true })

const { t } = useI18n()

const sortedSelectedCodes = computed(() => [...state.value.selectedCodes].sort((a, b) => a - b))

const confirmWarningForCode = (code: number) => {
  if (code !== 429 && code !== 529) {
    return true
  }

  const confirmFn =
    typeof globalThis.confirm === 'function' ? globalThis.confirm.bind(globalThis) : () => true

  return confirmFn(
    t(code === 429 ? 'admin.accounts.customErrorCodes429Warning' : 'admin.accounts.customErrorCodes529Warning')
  )
}

const toggleErrorCode = (code: number) => {
  const index = state.value.selectedCodes.indexOf(code)
  if (index !== -1) {
    state.value.selectedCodes.splice(index, 1)
    return
  }

  if (!confirmWarningForCode(code)) {
    return
  }

  state.value.selectedCodes.push(code)
}

const addCustomErrorCode = () => {
  const code = state.value.input
  if (code === null || code < 100 || code > 599) {
    props.showError(t('admin.accounts.invalidErrorCode'))
    return
  }

  if (state.value.selectedCodes.includes(code)) {
    props.showInfo(t('admin.accounts.errorCodeExists'))
    return
  }

  if (!confirmWarningForCode(code)) {
    return
  }

  state.value.selectedCodes.push(code)
  state.value.input = null
}

const removeErrorCode = (code: number) => {
  const index = state.value.selectedCodes.indexOf(code)
  if (index !== -1) {
    state.value.selectedCodes.splice(index, 1)
  }
}
</script>

<template>
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
        @click="state.enabled = !state.enabled"
        :class="[
          'relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
          state.enabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
        ]"
      >
        <span
          :class="[
            'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            state.enabled ? 'translate-x-5' : 'translate-x-0'
          ]"
        />
      </button>
    </div>

    <div v-if="state.enabled" class="space-y-3">
      <div class="rounded-lg bg-amber-50 p-3 dark:bg-amber-900/20">
        <p class="text-xs text-amber-700 dark:text-amber-400">
          <Icon name="exclamationTriangle" size="sm" class="mr-1 inline" :stroke-width="2" />
          {{ t('admin.accounts.customErrorCodesWarning') }}
        </p>
      </div>

      <div class="flex flex-wrap gap-2">
        <button
          v-for="code in errorCodeOptions"
          :key="code.value"
          type="button"
          @click="toggleErrorCode(code.value)"
          :class="[
            'rounded-lg px-3 py-1.5 text-sm font-medium transition-colors',
            state.selectedCodes.includes(code.value)
              ? 'bg-red-100 text-red-700 ring-1 ring-red-500 dark:bg-red-900/30 dark:text-red-400'
              : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-400 dark:hover:bg-dark-500'
          ]"
        >
          {{ code.value }} {{ code.label }}
        </button>
      </div>

      <div class="flex items-center gap-2">
        <input
          v-model.number="state.input"
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

      <div class="flex flex-wrap gap-1.5">
        <span
          v-for="code in sortedSelectedCodes"
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
        <span v-if="sortedSelectedCodes.length === 0" class="text-xs text-gray-400">
          {{ t('admin.accounts.noneSelectedUsesDefault') }}
        </span>
      </div>
    </div>
  </div>
</template>
