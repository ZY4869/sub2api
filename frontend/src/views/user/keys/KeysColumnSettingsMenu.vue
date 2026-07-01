<template>
  <div ref="menuRef" class="relative">
    <button
      type="button"
      class="btn btn-secondary px-2 md:px-3"
      :title="t('keys.columnSettings')"
      @click.stop="open = !open"
    >
      <Icon name="grid" size="sm" class="md:mr-1.5" :stroke-width="2" />
      <span class="hidden md:inline">{{ t("keys.columnSettings") }}</span>
    </button>

    <div
      v-if="open"
      class="absolute right-0 top-full z-50 mt-2 w-[min(18rem,calc(100vw-2rem))] rounded-lg border border-gray-200 bg-white p-3 shadow-xl dark:border-dark-600 dark:bg-dark-800"
    >
      <p class="mb-2 text-xs font-semibold text-gray-500 dark:text-gray-400">
        {{ t("keys.displaySettingsColumns") }}
      </p>
      <div class="grid max-h-64 grid-cols-1 gap-1 overflow-y-auto pr-1">
        <button
          v-for="column in toggleableColumns"
          :key="column.key"
          type="button"
          class="flex items-center justify-between rounded-md px-2.5 py-2 text-left text-sm transition hover:bg-gray-100 dark:hover:bg-dark-700"
          @click="$emit('toggle-column', column.key)"
        >
          <span class="text-gray-700 dark:text-gray-200">{{ column.label }}</span>
          <Icon
            :name="hiddenColumns.has(column.key) ? 'xCircle' : 'checkCircle'"
            size="sm"
            :class="hiddenColumns.has(column.key) ? 'text-gray-400 dark:text-gray-500' : 'text-primary-500'"
            :stroke-width="2"
          />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import Icon from "@/components/icons/Icon.vue";
import type { Column } from "@/components/common/types";

const props = defineProps<{
  hiddenColumns: Set<string>;
  columns: Column[];
  alwaysVisibleColumns: string[];
}>();

defineEmits<{
  "toggle-column": [key: string];
}>();

const { t } = useI18n();
const open = ref(false);
const menuRef = ref<HTMLElement | null>(null);

const alwaysVisibleSet = computed(() => new Set(props.alwaysVisibleColumns));
const toggleableColumns = computed(() =>
  props.columns.filter((column) => !alwaysVisibleSet.value.has(column.key)),
);

const handleClickOutside = (event: MouseEvent) => {
  if (!menuRef.value || menuRef.value.contains(event.target as Node)) {
    return;
  }
  open.value = false;
};

onMounted(() => {
  document.addEventListener("click", handleClickOutside);
});

onUnmounted(() => {
  document.removeEventListener("click", handleClickOutside);
});
</script>

