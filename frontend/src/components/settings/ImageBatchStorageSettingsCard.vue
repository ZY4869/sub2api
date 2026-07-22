<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.imageBatchStorage.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.settings.imageBatchStorage.description') }}
      </p>
    </div>

    <div class="space-y-5 p-6">
      <div v-if="loading" class="flex items-center gap-2 text-gray-500">
        <div class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"></div>
        {{ t('common.loading') }}
      </div>

      <template v-else>
        <label class="space-y-2">
          <span class="input-label">{{ t('admin.settings.imageBatchStorage.backend') }}</span>
          <Select v-model="form.backend" :options="backendOptions" />
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.settings.imageBatchStorage.backendHint') }}
          </span>
        </label>

        <div v-if="externalBackend" class="grid gap-4 md:grid-cols-2">
          <label class="space-y-2">
            <span class="input-label">{{ t('admin.settings.imageBatchStorage.endpoint') }}</span>
            <input
              v-model.trim="form.endpoint"
              type="url"
              class="input"
              placeholder="https://s3.example.com"
            />
          </label>

          <label class="space-y-2">
            <span class="input-label">{{ t('admin.settings.imageBatchStorage.region') }}</span>
            <input v-model.trim="form.region" type="text" class="input" placeholder="auto" />
          </label>

          <label class="space-y-2">
            <span class="input-label">{{ t('admin.settings.imageBatchStorage.bucket') }}</span>
            <input v-model.trim="form.bucket" type="text" class="input" />
          </label>

          <label class="space-y-2">
            <span class="input-label">{{ t('admin.settings.imageBatchStorage.prefix') }}</span>
            <input v-model.trim="form.prefix" type="text" class="input" placeholder="image-batches" />
          </label>

          <label class="space-y-2">
            <span class="input-label">{{ t('admin.settings.imageBatchStorage.accessKeyID') }}</span>
            <input v-model.trim="form.access_key_id" type="text" class="input" autocomplete="off" />
          </label>

          <label class="space-y-2">
            <span class="input-label">
              {{ t('admin.settings.imageBatchStorage.secretAccessKey') }}
            </span>
            <input
              v-model.trim="form.secret_access_key"
              type="password"
              class="input"
              autocomplete="new-password"
              :placeholder="secretPlaceholder"
            />
          </label>
        </div>

        <div
          v-if="externalBackend"
          class="flex items-center justify-between gap-4 border-t border-gray-100 pt-4 dark:border-dark-700"
        >
          <div>
            <label class="font-medium text-gray-900 dark:text-white">
              {{ t('admin.settings.imageBatchStorage.forcePathStyle') }}
            </label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.imageBatchStorage.forcePathStyleHint') }}
            </p>
          </div>
          <Toggle v-model="form.force_path_style" />
        </div>

        <div class="flex flex-wrap justify-end gap-2 border-t border-gray-100 pt-4 dark:border-dark-700">
          <button
            v-if="externalBackend"
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="testing || saving"
            @click="testSettings"
          >
            {{ testing ? t('admin.settings.imageBatchStorage.testing') : t('admin.settings.imageBatchStorage.test') }}
          </button>
          <button
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="saving || testing"
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
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api'
import type { ImageBatchStorageSettings } from '@/api/admin/settings'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(true)
const saving = ref(false)
const testing = ref(false)

const form = reactive<ImageBatchStorageSettings>({
  backend: 'local',
  endpoint: '',
  bucket: '',
  prefix: 'image-batches',
  region: 'auto',
  force_path_style: false,
  access_key_id: '',
  secret_access_key: '',
  secret_access_key_configured: false,
})

const backendOptions = computed(() => [
  { value: 'local', label: t('admin.settings.imageBatchStorage.backendLocal') },
  { value: 's3', label: t('admin.settings.imageBatchStorage.backendS3') },
  { value: 'r2', label: t('admin.settings.imageBatchStorage.backendR2') },
])

const externalBackend = computed(() => form.backend === 's3' || form.backend === 'r2')
const secretPlaceholder = computed(() =>
  form.secret_access_key_configured
    ? t('admin.settings.imageBatchStorage.secretConfigured')
    : t('admin.settings.imageBatchStorage.secretPlaceholder'),
)

const applySettings = (settings: ImageBatchStorageSettings) => {
  form.backend = settings.backend || 'local'
  form.endpoint = settings.endpoint || ''
  form.bucket = settings.bucket || ''
  form.prefix = settings.prefix || 'image-batches'
  form.region = settings.region || 'auto'
  form.force_path_style = !!settings.force_path_style
  form.access_key_id = settings.access_key_id || ''
  form.secret_access_key = ''
  form.secret_access_key_configured = !!settings.secret_access_key_configured
}

const payload = (): ImageBatchStorageSettings => ({
  backend: form.backend,
  endpoint: form.endpoint,
  bucket: form.bucket,
  prefix: form.prefix,
  region: form.region,
  force_path_style: form.force_path_style,
  access_key_id: form.access_key_id,
  secret_access_key: form.secret_access_key,
  secret_access_key_configured: form.secret_access_key_configured,
})

const loadSettings = async () => {
  loading.value = true
  try {
    applySettings(await adminAPI.settings.getImageBatchStorageSettings())
  } catch (error) {
    appStore.showError(
      (error as { message?: string })?.message || t('admin.settings.imageBatchStorage.loadFailed'),
    )
  } finally {
    loading.value = false
  }
}

const saveSettings = async () => {
  saving.value = true
  try {
    applySettings(await adminAPI.settings.updateImageBatchStorageSettings(payload()))
    appStore.showSuccess(t('admin.settings.imageBatchStorage.saved'))
  } catch (error) {
    appStore.showError(
      (error as { message?: string })?.message || t('admin.settings.imageBatchStorage.saveFailed'),
    )
  } finally {
    saving.value = false
  }
}

const testSettings = async () => {
  testing.value = true
  try {
    await adminAPI.settings.testImageBatchStorageSettings(payload())
    appStore.showSuccess(t('admin.settings.imageBatchStorage.testSuccess'))
  } catch (error) {
    appStore.showError(
      (error as { message?: string })?.message || t('admin.settings.imageBatchStorage.testFailed'),
    )
  } finally {
    testing.value = false
  }
}

onMounted(() => {
  void loadSettings()
})
</script>
