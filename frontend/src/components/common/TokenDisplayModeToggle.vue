<template>
  <div class="flex items-center gap-2">
    <span class="text-xs font-medium text-gray-500 dark:text-gray-400">
      {{ t("usage.tokenDisplay") }}:
    </span>
    <div
      class="inline-flex rounded-lg border border-gray-200 bg-white p-0.5 shadow-sm dark:border-dark-600 dark:bg-dark-800"
    >
      <button
        type="button"
        class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
        :class="
          tokenDisplayMode === 'full'
            ? 'bg-primary-500 text-white'
            : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'
        "
        @click="setTokenDisplayMode('full')"
      >
        {{ t("usage.tokenDisplayFull") }}
      </button>
      <button
        type="button"
        class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
        :class="
          tokenDisplayMode === 'compact'
            ? 'bg-primary-500 text-white'
            : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700'
        "
        @click="setTokenDisplayMode('compact')"
      >
        {{ t("usage.tokenDisplayCompact") }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import { useTokenDisplayMode } from "@/composables/useTokenDisplayMode";

const { t } = useI18n();
const { tokenDisplayMode, setTokenDisplayMode } = useTokenDisplayMode();
</script>
