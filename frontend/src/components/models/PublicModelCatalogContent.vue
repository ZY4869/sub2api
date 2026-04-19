<template>
  <div class="mx-auto max-w-[1700px] space-y-6 px-1">
    <section class="overflow-hidden rounded-[2rem] border border-slate-200 bg-[radial-gradient(circle_at_top_left,_rgba(14,116,144,0.12),_transparent_35%),linear-gradient(135deg,_rgba(255,255,255,0.98),_rgba(240,249,255,0.92))] p-6 shadow-sm dark:border-dark-700 dark:bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.12),_transparent_35%),linear-gradient(135deg,_rgba(15,23,42,0.96),_rgba(17,24,39,0.92))] md:p-8">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="max-w-3xl">
          <p class="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">
            {{ t("ui.modelCatalog.eyebrow") }}
          </p>
          <h1 class="mt-3 text-3xl font-semibold tracking-tight text-slate-950 dark:text-white">
            {{ t("nav.modelsCatalog") }}
          </h1>
          <p class="mt-3 text-sm leading-7 text-slate-700 dark:text-slate-200">
            {{ t("ui.modelCatalog.description") }}
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-2">
          <span class="rounded-full border border-slate-200 bg-white/80 px-4 py-2 text-sm text-slate-700 dark:border-dark-700 dark:bg-dark-900/80 dark:text-slate-200">
            {{ modelCountLabel }}
          </span>
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="loading"
            data-testid="public-models-refresh"
            @click="loadCatalog(true)"
          >
            {{ loading ? t("ui.modelCatalog.refreshing") : t("ui.modelCatalog.refresh") }}
          </button>
        </div>
      </div>
    </section>

    <div
      v-if="errorMessage"
      class="rounded-3xl border border-rose-200 bg-rose-50 px-6 py-4 text-sm text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-200"
    >
      {{ errorMessage }}
    </div>

    <section class="rounded-3xl border border-slate-200 bg-white/90 p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900/80">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-center xl:justify-between">
        <label class="relative block w-full xl:max-w-xl">
          <span class="sr-only">{{ t("ui.modelCatalog.searchPlaceholder") }}</span>
          <Icon
            name="search"
            size="sm"
            class="pointer-events-none absolute left-4 top-1/2 -translate-y-1/2 text-slate-400"
          />
          <input
            v-model.trim="searchQuery"
            type="search"
            class="input h-12 w-full rounded-2xl border-slate-200 bg-slate-50/80 pl-11 pr-4 text-sm text-slate-700 dark:border-dark-700 dark:bg-dark-800/80 dark:text-slate-100"
            :placeholder="t('ui.modelCatalog.searchPlaceholder')"
            data-testid="public-models-search"
          />
        </label>

        <div class="flex flex-wrap items-center justify-between gap-3 xl:justify-end">
          <span class="text-sm text-slate-500 dark:text-slate-400">
            {{ filteredItems.length }} / {{ catalog?.items.length || 0 }}
          </span>
          <div class="inline-flex rounded-2xl border border-slate-200 bg-slate-50/90 p-1 dark:border-dark-700 dark:bg-dark-800/80">
            <button
              type="button"
              class="inline-flex items-center gap-2 rounded-xl px-3 py-2 text-sm font-medium transition"
              :class="viewMode === 'grid' ? activeToggleClass : inactiveToggleClass"
              data-testid="public-models-view-grid"
              @click="viewMode = 'grid'"
            >
              <Icon name="grid" size="sm" />
              {{ t("ui.modelCatalog.viewModes.grid") }}
            </button>
            <button
              type="button"
              class="inline-flex items-center gap-2 rounded-xl px-3 py-2 text-sm font-medium transition"
              :class="viewMode === 'list' ? activeToggleClass : inactiveToggleClass"
              data-testid="public-models-view-list"
              @click="viewMode = 'list'"
            >
              <Icon name="menu" size="sm" />
              {{ t("ui.modelCatalog.viewModes.list") }}
            </button>
          </div>
        </div>
      </div>
    </section>

    <div class="grid gap-6 xl:grid-cols-[360px_minmax(0,1fr)]">
      <aside class="space-y-4 rounded-3xl border border-slate-200 bg-white/90 p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900/80 xl:sticky xl:top-24 xl:self-start">
        <section class="space-y-3">
          <div class="text-sm font-semibold text-slate-900 dark:text-white">
            {{ t("ui.modelCatalog.filters.provider") }}
          </div>
          <div class="grid gap-2 sm:grid-cols-2 xl:grid-cols-1">
            <button
              type="button"
              class="group rounded-2xl border px-4 py-3 text-left transition"
              :class="selectedProvider === '' ? activeFilterClass : inactiveFilterClass"
              data-testid="models-filter-provider-all"
              @click="selectedProvider = ''"
            >
              <div class="flex items-center gap-3">
                <span class="filter-icon-shell">
                  <Icon name="filter" size="sm" />
                </span>
                <div class="min-w-0">
                  <div class="truncate text-sm font-semibold">
                    {{ t("ui.modelCatalog.filters.all") }}
                  </div>
                  <div class="text-xs opacity-70">
                    {{ catalog?.items.length || 0 }}
                  </div>
                </div>
              </div>
            </button>
            <button
              v-for="provider in providerOptions"
              :key="provider.id"
              type="button"
              class="group rounded-2xl border px-4 py-3 text-left transition"
              :class="selectedProvider === provider.id ? activeFilterClass : inactiveFilterClass"
              :data-testid="`models-filter-provider-${provider.id}`"
              @click="selectedProvider = provider.id"
            >
              <div class="flex items-center gap-3">
                <span class="filter-icon-shell">
                  <ModelPlatformIcon :platform="provider.iconKey" size="md" />
                </span>
                <div class="min-w-0">
                  <div class="truncate text-sm font-semibold">
                    {{ provider.label }}
                  </div>
                  <div class="text-xs opacity-70">
                    {{ provider.count }}
                  </div>
                </div>
              </div>
            </button>
          </div>
        </section>

        <section class="space-y-3">
          <div class="text-sm font-semibold text-slate-900 dark:text-white">
            {{ t("ui.modelCatalog.filters.protocol") }}
          </div>
          <div class="grid gap-2 sm:grid-cols-2 xl:grid-cols-1">
            <button
              type="button"
              class="group rounded-2xl border px-4 py-3 text-left transition"
              :class="selectedProtocol === '' ? activeFilterClass : inactiveFilterClass"
              data-testid="models-filter-protocol-all"
              @click="selectedProtocol = ''"
            >
              <div class="flex items-center gap-3">
                <span class="filter-icon-shell">
                  <Icon name="swap" size="sm" />
                </span>
                <div class="min-w-0">
                  <div class="truncate text-sm font-semibold">
                    {{ t("ui.modelCatalog.filters.all") }}
                  </div>
                  <div class="text-xs opacity-70">
                    {{ protocolItemCount }}
                  </div>
                </div>
              </div>
            </button>
            <button
              v-for="protocol in protocolOptions"
              :key="protocol.id"
              type="button"
              class="group rounded-2xl border px-4 py-3 text-left transition"
              :class="selectedProtocol === protocol.id ? activeFilterClass : inactiveFilterClass"
              :data-testid="`models-filter-protocol-${protocol.id}`"
              @click="selectedProtocol = protocol.id"
            >
              <div class="flex items-center gap-3">
                <span class="filter-icon-shell">
                  <ModelPlatformIcon :platform="protocol.iconKey" size="md" />
                </span>
                <div class="min-w-0">
                  <div class="truncate text-sm font-semibold">
                    {{ protocol.label }}
                  </div>
                  <div class="text-xs opacity-70">
                    {{ protocol.count }}
                  </div>
                </div>
              </div>
            </button>
          </div>
        </section>

        <section class="space-y-3">
          <div class="text-sm font-semibold text-slate-900 dark:text-white">
            {{ t("ui.modelCatalog.filters.multiplier") }}
          </div>
          <div class="grid gap-2 sm:grid-cols-2 xl:grid-cols-1">
            <button
              type="button"
              class="group rounded-2xl border px-4 py-3 text-left transition"
              :class="selectedMultiplier === '' ? activeFilterClass : inactiveFilterClass"
              data-testid="models-filter-multiplier-all"
              @click="selectedMultiplier = ''"
            >
              <div class="flex items-center gap-3">
                <span class="filter-icon-shell">
                  <Icon name="calculator" size="sm" />
                </span>
                <div class="min-w-0">
                  <div class="truncate text-sm font-semibold">
                    {{ t("ui.modelCatalog.filters.all") }}
                  </div>
                  <div class="text-xs opacity-70">
                    {{ catalog?.items.length || 0 }}
                  </div>
                </div>
              </div>
            </button>
            <button
              v-for="option in multiplierOptions"
              :key="option.id"
              type="button"
              class="group rounded-2xl border px-4 py-3 text-left transition"
              :class="selectedMultiplier === option.id ? activeFilterClass : inactiveFilterClass"
              :data-testid="`models-filter-multiplier-${option.id}`"
              @click="selectedMultiplier = option.id"
            >
              <div class="flex items-center gap-3">
                <span class="filter-icon-shell">
                  <Icon :name="option.iconName" size="sm" />
                </span>
                <div class="min-w-0">
                  <div class="truncate text-sm font-semibold">
                    {{ option.label }}
                  </div>
                  <div class="text-xs opacity-70">
                    {{ option.count }}
                  </div>
                </div>
              </div>
            </button>
          </div>
        </section>
      </aside>

      <section class="space-y-4">
        <div
          class="gap-4"
          :class="viewMode === 'grid' ? 'grid md:grid-cols-2 2xl:grid-cols-3' : 'flex flex-col'"
          data-testid="public-model-results"
          :data-view-mode="viewMode"
        >
          <article
            v-for="item in filteredItems"
            :key="item.model"
            class="relative overflow-hidden rounded-3xl border border-slate-200 bg-white/90 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md dark:border-dark-700 dark:bg-dark-900/80"
            :class="viewMode === 'list' ? 'p-1' : ''"
            :data-testid="`public-model-card-${item.model}`"
          >
            <button
              type="button"
              class="absolute right-4 top-4 z-10 inline-flex h-10 w-10 items-center justify-center rounded-2xl border border-slate-200 bg-white/90 text-slate-500 shadow-sm transition hover:border-primary-300 hover:text-primary-600 dark:border-dark-700 dark:bg-dark-800/90 dark:text-slate-300 dark:hover:border-primary-500 dark:hover:text-primary-200"
              :title="t('ui.modelCatalog.detailButton')"
              :data-testid="`public-model-detail-${item.model}`"
              @click.stop="openDetail(item)"
            >
              <Icon name="more" size="sm" />
            </button>

            <button
              type="button"
              class="block w-full text-left"
              :class="viewMode === 'grid' ? 'p-5 pr-16' : 'p-5 pr-16'"
              :data-testid="`public-model-copy-${item.model}`"
              @click="copyModelID(item)"
            >
              <div
                class="gap-5"
                :class="viewMode === 'grid' ? 'flex flex-col' : 'flex flex-col xl:flex-row xl:items-start xl:justify-between'"
              >
                <div class="min-w-0 space-y-4">
                  <div class="flex items-start gap-3">
                    <ModelIcon
                      :model="item.model"
                      :provider="item.provider"
                      :display-name="item.display_name"
                      size="24px"
                    />
                    <div class="min-w-0">
                      <div class="break-words text-lg font-semibold text-slate-950 dark:text-white">
                        {{ item.display_name || item.model }}
                      </div>
                      <div class="mt-1 break-all text-sm text-slate-500 dark:text-slate-400">
                        {{ item.model }}
                      </div>
                    </div>
                  </div>

                  <div class="flex flex-wrap gap-2 text-xs">
                    <span class="inline-flex items-center gap-1.5 rounded-full border border-slate-200 bg-slate-50 px-2.5 py-1 text-slate-700 dark:border-dark-700 dark:bg-dark-800 dark:text-slate-200">
                      <ModelPlatformIcon :platform="item.provider_icon_key || item.provider || ''" size="sm" />
                      {{ providerLabel(item) }}
                    </span>
                    <span
                      v-for="protocol in item.request_protocols || []"
                      :key="protocol"
                      class="inline-flex items-center gap-1.5 rounded-full border border-sky-200 bg-sky-50 px-2.5 py-1 text-sky-700 dark:border-sky-500/30 dark:bg-sky-500/10 dark:text-sky-200"
                    >
                      <ModelPlatformIcon :platform="protocol" size="xs" />
                      {{ protocolLabel(protocol) }}
                    </span>
                    <span class="inline-flex items-center gap-1.5 rounded-full border border-emerald-200 bg-emerald-50 px-2.5 py-1 text-emerald-700 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-200">
                      <Icon name="calculator" size="xs" />
                      {{ multiplierSummaryLabel(item.multiplier_summary) }}
                    </span>
                    <span
                      v-if="item.mode"
                      class="inline-flex rounded-full border border-fuchsia-200 bg-fuchsia-50 px-2.5 py-1 text-fuchsia-700 dark:border-fuchsia-500/30 dark:bg-fuchsia-500/10 dark:text-fuchsia-200"
                    >
                      {{ item.mode }}
                    </span>
                  </div>

                  <div
                    v-if="item.source_ids?.length"
                    class="flex flex-wrap gap-2 text-xs text-slate-500 dark:text-slate-400"
                  >
                    <span
                      v-for="sourceID in item.source_ids"
                      :key="sourceID"
                      class="rounded-full border border-slate-200 px-2.5 py-1 dark:border-dark-700"
                    >
                      {{ sourceID }}
                    </span>
                  </div>
                </div>

                <div class="min-w-0 rounded-[1.5rem] border border-slate-200 bg-slate-50/90 p-4 dark:border-dark-700 dark:bg-dark-800/80 xl:w-[310px]">
                  <div class="flex items-center justify-between gap-3">
                    <span class="text-[11px] font-semibold uppercase tracking-[0.24em] text-slate-500 dark:text-slate-400">
                      {{ item.currency }}
                    </span>
                    <span class="text-xs text-slate-400 dark:text-slate-500">
                      ID
                    </span>
                  </div>
                  <div class="mt-3 space-y-2">
                    <div
                      v-for="entry in item.price_display.primary"
                      :key="entry.id"
                      class="flex items-center justify-between gap-3 rounded-2xl bg-white/90 px-3 py-2 text-sm dark:bg-dark-900/80"
                    >
                      <span class="text-slate-600 dark:text-slate-300">
                        {{ priceEntryLabel(entry.id) }}
                      </span>
                      <span class="font-semibold" :class="primaryPriceClass(entry.id)">
                        {{ formatCatalogPrice(entry, item.currency) }}
                      </span>
                    </div>
                  </div>

                  <div
                    v-if="item.price_display.secondary?.length"
                    class="mt-3 flex flex-wrap gap-2 text-xs text-slate-500 dark:text-slate-400"
                  >
                    <span
                      v-for="entry in item.price_display.secondary"
                      :key="entry.id"
                      class="rounded-full border border-slate-200 bg-white/80 px-2.5 py-1 dark:border-dark-700 dark:bg-dark-900/80"
                    >
                      {{ priceEntryLabel(entry.id) }}: {{ formatCatalogPrice(entry, item.currency) }}
                    </span>
                  </div>
                </div>
              </div>
            </button>
          </article>
        </div>

        <div
          v-if="!loading && filteredItems.length === 0"
          class="rounded-3xl border border-dashed border-slate-300 bg-white/80 px-6 py-12 text-center text-sm text-slate-500 dark:border-dark-700 dark:bg-dark-900/70 dark:text-slate-400"
        >
          {{ t("ui.modelCatalog.empty") }}
        </div>
      </section>
    </div>

    <PublicModelCatalogDetailDialog
      :show="showDetailDialog"
      :model="selectedItem?.model || null"
      :catalog-item="selectedItem"
      :usd-to-cny-rate="usdToCnyRate"
      @close="closeDetail"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import {
  getModelCatalog,
  getUSDCNYExchangeRate,
  type PublicModelCatalogItem,
  type PublicModelCatalogMultiplierSummary,
  type PublicModelCatalogPriceEntry,
  type PublicModelCatalogSnapshot,
} from "@/api/meta";
import ModelIcon from "@/components/common/ModelIcon.vue";
import ModelPlatformIcon from "@/components/common/ModelPlatformIcon.vue";
import Icon from "@/components/icons/Icon.vue";
import PublicModelCatalogDetailDialog from "@/components/models/PublicModelCatalogDetailDialog.vue";
import { useAppStore } from "@/stores/app";
import { formatProviderLabel, normalizeProviderSlug } from "@/utils/providerLabels";
import {
  formatCatalogPrice as renderCatalogPrice,
  multiplierSummaryLabel as renderMultiplierSummaryLabel,
  priceEntryLabel as renderPriceEntryLabel,
  PUBLIC_MODEL_PROTOCOL_ORDER,
} from "@/utils/publicModelCatalog";

