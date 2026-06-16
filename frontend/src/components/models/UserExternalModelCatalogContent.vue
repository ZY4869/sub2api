<template>
  <div class="mx-auto max-w-[1700px] space-y-6 px-1" data-testid="user-model-catalog">
    <section class="rounded-3xl border border-slate-200 bg-white/90 p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900/80">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="max-w-3xl">
          <p class="text-xs font-semibold uppercase tracking-[0.18em] text-sky-700 dark:text-sky-300">
            {{ t('ui.modelCatalog.eyebrow') }}
          </p>
          <h1 class="mt-3 text-3xl font-semibold tracking-tight text-slate-950 dark:text-white">
            {{ effectiveMode === 'group_first' ? t('ui.modelCatalog.groupFirst.title') : t('nav.modelsCatalog') }}
          </h1>
          <p class="mt-3 text-sm leading-7 text-slate-700 dark:text-slate-200">
            {{ effectiveMode === 'group_first' ? t('ui.modelCatalog.groupFirst.description') : t('ui.modelCatalog.description') }}
          </p>
        </div>
        <button
          type="button"
          class="btn btn-secondary"
          :disabled="loading"
          data-testid="user-models-refresh"
          @click="refresh"
        >
          {{ loading ? t('ui.modelCatalog.refreshing') : t('ui.modelCatalog.refresh') }}
        </button>
      </div>
    </section>

    <div
      v-if="errorMessage"
      class="rounded-3xl border border-rose-200 bg-rose-50 px-6 py-4 text-sm text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-200"
    >
      {{ errorMessage }}
    </div>

    <div v-if="loading && groups.length === 0" class="rounded-3xl border border-slate-200 bg-white/90 px-6 py-8 text-sm text-slate-500 dark:border-dark-700 dark:bg-dark-900/80 dark:text-slate-300">
      {{ t('ui.modelCatalog.groupFirst.loading') }}
    </div>

    <template v-else-if="effectiveMode === 'group_first' && !selectedGroup">
      <section
        v-if="groups.length > 0"
        class="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
        data-testid="user-model-group-list"
      >
        <button
          v-for="group in groups"
          :key="group.id"
          type="button"
          class="rounded-2xl border border-slate-200 bg-white p-5 text-left shadow-sm transition hover:border-primary-300 hover:bg-primary-50/40 dark:border-dark-700 dark:bg-dark-900 dark:hover:border-primary-500/60 dark:hover:bg-primary-500/10"
          :data-testid="`user-model-group-${group.id}`"
          @click="selectGroup(group.id)"
        >
          <div class="flex items-center justify-between gap-3">
            <div class="min-w-0">
              <div class="truncate text-base font-semibold text-slate-950 dark:text-white">{{ group.name }}</div>
              <div class="mt-1 text-xs text-slate-500 dark:text-slate-400">{{ group.platform }}</div>
            </div>
            <span class="rounded-full border border-slate-200 px-3 py-1 text-xs font-medium text-slate-600 dark:border-dark-700 dark:text-slate-300">
              {{ groupModelCountLabel(group.id) }}
            </span>
          </div>
          <p v-if="group.description" class="mt-3 line-clamp-2 text-sm text-slate-600 dark:text-slate-300">
            {{ group.description }}
          </p>
          <div class="mt-4 text-sm font-medium text-primary-700 dark:text-primary-200">
            {{ t('ui.modelCatalog.groupFirst.selectGroup') }}
          </div>
        </button>
      </section>

      <div
        v-else
        class="rounded-3xl border border-dashed border-slate-300 bg-white/80 px-6 py-12 text-center text-sm text-slate-500 dark:border-dark-700 dark:bg-dark-900/70 dark:text-slate-400"
      >
        {{ t('ui.modelCatalog.groupFirst.emptyGroups') }}
      </div>
    </template>

    <template v-else>
      <div v-if="selectedGroup" class="flex flex-wrap items-center justify-between gap-3 rounded-3xl border border-slate-200 bg-white/90 px-5 py-4 text-sm shadow-sm dark:border-dark-700 dark:bg-dark-900/80">
        <span class="font-medium text-slate-700 dark:text-slate-200">
          {{ t('ui.modelCatalog.groupFirst.selectedGroup', { name: selectedGroup.name }) }}
        </span>
        <button type="button" class="btn btn-secondary btn-sm" data-testid="user-models-back-to-groups" @click="selectedGroupID = null">
          {{ t('ui.modelCatalog.groupFirst.backToGroups') }}
        </button>
      </div>
      <UserModelCatalogGrid :items="currentItems" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import userGroupsAPI from '@/api/groups'
