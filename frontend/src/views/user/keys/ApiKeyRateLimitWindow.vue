<template>
  <div>
    <div class="flex items-center justify-between text-xs">
      <span class="text-gray-500 dark:text-gray-400">{{ label }}</span>
      <span
        :class="[
          'font-medium tabular-nums',
          usage >= limit
            ? 'text-red-500'
            : usage >= limit * 0.8
              ? 'text-yellow-500'
              : 'text-gray-700 dark:text-gray-300',
        ]"
      >
        ${{ usage?.toFixed(2) || "0.00" }}/{{ limit?.toFixed(2) }}
      </span>
    </div>
    <div class="h-1 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
      <div
        :class="[
          'h-full rounded-full transition-all',
          usage >= limit
            ? 'bg-red-500'
            : usage >= limit * 0.8
              ? 'bg-yellow-500'
              : 'bg-emerald-500',
        ]"
        :style="{ width: Math.min((usage / limit) * 100, 100) + '%' }"
      />
    </div>
    <div
      v-if="resetAt && formatResetTime(resetAt)"
      class="text-[10px] text-gray-400 dark:text-gray-500 tabular-nums"
    >
      ⟳ {{ formatResetTime(resetAt) }}
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string;
  usage: number;
  limit: number;
  resetAt: string | null;
  formatResetTime: (resetAt: string | null) => string;
}>();
</script>