type CatalogViewMode = "grid" | "list";

interface CatalogFilterOption {
  id: string;
  label: string;
  count: number;
  iconKey: string;
}

interface MultiplierFilterOption {
  id: string;
  label: string;
  count: number;
  iconName: "calculator" | "badge" | "sparkles";
}

const PUBLIC_MODEL_CATALOG_VIEW_MODE_KEY = "public-model-catalog:view-mode";

const { t } = useI18n();
const appStore = useAppStore();

const activeFilterClass =
  "border-primary-300 bg-primary-50 text-primary-900 shadow-sm ring-2 ring-primary-400/25 dark:border-primary-500/60 dark:bg-primary-500/10 dark:text-primary-100";
const inactiveFilterClass =
  "border-slate-200 bg-slate-50/80 text-slate-600 hover:border-primary-200 hover:bg-white hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800/70 dark:text-slate-300 dark:hover:border-primary-500 dark:hover:text-primary-200";
const activeToggleClass =
  "bg-primary-600 text-white shadow-sm dark:bg-primary-500 dark:text-white";
const inactiveToggleClass =
  "text-slate-500 hover:text-primary-600 dark:text-slate-300 dark:hover:text-primary-200";

const loading = ref(false);
const errorMessage = ref("");
const etag = ref<string | null>(null);
const catalog = ref<PublicModelCatalogSnapshot | null>(null);
const usdToCnyRate = ref<number | null>(null);
const showDetailDialog = ref(false);
const selectedItem = ref<PublicModelCatalogItem | null>(null);

