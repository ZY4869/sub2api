<template>
  <span
    :class="[
      'inline-flex max-w-full items-center gap-1.5 rounded-md px-2 py-0.5 text-xs font-medium',
      badgeClass,
    ]"
    :title="label"
    :aria-label="label"
  >
    <PlatformIcon :platform="platform" size="sm" />
    <span class="min-w-0 truncate">{{ name }}</span>
    <span
      v-if="effectiveRate !== undefined"
      class="rounded bg-black/10 px-1.5 py-0.5 text-[10px] font-semibold dark:bg-white/10"
    >
      {{ effectiveRate }}x
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import PlatformIcon from "@/components/common/PlatformIcon.vue";
import type { GroupPlatform, SubscriptionType } from "@/types";

const props = defineProps<{
  name: string;
  platform?: GroupPlatform | string;
  subscriptionType?: SubscriptionType;
  rateMultiplier?: number;
  userRateMultiplier?: number | null;
}>();

const effectiveRate = computed(() => {
  return props.userRateMultiplier ?? props.rateMultiplier;
});

const label = computed(() => {
  const parts = [props.name];
  if (effectiveRate.value !== undefined) {
    parts.push(`${effectiveRate.value}x`);
  }
  return parts.join(" ");
});

const badgeClass = computed(() => {
  if (props.subscriptionType === "subscription") {
    return "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300";
  }
  if (props.platform === "anthropic" || props.platform === "kiro") {
    return "bg-amber-50 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300";
  }
  if (props.platform === "gemini") {
    return "bg-sky-50 text-sky-700 dark:bg-sky-900/20 dark:text-sky-300";
  }
  if (props.platform === "openai") {
    return "bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-300";
  }
  return "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300";
});
</script>
