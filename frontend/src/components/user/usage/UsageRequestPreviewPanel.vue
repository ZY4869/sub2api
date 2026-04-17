<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  title: string;
  content?: string | null;
  emptyMessage: string;
}>();

const { t } = useI18n();

const formattedContent = computed(() => {
  const trimmed = String(props.content || "").trim();
  if (!trimmed) {
    return "";
  }
  try {
    return JSON.stringify(JSON.parse(trimmed), null, 2);
  } catch {
    return trimmed;
  }
});

const hasContent = computed(() => formattedContent.value.length > 0);
</script>

<template>
  <section class="rounded-2xl border border-gray-200 dark:border-dark-700">
    <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
      <div class="text-sm font-semibold text-gray-900 dark:text-white">
        {{ title }}
      </div>
      <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
        {{
          hasContent
            ? t("usage.requestPreview.previewReady")
            : t("usage.requestPreview.empty")
        }}
      </div>
    </div>

    <div class="p-4">
      <pre
        v-if="hasContent"
        class="max-h-[280px] overflow-auto rounded-2xl bg-gray-50 p-4 text-xs text-gray-800 dark:bg-dark-800 dark:text-gray-200"
      ><code>{{ formattedContent }}</code></pre>
      <div
        v-else
        class="flex min-h-[180px] items-center justify-center rounded-2xl border border-dashed border-gray-200 bg-gray-50 px-6 text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400"
      >
        {{ emptyMessage }}
      </div>
    </div>
  </section>
</template>