const selectedProvider = ref("");
const selectedProtocol = ref("");
const selectedMultiplier = ref("");
const searchQuery = ref("");
const viewMode = ref<CatalogViewMode>("grid");

const modelCountLabel = computed(() =>
  t("ui.modelCatalog.modelCount", { count: catalog.value?.items.length || 0 }),
);

const providerOptions = computed<CatalogFilterOption[]>(() => {
  const seen = new Map<string, CatalogFilterOption>();
  for (const item of catalog.value?.items || []) {
    const id = normalizeProviderSlug(item.provider || item.provider_icon_key || "");
    if (!id) {
      continue;
    }
    const current = seen.get(id);
    const label = formatProviderLabel(id);
    if (current) {
      current.count += 1;
      continue;
    }
    seen.set(id, {
      id,
      label,
      count: 1,
      iconKey: item.provider_icon_key || id,
    });
  }
  return Array.from(seen.values()).sort((left, right) =>
    left.label.localeCompare(right.label),
  );
});

const protocolOptions = computed<CatalogFilterOption[]>(() => {
  const seen = new Map<string, CatalogFilterOption>();
  for (const item of catalog.value?.items || []) {
    for (const protocolValue of item.request_protocols || []) {
      const protocol = normalizeProviderSlug(protocolValue);
      if (!protocol) {
        continue;
      }
      const current = seen.get(protocol);
      if (current) {
        current.count += 1;
        continue;
      }
      seen.set(protocol, {
        id: protocol,
        label: protocolLabel(protocol),
        count: 1,
        iconKey: protocol,
      });
    }
  }
  return Array.from(seen.values()).sort((left, right) => {
    const leftIndex = PUBLIC_MODEL_PROTOCOL_ORDER.indexOf(
      left.id as (typeof PUBLIC_MODEL_PROTOCOL_ORDER)[number],
    );
    const rightIndex = PUBLIC_MODEL_PROTOCOL_ORDER.indexOf(
      right.id as (typeof PUBLIC_MODEL_PROTOCOL_ORDER)[number],
    );
    const safeLeft = leftIndex >= 0 ? leftIndex : Number.MAX_SAFE_INTEGER;
    const safeRight = rightIndex >= 0 ? rightIndex : Number.MAX_SAFE_INTEGER;
    if (safeLeft !== safeRight) {
      return safeLeft - safeRight;
    }
    return left.label.localeCompare(right.label);
  });
});

