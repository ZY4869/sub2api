<template>
  <div>
    <div class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ label }}</div>
    <div class="mt-1 break-words text-sm text-gray-800 dark:text-gray-100" :class="{ 'font-mono': mono }">
      {{ value || '-' }}
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string
  value: string
  mono?: boolean
}>()
</script>
