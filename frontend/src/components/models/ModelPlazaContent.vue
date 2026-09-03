<template>
  <div class="model-library-shell mx-auto max-w-[1760px] space-y-4 px-2 sm:px-4 2xl:space-y-5">
    <section class="rounded-lg border border-slate-200 bg-white/95 p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900/90 2xl:p-6">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div class="min-w-0 flex-1 max-w-3xl">
          <p class="text-xs font-semibold uppercase tracking-[0.22em] text-primary-600 dark:text-primary-300">
            {{ t('ui.modelPlaza.eyebrow') }}
          </p>
          <h1 class="mt-2 text-2xl font-semibold text-slate-950 dark:text-white md:text-3xl">
            {{ t('ui.modelPlaza.title') }}
          </h1>
          <p class="mt-2 max-w-4xl break-words text-sm leading-6 text-slate-600 dark:text-slate-300">
            {{ t('ui.modelPlaza.description') }}
          </p>
        </div>

        <div class="flex w-full min-w-0 flex-wrap items-center gap-2 sm:w-auto xl:justify-end">
          <span class="stat-pill">
            {{ t('ui.modelPlaza.groups', { count: groups.length }) }}
          </span>
          <span class="stat-pill">
            {{ t('ui.modelPlaza.models', { count: totalModelCount }) }}
          </span>
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading"
            @click="load(true)"
          >
            <Icon name="refresh" size="sm" :class="{ 'animate-spin': loading }" />
            {{ loading ? t('ui.modelPlaza.refreshing') : t('ui.modelPlaza.refresh') }}
          </button>
        </div>
      </div>
    </section>

    <section class="rounded-lg border border-slate-200 bg-white/95 p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900/90 2xl:p-5">
      <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center">
        <label class="relative block min-w-0">
          <span class="sr-only">{{ t('ui.modelPlaza.searchPlaceholder') }}</span>
          <Icon
            name="search"
            size="sm"
            class="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-slate-400"
          />
          <input
            v-model.trim="searchQuery"
            type="search"
            class="input h-11 w-full rounded-lg border-slate-200 bg-slate-50/80 pl-11 pr-4 text-sm dark:border-dark-700 dark:bg-dark-800/80"
            :placeholder="t('ui.modelPlaza.searchPlaceholder')"
          />
        </label>

        <div class="flex min-w-0 flex-wrap gap-2">
          <button
            v-for="option in accessOptions"
            :key="option.id"
            type="button"
            class="filter-button"
            :class="accessMode === option.id ? activeFilterClass : inactiveFilterClass"
            @click="accessMode = option.id"
          >
            {{ option.label }}
          </button>
        </div>
      </div>

      <div class="mt-4 flex min-w-0 flex-wrap gap-2">
        <button
          type="button"
          class="filter-button"
          :class="selectedPlatform === '' ? activeFilterClass : inactiveFilterClass"
          @click="selectedPlatform = ''"
        >
          <Icon name="globe" size="sm" />
          {{ t('ui.modelPlaza.allPlatforms') }}
        </button>
        <button
          v-for="platform in platformOptions"
          :key="platform"
          type="button"
          class="filter-button"
          :class="selectedPlatform === platform ? activeFilterClass : inactiveFilterClass"
          @click="selectedPlatform = platform"
        >
          <ModelPlatformIcon :platform="platform" size="md" />
          <span>{{ formatPlatform(platform) }}</span>
        </button>
      </div>
    </section>

    <div
      v-if="errorMessage"
      class="min-w-0 break-words rounded-lg border border-rose-200 bg-rose-50 px-5 py-4 text-sm text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-200"
    >
      {{ errorMessage }}
    </div>

    <div v-if="loading && groups.length === 0" class="flex items-center justify-center py-14">
      <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
    </div>

    <div
      v-else-if="filteredGroups.length === 0"
      class="rounded-lg border border-dashed border-slate-300 bg-white/80 px-6 py-14 text-center text-sm text-slate-500 dark:border-dark-700 dark:bg-dark-900/70 dark:text-slate-400"
    >
      {{ t('ui.modelPlaza.empty') }}
    </div>

    <section v-else class="space-y-5">
      <article
        v-for="group in filteredGroups"
        :key="group.id"
        class="rounded-lg border border-slate-200 bg-white/95 p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900/90 2xl:p-5"
      >
        <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <ModelPlatformIcon :platform="group.platform" size="lg" />
              <h2 class="truncate text-lg font-semibold text-slate-950 dark:text-white">
                {{ group.name }}
              </h2>
              <span class="tag-pill">
                {{ group.is_exclusive ? t('ui.modelPlaza.exclusive') : t('ui.modelPlaza.public') }}
              </span>
              <span v-if="group.subscription_type === 'subscription'" class="tag-pill">
                {{ t('ui.modelPlaza.subscription') }}
              </span>
            </div>
            <p v-if="group.description" class="mt-2 max-w-4xl break-words text-sm leading-6 text-slate-500 dark:text-slate-400">
              {{ group.description }}
            </p>
          </div>

          <div class="flex flex-wrap gap-2 text-xs">
            <span class="metric-pill">
              {{ t('ui.modelPlaza.rate') }} {{ formatMultiplier(group.rate_multiplier) }}
            </span>
            <span v-if="group.peak_rate_enabled" class="metric-pill">
              {{ t('ui.modelPlaza.peak') }} {{ formatMultiplier(group.peak_rate_multiplier) }}
            </span>
            <span v-if="group.image_rate_independent" class="metric-pill">
              {{ t('ui.modelPlaza.independentImages') }}
            </span>
          </div>
        </div>

        <div v-if="imagePriceItems(group).length > 0" class="mt-4 flex flex-wrap gap-2 text-xs">
          <span
            v-for="item in imagePriceItems(group)"
            :key="item.label"
            class="max-w-full break-words rounded-full border border-sky-200 bg-sky-50 px-3 py-1 text-sky-700 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200"
          >
            {{ item.label }} {{ item.value }}
          </span>
        </div>

        <div class="model-plaza-grid mt-4 grid gap-3 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
          <div
            v-for="model in group.models"
            :key="`${group.id}:${model.platform}:${model.display_model_id}`"
            class="min-w-0 rounded-lg border border-slate-200 bg-slate-50/70 p-3 dark:border-dark-700 dark:bg-dark-800/70 2xl:p-4"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="flex min-w-0 items-center gap-2">
                <ModelIcon
                  :model="model.display_model_id"
                  :provider="model.platform || group.platform"
                  :display-name="model.display_model_id"
                  size="18px"
                />
                <span class="truncate font-mono text-sm font-semibold text-slate-900 dark:text-white">
                  {{ model.display_model_id }}
                </span>
              </div>
              <button
                type="button"
                class="inline-flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500 transition hover:border-primary-300 hover:text-primary-600 dark:border-dark-700 dark:bg-dark-900 dark:text-slate-300 dark:hover:text-primary-200"
                :title="t('ui.modelPlaza.copyModelId')"
                @click="copyModel(model.display_model_id)"
              >
                <Icon name="copy" size="sm" />
              </button>
            </div>

            <div class="mt-3 flex flex-wrap gap-2 text-xs">
              <span
                v-for="item in priceItems(model.pricing)"
                :key="`${model.display_model_id}:${item.label}`"
                class="max-w-full break-words rounded-full border border-slate-200 bg-white px-2.5 py-1 text-slate-600 dark:border-dark-700 dark:bg-dark-900 dark:text-slate-300"
              >
                {{ item.label }} {{ item.value }}
              </span>
              <span
                v-if="priceItems(model.pricing).length === 0"
                class="max-w-full break-words rounded-full border border-slate-200 bg-white px-2.5 py-1 text-slate-400 dark:border-dark-700 dark:bg-dark-900"
              >
                {{ t('ui.modelPlaza.pricingUnavailable') }}
              </span>
            </div>
          </div>
        </div>
      </article>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { getModelPlaza, type ModelPlazaGroup, type ModelPlazaModelPricing } from '@/api/modelPlaza'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import { formatProviderLabel } from '@/utils/providerLabels'