const protocolItemCount = computed(() =>
  (catalog.value?.items || []).filter((item) => (item.request_protocols || []).length > 0).length,
);

const multiplierOptions = computed<MultiplierFilterOption[]>(() => {
  const values = new Map<string, MultiplierFilterOption>();
  for (const item of catalog.value?.items || []) {
    const option = multiplierFilterOptionForItem(item);
    const current = values.get(option.id);
    if (current) {
      current.count += 1;
      continue;
    }
    values.set(option.id, option);
  }
  return Array.from(values.values()).sort((left, right) => {
    const leftWeight = multiplierSortWeight(left.id);
    const rightWeight = multiplierSortWeight(right.id);
    if (leftWeight !== rightWeight) {
      return leftWeight - rightWeight;
    }
    return left.label.localeCompare(right.label);
  });
});

const normalizedSearchQuery = computed(() => searchQuery.value.trim().toLowerCase());

const filteredItems = computed(() =>
  (catalog.value?.items || []).filter((item) => {
    if (selectedProvider.value && normalizeProviderSlug(item.provider) !== selectedProvider.value) {
      return false;
    }
    if (
      selectedProtocol.value &&
      !(item.request_protocols || []).some(
        (protocol) => normalizeProviderSlug(protocol) === selectedProtocol.value,
      )
    ) {
      return false;
    }
    if (selectedMultiplier.value && multiplierFilterID(item) !== selectedMultiplier.value) {
      return false;
    }
    if (normalizedSearchQuery.value) {
      const haystack = [
        item.display_name,
        item.model,
        ...(item.source_ids || []),
      ]
        .join("\n")
        .toLowerCase();
      if (!haystack.includes(normalizedSearchQuery.value)) {
        return false;
      }
    }
    return true;
  }),
);

