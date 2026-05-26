<template>
  <BaseDialog
    :show="show"
    :title="t('admin.billing.publicCatalog.dialog.title')"
    width="wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <div v-if="draft" class="space-y-5">
      <div class="overflow-hidden rounded-2xl border border-slate-200 bg-slate-50/80 shadow-sm dark:border-dark-700 dark:bg-dark-900/40">
        <div class="flex items-start justify-between gap-4 border-b border-slate-100 bg-white px-5 py-4 dark:border-dark-700 dark:bg-dark-800">
          <div class="flex min-w-0 items-center gap-3">
            <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-slate-100 bg-slate-50 shadow-sm dark:border-dark-700 dark:bg-dark-900">
              <ModelIcon
                :model="baseModel"
                :provider="item?.provider"
                :icon-key="item?.provider_icon_key"
                :display-name="item?.display_name"
                size="28px"
              />
            </div>
            <div class="min-w-0">
              <div class="truncate text-base font-bold text-slate-900 dark:text-white">
                {{ item?.display_name || baseModel }}
              </div>
              <div class="truncate font-mono text-xs text-slate-400 dark:text-slate-500">
                {{ draft.entry_id }}
              </div>
            </div>
          </div>
          <span class="inline-flex shrink-0 items-center gap-1 rounded-lg bg-slate-100 px-2 py-1 text-xs font-medium text-slate-500 dark:bg-dark-700 dark:text-slate-300">
            <ModelPlatformIcon :platform="item?.provider || item?.source_protocol || draft.source_protocol" size="xs" />
            {{ providerLabel }}
          </span>
        </div>

        <div class="grid gap-2 px-5 py-4 text-xs text-slate-500 dark:text-slate-400 sm:grid-cols-3">
          <div class="truncate">{{ t('admin.billing.publicCatalog.card.baseModel', { value: baseModel }) }}</div>
          <div class="truncate">{{ t('admin.billing.publicCatalog.card.protocol', { value: draft.source_protocol || item?.source_protocol || '-' }) }}</div>
          <div class="truncate">{{ t('admin.billing.publicCatalog.card.account', { value: item?.source_account_name || draft.source_alias || '-' }) }}</div>
        </div>
      </div>

      <div class="grid gap-4 md:grid-cols-2">
        <label class="space-y-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          <span>{{ t('admin.billing.publicCatalog.dialog.publicModelId') }}</span>
          <input
            v-model.trim="draft.public_model_id"
            class="input font-mono"
            data-testid="catalog-dialog-public-id"
          />
        </label>
        <label class="space-y-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
          <span>{{ t('admin.billing.publicCatalog.dialog.sourceAlias') }}</span>
          <input
            v-model.trim="draft.source_alias"
            class="input"
            data-testid="catalog-dialog-source-alias"
          />
        </label>
      </div>

      <div class="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="mb-3 flex items-center gap-2 text-sm font-bold text-slate-800 dark:text-white">
          <Icon name="dollar" size="sm" class="text-amber-500" />
          {{ t('admin.billing.publicCatalog.dialog.pricingTitle') }}
        </div>
        <PublicCatalogPriceEditor
          :official="item?.official_price_display || item?.price_display"
          :sale="draft.sale_price_display || item?.sale_price_display || item?.price_display"
          :currency="item?.currency || 'USD'"
          editable
          testid-prefix="catalog-dialog-price"
          @update:sale="draft.sale_price_display = $event"
        />
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('admin.billing.publicCatalog.dialog.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          data-testid="catalog-dialog-save"
          :disabled="!draft?.public_model_id"
          @click="save"
        >
          {{ t('admin.billing.publicCatalog.dialog.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BillingPublicCatalogAdminEntry, BillingPublicCatalogEntryDraft } from '@/api/admin/billing'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { clonePriceDisplay } from './publicCatalogPricing'
import PublicCatalogPriceEditor from './PublicCatalogPriceEditor.vue'

const props = defineProps<{
  show: boolean
  item: BillingPublicCatalogAdminEntry | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', entryID: string, patch: Partial<BillingPublicCatalogEntryDraft>): void
}>()

const draft = ref<BillingPublicCatalogEntryDraft | null>(null)
const { t } = useI18n()

const baseModel = computed(() => (
  props.item?.base_model || props.item?.source_model_id || props.item?.model || draft.value?.base_model || '-'
))
const providerLabel = computed(() => formatProviderName(props.item?.provider || props.item?.source_protocol || draft.value?.source_protocol || '-'))

watch(
  () => [props.show, props.item] as const,
  () => {
    if (!props.show || !props.item) {
      draft.value = null
      return
    }
    draft.value = {
      entry_id: props.item.entry_id || props.item.model,
      public_model_id: props.item.public_model_id || props.item.model,
      source_account_id: props.item.source_account_id,
      source_alias: props.item.source_alias || '',
      source_model_id: props.item.source_model_id || props.item.base_model || props.item.model,
      base_model: props.item.base_model || props.item.source_model_id || props.item.model,
      source_protocol: props.item.source_protocol || props.item.request_protocols?.[0] || '',
      sale_price_display: clonePriceDisplay(props.item.sale_price_display || props.item.price_display),
    }
  },
  { immediate: true },
)

function save() {
  if (!draft.value) return
  emit('save', draft.value.entry_id, {
    public_model_id: draft.value.public_model_id,
    source_alias: draft.value.source_alias,
    source_model_id: draft.value.source_model_id,
    base_model: draft.value.base_model,
    source_protocol: draft.value.source_protocol,
    sale_price_display: clonePriceDisplay(draft.value.sale_price_display),
  })
  emit('close')
}

function formatProviderName(value: string): string {
  const normalized = value.trim().toLowerCase()
  const labels: Record<string, string> = {
    openai: 'OpenAI',
    anthropic: 'Anthropic',
    gemini: 'Gemini',
    google: 'Google',
    deepseek: 'DeepSeek',
  }
  return labels[normalized] || value
}
</script>
