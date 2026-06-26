<template>
  <button
    v-if="rawUserAgent && displayMode === 'compact'"
    type="button"
    class="inline-flex max-w-[7.5rem] items-center gap-1.5 rounded-md border border-gray-200 bg-white px-2 py-1 text-xs font-medium text-gray-700 shadow-sm transition hover:border-primary-300 hover:text-primary-600 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-primary-500 dark:hover:text-primary-300"
    :title="rawUserAgent"
    :aria-label="copyLabel"
    @click.stop="copyUserAgent"
  >
    <Icon :name="iconName" size="xs" :stroke-width="2" />
    <span class="truncate">{{ compactLabel }}</span>
  </button>
  <span
    v-else-if="rawUserAgent"
    class="block max-w-[18rem] whitespace-normal break-all text-sm text-gray-600 dark:text-gray-400"
    :title="rawUserAgent"
  >
    {{ formatUserAgent(rawUserAgent) }}
  </span>
  <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useClipboard } from "@/composables/useClipboard";
import Icon from "@/components/icons/Icon.vue";
import type { UsageViewUserAgentDisplayMode } from "@/types";

const props = withDefaults(defineProps<{
  userAgent?: string | null;
  displayMode?: UsageViewUserAgentDisplayMode;
  formatUserAgent: (ua: string) => string;
}>(), {
  displayMode: "compact",
});

const { t } = useI18n();
const { copyToClipboard } = useClipboard();

const rawUserAgent = computed(() => props.userAgent?.trim() || "");
const compactLabel = computed(() => props.formatUserAgent(rawUserAgent.value));
const copyLabel = computed(() => `${t("common.copy")}: ${compactLabel.value}`);

const iconName = computed(() => {
  const normalized = rawUserAgent.value.toLowerCase();
  if (
    normalized.includes("mozilla/") ||
    normalized.includes("chrome/") ||
    normalized.includes("safari/") ||
    normalized.includes("firefox/") ||
    normalized.includes("edg/")
  ) {
    return "globe";
  }
  if (normalized.startsWith("curl/") || normalized.includes("postman")) {
    return "terminal";
  }
  return "clipboard";
});

const copyUserAgent = async () => {
  await copyToClipboard(rawUserAgent.value, t("usage.userAgentCopied"));
};
</script>