watch(viewMode, (nextMode) => {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.setItem(PUBLIC_MODEL_CATALOG_VIEW_MODE_KEY, nextMode);
});

onMounted(() => {
  hydrateViewMode();
  loadCatalog().catch(() => undefined);
});

async function loadCatalog(force = false) {
  loading.value = true;
  errorMessage.value = "";
  try {
    const response = await getModelCatalog(force ? null : etag.value);
    if (!response.notModified && response.data) {
      catalog.value = response.data;
    }
    etag.value = response.etag;
    if ((catalog.value?.items || []).some((item) => item.currency === "CNY")) {
      const rate = await getUSDCNYExchangeRate();
      usdToCnyRate.value = rate.rate;
    }
  } catch (error) {
    errorMessage.value = resolveErrorMessage(
      error,
      t("ui.modelCatalog.loadFailed"),
    );
  } finally {
    loading.value = false;
  }
}

function hydrateViewMode() {
  if (typeof window === "undefined") {
    return;
  }
  const stored = window.localStorage.getItem(PUBLIC_MODEL_CATALOG_VIEW_MODE_KEY);
  if (stored === "grid" || stored === "list") {
    viewMode.value = stored;
  }
}

async function copyModelID(item: PublicModelCatalogItem) {
  const modelID = String(item.model || "").trim();
  if (!modelID) {
    return;
  }
  try {
    if (!navigator?.clipboard?.writeText) {
      throw new Error("clipboard_unavailable");
    }
    await navigator.clipboard.writeText(modelID);
    appStore.showSuccess(t("ui.modelCatalog.copySuccess", { model: modelID }));
  } catch {
    appStore.showError(t("ui.modelCatalog.copyFailed"));
  }
}

