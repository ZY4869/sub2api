<template>
  <div>
    <label class="input-label">{{ label }}</label>
    <div class="relative">
      <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
      <input
        :value="modelValue"
        type="number"
        step="0.01"
        min="0"
        class="input pl-7"
        :placeholder="'0'"
        @input="$emit('update:modelValue', parseNumber(($event.target as HTMLInputElement).value))"
      />
    </div>
    <div v-if="showUsage" class="mt-2">
      <div class="flex items-center gap-2">
        <div class="flex-1 rounded-lg bg-gray-100 px-3 py-2 text-sm dark:bg-dark-700">
          <span
            :class="[
              'font-medium',
              usage >= limit
                ? 'text-red-500'
                : usage >= limit * 0.8
                  ? 'text-yellow-500'
                  : 'text-gray-900 dark:text-white',
            ]"
          >
            ${{ usage?.toFixed(4) || "0.0000" }}
          </span>
          <span class="mx-2 text-gray-400">/</span>
          <span class="text-gray-500 dark:text-gray-400">${{ limit?.toFixed(2) || "0.00" }}</span>
        </div>
      </div>
      <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
        <div
          :class="[
            'h-full rounded-full transition-all',
            usage >= limit ? 'bg-red-500' : usage >= limit * 0.8 ? 'bg-yellow-500' : 'bg-green-500',
          ]"
          :style="{ width: Math.min((usage / limit) * 100, 100) + '%' }"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { ApiKey } from "@/types";

const props = defineProps<{
  modelValue: number | null;
  label: string;
  selectedKey: ApiKey | null;
  usageKey: "usage_5h" | "usage_1d" | "usage_7d";
  limitKey: "rate_limit_5h" | "rate_limit_1d" | "rate_limit_7d";
  showEditModal: boolean;
}>();

defineEmits<{
  "update:modelValue": [value: number | null];
}>();

const usage = computed(() => Number(props.selectedKey?.[props.usageKey] ?? 0));
const limit = computed(() => Number(props.selectedKey?.[props.limitKey] ?? 0));
const showUsage = computed(() => props.showEditModal && !!props.selectedKey && limit.value > 0);

function parseNumber(value: string): number | null {
  if (value === "") return null;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}
</script>
