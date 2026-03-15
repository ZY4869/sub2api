<template>
  <BaseDialog :show="show" :title="dialogTitle" width="extra-wide" close-on-click-outside @close="emit('close')">
    <div v-if="loading" class="py-12 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('common.loading') }}
    </div>

    <div v-else-if="detail" class="space-y-6">
      <section class="rounded-2xl border border-gray-200 bg-gray-50/70 p-4 dark:border-dark-700 dark:bg-dark-900/70">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <ModelCatalogModelLabel
            :model="detail.model"
            :display-name="detail.display_name"
            :icon-key="detail.icon_key"
            :provider="detail.provider"
            :platforms="detail.default_platforms"
          />
          <button
            v-if="view === 'official'"
            class="btn btn-secondary"
            :disabled="saving"
            @click="emit('copy-official-to-sale', detail.model)"
          >
            {{ t('admin.models.copyToSale') }}
          </button>
        </div>
      </section>

      <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900">
          <p class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.meta.provider') }}</p>
          <div class="mt-2">
            <span class="inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-xs font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300">
              {{ formatProvider(detail.provider) }}
            </span>
          </div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900">
          <p class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.meta.mode') }}</p>
          <div class="mt-2">
            <span class="inline-flex rounded-full bg-indigo-100 px-2.5 py-1 text-xs font-medium text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300">
              {{ formatMode(detail.mode) }}
            </span>
          </div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900">
          <p class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.meta.defaultPlatforms') }}</p>
          <div class="mt-2">
            <ModelPlatformsInline :platforms="detail.default_platforms" />
          </div>
        </div>
        <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900">
          <p class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.meta.pricingSource') }}</p>
          <div class="mt-2">
            <span :class="sourceClass(detail.pricing_source)">{{ t(`admin.models.sources.${detail.pricing_source}`) }}</span>
          </div>
        </div>
      </section>

      <ModelCatalogPricingComparison
        :detail="detail"
        :view="view"
        :exchange-rate="exchangeRate"
        :price-display-mode="priceDisplayMode"
      />

      <ModelCatalogPricingEditorSection
        v-if="view === 'official'"
        :detail="detail"
        layer="official"
        :saving="saving"
        @save="emit('save-official', $event)"
        @reset="emit('reset-official', $event)"
      />
      <ModelCatalogPricingEditorSection
        v-else
        :detail="detail"
        layer="sale"
        :saving="saving"
        @save="emit('save-sale', $event)"
        @reset="emit('reset-sale', $event)"
      />

      <section class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
        <div class="mb-3 flex items-center justify-between gap-2">
          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.models.routeReferences') }}</h4>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.models.routeReferenceCount', { count: detail.route_reference_count }) }}</span>
        </div>
        <div v-if="detail.route_references.length" class="space-y-3">
          <div v-for="reference in detail.route_references" :key="`${reference.group_id}-${reference.group_name}`" class="rounded-lg bg-gray-50 p-3 dark:bg-dark-800">
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div>
                <p class="text-sm font-medium text-gray-900 dark:text-white">{{ reference.group_name }}</p>
                <p class="text-xs text-gray-500 dark:text-gray-400">{{ reference.platform }}</p>
              </div>
              <div class="flex flex-wrap gap-2">
                <span v-for="type in reference.reference_types" :key="type" class="rounded-full bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-500/15 dark:text-primary-300">
                  {{ t(`admin.models.referenceTypes.${type}`) }}
                </span>
              </div>
            </div>
            <p v-if="reference.matched_routing_patterns?.length" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
              {{ reference.matched_routing_patterns.join(', ') }}
            </p>
          </div>
        </div>
        <p v-else class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.models.noRouteReferences') }}</p>
      </section>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelCatalogDetail, ModelCatalogExchangeRate, UpdatePricingOverridePayload } from '@/api/admin/models'
import type { ModelCatalogPricingLayer } from '@/composables/useModelCatalogPage'
import type { ModelCatalogPriceDisplayMode } from '@/utils/modelCatalogPresentation'
import { formatModelCatalogProvider } from '@/utils/modelCatalogPresentation'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ModelPlatformsInline from '@/components/common/ModelPlatformsInline.vue'
import ModelCatalogModelLabel from './ModelCatalogModelLabel.vue'
import ModelCatalogPricingComparison from './ModelCatalogPricingComparison.vue'
import ModelCatalogPricingEditorSection from './ModelCatalogPricingEditorSection.vue'

const props = withDefaults(
  defineProps<{
    show: boolean
    detail: ModelCatalogDetail | null
    loading: boolean
    saving: boolean
    view: ModelCatalogPricingLayer
    exchangeRate?: ModelCatalogExchangeRate | null
    priceDisplayMode?: ModelCatalogPriceDisplayMode
  }>(),
  {
    exchangeRate: null,
    priceDisplayMode: 'usd'
  }
)

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save-official', payload: UpdatePricingOverridePayload): void
  (e: 'reset-official', model: string): void
  (e: 'save-sale', payload: UpdatePricingOverridePayload): void
  (e: 'reset-sale', model: string): void
  (e: 'copy-official-to-sale', model: string): void
}>()

const { t } = useI18n()
const dialogTitle = computed(() => props.detail ? props.detail.display_name || props.detail.model : t('admin.models.detail'))

function formatProvider(provider?: string) {
  return formatModelCatalogProvider(provider)
}

function formatMode(mode?: string) {
  const labels: Record<string, string> = {
    chat: t('admin.models.modes.chat'),
    image: t('admin.models.modes.image'),
    video: t('admin.models.modes.video'),
    prompt_enhance: t('admin.models.modes.promptEnhance')
  }
  return mode ? labels[mode] || mode : '-'
}

function sourceClass(source?: string) {
  const classes: Record<string, string> = {
    override: 'inline-flex rounded-full bg-primary-100 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-500/15 dark:text-primary-300',
    dynamic: 'inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-xs font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-300',
    fallback: 'inline-flex rounded-full bg-amber-100 px-2.5 py-1 text-xs font-medium text-amber-700 dark:bg-amber-500/15 dark:text-amber-300',
    none: 'inline-flex rounded-full bg-gray-100 px-2.5 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
  return classes[source || 'none'] || classes.none
}
</script>