import type { PublicModelCatalogItem } from '@/api/meta'
import type { ExternalModelCatalogGroupSummary } from '@/api/groups'
import type { EffectiveExternalModelCatalogViewMode } from '@/types'
import UserModelCatalogGrid from '@/components/models/UserModelCatalogGrid.vue'
import { useAuthStore } from '@/stores/auth'
import { resolveEffectiveExternalModelCatalogViewMode } from '@/utils/externalModelCatalogViewMode'
import {
  MODEL_CATALOG_PUBLISHED_EVENT,
  subscribeModelCatalogPublishedEvents,
  type ModelCatalogPublishedEventSubscription,
} from '@/utils/modelCatalogPublishedEvent'

const { t } = useI18n()
const authStore = useAuthStore()
const groups = ref<ExternalModelCatalogGroupSummary[]>([])
const catalogsByGroup = ref<Record<number, PublicModelCatalogItem[]>>({})
const modelOnlyItems = ref<PublicModelCatalogItem[]>([])
const serverEffectiveMode = ref<EffectiveExternalModelCatalogViewMode | null>(null)
const selectedGroupID = ref<number | null>(null)
const loading = ref(false)
const errorMessage = ref('')
let eventSubscription: ModelCatalogPublishedEventSubscription | null = null

const effectiveMode = computed(() =>
  serverEffectiveMode.value || resolveEffectiveExternalModelCatalogViewMode(authStore.user),
)
const selectedGroup = computed(() => groups.value.find((group) => group.id === selectedGroupID.value) || null)
const currentItems = computed(() => {
  if (effectiveMode.value === 'group_first' && selectedGroupID.value) {
    return catalogsByGroup.value[selectedGroupID.value] || []
  }
  return modelOnlyItems.value
})

watch(effectiveMode, () => {
  selectedGroupID.value = null
  void refresh()
})

onMounted(() => {
  void refresh()
  eventSubscription = subscribeModelCatalogPublishedEvents()
  window.addEventListener('focus', handleCatalogRefreshSignal)
  window.addEventListener(MODEL_CATALOG_PUBLISHED_EVENT, handleCatalogRefreshSignal)
})

onUnmounted(() => {
  eventSubscription?.close()
  eventSubscription = null
  window.removeEventListener('focus', handleCatalogRefreshSignal)
  window.removeEventListener(MODEL_CATALOG_PUBLISHED_EVENT, handleCatalogRefreshSignal)
})

async function refresh() {
  loading.value = true
  errorMessage.value = ''
  try {
    const view = await userGroupsAPI.getExternalModelCatalog()
    serverEffectiveMode.value = view.effective_external_model_catalog_view_mode
    groups.value = view.groups || []
    modelOnlyItems.value = view.items || []
    catalogsByGroup.value = normalizeGroupCatalogs(view.group_catalogs || {})
    if (selectedGroupID.value && !groups.value.some((group) => group.id === selectedGroupID.value)) {
      selectedGroupID.value = null
    }
  } catch {
    errorMessage.value = t('ui.modelCatalog.groupFirst.loadFailed')
  } finally {
    loading.value = false
  }
}

async function selectGroup(groupID: number) {
  selectedGroupID.value = groupID
}

function groupModelCountLabel(groupID: number): string {
  const group = groups.value.find((item) => item.id === groupID)
  return t('ui.modelCatalog.groupFirst.modelCount', {
    count: group?.model_count || catalogsByGroup.value[groupID]?.length || 0,
  })
}

function normalizeGroupCatalogs(input: Record<string, PublicModelCatalogItem[]>): Record<number, PublicModelCatalogItem[]> {
  const output: Record<number, PublicModelCatalogItem[]> = {}
  for (const [key, items] of Object.entries(input)) {
    const groupID = Number(key)
    if (!Number.isFinite(groupID) || groupID <= 0) {
      continue
    }
    output[groupID] = items || []
  }
  return output
}

function handleCatalogRefreshSignal() {
  void refresh()
}
</script>
