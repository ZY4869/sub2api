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
          :image-fixed-pricing="draft.image_fixed_pricing || item?.image_fixed_pricing"
          :currency="item?.currency || 'USD'"
          editable
          testid-prefix="catalog-dialog-price"
          @update:sale="draft.sale_price_display = $event"
          @update:image-fixed-pricing="draft.image_fixed_pricing = $event"
        />
      </div>

      <div class="space-y-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex items-center gap-2 text-sm font-bold text-slate-800 dark:text-white">
          <Icon name="badge" size="sm" class="text-rose-500" />
          {{ t('admin.billing.publicCatalog.dialog.discountTitle') }}
        </div>
        <label class="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50/70 p-3 dark:border-dark-700 dark:bg-dark-900/30">
          <input
            v-model="discountEnabled"
            type="checkbox"
            class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            data-testid="catalog-dialog-discount-enabled"
          />
          <div>
            <div class="text-sm font-medium text-slate-900 dark:text-white">
              {{ t('admin.billing.publicCatalog.dialog.discountEnabled') }}
            </div>
            <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
              {{ t('admin.billing.publicCatalog.dialog.discountHint') }}
            </p>
          </div>
        </label>
        <div v-if="discountEnabled && draft.discount_policy" class="space-y-4">
          <div class="grid gap-3 md:grid-cols-2">
            <label class="space-y-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
              <span>{{ t('admin.billing.publicCatalog.dialog.discountPercent') }}</span>
              <input
                v-model.number="draft.discount_policy.reduction_percent"
                type="number"
                min="0.01"
                max="100"
                step="0.01"
                class="input"
                data-testid="catalog-dialog-discount-percent"
              />
            </label>
            <label class="space-y-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
              <span>{{ t('admin.billing.publicCatalog.dialog.discountTimezone') }}</span>
              <input
                v-model.trim="draft.discount_policy.timezone"
                class="input"
                placeholder="Asia/Singapore"
                data-testid="catalog-dialog-discount-timezone"
              />
            </label>
          </div>
          <div class="flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary btn-sm" @click="addDiscountWindow('once')">
              {{ t('admin.billing.publicCatalog.dialog.addOnceDiscount') }}
            </button>
            <button type="button" class="btn btn-secondary btn-sm" @click="addDiscountWindow('daily')">
              {{ t('admin.billing.publicCatalog.dialog.addDailyDiscount') }}
            </button>
          </div>
          <div class="space-y-3">
            <div
              v-for="(window, index) in draft.discount_policy.windows"
              :key="window.id || index"
              class="rounded-xl border border-slate-200 bg-slate-50/70 p-3 dark:border-dark-700 dark:bg-dark-900/30"
            >
              <div class="mb-3 flex items-center justify-between gap-3">
                <select v-model="window.type" class="input max-w-[10rem]">
                  <option value="once">{{ t('admin.billing.publicCatalog.dialog.onceDiscount') }}</option>
                  <option value="daily">{{ t('admin.billing.publicCatalog.dialog.dailyDiscount') }}</option>
                </select>
                <button type="button" class="btn btn-secondary btn-sm" @click="removeDiscountWindow(index)">
                  {{ t('admin.billing.publicCatalog.dialog.removeWindow') }}
                </button>
              </div>
              <div v-if="window.type === 'once'" class="grid gap-3 md:grid-cols-2">
                <label class="space-y-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
                  <span>{{ t('admin.billing.publicCatalog.dialog.discountStartAt') }}</span>
                  <input
                    :value="formatDateTimeLocal(window.start_at)"
                    type="datetime-local"
                    class="input"
                    @input="window.start_at = dateTimeLocalToISOString(($event.target as HTMLInputElement).value) || ''"
                  />
                </label>
                <label class="space-y-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
                  <span>{{ t('admin.billing.publicCatalog.dialog.discountEndAt') }}</span>
                  <input
                    :value="formatDateTimeLocal(window.end_at)"
                    type="datetime-local"
                    class="input"
                    @input="window.end_at = dateTimeLocalToISOString(($event.target as HTMLInputElement).value) || ''"
                  />
                </label>
              </div>
              <div v-else class="space-y-3">
                <div class="grid gap-3 md:grid-cols-2">
                  <label class="space-y-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
                    <span>{{ t('admin.billing.publicCatalog.dialog.discountStartTime') }}</span>
                    <input v-model="window.start_time" type="time" step="1" class="input" />
                  </label>
                  <label class="space-y-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
                    <span>{{ t('admin.billing.publicCatalog.dialog.discountEndTime') }}</span>
                    <input v-model="window.end_time" type="time" step="1" class="input" />
                  </label>
                </div>
                <div class="flex flex-wrap gap-2">
                  <label
                    v-for="day in weekDays"
                    :key="day.value"
                    class="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-2 py-1 text-xs font-medium text-slate-600 dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300"
                  >
                    <input
                      type="checkbox"
                      class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                      :checked="(window.days || []).includes(day.value)"
                      @change="toggleDiscountDay(window, day.value)"
                    />
                    {{ day.label }}
                  </label>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="space-y-4 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex items-center gap-2 text-sm font-bold text-slate-800 dark:text-white">
          <Icon name="calendar" size="sm" class="text-blue-500" />
          {{ t('admin.billing.publicCatalog.dialog.scheduleTitle') }}
        </div>
        <div class="grid gap-3 md:grid-cols-2">
          <label class="space-y-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
            <span>{{ t('admin.billing.publicCatalog.dialog.availableFrom') }}</span>
            <input
              :value="formatDateTimeLocal(draft.available_from)"
              type="datetime-local"
              class="input"
              data-testid="catalog-dialog-available-from"
              @input="draft.available_from = dateTimeLocalToISOString(($event.target as HTMLInputElement).value) || ''"
            />
          </label>
          <label class="space-y-1.5 text-sm font-medium text-slate-700 dark:text-slate-200">
            <span>{{ t('admin.billing.publicCatalog.dialog.availableUntil') }}</span>
            <input
              :value="formatDateTimeLocal(draft.available_until)"
              type="datetime-local"
              class="input"
              data-testid="catalog-dialog-available-until"
              @input="draft.available_until = dateTimeLocalToISOString(($event.target as HTMLInputElement).value) || ''"
            />
          </label>
        </div>
        <label class="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50/70 p-3 dark:border-dark-700 dark:bg-dark-900/30">
          <input
            v-model="timeAccessEnabled"
            type="checkbox"
            class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
            data-testid="catalog-dialog-time-access-enabled"
          />
          <div>
            <div class="text-sm font-medium text-slate-900 dark:text-white">
              {{ t('admin.billing.publicCatalog.dialog.timeAccess') }}
            </div>
            <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
              {{ t('admin.billing.publicCatalog.dialog.timeAccessHint') }}
            </p>
          </div>
        </label>
        <TimeAccessPolicyEditor
          v-if="timeAccessEnabled"
          v-model="draft.access_time_policy"
          :hint="t('admin.billing.publicCatalog.dialog.timeAccessHint')"
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
import type { PublicModelCatalogDiscountWindow } from '@/api/meta'
import BaseDialog from '@/components/common/BaseDialog.vue'
import TimeAccessPolicyEditor from '@/components/common/TimeAccessPolicyEditor.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { clonePriceDisplay } from './publicCatalogPricing'
import { cloneImageFixedPricing } from './publicCatalogDraft'
import PublicCatalogPriceEditor from './PublicCatalogPriceEditor.vue'
import {
  createDailyDiscountWindow,
  createDefaultDiscountPolicy,
  createOnceDiscountWindow,
  discountPolicyToPayload,
  ensureDiscountPolicy,
  normalizeDiscountPolicy,
} from './publicCatalogDiscount'
import {
  buildPresetTimeAccessPolicy,
  dateTimeLocalToISOString,
  ensureEnabledTimeAccessPolicy,
  formatDateTimeLocal,
  normalizeTimeAccessPolicy,
  policyToPayload,
} from '@/utils/timeAccessPolicy'

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
const timeAccessEnabled = ref(false)
const discountEnabled = ref(false)
const dayKeys = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat']
const weekDays = computed(() => [0, 1, 2, 3, 4, 5, 6].map((value) => ({
  value,
  label: t(`common.timeAccess.daysShort.${dayKeys[value]}`),
})))

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
      image_fixed_pricing: cloneImageFixedPricing(props.item.image_fixed_pricing),
      discount_policy: normalizeDiscountPolicy(props.item.discount_policy),
      available_from: props.item.available_from || '',
      available_until: props.item.available_until || '',
      access_time_policy: normalizeTimeAccessPolicy(props.item.access_time_policy || null),
    }
    timeAccessEnabled.value = !!props.item.access_time_policy?.enabled
    discountEnabled.value = !!props.item.discount_policy?.enabled
  },
  { immediate: true },
)

