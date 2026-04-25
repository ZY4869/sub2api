<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('availableChannels.title') }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('availableChannels.description') }}
          </p>
        </div>

        <div class="p-6">
          <div v-if="loading" class="flex items-center justify-center py-12">
            <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
          </div>

          <div
            v-else-if="!featureEnabled"
            class="flex items-center justify-center p-10 text-center"
          >
            <div class="max-w-md">
              <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
                <Icon name="globe" size="lg" class="text-gray-400" />
              </div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('availableChannels.notEnabledTitle') }}
              </h3>
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                {{ t('availableChannels.notEnabledDesc') }}
              </p>
            </div>
          </div>

          <div
            v-else-if="items.length === 0"
            class="flex items-center justify-center p-10 text-center"
          >
            <div class="max-w-md">
              <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
                <Icon name="database" size="lg" class="text-gray-400" />
              </div>
              <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('availableChannels.emptyTitle') }}
              </h3>
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
                {{ t('availableChannels.emptyDesc') }}
              </p>
            </div>
          </div>

          <div v-else class="space-y-6">
            <div
              v-for="channel in items"
              :key="channel.name"
              class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800"
            >
              <div class="flex items-start justify-between gap-4">
                <div class="min-w-0">
                  <div class="text-base font-semibold text-gray-900 dark:text-white">
                    {{ channel.name }}
                  </div>
                  <div v-if="channel.description" class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{ channel.description }}
                  </div>
                </div>
              </div>

              <div class="mt-4 space-y-4">
                <div
                  v-for="section in channel.platforms"
                  :key="section.platform"
                  class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900"
                >
                  <div class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100">
                    <ModelPlatformIcon :platform="section.platform" size="sm" />
                    <span class="sr-only">{{ section.platform }}</span>
                    <span class="truncate">{{ section.platform }}</span>
                  </div>

                  <div class="mt-3">
                    <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                      {{ t('availableChannels.groupsLabel') }}
                    </div>
                    <div class="mt-2 flex flex-wrap gap-2">
                      <span
                        v-for="g in section.groups"
                        :key="g.id"
                        class="inline-flex items-center gap-1 rounded-full border border-gray-200 bg-white px-2 py-1 text-xs text-gray-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200"
                      >
                        <span class="max-w-[180px] truncate">{{ g.name }}</span>
                        <span
                          v-if="g.subscription_type === 'subscription'"
                          class="rounded-full bg-blue-50 px-1.5 py-0.5 text-[10px] font-semibold text-blue-700 dark:bg-blue-950 dark:text-blue-200"
                        >
                          {{ t('availableChannels.subscriptionTag') }}
                        </span>
                        <span
                          v-if="g.is_exclusive"
                          class="rounded-full bg-amber-50 px-1.5 py-0.5 text-[10px] font-semibold text-amber-700 dark:bg-amber-950 dark:text-amber-200"
                        >
                          {{ t('availableChannels.exclusiveTag') }}
                        </span>
                      </span>
                    </div>
                  </div>

                  <div class="mt-4">
                    <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
                      {{ t('availableChannels.modelsLabel') }}
                    </div>
                    <div class="mt-2 flex flex-wrap gap-2">
                      <span
                        v-for="m in section.supported_models"
                        :key="`${m.platform}:${m.name}`"
                        class="inline-flex items-center gap-1.5 rounded-full border border-gray-200 bg-white px-2 py-1 text-xs text-gray-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200"
                        :title="m.name"
                      >
                        <ModelIcon
                          :model="m.name"
                          :provider="m.platform"
                          :display-name="m.name"
                          size="14px"
                        />
                        <span class="font-mono">{{ m.name }}</span>
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import { channelsAPI, type AvailableChannel } from '@/api/channels'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const items = ref<AvailableChannel[]>([])

const featureEnabled = computed(() => appStore.cachedPublicSettings?.available_channels_enabled === true)

onMounted(async () => {
  if (!appStore.publicSettingsLoaded) {
    loading.value = true
    try {
      await appStore.fetchPublicSettings()
    } finally {
      loading.value = false
    }
  }

  if (!featureEnabled.value) return

  loading.value = true
  try {
    items.value = await channelsAPI.getAvailableChannels()
  } catch (err: any) {
    appStore.showError(t('availableChannels.loadFailed') + ': ' + (err?.message || t('common.unknownError')))
    items.value = []
  } finally {
    loading.value = false
  }
})
</script>

