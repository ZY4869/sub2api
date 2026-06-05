<template>
  <span class="relative flex h-3 w-3 items-center justify-center">
    <svg
      :class="['absolute h-3 w-3 opacity-20', className]"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      stroke-width="3.2"
    >
      <path stroke-linecap="round" stroke-linejoin="round" d="M3 12h3l2-5 4 10 3-7 2 2h4" />
    </svg>
    <svg
      :class="['absolute h-3 w-3 account-today-stats-ecg', className]"
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
      stroke-width="3.2"
    >
      <path stroke-linecap="round" stroke-linejoin="round" d="M3 12h3l2-5 4 10 3-7 2 2h4" />
    </svg>
  </span>
</template>

<script setup lang="ts">
defineProps<{
  className: string
}>()
</script>

<style scoped>
.account-today-stats-ecg path {
  stroke-dasharray: 15 100;
  animation: account-today-stats-sweep 2s linear infinite;
}

@keyframes account-today-stats-sweep {
  0% {
    stroke-dashoffset: 100;
    opacity: 0;
  }
  15% {
    opacity: 1;
  }
  85% {
    opacity: 1;
  }
  100% {
    stroke-dashoffset: -20;
    opacity: 0;
  }
}
</style>
