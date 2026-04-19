<template>
  <BaseDialog
    :show="show"
    :title="dialogTitle"
    width="extra-wide"
    @close="emit('close')"
  >
    <div class="space-y-5">
      <div class="flex flex-wrap items-start justify-between gap-4">
        <div class="min-w-0">
          <div class="flex items-center gap-3">
            <ModelIcon
              v-if="displayItem"
              :model="displayItem.model"
              :provider="displayItem.provider"
              :display-name="displayItem.display_name"
              size="22px"
            />
            <div class="min-w-0">
              <p class="truncate text-lg font-semibold text-slate-950 dark:text-white">
                {{ displayItem?.display_name || displayItem?.model || "-" }}
              </p>
              <p class="truncate text-sm text-slate-500 dark:text-slate-400">
                {{ displayItem?.model || "-" }}
              </p>
            </div>
          </div>
          <div
            v-if="displayItem?.request_protocols?.length"
            class="mt-3 flex flex-wrap gap-2 text-xs"
          >
            <span
              v-for="protocol in displayItem.request_protocols"
              :key="protocol"
              class="rounded-full bg-sky-100 px-2.5 py-1 text-sky-700 dark:bg-sky-500/15 dark:text-sky-200"
            >
              {{ protocol }}
            </span>
          </div>
        </div>

        <div class="rounded-2xl bg-slate-50 px-4 py-3 text-right dark:bg-dark-800">
          <div class="text-xs uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">
            {{ displayItem?.currency || "USD" }}
          </div>
          <div class="mt-2 text-sm font-semibold text-slate-900 dark:text-white">
            {{ multiplierLabel }}
          </div>
        </div>
      </div>

      <div class="grid gap-5 xl:grid-cols-[minmax(0,0.88fr)_minmax(0,1.12fr)]">
        <section class="rounded-3xl border border-slate-200 bg-slate-50/70 p-5 dark:border-dark-700 dark:bg-dark-900/60">
          <div class="flex items-center justify-between gap-3">
            <h3 class="text-sm font-semibold text-slate-900 dark:text-white">
              {{ t("ui.modelCatalog.detail.pricingTitle") }}
            </h3>
            <span class="text-xs text-slate-500 dark:text-slate-400">
              {{ displayItem?.currency || "USD" }}
            </span>
          </div>

          <div v-if="displayItem" class="mt-4 space-y-3">
            <div
              v-for="entry in displayItem.price_display.primary"
              :key="entry.id"
              class="flex items-center justify-between gap-4 rounded-2xl bg-white px-4 py-3 text-sm shadow-sm dark:bg-dark-800"
            >
              <span class="text-slate-600 dark:text-slate-300">
                {{ renderPriceEntryLabel(entry.id) }}
              </span>
              <span class="font-semibold text-slate-950 dark:text-white">
                {{ renderPrice(entry, displayItem.currency) }}
              </span>
            </div>

            <div v-if="displayItem.price_display.secondary?.length" class="space-y-2">
              <p class="text-xs uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">
                {{ t("ui.modelCatalog.detail.secondaryPricingTitle") }}
              </p>
              <div class="space-y-2">
                <div
                  v-for="entry in displayItem.price_display.secondary"
                  :key="entry.id"
                  class="flex items-center justify-between gap-4 rounded-2xl border border-slate-200 px-4 py-3 text-sm dark:border-dark-700"
                >
                  <span class="text-slate-600 dark:text-slate-300">
                    {{ renderPriceEntryLabel(entry.id) }}
                  </span>
                  <span class="font-medium text-slate-950 dark:text-white">
                    {{ renderPrice(entry, displayItem.currency) }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="rounded-3xl border border-slate-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900/70">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h3 class="text-sm font-semibold text-slate-900 dark:text-white">
                {{ t("ui.modelCatalog.detail.exampleTitle") }}
              </h3>
              <p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
                {{ exampleCaption }}
              </p>
            </div>

            <div
              v-if="supportedKeys.length > 1"
              class="w-full max-w-xs"
            >
              <label class="mb-1 block text-xs font-medium text-slate-600 dark:text-slate-300">
                {{ t("ui.modelCatalog.detail.keySelector") }}
              </label>
              <select
                v-model="selectedKeyID"
                class="input"
              >
                <option
                  v-for="item in supportedKeys"
                  :key="item.id"
                  :value="item.id"
                >
                  {{ item.name }}
                </option>
              </select>
            </div>
          </div>

          <div class="mt-4 space-y-4">
            <div
              v-if="loading"
              class="rounded-2xl border border-dashed border-slate-300 px-4 py-8 text-sm text-slate-500 dark:border-dark-700 dark:text-slate-300"
            >
              {{ t("ui.modelCatalog.detail.loading") }}
            </div>

            <div
              v-else-if="errorMessage"
              class="rounded-2xl border border-rose-200 bg-rose-50 px-4 py-4 text-sm text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/20 dark:text-rose-200"
            >
              {{ errorMessage }}
            </div>

            <div
              v-else-if="exampleResult.group"
              class="space-y-3"
            >
              <div class="flex flex-wrap gap-2 text-xs">
                <span class="rounded-full bg-slate-100 px-2.5 py-1 text-slate-700 dark:bg-dark-800 dark:text-slate-200">
                  {{ detail?.example_protocol || displayItem?.provider || "openai" }}
                </span>
                <span
                  v-if="detail?.example_source"
                  class="rounded-full bg-emerald-100 px-2.5 py-1 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-200"
                >
                  {{
                    detail.example_source === "docs_section"
                      ? t("ui.modelCatalog.detail.exampleSourceDocs")
                      : t("ui.modelCatalog.detail.exampleSourceOverride")
                  }}
                </span>
              </div>

              <div
                class="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-xs text-slate-600 dark:border-dark-700 dark:bg-dark-800 dark:text-slate-300"
              >
                {{ keyHint }}
              </div>

              <DocsCodeTabs
                :group="exampleResult.group"
                :theme="docsTheme"
              />
            </div>

            <div
              v-else
              class="rounded-2xl border border-dashed border-slate-300 px-4 py-8 text-sm text-slate-500 dark:border-dark-700 dark:text-slate-300"
            >
              {{ t("ui.modelCatalog.detail.exampleUnavailable") }}
            </div>
          </div>
        </section>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { getModelCatalogDetail, type PublicModelCatalogItem } from "@/api/meta";
import keysAPI from "@/api/keys";
import userGroupsAPI from "@/api/groups";
import BaseDialog from "@/components/common/BaseDialog.vue";
import DocsCodeTabs from "@/components/docs/DocsCodeTabs.vue";
import { getDocsTheme } from "@/components/docs/docsTheme";
import ModelIcon from "@/components/common/ModelIcon.vue";
import { useAppStore } from "@/stores/app";
import { useAuthStore } from "@/stores/auth";
import { buildPublicModelExample } from "@/utils/publicModelCatalogExamples";
import { findSupportedKeysForModel } from "@/utils/publicModelCatalogKeys";
import {
  formatCatalogPrice,
  multiplierSummaryLabel,
  priceEntryLabel,
} from "@/utils/publicModelCatalog";
import type { ApiKey, UserGroupModelOptionGroup } from "@/types";
import type {
  PublicModelCatalogDetailResponse,
  PublicModelCatalogPriceEntry,
} from "@/api/meta";

const props = defineProps<{
  show: boolean;
  model: string | null;
  catalogItem?: PublicModelCatalogItem | null;
  usdToCnyRate?: number | null;
}>();

const emit = defineEmits<{
  close: [];
}>();

const { t } = useI18n();
const appStore = useAppStore();
const authStore = useAuthStore();

const detail = ref<PublicModelCatalogDetailResponse | null>(null);
const loading = ref(false);
const errorMessage = ref("");
const userKeys = ref<ApiKey[]>([]);
const userGroupOptions = ref<UserGroupModelOptionGroup[]>([]);
const selectedKeyID = ref<number | null>(null);
let requestToken = 0;

const displayItem = computed(
  () => detail.value?.item || props.catalogItem || null,
);
const dialogTitle = computed(
  () => displayItem.value?.display_name || displayItem.value?.model || t("nav.modelsCatalog"),
);
const multiplierLabel = computed(() =>
  displayItem.value
    ? multiplierSummaryLabel(t, displayItem.value.multiplier_summary)
    : "-",
);
const supportedKeys = computed(() =>
  findSupportedKeysForModel(userKeys.value, userGroupOptions.value, detail.value),
);
const selectedKey = computed(
  () => supportedKeys.value.find((item) => item.id === selectedKeyID.value) || supportedKeys.value[0] || null,
);
const effectiveAPIKey = computed(
  () => selectedKey.value?.key || "sk-your-key",
);
const exampleResult = computed(() =>
  buildPublicModelExample(detail.value, effectiveAPIKey.value, resolvedBaseUrl()),
);
const docsTheme = computed(() => getDocsTheme(exampleResult.value.pageId));
const exampleCaption = computed(() => {
  if (!detail.value?.example_protocol) {
    return t("ui.modelCatalog.detail.exampleCaptionFallback");
  }
  return t("ui.modelCatalog.detail.exampleCaption", {
    protocol: detail.value.example_protocol,
  });
});
const keyHint = computed(() => {
  if (!authStore.isAuthenticated) {
    return t("ui.modelCatalog.detail.keyHintGuest");
  }
  if (selectedKey.value) {
    return t("ui.modelCatalog.detail.keyHintMatched", {
      name: selectedKey.value.name,
    });
  }
  return t("ui.modelCatalog.detail.keyHintMissing");
});

watch(
  () => [props.show, props.model, authStore.isAuthenticated] as const,
  async ([show, model]) => {
    if (!show || !model) {
      return;
    }
    const currentToken = ++requestToken;
    loading.value = true;
    errorMessage.value = "";
    try {
      const nextDetail = await getModelCatalogDetail(model);
      if (currentToken !== requestToken) {
        return;
      }
      detail.value = nextDetail;
      if (authStore.isAuthenticated) {
        await loadUserContext(currentToken);
      } else {
        userKeys.value = [];
        userGroupOptions.value = [];
      }
    } catch (error) {
      if (currentToken !== requestToken) {
        return;
      }
      detail.value = null;
      errorMessage.value = resolveErrorMessage(
        error,
        t("ui.modelCatalog.detail.loadFailed"),
      );
    } finally {
      if (currentToken === requestToken) {
        loading.value = false;
      }
    }
  },
  { immediate: true },
);

watch(
  supportedKeys,
  (items) => {
    if (items.length === 0) {
      selectedKeyID.value = null;
      return;
    }
    if (!items.some((item) => item.id === selectedKeyID.value)) {
      selectedKeyID.value = items[0].id;
    }
  },
  { immediate: true },
);

function renderPriceEntryLabel(fieldID: string): string {
  return priceEntryLabel(t, fieldID);
}

function renderPrice(entry: PublicModelCatalogPriceEntry, currency: string): string {
  return formatCatalogPrice(t, entry, currency, props.usdToCnyRate ?? null);
}

async function loadUserContext(currentToken: number) {
  try {
    const [keysResponse, groupOptions] = await Promise.all([
      keysAPI.list(1, 1000),
      userGroupsAPI.getModelOptions(),
    ]);
    if (currentToken !== requestToken) {
      return;
    }
    userKeys.value = keysResponse.items || [];
    userGroupOptions.value = groupOptions || [];
  } catch (error) {
    if (currentToken !== requestToken) {
      return;
    }
    userKeys.value = [];
    userGroupOptions.value = [];
  }
}

function resolvedBaseUrl(): string {
  const configured = String(appStore.apiBaseUrl || "").trim();
  if (configured) {
    return configured.replace(/\/+$/g, "");
  }
  if (typeof window !== "undefined" && window.location?.origin) {
    return window.location.origin.replace(/\/+$/g, "");
  }
  return "https://api.zyxai.de";
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
