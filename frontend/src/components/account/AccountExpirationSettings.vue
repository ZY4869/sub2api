<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AccountAutoRenewSettings from '@/components/account/AccountAutoRenewSettings.vue'
import AccountProbeExtensionSettings from '@/components/account/AccountProbeExtensionSettings.vue'
import { useAccountExpirationShortcuts } from '@/composables/useAccountExpirationShortcuts'
import type { AccountAutoRenewPeriod } from '@/types'

const expiresAtInput = defineModel<string>('expiresAtInput', { required: true })
const expiryProbeExtensionDays = defineModel<number>('expiryProbeExtensionDays', { required: true })
const autoRenewEnabled = defineModel<boolean>('autoRenewEnabled', { required: true })
const autoRenewPeriod = defineModel<AccountAutoRenewPeriod>('autoRenewPeriod', { required: true })

const { t } = useI18n()

const {
  expirationEnabled,
  expiresAtPreview,
  applyQuickExpiry,
} = useAccountExpirationShortcuts({
  expiresAtInput,
  autoRenewEnabled
})
</script>

<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="flex items-center justify-between gap-3">
      <label class="input-label mb-0">{{ t('admin.accounts.expiresAt') }}</label>
      <label class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
        <input v-model="expirationEnabled" type="checkbox" class="h-4 w-4 rounded border-gray-300" />
        <span>{{ t('admin.accounts.expirationEnabled') }}</span>
      </label>
    </div>
    <div v-if="expirationEnabled" class="mt-3 space-y-3">
      <div class="flex flex-wrap gap-2">
        <button type="button" class="btn btn-secondary btn-sm" @click="applyQuickExpiry(7, 'day')">
          {{ t('admin.accounts.expirationQuickWeek') }}
        </button>
        <button type="button" class="btn btn-secondary btn-sm" @click="applyQuickExpiry(1, 'month')">
          {{ t('admin.accounts.expirationQuickMonth') }}
        </button>
        <button type="button" class="btn btn-secondary btn-sm" @click="applyQuickExpiry(1, 'year')">
          {{ t('admin.accounts.expirationQuickYear') }}
        </button>
      </div>
      <input v-model="expiresAtInput" type="datetime-local" class="input" />
      <p v-if="expiresAtPreview" class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.expiresAtPreview', { value: expiresAtPreview.replace('T', ' ') }) }}
      </p>
      <AccountProbeExtensionSettings
        v-model:expiry-probe-extension-days="expiryProbeExtensionDays"
      />
      <AccountAutoRenewSettings
        v-model:auto-renew-enabled="autoRenewEnabled"
        v-model:auto-renew-period="autoRenewPeriod"
        :expires-at-input="expiresAtInput"
        @update:expires-at-input="expiresAtInput = $event"
      />
    </div>
    <p class="input-hint">{{ t('admin.accounts.expiresAtHint') }}</p>
  </div>
</template>
