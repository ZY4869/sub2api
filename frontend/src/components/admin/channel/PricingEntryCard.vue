<template>
  <div class="rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
    <div class="mb-4 flex items-center justify-between gap-3">
      <div class="flex items-center gap-2">
        <PlatformIcon :platform="platform" size="sm" />
        <span class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t(`admin.groups.platforms.${platform}`, platform) }}
        </span>
      </div>
      <button type="button" class="rounded p-1 text-gray-400 hover:text-red-500" @click="emit('remove')">
        <Icon name="trash" size="sm" />
      </button>
    </div>

    <div class="space-y-4">
      <div>
        <label class="input-label text-xs">{{ t('admin.channels.form.models', 'Models') }}</label>
        <textarea
          :value="modelsText"
          rows="2"
          class="input"
          :placeholder="t('admin.channels.form.modelsPlaceholder', 'One per line or comma separated, supports wildcard *')"
          @input="updateModels(($event.target as HTMLTextAreaElement).value)"
        />
      </div>

      <div>
        <label class="input-label text-xs">{{ t('admin.channels.form.billingMode', 'Billing Mode') }}</label>
        <select v-model="local.billing_mode" class="input" @change="sync">
          <option value="token">{{ t('admin.channels.form.billingModeToken', 'Token') }}</option>
          <option value="per_request">{{ t('admin.channels.form.billingModePerRequest', 'Per Request') }}</option>
          <option value="image">{{ t('admin.channels.form.billingModeImage', 'Image') }}</option>
        </select>
      </div>

      <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <div v-for="field in visiblePriceFields" :key="field.key">
          <label class="input-label text-xs">{{ field.label }}</label>
          <input
            :value="readField(field.key)"
            type="number"
            step="any"
            class="input"
            @input="writeField(field.key, ($event.target as HTMLInputElement).value)"
          />
        </div>
      </div>

      <div class="space-y-3 rounded-lg border border-dashed border-gray-300 p-3 dark:border-dark-500">
        <div class="flex items-center justify-between gap-3">
          <label class="input-label mb-0 text-xs">{{ t('admin.channels.form.billingIntervals', 'Billing Intervals') }}</label>
          <button type="button" class="text-xs text-primary-600 hover:text-primary-700" @click="addInterval">
            + {{ t('common.add', 'Add') }}
          </button>
        </div>

        <div v-if="local.intervals.length === 0" class="text-xs text-gray-400">
          {{ t('admin.channels.form.noIntervals', 'No interval rules configured.') }}
        </div>

        <div v-for="(interval, idx) in local.intervals" :key="idx" class="space-y-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <div class="flex items-center justify-between gap-3">
            <span class="text-xs font-medium text-gray-500 dark:text-gray-300">
              {{ t('admin.channels.form.intervalTitle', { index: idx + 1 }, `Interval ${idx + 1}`) }}
            </span>
            <button type="button" class="rounded p-1 text-gray-400 hover:text-red-500" @click="removeInterval(idx)">
              <Icon name="trash" size="sm" />
            </button>
          </div>

          <div class="grid gap-3 sm:grid-cols-3">
            <div>
              <label class="input-label text-xs">{{ t('admin.channels.form.minTokens', 'Min Tokens') }}</label>
              <input v-model.number="interval.min_tokens" type="number" min="0" class="input" @input="sync" />
            </div>
            <div>
              <label class="input-label text-xs">{{ t('admin.channels.form.maxTokens', 'Max Tokens') }}</label>
              <input :value="interval.max_tokens ?? ''" type="number" min="0" class="input" @input="updateIntervalMax(idx, ($event.target as HTMLInputElement).value)" />
            </div>
            <div>
              <label class="input-label text-xs">{{ t('admin.channels.form.tierLabel', 'Tier Label') }}</label>
              <input v-model="interval.tier_label" type="text" class="input" @input="sync" />
            </div>
          </div>

          <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <div v-for="field in intervalPriceFields" :key="field.key">
              <label class="input-label text-xs">{{ field.label }}</label>
              <input
                :value="readIntervalField(interval, field.key)"
                type="number"
                step="any"
                class="input"
                @input="writeIntervalField(idx, field.key, ($event.target as HTMLInputElement).value)"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import type { GroupPlatform } from '@/types'
import type { IntervalFormEntry, PricingFormEntry } from './types'