type AccessMode = 'all' | 'public' | 'exclusive'
type PriceItem = { label: string; value: string }

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const loading = ref(false)
const errorMessage = ref('')
const plazaGroups = ref<ModelPlazaGroup[]>([])
const searchQuery = ref('')
const selectedPlatform = ref('')
const accessMode = ref<AccessMode>('all')

const activeFilterClass =
  'border-primary-300 bg-primary-50 text-primary-800 ring-2 ring-primary-400/20 dark:border-primary-500/60 dark:bg-primary-500/10 dark:text-primary-100'
const inactiveFilterClass =
  'border-slate-200 bg-slate-50/80 text-slate-600 hover:border-primary-200 hover:bg-white hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800/70 dark:text-slate-300 dark:hover:border-primary-500 dark:hover:text-primary-200'

const groups = computed(() => plazaGroups.value)
const totalModelCount = computed(() => groups.value.reduce((sum, group) => sum + group.models.length, 0))
const normalizedSearch = computed(() => searchQuery.value.trim().toLowerCase())
const accessOptions = computed(() => [
  { id: 'all' as const, label: t('ui.modelPlaza.allAccess') },
  { id: 'public' as const, label: t('ui.modelPlaza.publicAccess') },
  { id: 'exclusive' as const, label: t('ui.modelPlaza.exclusiveAccess') }
])

const platformOptions = computed(() => {
  const seen = new Set<string>()
  for (const group of groups.value) {
    if (group.platform) {
      seen.add(group.platform)
    }
  }
  return Array.from(seen).sort((left, right) => formatPlatform(left).localeCompare(formatPlatform(right)))
})