watch(discountEnabled, (enabled) => {
  if (!draft.value) return
  draft.value.discount_policy = enabled
    ? ensureDiscountPolicy(draft.value.discount_policy || createDefaultDiscountPolicy())
    : createDefaultDiscountPolicy()
})

function save() {
  if (!draft.value) return
  emit('save', draft.value.entry_id, {
    public_model_id: draft.value.public_model_id,
    source_alias: draft.value.source_alias,
    source_model_id: draft.value.source_model_id,
    base_model: draft.value.base_model,
    source_protocol: draft.value.source_protocol,
    sale_price_display: clonePriceDisplay(draft.value.sale_price_display),
    image_fixed_pricing: cloneImageFixedPricing(draft.value.image_fixed_pricing),
    discount_policy: discountEnabled.value ? discountPolicyToPayload(draft.value.discount_policy) : null,
    available_from: draft.value.available_from || '',
    available_until: draft.value.available_until || '',
    access_time_policy: timeAccessEnabled.value
      ? policyToPayload(ensureEnabledTimeAccessPolicy(draft.value.access_time_policy || buildPresetTimeAccessPolicy('daytime')))
      : null,
  })
  emit('close')
}

function addDiscountWindow(type: 'once' | 'daily') {
  if (!draft.value) return
  draft.value.discount_policy = ensureDiscountPolicy(draft.value.discount_policy)
  draft.value.discount_policy.windows = [
    ...(draft.value.discount_policy.windows || []),
    type === 'once' ? createOnceDiscountWindow() : createDailyDiscountWindow(),
  ]
}

function removeDiscountWindow(index: number) {
  if (!draft.value?.discount_policy?.windows) return
  draft.value.discount_policy.windows.splice(index, 1)
}

function toggleDiscountDay(window: PublicModelCatalogDiscountWindow, day: number) {
  const days = new Set(window.days || [])
  if (days.has(day)) {
    days.delete(day)
  } else {
    days.add(day)
  }
  window.days = Array.from(days).sort()
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
