<template>
  <article
    class="group relative overflow-hidden rounded-xl border bg-white transition duration-200 dark:bg-dark-800"
    :class="cardClass"
  >
    <div class="flex">
      <div
        v-if="showHandle"
        class="public-catalog-drag-handle flex w-10 shrink-0 cursor-grab flex-col items-center justify-center border-r border-slate-100 bg-slate-50/70 text-slate-300 transition active:cursor-grabbing group-hover:bg-slate-100 group-hover:text-slate-500 dark:border-dark-700 dark:bg-dark-900/50 dark:group-hover:bg-dark-700"
        :aria-label="t('admin.billing.publicCatalog.card.dragLabel')"
        role="button"
      >
        <span class="mb-1 text-[10px] font-bold text-slate-400">{{ indexLabel }}</span>
        <Icon name="arrowsUpDown" size="sm" />
      </div>

      <div class="min-w-0 flex-1 p-4" :class="mode === 'available' ? 'pr-16' : ''">
        <PublicCatalogCardHeader
          :item="item"
          :mode="mode"
          :entry-id="entryId"
          :base-model="baseModel"
          :missing="missing"
          @edit="emit('edit')"
          @remove="emit('remove')"
        />

        <PublicCatalogSelectedFields
          v-if="mode === 'selected'"
          :entry-id="entryId"
          :public-model-label="publicModelLabel"
          :source-alias-label="sourceAliasLabel"
          :duplicate="duplicate"
          @update-public-id="emit('update-public-id', $event)"
          @update-source-alias="emit('update-source-alias', $event)"
        />

        <div class="mt-3 grid gap-2 text-[11px] text-slate-500 dark:text-slate-400 sm:grid-cols-2">
          <div class="truncate">{{ t('admin.billing.publicCatalog.card.baseModel', { value: baseModel }) }}</div>
          <div class="truncate">{{ t('admin.billing.publicCatalog.card.account', { value: sourceContextLabel || '-' }) }}</div>
        </div>

        <div
          v-if="missing"
          class="mt-3 rounded-xl border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-100"
        >
          {{ t('admin.billing.publicCatalog.card.missing') }}
        </div>

        <PublicCatalogPriceEditor
          class="mt-3"
          :official="officialPrice"
          :sale="salePrice"
          :currency="item.currency"
          :editable="mode === 'selected'"
          :testid-prefix="`price-${entryId}`"
          @update:sale="emit('update-sale-price', $event)"
        />

        <PublicCatalogMoveButtons
          v-if="mode === 'selected'"
          :move-up-disabled="moveUpDisabled"
          :move-down-disabled="moveDownDisabled"
          @move-up="emit('move-up')"
          @move-down="emit('move-down')"
        />
      </div>
    </div>

    <div
      v-if="mode === 'available'"
      class="absolute inset-y-0 right-0 flex w-16 items-center justify-end bg-gradient-to-l from-white via-white to-transparent pr-4 opacity-0 transition group-hover:opacity-100 dark:from-dark-800 dark:via-dark-800"
    >
      <button
        type="button"
        class="rounded-full p-2.5 shadow-sm transition"
        :class="alreadySelected ? 'cursor-default bg-slate-100 text-slate-400 dark:bg-dark-700' : 'bg-emerald-50 text-emerald-600 hover:bg-emerald-500 hover:text-white'"
        :disabled="alreadySelected"
        :aria-label="alreadySelected ? t('admin.billing.publicCatalog.card.added') : t('admin.billing.publicCatalog.card.add')"
        :data-testid="`add-entry-${entryId}`"
        @click="emit('add')"
      >
        <Icon :name="alreadySelected ? 'check' : 'plus'" size="sm" :stroke-width="2.5" />
      </button>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { BillingPublicCatalogAdminEntry } from '@/api/admin/billing'
import type { PublicModelCatalogPriceDisplay } from '@/api/meta'
import Icon from '@/components/icons/Icon.vue'
import PublicCatalogCardHeader from './PublicCatalogCardHeader.vue'
import PublicCatalogMoveButtons from './PublicCatalogMoveButtons.vue'
import PublicCatalogPriceEditor from './PublicCatalogPriceEditor.vue'
import PublicCatalogSelectedFields from './PublicCatalogSelectedFields.vue'

const props = withDefaults(defineProps<{
  item: BillingPublicCatalogAdminEntry
  mode: 'available' | 'selected'
  index?: number
  missing?: boolean
  duplicate?: boolean
  alreadySelected?: boolean
  moveUpDisabled?: boolean
  moveDownDisabled?: boolean
}>(), {
  index: 0,
  missing: false,
  duplicate: false,
  alreadySelected: false,
  moveUpDisabled: false,
  moveDownDisabled: false,
})

const emit = defineEmits<{
  (e: 'add'): void
  (e: 'remove'): void
  (e: 'edit'): void
  (e: 'move-up'): void
  (e: 'move-down'): void
  (e: 'update-public-id', value: string): void
  (e: 'update-source-alias', value: string): void
  (e: 'update-sale-price', value: PublicModelCatalogPriceDisplay): void
}>()

const { t } = useI18n()
const entryId = computed(() => props.item.entry_id || props.item.model)
const showHandle = computed(() => props.mode === 'selected')
const indexLabel = computed(() => String(props.index + 1).padStart(2, '0'))
const baseModel = computed(() => props.item.base_model || props.item.source_model_id || props.item.model)
const publicModelLabel = computed(() => props.item.public_model_id || props.item.model)
const sourceAliasLabel = computed(() => props.item.source_alias || t('admin.billing.publicCatalog.card.defaultSource'))
const officialPrice = computed(() => props.item.official_price_display || props.item.price_display)
const salePrice = computed(() => props.item.sale_price_display || props.item.price_display)
const sourceContextLabel = computed(() => props.item.source_account_name || props.item.source_alias || '')

const cardClass = computed(() => {
  if (props.missing) {
    return 'border-amber-300 shadow-sm dark:border-amber-500/40'
  }
  if (props.duplicate) {
    return 'border-rose-300 shadow-sm dark:border-rose-500/40'
  }
  if (props.alreadySelected) {
    return 'border-emerald-200 shadow-sm ring-1 ring-emerald-100 dark:border-emerald-500/30 dark:ring-emerald-500/10'
  }
  return 'border-slate-200 shadow-sm hover:border-emerald-300 hover:shadow-md dark:border-dark-700 dark:hover:border-emerald-500/40'
})
</script>
