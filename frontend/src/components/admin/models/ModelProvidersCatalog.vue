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

          <button class="btn btn-secondary" :disabled="loading" @click="refreshAll">
            {{ t('common.refresh') }}
          </button>
        </div>
      </div>
    </template>

    <template #table>
      <div v-if="loading" class="flex items-center justify-center py-10">
        <LoadingSpinner />
      </div>

      <template v-else>
        <ModelProvidersGrid v-if="viewMode === 'grid'" :providers="providerGroups" @open="openProviderDialog" />
        <ModelProvidersList
          v-else
          :providers="providerGroups"
          :is-activating="isActivating"
          @activate="activateModel"
        />
      </template>
    </template>
  </TablePageLayout>

  <ModelProviderDialog
    :show="providerDialogOpen"
    :loading="loading"
    :providers="providerGroups"
    :active-provider="activeProvider"
    :is-activating="isActivating"
    @close="providerDialogOpen = false"
    @refresh="refreshAll"
    @select-provider="activeProvider = $event"
    @activate="activateModel"
  />
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import { useAdminModelRegistryProviders } from '@/composables/useAdminModelRegistryProviders'
import ModelProviderDialog from '@/components/admin/models/ModelProviderDialog.vue'
import ModelProvidersGrid from '@/components/admin/models/ModelProvidersGrid.vue'
import ModelProvidersList from '@/components/admin/models/ModelProvidersList.vue'

const { t } = useI18n()

const viewMode = ref<'grid' | 'list'>('grid')
const providerDialogOpen = ref(false)
const activeProvider = ref('')

const {
  loading,
  providerGroups,
  isActivating,
  loadAll,
  refreshAll,
  activateModel
} = useAdminModelRegistryProviders()

const firstProvider = computed(() => providerGroups.value[0]?.provider || '')

onMounted(() => {
  void loadAll()
})

watch(
  () => providerGroups.value.length,
  () => {
    if (!activeProvider.value) {
      activeProvider.value = firstProvider.value
    }
  },
  { immediate: true }
)

function openProviderDialog(provider: string) {
  activeProvider.value = provider
  providerDialogOpen.value = true
}
</script>

