<template>
  <TablePageLayout>
    <template #actions>
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div class="rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300">
          {{ t('admin.models.pages.all.description') }}
        </div>

        <div class="flex flex-wrap items-center gap-3">
          <div class="flex items-center gap-1 rounded-2xl border border-gray-200 bg-white p-1 shadow-sm dark:border-dark-700 dark:bg-dark-800">
            <button
              type="button"
              class="rounded-xl px-4 py-2 text-sm font-medium transition-colors"
              :class="viewMode === 'grid' ? 'bg-primary-600 text-white shadow-sm' : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white'"
              @click="viewMode = 'grid'"
            >
              {{ t('admin.models.pages.all.viewModes.grid') }}
            </button>
            <button
              type="button"
              class="rounded-xl px-4 py-2 text-sm font-medium transition-colors"
              :class="viewMode === 'list' ? 'bg-primary-600 text-white shadow-sm' : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white'"
              @click="viewMode = 'list'"
            >
              {{ t('admin.models.pages.all.viewModes.list') }}
            </button>
          </div>

          <button class="btn btn-secondary" :disabled="loading" @click="handleRefreshAll">
            {{ t('common.refresh') }}
          </button>
        </div>
      </div>
    </template>

    <template #table>
      <div v-if="loading" class="flex items-center justify-center py-10">
        <LoadingSpinner />
      </div>

      <div v-else-if="providerGroups.length === 0" class="p-8">
        <EmptyState
          :title="t('admin.models.registry.emptyTitle')"
          :description="t('admin.models.registry.emptyDescription')"
        />
      </div>

      <template v-else>
        <ModelProvidersGrid v-if="viewMode === 'grid'" :providers="providerGroups" @open="openProviderDialog" />
        <ModelProvidersList
          v-else
          :providers="providerGroups"
          :get-models="getProviderModels"
          :is-provider-loading="isProviderLoading"
          :provider-has-more-models="providerHasMoreModels"
          :is-activating="isActivating"
          @expand="ensureProviderModels"
          @load-more="loadMoreProviderModels"
          @activate="activateModel"
        />

        <div class="flex flex-col items-center gap-3 px-4 pb-6 pt-2">
          <div
            v-if="loadingMore"
            class="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400"
          >
            <LoadingSpinner />
            <span>{{ t('common.loading') }}</span>
          </div>
          <button
            v-else-if="hasMoreProviders"
            type="button"
            class="btn btn-secondary btn-sm"
            @click="loadMoreProviders"
          >
            {{ t('admin.models.pages.all.loadMore') }}
          </button>
          <div ref="loadMoreSentinel" class="h-1 w-full" />
        </div>
      </template>
    </template>
  </TablePageLayout>

  <ModelProviderDialog
    :show="providerDialogOpen"
    :loading="isProviderLoading(activeProvider)"
    :providers="providerGroups"
    :active-provider="activeProvider"
    :active-models="getProviderModels(activeProvider)"
    :has-more="providerHasMoreModels(activeProvider)"
    :is-activating="isActivating"
    @close="providerDialogOpen = false"
    @refresh="handleRefreshAll"
    @select-provider="selectProvider"
    @load-more="loadMoreProviderModels"
    @activate="activateModel"
  />
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { useAdminModelRegistryProviders } from '@/composables/useAdminModelRegistryProviders'
import ModelProviderDialog from '@/components/admin/models/ModelProviderDialog.vue'
import ModelProvidersGrid from '@/components/admin/models/ModelProvidersGrid.vue'
import ModelProvidersList from '@/components/admin/models/ModelProvidersList.vue'

const { t } = useI18n()

const viewMode = ref<'grid' | 'list'>('grid')
const providerDialogOpen = ref(false)
const activeProvider = ref('')
const loadMoreSentinel = ref<HTMLElement | null>(null)
let loadMoreObserver: IntersectionObserver | null = null

const {
  loading,
  loadingMore,
  providerGroups,
  hasMoreProviders,
  isActivating,
  loadAll,
  loadMoreProviders,
  refreshAll,
  ensureProviderModels,
  loadMoreProviderModels,
  getProviderModels,
  isProviderLoading,
  providerHasMoreModels,
  activateModel
} = useAdminModelRegistryProviders()

const firstProvider = computed(() => providerGroups.value[0]?.provider || '')

onMounted(() => {
  void loadAll()
  if (typeof IntersectionObserver !== 'undefined') {
    loadMoreObserver = new IntersectionObserver((entries) => {
      if (!entries.some((entry) => entry.isIntersecting)) {
        return
      }
      if (loading.value || loadingMore.value || !hasMoreProviders.value) {
        return
      }
      void loadMoreProviders()
    })
  }
})

onUnmounted(() => {
  loadMoreObserver?.disconnect()
})

watch(
  () => providerGroups.value.map((group) => group.provider).join('|'),
  () => {
    if (!providerGroups.value.length) {
      activeProvider.value = ''
      return
    }
    if (!activeProvider.value || !providerGroups.value.some((group) => group.provider === activeProvider.value)) {
      activeProvider.value = firstProvider.value
    }
  },
  { immediate: true }
)

watch(loadMoreSentinel, (next, previous) => {
  if (!loadMoreObserver) {
    return
  }
  if (previous) {
    loadMoreObserver.unobserve(previous)
  }
  if (next) {
    loadMoreObserver.observe(next)
  }
})

async function openProviderDialog(provider: string) {
  activeProvider.value = provider
  providerDialogOpen.value = true
  await ensureProviderModels(provider)
}

async function selectProvider(provider: string) {
  activeProvider.value = provider
  await ensureProviderModels(provider)
}

async function handleRefreshAll() {
  await refreshAll()
  if (!providerDialogOpen.value || !activeProvider.value) {
    return
  }
  await ensureProviderModels(activeProvider.value)
}

watch(
  () => viewMode.value,
  async () => {
    await nextTick()
    if (loadMoreSentinel.value && loadMoreObserver) {
      loadMoreObserver.observe(loadMoreSentinel.value)
    }
  }
)
</script>
