<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-medium text-gray-900 dark:text-white">
        {{ t('profile.editProfile') }}
      </h2>
    </div>
    <div class="px-6 py-6">
      <form @submit.prevent="handleUpdateProfile" class="space-y-4">
        <div>
          <label for="username" class="input-label">
            {{ t('profile.username') }}
          </label>
          <input
            id="username"
            v-model="username"
            type="text"
            class="input"
            :placeholder="t('profile.enterUsername')"
          />
        </div>

        <div>
          <label class="input-label">
            {{ t('profile.usageModelDisplayMode') }}
          </label>
          <UsageModelDisplayModeToggle
            v-model="usageModelDisplayMode"
            :show-label="false"
          />
        </div>

        <div>
          <label class="input-label">
            {{ t('profile.visualPresetPreference') }}
          </label>
          <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
            <button
              v-for="option in visualPresetOptions"
              :key="option.value"
              type="button"
              class="rounded-xl border px-3 py-2 text-sm font-medium transition"
              :class="
                visualPresetPreference === option.value
                  ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-500/15 dark:text-primary-200'
                  : 'border-gray-200 bg-white text-gray-700 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700'
              "
              @click="visualPresetPreference = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>

        <div class="flex justify-end pt-4">
          <button type="submit" :disabled="loading" class="btn btn-primary">
            {{ loading ? t('profile.updating') : t('profile.updateProfile') }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import UsageModelDisplayModeToggle from '@/components/common/UsageModelDisplayModeToggle.vue'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { userAPI } from '@/api'
import type { VisualPresetPreference } from '@/types'
import {
  normalizeUsageModelDisplayMode,
} from '@/utils/usageModelPresentation'
import { normalizeVisualPresetPreference } from '@/utils/visualPreset'

const props = defineProps<{
  initialUsername: string
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const username = ref(props.initialUsername)
const usageModelDisplayMode = ref(
  normalizeUsageModelDisplayMode(authStore.user?.usage_model_display_mode)
)
const visualPresetPreference = ref<VisualPresetPreference>(
  normalizeVisualPresetPreference(authStore.user?.visual_preset_preference)
)
const loading = ref(false)
const visualPresetOptions = [
  { value: 'inherit', label: t('profile.visualPresetFollowSite') },
  { value: 'classic', label: t('profile.visualPresetClassic') },
  { value: 'airy', label: t('profile.visualPresetAiry') },
] as const

watch(() => props.initialUsername, (val) => {
  username.value = val
})

watch(
  () => authStore.user?.usage_model_display_mode,
  (val) => {
    usageModelDisplayMode.value = normalizeUsageModelDisplayMode(val)
  }
)

watch(
  () => authStore.user?.visual_preset_preference,
  (val) => {
    visualPresetPreference.value = normalizeVisualPresetPreference(val)
  }
)

const handleUpdateProfile = async () => {
  if (!username.value.trim()) {
    appStore.showError(t('profile.usernameRequired'))
    return
  }

  loading.value = true
  try {
    const updatedUser = await userAPI.updateProfile({
      username: username.value,
      usage_model_display_mode: usageModelDisplayMode.value,
      visual_preset_preference: visualPresetPreference.value,
    })
    authStore.setCurrentUser(updatedUser)
    appStore.showSuccess(t('profile.updateSuccess'))
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('profile.updateFailed'))
  } finally {
    loading.value = false
  }
}
</script>