const filteredGroups = computed(() => {
  return groups.value
    .filter((group) => {
      if (selectedPlatform.value && group.platform !== selectedPlatform.value) {
        return false
      }
      if (accessMode.value === 'public' && group.is_exclusive) {
        return false
      }
      if (accessMode.value === 'exclusive' && !group.is_exclusive) {
        return false
      }
      return true
    })
    .map((group) => {
      if (!normalizedSearch.value) {
        return group
      }
      const groupMatches = [group.name, group.description, group.platform]
        .some((value) => String(value || '').toLowerCase().includes(normalizedSearch.value))
      const models = groupMatches
        ? group.models
        : group.models.filter((model) => [model.display_model_id, model.platform]
          .some((value) => String(value || '').toLowerCase().includes(normalizedSearch.value)))
      return { ...group, models }
    })
    .filter((group) => group.models.length > 0)
})

onMounted(() => {
  void load(false)
})

async function load(force: boolean) {
  if (loading.value && !force) {
    return
  }
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await getModelPlaza()
    plazaGroups.value = response.groups || []
  } catch (err: any) {
    plazaGroups.value = []
    errorMessage.value = `${t('ui.modelPlaza.loadFailed')}: ${err?.message || t('common.unknownError')}`
    appStore.showError(errorMessage.value)
  } finally {
    loading.value = false
  }
}

async function copyModel(model: string) {
  await copyToClipboard(model, t('ui.modelPlaza.copySuccess', { model }))
}

function formatPlatform(platform: string): string {
  return formatProviderLabel(platform || '')
}

function formatMultiplier(value: number): string {
  const normalized = Number.isFinite(value) && value > 0 ? value : 1
  return `${new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: 4
  }).format(normalized)}x`
}

function imagePriceItems(group: ModelPlazaGroup): PriceItem[] {
  const items: PriceItem[] = []
  if (group.image_price_1k !== null && group.image_price_1k !== undefined) {
    items.push({ label: '1K', value: formatCurrency(group.image_price_1k) })
  }
  if (group.image_price_2k !== null && group.image_price_2k !== undefined) {
    items.push({ label: '2K', value: formatCurrency(group.image_price_2k) })
  }
  if (group.image_price_4k !== null && group.image_price_4k !== undefined) {
    items.push({ label: '4K', value: formatCurrency(group.image_price_4k) })
  }
  if (group.web_search_price_per_call !== null && group.web_search_price_per_call !== undefined) {
    items.push({ label: t('ui.modelPlaza.webSearch'), value: formatCurrency(group.web_search_price_per_call) })
  }
  return items
}

function priceItems(pricing: ModelPlazaModelPricing | null): PriceItem[] {
  if (!pricing) {
    return []
  }
  const items: PriceItem[] = []
  addTokenPrice(items, t('ui.modelPlaza.input'), pricing.input_price)
  addTokenPrice(items, t('ui.modelPlaza.output'), pricing.output_price)
  addTokenPrice(items, t('ui.modelPlaza.cacheWrite'), pricing.cache_write_price)
  addTokenPrice(items, t('ui.modelPlaza.cacheRead'), pricing.cache_read_price)
  if (pricing.image_output_price !== null && pricing.image_output_price !== undefined) {
    items.push({ label: t('ui.modelPlaza.imageOutput'), value: formatCurrency(pricing.image_output_price) })
  }
  if (pricing.per_request_price !== null && pricing.per_request_price !== undefined) {
    items.push({ label: t('ui.modelPlaza.request'), value: formatCurrency(pricing.per_request_price) })
  }
  return items
}

function addTokenPrice(items: PriceItem[], label: string, value: number | null | undefined) {
  if (value === null || value === undefined) {
    return
  }
  items.push({
    label,
    value: `${formatCurrency(value * 1_000_000)} ${t('ui.modelPlaza.perMillion')}`
  })
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: value >= 1 ? 4 : 8
  }).format(value)
}
</script>

<style scoped>
.stat-pill {
  @apply inline-flex max-w-full min-w-0 min-h-9 items-center rounded-lg border border-slate-200 bg-slate-50 px-3 py-1.5 text-sm font-medium text-slate-700 dark:border-dark-700 dark:bg-dark-800 dark:text-slate-200;
}

.filter-button {
  @apply inline-flex max-w-full min-w-0 min-h-10 items-center justify-center gap-2 rounded-lg border px-3 text-left text-sm font-medium leading-snug transition;
}

.tag-pill {
  @apply inline-flex max-w-full min-w-0 items-center rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-xs font-medium text-slate-600 dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300;
}

.metric-pill {
  @apply inline-flex max-w-full min-w-0 items-center rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1 font-medium text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200;
}
</style>
