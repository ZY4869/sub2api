<template>
  <div class="rounded-2xl border border-gray-200 bg-gray-50/70 px-4 py-4 dark:border-dark-700 dark:bg-dark-900/40">
    <div class="text-xs font-medium uppercase tracking-[0.18em] text-gray-500 dark:text-gray-400">{{ title }}</div>
    <div class="mt-2 text-lg font-semibold text-gray-900 dark:text-white">{{ value }}</div>
    <div class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ meta }}</div>
    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ foot }}</div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  value: string
  meta: string
  foot: string
}>()
</script>
