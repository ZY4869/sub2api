<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.billingProbe.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.settings.billingProbe.description') }}
      </p>
    </div>

    <div class="space-y-5 p-6">
      <div v-if="loading" class="flex items-center gap-2 text-gray-500">
        <div class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"></div>
        {{ t('common.loading') }}
      </div>

      <template v-else>
        <div class="flex items-center justify-between gap-4">
          <div>
            <label class="font-medium text-gray-900 dark:text-white">
              {{ t('admin.settings.billingProbe.enabled') }}
            </label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.billingProbe.enabledHint') }}
            </p>
          </div>
          <Toggle v-model="form.enabled" />
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <label class="space-y-2">
            <span class="input-label">{{ t('admin.settings.billingProbe.batchConcurrency') }}</span>
            <input
              v-model.number="form.batch_concurrency"
              type="number"
              min="1"
              max="10"
              class="input"
            />
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.billingProbe.batchConcurrencyHint') }}
            </span>
          </label>

          <label class="space-y-2">
            <span class="input-label">{{ t('admin.settings.billingProbe.timeoutSeconds') }}</span>
            <input
              v-model.number="form.timeout_seconds"
              type="number"
              min="1"
              max="120"
              class="input"
            />
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.billingProbe.timeoutSecondsHint') }}
            </span>
          </label>
        </div>

        <div class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700">
          <button
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="saving"
            @click="saveSettings"
          >
            <svg
              v-if="saving"
              class="mr-1 h-4 w-4 animate-spin"
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
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api'
import { useAppStore } from '@/stores'
import Toggle from '@/components/common/Toggle.vue'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(true)
const saving = ref(false)
const form = reactive({
  enabled: true,
  batch_concurrency: 2,
  timeout_seconds: 20
})

const applySettings = (settings: typeof form) => {
  form.enabled = settings.enabled
  form.batch_concurrency = settings.batch_concurrency
  form.timeout_seconds = settings.timeout_seconds
}

const loadSettings = async () => {
  loading.value = true
  try {
    applySettings(await adminAPI.settings.getUpstreamBillingProbeSettings())
  } catch (error) {
    appStore.showError(
      (error as { message?: string })?.message || t('admin.settings.billingProbe.loadFailed'),
    )
  } finally {
    loading.value = false
  }
}

const saveSettings = async () => {
  saving.value = true
  try {
    applySettings(await adminAPI.settings.updateUpstreamBillingProbeSettings({ ...form }))
    appStore.showSuccess(t('admin.settings.billingProbe.saved'))
  } catch (error) {
    appStore.showError(
      (error as { message?: string })?.message || t('admin.settings.billingProbe.saveFailed'),
    )
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void loadSettings()
})
</script>
