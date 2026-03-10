<template>
  <BaseDialog :show="show" :title="dialogTitle" width="extra-wide" close-on-click-outside @close="emit('close')">
    <div v-if="loading" class="py-12 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('common.loading') }}
    </div>

    <div v-else-if="detail" class="space-y-6">
      <section class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <p class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.meta.provider') }}</p>
          <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ detail.provider || '-' }}</p>
        </div>
        <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <p class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.meta.mode') }}</p>
          <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ detail.mode || '-' }}</p>
        </div>
        <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <p class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.meta.defaultPlatforms') }}</p>
          <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ (detail.default_platforms || []).join(', ') || '-' }}</p>
        </div>
        <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <p class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.meta.pricingSource') }}</p>
          <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ t(`admin.models.sources.${detail.pricing_source}`) }}</p>
        </div>
      </section>

      <section class="grid gap-4 md:grid-cols-3">
        <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <p class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.meta.promptCaching') }}</p>
          <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ detail.supports_prompt_caching ? t('common.yes') : t('common.no') }}</p>
        </div>
        <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <p class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.meta.serviceTier') }}</p>
          <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ detail.supports_service_tier ? t('common.yes') : t('common.no') }}</p>
        </div>
        <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <p class="text-xs uppercase tracking-wider text-gray-500 dark:text-gray-400">{{ t('admin.models.meta.longContext') }}</p>
          <p class="mt-2 text-sm font-medium text-gray-900 dark:text-white">{{ detail.long_context_input_token_threshold || '-' }}</p>
        </div>
      </section>

      <ModelCatalogPricingComparison :detail="detail" />
      <ModelCatalogPricingEditor :detail="detail" :saving="saving" @save="emit('save', $event)" @reset="emit('reset', $event)" />

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
import type { ModelCatalogDetail, UpdatePricingOverridePayload } from '@/api/admin/models'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ModelCatalogPricingComparison from './ModelCatalogPricingComparison.vue'
import ModelCatalogPricingEditor from './ModelCatalogPricingEditor.vue'

const props = defineProps<{
  show: boolean
  detail: ModelCatalogDetail | null
  loading: boolean
  saving: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', payload: UpdatePricingOverridePayload): void
  (e: 'reset', model: string): void
}>()

const { t } = useI18n()

const dialogTitle = computed(() =>
  props.detail ? t('admin.models.detailTitle', { model: props.detail.model }) : t('admin.models.detail')
)
</script>
