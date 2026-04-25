<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <div class="flex items-start justify-between gap-4">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('channelStatus.title') }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('channelStatus.description') }}
              </p>
            </div>

            <div v-if="featureEnabled" class="flex items-center gap-2">
              <button
                type="button"
                class="rounded-lg border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700 hover:bg-gray-50 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700"
                :disabled="loading"
                @click="refreshAll"
              >
                {{ t('channelStatus.refresh') }}
              </button>

              <div class="flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-3 py-2 dark:border-dark-700 dark:bg-dark-800">
                <span class="text-sm text-gray-700 dark:text-gray-200">{{ t('channelStatus.autoRefresh') }}</span>
                <Toggle v-model="autoRefresh" />
              </div>
            </div>
          </div>
        </div>

        <div class="p-6">
          <ChannelStatusCards
            :loading="loading"
            :feature-enabled="featureEnabled"
            :items="items"
            @openDetail="openDetail"
          />
        </div>
      </div>
    </div>

    <ChannelStatusDetailDialog
      :open="detailOpen"
      :loading="detailLoading"
      :detail="detail"
      @close="closeDetail"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import AppLayout from '@/components/layout/AppLayout.vue'
import Toggle from '@/components/common/Toggle.vue'
import ChannelStatusCards from '@/components/user/channel-status/ChannelStatusCards.vue'
import ChannelStatusDetailDialog from '@/components/user/channel-status/ChannelStatusDetailDialog.vue'
import { channelMonitorsAPI, type ChannelMonitorUserDetail, type ChannelMonitorUserListItem } from '@/api/channelMonitors'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const items = ref<ChannelMonitorUserListItem[]>([])

const detailOpen = ref(false)
const detailLoading = ref(false)
const detail = ref<ChannelMonitorUserDetail | null>(null)
const selectedId = ref<number | null>(null)

const autoRefresh = ref(true)
const refreshIntervalMs = ref(60_000)
let refreshTimer: number | null = null

const featureEnabled = computed(() => appStore.cachedPublicSettings?.channel_monitor_enabled === true)

async function fetchPublicSettingsIfNeeded() {
  if (appStore.publicSettingsLoaded) return
  await appStore.fetchPublicSettings()
}

async function loadList() {
  if (!featureEnabled.value) {
    items.value = []
    return
  }

  loading.value = true
  try {
    items.value = await channelMonitorsAPI.getChannelMonitors()
  } catch (err: any) {
    appStore.showError((err?.message || t('common.unknownError')) as string)
    items.value = []
  } finally {
    loading.value = false
  }
}

async function loadDetail(id: number) {
  detailLoading.value = true
  try {
    detail.value = await channelMonitorsAPI.getChannelMonitorStatus(id)
  } catch (err: any) {
    appStore.showError((err?.message || t('common.unknownError')) as string)
    detail.value = null
  } finally {
    detailLoading.value = false
  }
}

async function refreshAll() {
  await loadList()
  if (detailOpen.value && selectedId.value != null) {
    await loadDetail(selectedId.value)
  }
}

function openDetail(id: number) {
  selectedId.value = id
  detailOpen.value = true
  loadDetail(id)
}

function closeDetail() {
  detailOpen.value = false
  detail.value = null
  selectedId.value = null
}

function clearTimer() {
  if (refreshTimer != null) {
    window.clearInterval(refreshTimer)
    refreshTimer = null
  }
}

function setupTimer() {
  clearTimer()
  if (!featureEnabled.value || !autoRefresh.value) return
  refreshTimer = window.setInterval(() => {
    refreshAll()
  }, refreshIntervalMs.value)
}

watch([autoRefresh, featureEnabled, refreshIntervalMs], setupTimer, { immediate: true })

onMounted(async () => {
  loading.value = true
  try {
    await fetchPublicSettingsIfNeeded()
  } finally {
    loading.value = false
  }
  await loadList()
})

onUnmounted(() => {
  clearTimer()
})
</script>
