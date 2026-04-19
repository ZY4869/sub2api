<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { parseRequestPreviewContent } from "@/utils/requestPreview";

const props = defineProps<{
  title: string;
  content?: string | null;
  emptyMessage: string;
}>();

const { t } = useI18n();
const parsed = computed(() => parseRequestPreviewContent(props.content));
const hasContent = computed(() => parsed.value.hasContent);
const headerText = computed(() => {
  switch (parsed.value.renderState) {
    case "ready":
      return t("usage.requestPreview.previewReady");
    case "raw_only":
      return t("usage.requestPreview.rawOnlyStatus");
    case "empty":
      return t("usage.requestPreview.capturedEmptyStatus");
    default:
      return t("usage.requestPreview.empty");
  }
});
const emptyText = computed(() => {
  switch (parsed.value.renderState) {
    case "empty":
      return t("usage.requestPreview.capturedEmptyDescription");
    case "raw_only":
      return t("usage.requestPreview.rawOnlyDescription");
    default:
      return props.emptyMessage;
  }
});
const notices = computed(() => {
  const items: string[] = [];
  if (parsed.value.renderState === "raw_only") {
    items.push(t("usage.requestPreview.rawOnlyNotice"));
  }
  if (parsed.value.truncated) {
    items.push(t("usage.requestPreview.truncatedNotice"));
  }
  return items;
});
</script>

<template>
  <section class="rounded-2xl border border-gray-200 dark:border-dark-700">
    <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
      <div class="text-sm font-semibold text-gray-900 dark:text-white">
        {{ title }}
      </div>
      <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {{ headerText }}
      </div>
      <div
        v-if="parsed.source || parsed.truncated || parsed.renderState === 'raw_only'"
        class="mt-2 flex flex-wrap gap-2 text-[11px]"
      >
        <span
          v-if="parsed.source"
          class="inline-flex items-center rounded-full bg-sky-100 px-2 py-0.5 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300"
        >
          {{ t("usage.requestPreview.metaSource") }}: {{ parsed.source }}
        </span>
        <span
          v-if="parsed.renderState === 'raw_only'"
          class="inline-flex items-center rounded-full bg-amber-100 px-2 py-0.5 text-amber-800 dark:bg-amber-900/30 dark:text-amber-200"
        >
          {{ t("usage.requestPreview.rawOnlyBadge") }}
        </span>
        <span
          v-if="parsed.truncated"
          class="inline-flex items-center rounded-full bg-rose-100 px-2 py-0.5 text-rose-700 dark:bg-rose-900/30 dark:text-rose-200"
        >
          {{ t("usage.requestPreview.truncatedBadge") }}
        </span>
      </div>
    </div>

    <div class="p-4">
      <div
        v-for="notice in notices"
        :key="notice"
        class="mb-3 rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200"
      >
        {{ notice }}
      </div>
      <pre
        v-if="hasContent"
        class="max-h-[280px] overflow-auto rounded-2xl bg-gray-50 p-4 text-xs text-gray-800 dark:bg-dark-800 dark:text-gray-200"
      ><code>{{ parsed.displayContent }}</code></pre>
      <div
        v-else
        class="flex min-h-[180px] items-center justify-center rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-6 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400"
      >
        {{ emptyText }}
      </div>
    </div>
  </section>
</template>