type PricingFieldKey =
  | 'input_price'
  | 'output_price'
  | 'cache_write_price'
  | 'cache_read_price'
  | 'image_output_price'
  | 'per_request_price'

type IntervalFieldKey =
  | 'input_price'
  | 'output_price'
  | 'cache_write_price'
  | 'cache_read_price'
  | 'per_request_price'

interface PriceFieldOption<T extends string> {
  key: T
  label: string
}

const props = defineProps<{
  entry: PricingFormEntry
  platform: GroupPlatform
}>()

const emit = defineEmits<{
  (e: 'update', value: PricingFormEntry): void
  (e: 'remove'): void
}>()

const { t } = useI18n()

const local = reactive(cloneEntry(props.entry))

watch(
  () => props.entry,
  (value) => Object.assign(local, cloneEntry(value)),
  { deep: true }
)

const modelsText = computed(() => local.models.join('\n'))

const visiblePriceFields = computed<PriceFieldOption<PricingFieldKey>[]>(() => {
  if (local.billing_mode === 'token') {
    return [
      { key: 'input_price', label: t('admin.channels.form.inputPrice', 'Input Price ($/MTok)') },
      { key: 'output_price', label: t('admin.channels.form.outputPrice', 'Output Price ($/MTok)') },
      { key: 'cache_write_price', label: t('admin.channels.form.cacheWritePrice', 'Cache Write ($/MTok)') },
      { key: 'cache_read_price', label: t('admin.channels.form.cacheReadPrice', 'Cache Read ($/MTok)') },
      { key: 'image_output_price', label: t('admin.channels.form.imageOutputPrice', 'Image Output ($/MTok)') }
    ]
  }
  return [
    { key: 'per_request_price', label: t('admin.channels.form.perRequestPrice', 'Per Request Price') },
    { key: 'image_output_price', label: t('admin.channels.form.imageOutputPrice', 'Image Output ($/MTok)') }
  ]
})

const intervalPriceFields = computed<PriceFieldOption<IntervalFieldKey>[]>(() => [
  { key: 'input_price', label: t('admin.channels.form.inputPrice', 'Input Price ($/MTok)') },
  { key: 'output_price', label: t('admin.channels.form.outputPrice', 'Output Price ($/MTok)') },
  { key: 'cache_write_price', label: t('admin.channels.form.cacheWritePrice', 'Cache Write ($/MTok)') },
  { key: 'cache_read_price', label: t('admin.channels.form.cacheReadPrice', 'Cache Read ($/MTok)') },
  { key: 'per_request_price', label: t('admin.channels.form.perRequestPrice', 'Per Request Price') }
])

function cloneEntry(entry: PricingFormEntry): PricingFormEntry {
  return {
    models: [...(entry.models || [])],
    billing_mode: entry.billing_mode,
    input_price: entry.input_price ?? null,
    output_price: entry.output_price ?? null,
    cache_write_price: entry.cache_write_price ?? null,
    cache_read_price: entry.cache_read_price ?? null,
    image_output_price: entry.image_output_price ?? null,
    per_request_price: entry.per_request_price ?? null,
    intervals: (entry.intervals || []).map((interval) => ({ ...interval }))
  }
}

function sync() {
  emit('update', cloneEntry(local))
}

function updateModels(value: string) {
  local.models = value
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
  sync()
}

function readField(key: PricingFieldKey) {
  const value = local[key]
  return value == null ? '' : value
}

function writeField(key: PricingFieldKey, value: string) {
  ;(local[key] as number | string | null) = value === '' ? null : Number(value)
  sync()
}

function addInterval() {
  local.intervals.push({
    min_tokens: 0,
    max_tokens: null,
    tier_label: '',
    input_price: null,
    output_price: null,
    cache_write_price: null,
    cache_read_price: null,
    per_request_price: null,
    sort_order: local.intervals.length
  })
  sync()
}

function removeInterval(index: number) {
  local.intervals.splice(index, 1)
  sync()
}

function updateIntervalMax(index: number, value: string) {
  local.intervals[index].max_tokens = value === '' ? null : Number(value)
  sync()
}

function readIntervalField(interval: IntervalFormEntry, key: IntervalFieldKey) {
  const value = interval[key]
  return value == null ? '' : value
}

function writeIntervalField(index: number, key: IntervalFieldKey, value: string) {
  ;(local.intervals[index][key] as number | string | null) = value === '' ? null : Number(value)
  sync()
}
</script>
