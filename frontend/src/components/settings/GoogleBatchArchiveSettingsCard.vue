<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.googleBatchArchive.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.settings.googleBatchArchive.description') }}
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
              {{ t('admin.settings.googleBatchArchive.enabled') }}
            </label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.googleBatchArchive.enabledHint') }}
            </p>
          </div>
          <Toggle v-model="form.enabled" />
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">
              {{ t('admin.settings.googleBatchArchive.pollMinIntervalSeconds') }}
            </label>
            <input
              v-model.number="form.poll_min_interval_seconds"
              type="number"
              min="30"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">
              {{ t('admin.settings.googleBatchArchive.pollMaxIntervalSeconds') }}
            </label>
            <input
              v-model.number="form.poll_max_interval_seconds"
              type="number"
              min="30"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">
              {{ t('admin.settings.googleBatchArchive.pollBackoffFactor') }}
            </label>
            <input
              v-model.number="form.poll_backoff_factor"
              type="number"
              min="1"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">
              {{ t('admin.settings.googleBatchArchive.pollJitterSeconds') }}
            </label>
            <input
              v-model.number="form.poll_jitter_seconds"
              type="number"
              min="0"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">
              {{ t('admin.settings.googleBatchArchive.pollMaxConcurrency') }}
            </label>
            <input
              v-model.number="form.poll_max_concurrency"
              type="number"
              min="1"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">
              {{ t('admin.settings.googleBatchArchive.prefetchAfterHours') }}
            </label>
            <input
              v-model.number="form.prefetch_after_hours"
              type="number"
              min="1"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">
              {{ t('admin.settings.googleBatchArchive.downloadTimeoutSeconds') }}
            </label>
            <input
              v-model.number="form.download_timeout_seconds"
              type="number"
              min="30"
              class="input"
            />
          </div>

          <div>
            <label class="input-label">
              {{ t('admin.settings.googleBatchArchive.cleanupIntervalMinutes') }}
            </label>
            <input
              v-model.number="form.cleanup_interval_minutes"
              type="number"
              min="1"
              class="input"
            />
          </div>

          <div class="md:col-span-2">
            <label class="input-label">
              {{ t('admin.settings.googleBatchArchive.localStorageRoot') }}
            </label>
            <input
              v-model.trim="form.local_storage_root"
              type="text"
              class="input"
            />
          </div>
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
  enabled: false,
  poll_min_interval_seconds: 300,
  poll_max_interval_seconds: 1800,
  poll_backoff_factor: 2,
  poll_jitter_seconds: 30,
  poll_max_concurrency: 2,
  prefetch_after_hours: 40,
  download_timeout_seconds: 180,
  cleanup_interval_minutes: 60,
  local_storage_root: '/app/data/google-batch',
})

const loadSettings = async () => {
  loading.value = true
  try {
    const settings = await adminAPI.settings.getGoogleBatchArchiveSettings()
    form.enabled = settings.enabled
    form.poll_min_interval_seconds = settings.poll_min_interval_seconds
    form.poll_max_interval_seconds = settings.poll_max_interval_seconds
    form.poll_backoff_factor = settings.poll_backoff_factor
    form.poll_jitter_seconds = settings.poll_jitter_seconds
    form.poll_max_concurrency = settings.poll_max_concurrency
    form.prefetch_after_hours = settings.prefetch_after_hours
    form.download_timeout_seconds = settings.download_timeout_seconds
    form.cleanup_interval_minutes = settings.cleanup_interval_minutes
    form.local_storage_root = settings.local_storage_root
  } catch (error) {
    appStore.showError(
      (error as { message?: string })?.message ||
        t('admin.settings.googleBatchArchive.loadFailed'),
    )
  } finally {
    loading.value = false
  }
}

const saveSettings = async () => {
  saving.value = true
  try {
    await adminAPI.settings.updateGoogleBatchArchiveSettings({ ...form })
    appStore.showSuccess(t('admin.settings.googleBatchArchive.saved'))
  } catch (error) {
    appStore.showError(
      (error as { message?: string })?.message ||
        t('admin.settings.googleBatchArchive.saveFailed'),
    )
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void loadSettings()
})
</script>
