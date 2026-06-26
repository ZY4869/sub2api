<template>
  <div v-if="lines.length > 0" class="inline-flex items-center gap-1.5">
    <span
      v-for="line in lines"
      :key="line.key"
      class="inline-flex h-7 w-7 items-center justify-center rounded-md border border-gray-200 bg-white text-gray-700 shadow-sm dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200"
      :title="lineTitle(line)"
      :aria-label="lineTitle(line)"
    >
      <PlatformIcon :platform="line.platform" size="md" />
    </span>
  </div>
  <span v-else class="text-sm text-gray-400 dark:text-gray-500">-</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import PlatformIcon from "@/components/common/PlatformIcon.vue";
import {
  resolveProtocolFamilyFromValue,
  type ProtocolFamily,
} from "@/utils/protocolDisplay";
import { formatUsageEndpointPath } from "@/utils/usageDisplay";
import type { AccountPlatform } from "@/types";

type EndpointLine = {
  key: "inbound" | "upstream";
  labelKey: "usage.inbound" | "usage.upstream";
  raw: string;
  display: string;
  platform: AccountPlatform;
};

const props = defineProps<{
  inboundPath?: string | null;
  upstreamPath?: string | null;
}>();

const { t } = useI18n();

const toPlatform = (family: ProtocolFamily): AccountPlatform => {
  switch (family) {
    case "openai":
      return "openai";
    case "anthropic":
      return "anthropic";
    case "gemini":
      return "gemini";
    default:
      return "protocol_gateway";
  }
};

const buildLine = (
  key: EndpointLine["key"],
  labelKey: EndpointLine["labelKey"],
  path: string | null | undefined,
): EndpointLine | null => {
  const raw = path?.trim();
  if (!raw) return null;
  return {
    key,
    labelKey,
    raw,
    display: formatUsageEndpointPath(raw),
    platform: toPlatform(resolveProtocolFamilyFromValue(raw)),
  };
};

const lines = computed(() =>
  [
    buildLine("inbound", "usage.inbound", props.inboundPath),
    buildLine("upstream", "usage.upstream", props.upstreamPath),
  ].filter((line): line is EndpointLine => Boolean(line)),
);

const lineTitle = (line: EndpointLine) =>
  `${t(line.labelKey)}: ${line.display} (${line.raw})`;
</script>
