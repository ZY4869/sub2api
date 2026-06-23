<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { BatchApiKeyUsageStats } from "@/api/usage";
import type { ApiKey } from "@/types";

const props = defineProps<{
  apiKey: ApiKey;
  stats?: BatchApiKeyUsageStats;
  isAdminMode?: boolean;
}>();

const { t } = useI18n();

const imageQuotaEnabled = computed(
  () =>
    !!props.isAdminMode &&
    props.apiKey.image_only_enabled &&
    props.apiKey.image_count_billing_enabled,
);
const imageUsed = computed(() => Math.max(0, Number(props.apiKey.image_count_used || 0)));
const imageMax = computed(() => Math.max(0, Number(props.apiKey.image_max_count || 0)));
const imageRemaining = computed(() => Math.max(imageMax.value - imageUsed.value, 0));
const imagePercent = computed(() => {
  if (imageMax.value <= 0) return 0;
  return Math.min((imageUsed.value / imageMax.value) * 100, 100);
});
const imageStatusClass = computed(() => {
  if (imageMax.value > 0 && imageUsed.value >= imageMax.value) return "text-red-500";
  if (imageMax.value > 0 && imageUsed.value >= imageMax.value * 0.8) return "text-yellow-500";
  return "text-gray-900 dark:text-white";
});
const quotaStatusClass = computed(() => {
  if (props.apiKey.quota_used >= props.apiKey.quota) return "text-red-500";
  if (props.apiKey.quota_used >= props.apiKey.quota * 0.8) return "text-yellow-500";
  return "text-gray-900 dark:text-white";
});
</script>

<template>
  <div class="min-w-[11rem] text-sm">
    <div class="flex items-center gap-1.5">
      <span class="text-gray-500 dark:text-gray-400">{{ t("keys.today") }}:</span>
      <span class="font-medium text-gray-900 dark:text-white">
        ${{ (stats?.today_actual_cost ?? 0).toFixed(4) }}
      </span>
    </div>
    <div class="mt-0.5 flex items-center gap-1.5">
      <span class="text-gray-500 dark:text-gray-400">{{ t("keys.total") }}:</span>
      <span class="font-medium text-gray-900 dark:text-white">
        ${{ (stats?.total_actual_cost ?? 0).toFixed(4) }}
      </span>
    </div>

    <div v-if="apiKey.quota > 0" class="mt-1.5">
      <div class="flex items-center gap-1.5">
        <span class="text-gray-500 dark:text-gray-400">{{ t("keys.quota") }}:</span>
        <span :class="['font-medium', quotaStatusClass]">
          ${{ apiKey.quota_used?.toFixed(2) || "0.00" }} / ${{ apiKey.quota?.toFixed(2) }}
        </span>
      </div>
      <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
        <div
          :class="[
            'h-full rounded-full transition-all',
            apiKey.quota_used >= apiKey.quota
              ? 'bg-red-500'
              : apiKey.quota_used >= apiKey.quota * 0.8
                ? 'bg-yellow-500'
                : 'bg-primary-500',
          ]"
          :style="{ width: Math.min((apiKey.quota_used / apiKey.quota) * 100, 100) + '%' }"
        />
      </div>
    </div>

    <div v-if="imageQuotaEnabled" class="mt-2 border-t border-gray-100 pt-1.5 dark:border-dark-700">
      <div class="flex flex-wrap items-center gap-x-1.5 gap-y-0.5">
        <span class="text-gray-500 dark:text-gray-400">{{ t("keys.imageCountUsage") }}:</span>
        <span :class="['font-medium', imageStatusClass]">
          {{ imageUsed }} / {{ imageMax > 0 ? imageMax : t("keys.unlimited") }}
        </span>
      </div>
      <div class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
        {{ t("keys.imageCountRemaining") }}: {{ imageMax > 0 ? imageRemaining : t("keys.unlimited") }}
      </div>
      <div
        v-if="imageMax > 0"
        class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600"
        :aria-label="t('keys.imageCountUsage')"
      >
        <div
          :class="[
            'h-full rounded-full transition-all',
            imageUsed >= imageMax ? 'bg-red-500' : imageUsed >= imageMax * 0.8 ? 'bg-yellow-500' : 'bg-emerald-500',
          ]"
          :style="{ width: imagePercent + '%' }"
        />
      </div>
    </div>
  </div>
</template>
