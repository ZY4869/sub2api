<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed z-[9999] pointer-events-none -translate-y-1/2"
      :style="{ left: position.x + 'px', top: position.y + 'px' }"
    >
      <div class="whitespace-nowrap rounded-lg border border-gray-700 bg-gray-900 px-3 py-2.5 text-xs text-white shadow-xl dark:border-gray-600 dark:bg-gray-800">
        <div class="space-y-1.5">
          <div>
            <div class="text-xs font-semibold text-gray-300 mb-1">
              {{ t("usage.tokenDetails") }}
            </div>
            <div v-if="data && data.input_tokens > 0" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t("admin.usage.inputTokens") }}</span>
              <span class="font-medium text-white">{{ data.input_tokens.toLocaleString() }}</span>
            </div>
            <div v-if="data && data.output_tokens > 0" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t("admin.usage.outputTokens") }}</span>
              <span class="font-medium text-white">{{ data.output_tokens.toLocaleString() }}</span>
            </div>
            <div v-if="data && data.cache_creation_tokens > 0">
              <template v-if="data.cache_creation_5m_tokens > 0 || data.cache_creation_1h_tokens > 0">
                <div v-if="data.cache_creation_5m_tokens > 0" class="flex items-center justify-between gap-4">
                  <span class="text-gray-400 flex items-center gap-1.5">
                    {{ t("admin.usage.cacheCreation5mTokens") }}
                    <span class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-amber-500/20 text-amber-400 ring-1 ring-inset ring-amber-500/30">5m</span>
                  </span>
                  <span class="font-medium text-white">{{ data.cache_creation_5m_tokens.toLocaleString() }}</span>
                </div>
                <div v-if="data.cache_creation_1h_tokens > 0" class="flex items-center justify-between gap-4">
                  <span class="text-gray-400 flex items-center gap-1.5">
                    {{ t("admin.usage.cacheCreation1hTokens") }}
                    <span class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-orange-500/20 text-orange-400 ring-1 ring-inset ring-orange-500/30">1h</span>
                  </span>
                  <span class="font-medium text-white">{{ data.cache_creation_1h_tokens.toLocaleString() }}</span>
                </div>
              </template>
              <div v-else class="flex items-center justify-between gap-4">
                <span class="text-gray-400">{{ getCacheCreationLabel(data) }}</span>
                <span class="font-medium text-white">{{ data.cache_creation_tokens.toLocaleString() }}</span>
              </div>
            </div>
            <div v-if="data && data.cache_ttl_overridden" class="flex items-center justify-between gap-4">
              <span class="text-gray-400 flex items-center gap-1.5">
                {{ t("usage.cacheTtlOverriddenLabel") }}
                <span class="inline-flex items-center rounded px-1 py-px text-[10px] font-medium leading-tight bg-rose-500/20 text-rose-400 ring-1 ring-inset ring-rose-500/30">
                  R-{{ data.cache_creation_1h_tokens > 0 ? "5m" : "1H" }}
                </span>
              </span>
              <span class="font-medium text-rose-400">
                {{ data.cache_creation_1h_tokens > 0 ? t("usage.cacheTtlOverridden1h") : t("usage.cacheTtlOverridden5m") }}
              </span>
            </div>
            <div v-if="data && data.cache_read_tokens > 0" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ getCacheReadLabel(data) }}</span>
              <span class="font-medium text-white">{{ data.cache_read_tokens.toLocaleString() }}</span>
            </div>
          </div>
          <div class="flex items-center justify-between gap-6 border-t border-gray-700 pt-1.5">
            <span class="text-gray-400">{{ t("usage.totalTokens") }}</span>
            <span class="font-semibold text-blue-400">
              {{
                (
                  (data?.input_tokens || 0) +
                  (data?.output_tokens || 0) +
                  (data?.cache_creation_tokens || 0) +
                  (data?.cache_read_tokens || 0)
                ).toLocaleString()
              }}
            </span>
          </div>
        </div>
        <div class="absolute right-full top-1/2 h-0 w-0 -translate-y-1/2 border-b-[6px] border-r-[6px] border-t-[6px] border-b-transparent border-r-gray-900 border-t-transparent dark:border-r-gray-800"></div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import type { UsageLog } from "@/types";

defineProps<{
  visible: boolean;
  position: { x: number; y: number };
  data: UsageLog | null;
  getCacheReadLabel: (row: Pick<UsageLog, "upstream_service"> | null | undefined) => string;
  getCacheCreationLabel: (row: Pick<UsageLog, "upstream_service"> | null | undefined) => string;
}>();

const { t } = useI18n();
</script>