function openDetail(item: PublicModelCatalogItem) {
  selectedItem.value = item;
  showDetailDialog.value = true;
}

function closeDetail() {
  showDetailDialog.value = false;
}

function providerLabel(item: PublicModelCatalogItem): string {
  return formatProviderLabel(item.provider);
}

function protocolLabel(protocol: string): string {
  switch (normalizeProviderSlug(protocol)) {
    case "openai":
      return "OpenAI";
    case "anthropic":
      return "Anthropic";
    case "gemini":
      return "Gemini";
    case "grok":
      return "Grok";
    case "antigravity":
      return "Antigravity";
    case "vertex-batch":
      return "Vertex Batch";
    default:
      return formatProviderLabel(protocol);
  }
}

function multiplierFilterOptionForItem(item: PublicModelCatalogItem): MultiplierFilterOption {
  const filterID = multiplierFilterID(item);
  if (filterID === "disabled") {
    return {
      id: filterID,
      label: t("ui.modelCatalog.multiplier.disabled"),
      count: 1,
      iconName: "calculator",
    };
  }
  if (filterID === "mixed") {
    return {
      id: filterID,
      label: t("ui.modelCatalog.multiplier.mixed"),
      count: 1,
      iconName: "sparkles",
    };
  }
  return {
    id: filterID,
    label: `${formatNumber(item.multiplier_summary.value || 1)}x`,
    count: 1,
    iconName: "badge",
  };
}

