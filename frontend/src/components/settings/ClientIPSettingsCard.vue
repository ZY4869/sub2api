<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.clientIP.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.settings.clientIP.description') }}
      </p>
    </div>

    <div class="space-y-5 p-6">
      <div v-if="loading" class="flex items-center gap-2 text-gray-500">
        <div class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"></div>
        {{ t('common.loading') }}
      </div>

      <template v-else>
        <label class="space-y-2">
          <span class="input-label">{{ t('admin.settings.clientIP.mode') }}</span>
          <Select v-model="form.mode" :options="modeOptions" />
          <span class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.settings.clientIP.modeHint') }}
          </span>
        </label>

        <div v-if="form.mode === 'headers'" class="space-y-4 border-t border-gray-100 pt-4 dark:border-dark-700">
          <label class="space-y-2">
            <span class="input-label">{{ t('admin.settings.clientIP.headers') }}</span>
            <textarea
              v-model="headersText"
              rows="4"
              class="input font-mono text-sm"
              placeholder="CF-Connecting-IP&#10;X-Real-IP&#10;X-Forwarded-For"
            ></textarea>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.clientIP.headersHint') }}
            </span>
          </label>

          <label class="space-y-2">
            <span class="input-label">{{ t('admin.settings.clientIP.xffHopIndex') }}</span>
            <input
              v-model.number="form.xff_hop_index"
              type="number"
              min="0"
              max="20"
              class="input w-32"
            />
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.clientIP.xffHopIndexHint') }}
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
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api'
import type { ClientIPSettings } from '@/api/admin/settings'
import Select from '@/components/common/Select.vue'
import { useAppStore } from '@/stores'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(true)
const saving = ref(false)

const form = reactive<ClientIPSettings>({
  mode: 'gin',
  headers: [],
  xff_hop_index: 0,
})

const modeOptions = computed(() => [
  { value: 'gin', label: t('admin.settings.clientIP.modeGin') },
  { value: 'headers', label: t('admin.settings.clientIP.modeHeaders') },
])

const headersText = computed({
  get: () => form.headers.join('\n'),
  set: (value: string) => {
    form.headers = value
      .split(/\r?\n|,/)
      .map((item) => item.trim())
      .filter(Boolean)
  },
})

const applySettings = (settings: ClientIPSettings) => {
  form.mode = settings.mode || 'gin'
  form.headers = Array.isArray(settings.headers) ? [...settings.headers] : []
  form.xff_hop_index = Math.max(0, Number(settings.xff_hop_index || 0))
}

const loadSettings = async () => {
  loading.value = true
  try {
    applySettings(await adminAPI.settings.getClientIPSettings())
  } catch (error) {
    appStore.showError(
      (error as { message?: string })?.message || t('admin.settings.clientIP.loadFailed'),
    )
  } finally {
    loading.value = false
  }
}

const saveSettings = async () => {
  saving.value = true
  try {
    applySettings(await adminAPI.settings.updateClientIPSettings({ ...form }))
    appStore.showSuccess(t('admin.settings.clientIP.saved'))
  } catch (error) {
    appStore.showError(
      (error as { message?: string })?.message || t('admin.settings.clientIP.saveFailed'),
    )
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void loadSettings()
})
</script>
