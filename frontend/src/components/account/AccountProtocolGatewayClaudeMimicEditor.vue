<template>
  <section class="space-y-4 rounded-2xl border border-gray-200 bg-white/80 p-4 dark:border-dark-600 dark:bg-dark-700/60">
    <div class="space-y-1">
      <div class="text-sm font-semibold text-gray-900 dark:text-gray-100">
        {{ t('admin.accounts.protocolGateway.claudeMimic.title') }}
      </div>
      <p class="text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.protocolGateway.claudeMimic.description') }}
      </p>
    </div>

    <div class="space-y-4">
      <div class="flex items-center justify-between gap-4">
        <div>
          <label class="input-label mb-0">
            {{ t('admin.accounts.protocolGateway.claudeMimic.fullMimic') }}
          </label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.protocolGateway.claudeMimic.fullMimicHint') }}
          </p>
        </div>
        <button
          type="button"
          :class="toggleClass(enabled)"
          @click="emit('update:enabled', !enabled)"
        >
          <span :class="thumbClass(enabled)" />
        </button>
      </div>

      <div class="space-y-4 rounded-2xl border border-dashed border-gray-200 bg-gray-50/70 p-4 dark:border-dark-500 dark:bg-dark-800/60">
        <div class="flex items-center justify-between gap-4 opacity-100" :class="{ 'opacity-60': !enabled }">
          <div>
            <label class="input-label mb-0">
              {{ t('admin.accounts.protocolGateway.claudeMimic.tlsFingerprint') }}
            </label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.protocolGateway.claudeMimic.tlsFingerprintHint') }}
            </p>
          </div>
          <button
            type="button"
            :disabled="!enabled"
            :class="toggleClass(tlsFingerprintEnabled, !enabled)"
            @click="emit('update:tlsFingerprintEnabled', !tlsFingerprintEnabled)"
          >
            <span :class="thumbClass(tlsFingerprintEnabled)" />
          </button>
        </div>

        <div class="flex items-center justify-between gap-4 opacity-100" :class="{ 'opacity-60': !enabled }">
          <div>
            <label class="input-label mb-0">
              {{ t('admin.accounts.protocolGateway.claudeMimic.sessionMasking') }}
            </label>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.accounts.protocolGateway.claudeMimic.sessionMaskingHint') }}
            </p>
          </div>
          <button
            type="button"
            :disabled="!enabled"
            :class="toggleClass(sessionIdMaskingEnabled, !enabled)"
            @click="emit('update:sessionIdMaskingEnabled', !sessionIdMaskingEnabled)"
          >
            <span :class="thumbClass(sessionIdMaskingEnabled)" />
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  enabled: boolean
  tlsFingerprintEnabled: boolean
  sessionIdMaskingEnabled: boolean
}>()

const emit = defineEmits<{
  'update:enabled': [value: boolean]
  'update:tlsFingerprintEnabled': [value: boolean]
  'update:sessionIdMaskingEnabled': [value: boolean]
}>()

const { t } = useI18n()

function toggleClass(enabled: boolean, disabled = false) {
  return [
    'relative inline-flex h-6 w-11 flex-shrink-0 rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
    disabled
      ? 'cursor-not-allowed bg-gray-200 dark:bg-dark-600'
      : 'cursor-pointer',
    enabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
  ]
}

function thumbClass(enabled: boolean) {
  return [
    'pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
    enabled ? 'translate-x-5' : 'translate-x-0'
  ]
}
</script>
