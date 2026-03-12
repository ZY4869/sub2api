<template>
  <div class="table-page-layout">
    <div v-if="$slots.actions" class="layout-section">
      <slot name="actions" />
    </div>

    <div v-if="$slots.filters" class="layout-section">
      <slot name="filters" />
    </div>

    <div class="layout-section">
      <div class="card table-scroll-container" :class="{ 'table-scroll-container-page-scroll': preferPageScroll }">
        <slot name="table" />
      </div>
    </div>

    <div v-if="$slots.pagination" class="layout-section">
      <slot name="pagination" />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{
  preferPageScroll?: boolean
}>(), {
  preferPageScroll: false
})
</script>

<style scoped>
.table-page-layout {
  @apply flex flex-col gap-6;
}

.layout-section {
  @apply min-w-0;
}

.table-scroll-container {
  @apply overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800;
}

.table-scroll-container-page-scroll {
  overflow: visible;
}

.table-scroll-container :deep(.table-wrapper) {
  @apply overflow-x-auto;
}

.table-scroll-container-page-scroll :deep(.table-wrapper) {
  overflow-y: hidden;
  scrollbar-gutter: auto;
}

.table-scroll-container :deep(table) {
  @apply w-full;
  min-width: max-content;
  display: table;
}

.table-scroll-container :deep(thead) {
  @apply bg-gray-50/80 dark:bg-dark-800/80 backdrop-blur-sm;
}

.table-scroll-container :deep(th) {
  @apply px-5 py-4 text-left text-sm font-bold uppercase tracking-wider text-gray-900 dark:text-white border-b border-gray-200 dark:border-dark-700;
}

.table-scroll-container :deep(td) {
  @apply px-5 py-4 text-sm text-gray-700 dark:text-gray-300 border-b border-gray-100 dark:border-dark-800;
}
</style>
