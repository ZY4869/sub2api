<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const expiryProbeExtensionDaysModel = defineModel<number>('expiryProbeExtensionDays', { required: true })

const { t } = useI18n()

const expiryProbeExtensionDays = computed({
  get: () => normalizeExpiryProbeExtensionDaysValue(expiryProbeExtensionDaysModel.value),
  set: (value: number) => {
    expiryProbeExtensionDaysModel.value = normalizeExpiryProbeExtensionDaysValue(value)
  }
})

function normalizeExpiryProbeExtensionDaysValue(value: number | null | undefined): number {
  const normalized = Number(value)
  if (!Number.isFinite(normalized) || normalized <= 0) {
    return 1
  }
  return Math.floor(normalized)
}

function applyQuickExtensionDays(days: number) {
  expiryProbeExtensionDays.value = days
}
</script>

<template>
  <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-600">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div class="min-w-0 flex-1">
        <label class="input-label">{{ t('admin.accounts.expiryProbeExtensionDays') }}</label>
        <input
          v-model.number="expiryProbeExtensionDays"
          type="number"
          min="1"
          step="1"
          class="input"
        />
        <p class="input-hint">{{ t('admin.accounts.expiryProbeExtensionDaysHint') }}</p>
      </div>
      <div class="flex flex-wrap gap-2 md:pt-7">
        <button type="button" class="btn btn-secondary btn-sm" @click="applyQuickExtensionDays(1)">
          {{ t('admin.accounts.expiryProbeExtensionQuick1d') }}
        </button>
        <button type="button" class="btn btn-secondary btn-sm" @click="applyQuickExtensionDays(7)">
          {{ t('admin.accounts.expiryProbeExtensionQuick7d') }}
        </button>
      </div>
    </div>
  </div>
</template>
