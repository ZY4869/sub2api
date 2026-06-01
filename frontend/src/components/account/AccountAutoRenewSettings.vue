<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { buildExpiryInput } from '@/composables/useAccountExpirationShortcuts'
import type { AccountAutoRenewPeriod } from '@/types'

const props = defineProps<{
  expiresAtInput: string
}>()

const emit = defineEmits<{
  'update:expiresAtInput': [value: string]
}>()

const autoRenewEnabled = defineModel<boolean>('autoRenewEnabled', { required: true })
const autoRenewPeriod = defineModel<AccountAutoRenewPeriod>('autoRenewPeriod', { required: true })

const { t } = useI18n()

const autoRenewPeriodOptions = computed(() => [
  { value: 'month' as const, label: t('admin.accounts.autoRenewPeriodMonth') },
  { value: 'quarter' as const, label: t('admin.accounts.autoRenewPeriodQuarter') },
  { value: 'year' as const, label: t('admin.accounts.autoRenewPeriodYear') },
])

const autoRenewReady = computed(() => Boolean(props.expiresAtInput))

const handleAutoRenewToggle = () => {
  if (autoRenewEnabled.value && !props.expiresAtInput) {
    emit('update:expiresAtInput', buildExpiryInput(1, 'month'))
  }
}
</script>

<template>
  <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-600">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0">
        <label class="input-label mb-0">{{ t('admin.accounts.autoRenewEnabled') }}</label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.autoRenewHint') }}
        </p>
      </div>
      <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
        <input
          v-model="autoRenewEnabled"
          type="checkbox"
          class="h-4 w-4 rounded border-gray-300"
          @change="handleAutoRenewToggle"
        />
        <span>{{ t('admin.accounts.autoRenewToggle') }}</span>
      </label>
    </div>
    <div v-if="autoRenewEnabled" class="mt-3">
      <label class="input-label">{{ t('admin.accounts.autoRenewPeriod') }}</label>
      <select v-model="autoRenewPeriod" class="input">
        <option
          v-for="option in autoRenewPeriodOptions"
          :key="option.value"
          :value="option.value"
        >
          {{ option.label }}
        </option>
      </select>
      <p class="input-hint">
        {{
          autoRenewReady
            ? t('admin.accounts.autoRenewPeriodHint')
            : t('admin.accounts.autoRenewRequiresExpiration')
        }}
      </p>
    </div>
  </div>
</template>
