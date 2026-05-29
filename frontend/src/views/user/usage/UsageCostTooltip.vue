<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed z-[9999] pointer-events-none -translate-y-1/2"
      :style="{ left: position.x + 'px', top: position.y + 'px' }"
    >
      <div class="whitespace-nowrap rounded-lg border border-gray-700 bg-gray-900 px-3 py-2.5 text-xs text-white shadow-xl dark:border-gray-600 dark:bg-gray-800">
        <div class="space-y-1.5">
          <div class="mb-2 border-b border-gray-700 pb-1.5">
            <div class="text-xs font-semibold text-gray-300 mb-1">
              {{ t("usage.costDetails") }}
            </div>
            <div v-if="data && hasPositiveUsageAmount(data.input_cost)" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t("admin.usage.inputCost") }}</span>
              <span class="font-medium text-white">${{ formatUsageAmount(data.input_cost) }}</span>
            </div>
            <div v-if="data && hasPositiveUsageAmount(data.output_cost)" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t("admin.usage.outputCost") }}</span>
              <span class="font-medium text-white">${{ formatUsageAmount(data.output_cost) }}</span>
            </div>
            <div v-if="data && data.input_tokens > 0" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t("usage.inputTokenPrice") }}</span>
              <span class="font-medium text-sky-300">
                {{ formatTokenPricePerMillion(data.input_cost, data.input_tokens) }}
                {{ t("usage.perMillionTokens") }}
              </span>
            </div>
            <div v-if="data && data.output_tokens > 0" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t("usage.outputTokenPrice") }}</span>
              <span class="font-medium text-violet-300">
                {{ formatTokenPricePerMillion(data.output_cost, data.output_tokens) }}
                {{ t("usage.perMillionTokens") }}
              </span>
            </div>
            <div v-if="data && hasPositiveUsageAmount(data.cache_creation_cost)" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t("admin.usage.cacheCreationCost") }}</span>
              <span class="font-medium text-white">${{ formatUsageAmount(data.cache_creation_cost) }}</span>
            </div>
            <div v-if="data && hasPositiveUsageAmount(data.cache_read_cost)" class="flex items-center justify-between gap-4">
              <span class="text-gray-400">{{ t("admin.usage.cacheReadCost") }}</span>
              <span class="font-medium text-white">${{ formatUsageAmount(data.cache_read_cost) }}</span>
            </div>
          </div>
          <div class="flex items-center justify-between gap-6">
            <span class="text-gray-400">{{ t("usage.serviceTier") }}</span>
            <span class="font-semibold text-cyan-300">{{ getUsageServiceTierLabel(data?.service_tier, t) }}</span>
          </div>
          <div class="flex items-center justify-between gap-6">
            <span class="text-gray-400">{{ t("usage.rate") }}</span>
            <span class="font-semibold text-blue-400">{{ formatUsageMultiplier(data?.rate_multiplier) }}x</span>
          </div>
          <div class="flex items-center justify-between gap-6">
            <span class="text-gray-400">{{ t("usage.original") }}</span>
            <span class="font-medium text-white">${{ formatUsageAmount(data?.total_cost) }}</span>
          </div>
          <div class="flex items-center justify-between gap-6 border-t border-gray-700 pt-1.5">
            <span class="text-gray-400">{{ t("usage.billed") }}</span>
            <span class="font-semibold text-green-400">${{ formatUsageAmount(data?.actual_cost) }}</span>
          </div>
          <div v-if="data?.billing_exempt_reason === 'admin_free'" class="flex items-center justify-between gap-6">
            <span class="text-gray-400">免扣原因</span>
            <span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[11px] font-medium text-emerald-300">
              <Icon name="crown" size="xs" class="h-3 w-3" />
              管理员免费
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
import Icon from "@/components/icons/Icon.vue";
import type { UsageLog } from "@/types";
import { formatTokenPricePerMillion } from "@/utils/usagePricing";
import { getUsageServiceTierLabel } from "@/utils/usageServiceTier";
import { formatUsageAmount, formatUsageMultiplier, hasPositiveUsageAmount } from "@/utils/usageCost";

defineProps<{
  visible: boolean;
  position: { x: number; y: number };
  data: UsageLog | null;
}>();

const { t } = useI18n();
</script>
