<template>
  <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
    <div class="mb-4">
      <h3 class="input-label mb-0 text-base font-semibold">
        {{ t('admin.accounts.googleBatchArchive.title') }}
      </h3>
      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {{
          mode === 'ai_studio'
            ? t('admin.accounts.googleBatchArchive.descriptionAIStudio')
            : t('admin.accounts.googleBatchArchive.descriptionVertex')
        }}
      </p>
    </div>

    <div class="space-y-4">
      <div class="flex items-center justify-between gap-4">
        <div>
          <label class="input-label mb-0">
            {{ t('admin.accounts.googleBatchArchive.archiveEnabled') }}
          </label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.googleBatchArchive.archiveEnabledHint') }}
          </p>
        </div>
        <Toggle
          :model-value="archiveEnabled"
          @update:model-value="emit('update:archiveEnabled', $event)"
        />
      </div>

      <div v-if="archiveEnabled" class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div v-if="mode === 'ai_studio'" class="md:col-span-2">
          <div class="flex items-center justify-between gap-4">
            <div>
              <label class="input-label mb-0">
                {{ t('admin.accounts.googleBatchArchive.autoPrefetchEnabled') }}
              </label>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.googleBatchArchive.autoPrefetchEnabledHint') }}
              </p>
            </div>
            <Toggle
              :model-value="autoPrefetchEnabled"
              @update:model-value="emit('update:autoPrefetchEnabled', $event)"
            />
          </div>
        </div>

        <div>
          <label class="input-label">
            {{ t('admin.accounts.googleBatchArchive.retentionDays') }}
          </label>
          <input
            :value="retentionDays"
            type="number"
            min="1"
            class="input"
            @input="onRetentionDaysInput"
          />
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.googleBatchArchive.retentionDaysHint') }}
          </p>
        </div>

        <div>
          <label class="input-label">
            {{ t('admin.accounts.googleBatchArchive.billingMode') }}
          </label>
          <select
            :value="billingMode"
            class="input"
            @change="onBillingModeChange"
          >
            <option value="log_only">
              {{ t('admin.accounts.googleBatchArchive.billingModeLogOnly') }}
            </option>
            <option value="archive_charge">
              {{ t('admin.accounts.googleBatchArchive.billingModeArchiveCharge') }}
            </option>
          </select>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.googleBatchArchive.billingModeHint') }}
          </p>
        </div>

        <div v-if="billingMode === 'archive_charge'" class="md:col-span-2">
          <label class="input-label">
            {{ t('admin.accounts.googleBatchArchive.downloadPrice') }}
          </label>
          <div class="relative">
            <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500 dark:text-gray-400">
              $
            </span>
            <input
              :value="downloadPriceUsd"
              type="number"
              min="0"
              step="0.000001"
              class="input pl-7"
              @input="onDownloadPriceInput"
            />
          </div>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.googleBatchArchive.downloadPriceHint') }}
          </p>
        </div>
      </div>

      <div
        v-if="mode === 'ai_studio'"
        class="flex items-center justify-between gap-4 border-t border-gray-100 pt-4 dark:border-dark-700"
      >
        <div>
          <label class="input-label mb-0">
            {{ t('admin.accounts.googleBatchArchive.allowVertexOverflow') }}
          </label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.googleBatchArchive.allowVertexOverflowHint') }}
          </p>
        </div>
        <Toggle
          :model-value="allowVertexBatchOverflow"
          @update:model-value="emit('update:allowVertexBatchOverflow', $event)"
        />
      </div>

      <div
        v-if="mode === 'vertex'"
        class="flex items-center justify-between gap-4 border-t border-gray-100 pt-4 dark:border-dark-700"
      >
        <div>
          <label class="input-label mb-0">
            {{ t('admin.accounts.googleBatchArchive.acceptAIStudioOverflow') }}
          </label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.googleBatchArchive.acceptAIStudioOverflowHint') }}
          </p>
        </div>
        <Toggle
          :model-value="acceptAiStudioBatchOverflow"
          @update:model-value="emit('update:acceptAiStudioBatchOverflow', $event)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import type { GoogleBatchArchiveBillingMode } from '@/utils/accountGoogleBatchArchive'

const { t } = useI18n()

withDefaults(
  defineProps<{
    mode: 'ai_studio' | 'vertex'
    archiveEnabled: boolean
    autoPrefetchEnabled?: boolean
    retentionDays: number | null
    billingMode: GoogleBatchArchiveBillingMode
    downloadPriceUsd: number | null
    allowVertexBatchOverflow?: boolean
    acceptAiStudioBatchOverflow?: boolean
  }>(),
  {
    autoPrefetchEnabled: false,
    allowVertexBatchOverflow: false,
    acceptAiStudioBatchOverflow: false,
  },
)

const emit = defineEmits<{
  'update:archiveEnabled': [value: boolean]
  'update:autoPrefetchEnabled': [value: boolean]
  'update:retentionDays': [value: number]
  'update:billingMode': [value: GoogleBatchArchiveBillingMode]
  'update:downloadPriceUsd': [value: number]
  'update:allowVertexBatchOverflow': [value: boolean]
  'update:acceptAiStudioBatchOverflow': [value: boolean]
}>()

const onRetentionDaysInput = (event: Event) => {
  const rawValue = (event.target as HTMLInputElement).valueAsNumber
  emit('update:retentionDays', Number.isFinite(rawValue) && rawValue > 0 ? rawValue : 1)
}

const onBillingModeChange = (event: Event) => {
  const nextValue = (event.target as HTMLSelectElement)
    .value as GoogleBatchArchiveBillingMode
  emit('update:billingMode', nextValue)
}

const onDownloadPriceInput = (event: Event) => {
  const rawValue = (event.target as HTMLInputElement).valueAsNumber
  emit(
    'update:downloadPriceUsd',
    Number.isFinite(rawValue) && rawValue >= 0 ? rawValue : 0,
  )
}
</script>