function multiplierFilterID(item: PublicModelCatalogItem): string {
  const summary = item.multiplier_summary;
  if (summary.kind === "disabled") {
    return "disabled";
  }
  if (summary.kind === "mixed") {
    return "mixed";
  }
  return `uniform:${summary.value ?? 1}`;
}

function multiplierSortWeight(value: string): number {
  if (value === "disabled") {
    return 0;
  }
  if (value === "mixed") {
    return 10000;
  }
  const numeric = Number(value.slice("uniform:".length));
  return Number.isFinite(numeric) ? numeric : 5000;
}

function priceEntryLabel(fieldID: string): string {
  return renderPriceEntryLabel(t, fieldID);
}

function multiplierSummaryLabel(
  summary: PublicModelCatalogMultiplierSummary,
): string {
  return renderMultiplierSummaryLabel(t, summary);
}

function formatCatalogPrice(
  entry: PublicModelCatalogPriceEntry,
  currency: string,
): string {
  return renderCatalogPrice(t, entry, currency, usdToCnyRate.value);
}

function primaryPriceClass(fieldID: string): string {
  switch (fieldID) {
    case "input_price":
    case "input_price_above_threshold":
    case "batch_input_price":
      return "text-sky-700 dark:text-sky-300";
    case "output_price":
    case "output_price_above_threshold":
    case "batch_output_price":
      return "text-emerald-700 dark:text-emerald-300";
    case "cache_price":
    case "batch_cache_price":
      return "text-amber-700 dark:text-amber-300";
    default:
      return "text-fuchsia-700 dark:text-fuchsia-300";
  }
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: value >= 1 ? 3 : 6,
  }).format(value);
}

function resolveErrorMessage(error: unknown, fallback: string): string {
  if (
    typeof error === "object" &&
    error &&
    "message" in error &&
    typeof (error as { message?: unknown }).message === "string"
  ) {
    return String((error as { message: string }).message);
  }
  return fallback;
}
</script>

<style scoped>
.filter-icon-shell {
  @apply inline-flex h-10 w-10 items-center justify-center rounded-2xl border border-slate-200 bg-white/70 text-current dark:border-dark-700 dark:bg-dark-900/70;
}
</style>
