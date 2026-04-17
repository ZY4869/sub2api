<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { usageAPI } from "@/api";
import type { UsageLog, UsageRequestPreviewResponse } from "@/types";
import { formatDateTime } from "@/utils/format";
import UsageRequestPreviewPanel from "./UsageRequestPreviewPanel.vue";

const props = defineProps<{
  show: boolean;
  usageLog: UsageLog | null;
}>();

const emit = defineEmits<{
  (e: "close"): void;
}>();

const { t } = useI18n();

const loading = ref(false);
const loadFailed = ref(false);
const preview = ref<UsageRequestPreviewResponse | null>(null);
let currentLoadID = 0;

const sections = computed(() => [
  {
    key: "inbound_request_json",
    title: t("usage.requestPreview.sections.inbound"),
    emptyMessage: t("usage.requestPreview.emptyStates.inbound"),
  },
  {
    key: "normalized_request_json",
    title: t("usage.requestPreview.sections.normalized"),
    emptyMessage: t("usage.requestPreview.emptyStates.normalized"),
  },
  {
    key: "upstream_request_json",
    title: t("usage.requestPreview.sections.upstreamRequest"),
    emptyMessage: t("usage.requestPreview.emptyStates.upstreamRequest"),
  },
  {
    key: "upstream_response_json",
    title: t("usage.requestPreview.sections.upstreamResponse"),
    emptyMessage: t("usage.requestPreview.emptyStates.upstreamResponse"),
  },
  {
    key: "gateway_response_json",
    title: t("usage.requestPreview.sections.gatewayResponse"),
    emptyMessage: t("usage.requestPreview.emptyStates.gatewayResponse"),
  },
  {
    key: "tool_trace_json",
    title: t("usage.requestPreview.sections.tools"),
    emptyMessage: t("usage.requestPreview.emptyStates.tools"),
  },
]);

const selectedRequestID = computed(
  () => preview.value?.request_id || props.usageLog?.request_id || "-",
);

const capturedAt = computed(() =>
  preview.value?.captured_at ? formatDateTime(preview.value.captured_at) : "-",
);

const resetState = () => {
  loading.value = false;
  loadFailed.value = false;
  preview.value = null;
};

const loadPreview = async () => {
  const usageID = props.usageLog?.id;
  if (!props.show || !usageID) {
    resetState();
    return;
  }

  const loadID = ++currentLoadID;
  loading.value = true;
  loadFailed.value = false;
  preview.value = null;

  try {
    const response = await usageAPI.getRequestPreview(usageID);
    if (loadID !== currentLoadID) {
      return;
    }
    preview.value = response;
  } catch {
    if (loadID !== currentLoadID) {
      return;
    }
    loadFailed.value = true;
  } finally {
    if (loadID === currentLoadID) {
      loading.value = false;
    }
  }
};

watch(
  () => [props.show, props.usageLog?.id] as const,
  ([show, usageID]) => {
    if (!show || !usageID) {
      currentLoadID += 1;
      resetState();
      return;
    }
    void loadPreview();
  },
  { immediate: true },
);
</script>

<template>
  <Teleport to="body">
    <div
      v-if="show"
      class="fixed inset-0 z-40 bg-black/50"
      @click="emit('close')"
    />

    <section
      v-if="show"
      class="fixed inset-x-4 top-6 z-50 mx-auto flex max-h-[calc(100vh-3rem)] w-full max-w-6xl flex-col overflow-hidden rounded-3xl border border-gray-200 bg-white shadow-2xl dark:border-dark-700 dark:bg-dark-900"
    >
      <header
        class="flex items-start justify-between gap-4 border-b border-gray-100 px-6 py-5 dark:border-dark-700"
      >
        <div class="min-w-0">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t("usage.requestPreview.title") }}
          </h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t("usage.requestPreview.description") }}
          </p>
          <div
            class="mt-3 flex flex-wrap gap-x-6 gap-y-2 text-xs text-gray-500 dark:text-gray-400"
          >
            <span>
              {{ t("usage.requestPreview.metaRequestId") }}:
              {{ selectedRequestID }}
            </span>
            <span>
              {{ t("usage.requestPreview.metaCapturedAt") }}:
              {{ capturedAt }}
            </span>
          </div>
        </div>
        <button
          class="btn btn-secondary btn-sm"
          type="button"
          @click="emit('close')"
        >
          {{ t("common.close") }}
        </button>
      </header>

      <div class="overflow-y-auto px-6 py-5">
        <div
          v-if="loading"
          class="flex min-h-[320px] items-center justify-center text-sm text-gray-500 dark:text-gray-400"
        >
          {{ t("common.loading") }}
        </div>

        <div
          v-else-if="loadFailed"
          class="rounded-2xl border border-rose-200 bg-rose-50 px-5 py-4 text-sm text-rose-700 dark:border-rose-900/40 dark:bg-rose-900/20 dark:text-rose-200"
        >
          <p>{{ t("usage.requestPreview.failedToLoad") }}</p>
          <button
            class="btn btn-secondary btn-sm mt-3"
            type="button"
            @click="loadPreview"
          >
            {{ t("common.refresh") }}
          </button>
        </div>

        <div
          v-else-if="!preview || !preview.available"
          class="rounded-2xl border border-amber-200 bg-amber-50 px-5 py-4 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200"
        >
          <p class="font-medium">
            {{ t("usage.requestPreview.unavailableTitle") }}
          </p>
          <p class="mt-1">
            {{ t("usage.requestPreview.unavailableDescription") }}
          </p>
        </div>

        <div v-else class="grid gap-4 lg:grid-cols-2">
          <UsageRequestPreviewPanel
            v-for="section in sections"
            :key="section.key"
            :title="section.title"
            :content="
              preview[
                section.key as keyof UsageRequestPreviewResponse
              ] as string
            "
            :empty-message="section.emptyMessage"
          />
        </div>
      </div>
    </section>
  </Teleport>
</template>
