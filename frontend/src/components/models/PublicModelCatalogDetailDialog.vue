<template>
  <Teleport to="body">
    <Transition name="modal">
      <div
        v-if="show"
        class="fixed inset-0 z-40 bg-slate-900/40 backdrop-blur-sm"
        @click="emit('close')"
      ></div>
    </Transition>

    <Transition name="drawer">
      <aside
        v-if="show"
        class="fixed right-0 top-0 z-50 flex h-full w-full max-w-[1000px] flex-col border-l border-slate-200/80 bg-[#FAFAFA] shadow-2xl dark:border-dark-700 dark:bg-dark-950"
        role="dialog"
        aria-modal="true"
        :aria-label="dialogTitle"
      >
        <PublicModelDetailHeader
          :item="sourceItem"
          :title="dialogTitle"
          :provider-label="providerLabel"
          :status-class="statusDotClass"
          :copy-title="t('ui.modelCatalog.card.copyModelId')"
          :close-title="t('common.close')"
          :hosted-summary="t('ui.modelCatalog.detail.hostedSummary')"
          @copy="copyModelId"
          @close="emit('close')"
        />
        <PublicModelDetailTabs v-model="activeTab" :tabs="tabs" />

        <main class="flex-1 overflow-y-auto bg-slate-50/50 px-6 py-8 dark:bg-dark-950 md:px-10">
          <div v-if="!sourceItem" class="rounded-3xl border border-dashed border-slate-300 p-8 text-center text-sm text-slate-500 dark:border-dark-700 dark:text-slate-400">
            {{ t('ui.modelCatalog.detail.loading') }}
          </div>
          <PublicModelDetailOverview
            v-else-if="activeTab === 'overview'"
            :item="sourceItem"
            :health="props.health"
            :prices="displayItem?.primaryPrices || []"
            :multiplier-label="multiplierLabel"
            :protocol-summary="protocolSummary"
            :labels="overviewLabels"
            :price-entry-label="renderPriceEntryLabel"
            :format-catalog-price="renderPrice"
          />
          <PublicModelDetailMonitor
            v-else-if="activeTab === 'monitor'"
            :health="props.health"
            :labels="monitorLabels"
            :t="t"
          />
          <PublicModelDetailRouting
            v-else
            :labels="routingLabels"
            :endpoints="sourceItem?.protocol_endpoints || []"
            :t="t"
            :params="parameterRows"
          >
            <template #example>
              <PublicModelDetailExamplePanel
                v-model:selected-key-i-d="selectedKeyID"
                :supported-keys="supportedKeys"
                :loading="loading"
                :error-message="errorMessage"
                :example-group="exampleResult.group"
                :docs-theme="docsTheme"
                :protocol="detail?.example_protocol || sourceItem?.provider || 'openai'"
                :example-source="detail?.example_source"
                :key-hint="keyHint"
                :labels="exampleLabels"
              />
            </template>
          </PublicModelDetailRouting>
        </main>
      </aside>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, toRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type {
  PublicModelCatalogItem,
  PublicModelCatalogPriceEntry,
  PublicModelCatalogStatusItem,
} from '@/api/meta'
import { getDocsTheme } from '@/components/docs/docsTheme'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { formatProviderLabel } from '@/utils/providerLabels'
import {
  buildPublicModelCatalogDisplayItem,
  formatCatalogPrice,
  multiplierSummaryLabel,
  priceEntryLabel,
  type PublicModelCatalogDisplayItem,
} from '@/utils/publicModelCatalog'
import PublicModelDetailExamplePanel from './public-catalog/PublicModelDetailExamplePanel.vue'
import PublicModelDetailHeader from './public-catalog/PublicModelDetailHeader.vue'
import PublicModelDetailMonitor from './public-catalog/PublicModelDetailMonitor.vue'
import PublicModelDetailOverview from './public-catalog/PublicModelDetailOverview.vue'
import PublicModelDetailRouting from './public-catalog/PublicModelDetailRouting.vue'
import PublicModelDetailTabs, { type PublicModelDetailTab } from './public-catalog/PublicModelDetailTabs.vue'
import { healthDotClass } from './public-catalog/publicModelCatalogView'
import { usePublicModelDetail } from './public-catalog/usePublicModelDetail'
import { usePublicModelDetailLabels } from './public-catalog/usePublicModelDetailLabels'

type DetailTab = 'overview' | 'monitor' | 'routing'

