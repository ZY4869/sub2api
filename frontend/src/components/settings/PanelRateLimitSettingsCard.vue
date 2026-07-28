<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.panelRateLimit.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.settings.panelRateLimit.description') }}
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
              {{ t('admin.settings.panelRateLimit.enabled') }}
            </label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.panelRateLimit.enabledHint') }}
            </p>
          </div>
          <Toggle v-model="form.enabled" />
        </div>

        <div class="grid gap-4 md:grid-cols-3">
          <label class="space-y-2">
            <span class="input-label">{{ t('admin.settings.panelRateLimit.userRpm') }}</span>
            <input
              v-model.number="form.user_rpm"
              type="number"
              min="1"
              max="100000"
              class="input"
            />
          </label>

          <label class="space-y-2">
            <span class="input-label">{{ t('admin.settings.panelRateLimit.heavyRpm') }}</span>
            <input
              v-model.number="form.heavy_rpm"
              type="number"
              min="1"
              max="100000"
              class="input"
            />
          </label>

          <label class="space-y-2">
            <span class="input-label">{{ t('admin.settings.panelRateLimit.publicIpRpm') }}</span>
            <input
              v-model.number="form.public_ip_rpm"
              type="number"
              min="1"
              max="100000"
              class="input"
            />
          </label>
        </div>

        <div class="flex items-center justify-between gap-4 border-t border-gray-100 pt-4 dark:border-dark-700">
          <div>
            <label class="font-medium text-gray-900 dark:text-white">
              {{ t('admin.settings.panelRateLimit.exemptAdmin') }}
            </label>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.panelRateLimit.exemptAdminHint') }}
            </p>
          </div>
          <Toggle v-model="form.exempt_admin" />
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
import type { PanelRateLimitSettings } from '@/api/admin/settings'
import Toggle from '@/components/common/Toggle.vue'
import { useAppStore } from '@/stores'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(true)
const saving = ref(false)

const form = reactive<PanelRateLimitSettings>({
  enabled: false,
  user_rpm: 240,
  heavy_rpm: 60,
  public_ip_rpm: 300,
  exempt_admin: true,
})

const normalizeRPM = (value: number, fallback: number) => {
  const numeric = Number(value)
  if (!Number.isFinite(numeric) || numeric <= 0) {
    return fallback
  }
  return Math.min(Math.trunc(numeric), 100000)
}

const applySettings = (settings: PanelRateLimitSettings) => {
  form.enabled = Boolean(settings.enabled)
  form.user_rpm = normalizeRPM(settings.user_rpm, 240)
  form.heavy_rpm = normalizeRPM(settings.heavy_rpm, 60)
  form.public_ip_rpm = normalizeRPM(settings.public_ip_rpm, 300)
  form.exempt_admin = settings.exempt_admin !== false
}

const loadSettings = async () => {
  loading.value = true
  try {
    applySettings(await adminAPI.settings.getPanelRateLimitSettings())
  } catch (error) {
    appStore.showError(
      (error as { message?: string })?.message || t('admin.settings.panelRateLimit.loadFailed'),
    )
  } finally {
    loading.value = false
  }
}

const saveSettings = async () => {
  saving.value = true
  try {
    const payload: PanelRateLimitSettings = {
      enabled: form.enabled,
      user_rpm: normalizeRPM(form.user_rpm, 240),
      heavy_rpm: normalizeRPM(form.heavy_rpm, 60),
      public_ip_rpm: normalizeRPM(form.public_ip_rpm, 300),
      exempt_admin: form.exempt_admin,
    }
    applySettings(await adminAPI.settings.updatePanelRateLimitSettings(payload))
    appStore.showSuccess(t('admin.settings.panelRateLimit.saved'))
  } catch (error) {
    appStore.showError(
      (error as { message?: string })?.message || t('admin.settings.panelRateLimit.saveFailed'),
    )
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  void loadSettings()
})
</script>