const props = defineProps<{
  show: boolean
  model: string | null
  catalogItem?: PublicModelCatalogItem | null
  health?: PublicModelCatalogStatusItem
  usdToCnyRate?: number | null
}>()

const emit = defineEmits<{ close: [] }>()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const activeTab = ref<DetailTab>('overview')
const baseUrl = computed(resolvedBaseUrl)
const {
  overviewLabels,
  monitorLabels,
  routingLabels,
  exampleLabels,
  parameterRows,
} = usePublicModelDetailLabels(t)

const {
  detail,
  loading,
  errorMessage,
  selectedKey,
  selectedKeyID,
  supportedKeys,
  exampleResult,
} = usePublicModelDetail({
  show: toRef(props, 'show'),
  model: toRef(props, 'model'),
  isAuthenticated: computed(() => authStore.isAuthenticated),
  baseUrl,
  missingKey: 'sk-your-key',
  resolveErrorMessage: (error) => resolveErrorMessage(error, t('ui.modelCatalog.detail.loadFailed')),
})

const sourceItem = computed(() => detail.value?.item || props.catalogItem || null)
const displayItem = computed<PublicModelCatalogDisplayItem | null>(() =>
  sourceItem.value ? buildPublicModelCatalogDisplayItem(sourceItem.value) : null,
)
const dialogTitle = computed(() => displayItem.value?.title || sourceItem.value?.model || t('nav.modelsCatalog'))
const providerLabel = computed(() => formatProviderLabel(sourceItem.value?.provider || sourceItem.value?.provider_icon_key || ''))
const statusDotClass = computed(() => healthDotClass(props.health?.health_status || sourceItem.value?.health_status))
const multiplierLabel = computed(() =>
  sourceItem.value ? multiplierSummaryLabel(t, sourceItem.value.multiplier_summary) : '-',
)
const protocolSummary = computed(() => (sourceItem.value?.request_protocols || []).map((protocol) => formatProviderLabel(protocol)).join(' / ') || '-')
const docsTheme = computed(() => getDocsTheme(exampleResult.value.pageId))
const keyHint = computed(() => {
  if (!authStore.isAuthenticated) return t('ui.modelCatalog.detail.keyHintGuest')
  if (selectedKey.value) return t('ui.modelCatalog.detail.keyHintMatched', { name: selectedKey.value.name })
  return t('ui.modelCatalog.detail.keyHintMissing')
})

const tabs = computed<PublicModelDetailTab[]>(() => [
  { id: 'overview', label: t('ui.modelCatalog.detail.tabs.overview'), icon: 'infoCircle' },
  { id: 'monitor', label: t('ui.modelCatalog.detail.tabs.monitor'), icon: 'chart' },
  { id: 'routing', label: t('ui.modelCatalog.detail.tabs.routing'), icon: 'terminal' },
])

watch(
  () => [props.show, props.model] as const,
  ([show]) => {
    if (show) activeTab.value = 'overview'
  },
  { immediate: true },
)

function renderPriceEntryLabel(fieldID: string): string {
  return priceEntryLabel(t, fieldID)
}

function renderPrice(entry: PublicModelCatalogPriceEntry, currency: string): string {
  return formatCatalogPrice(t, entry, currency, props.usdToCnyRate ?? null)
}

async function copyModelId() {
  const modelID = String(sourceItem.value?.model || '').trim()
  if (!modelID) return
  try {
    await navigator.clipboard.writeText(modelID)
    appStore.showSuccess(t('ui.modelCatalog.copySuccess', { model: modelID }))
  } catch {
    appStore.showError(t('ui.modelCatalog.copyFailed'))
  }
}

function resolvedBaseUrl(): string {
  const configured = String(appStore.apiBaseUrl || '').trim()
  if (configured) return configured.replace(/\/+$/g, '')
  if (typeof window !== 'undefined' && window.location?.origin) {
    return window.location.origin.replace(/\/+$/g, '')
  }
  return 'https://api.zyxai.de'
}

function resolveErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error && 'message' in error && typeof (error as { message?: unknown }).message === 'string') {
    return String((error as { message: string }).message)
  }
  return fallback
}
</script>

<style scoped>
.drawer-enter-active,
.drawer-leave-active {
  transition: transform 0.35s ease, opacity 0.35s ease;
}

.drawer-enter-from,
.drawer-leave-to {
  transform: translateX(100%);
  opacity: 0;
}
</style>
