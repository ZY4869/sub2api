<template>
  <div ref="menuRef" class="relative">
    <button
      type="button"
      class="btn btn-secondary px-2 md:px-3"
      :disabled="disabled"
      :title="t('usage.displaySettings')"
      @click.stop="open = !open"
    >
      <Icon name="cog" size="sm" class="md:mr-1.5" :stroke-width="2" />
      <span class="hidden md:inline">{{ t("usage.displaySettings") }}</span>
    </button>

    <div
      v-if="open"
      class="absolute right-0 top-full z-50 mt-2 w-[min(22rem,calc(100vw-2rem))] rounded-lg border border-gray-200 bg-white p-3 shadow-xl dark:border-dark-600 dark:bg-dark-800"
    >
      <div class="space-y-4">
        <section class="space-y-2">
          <p class="text-xs font-semibold text-gray-500 dark:text-gray-400">
            {{ t("usage.displaySettingsAppearance") }}
          </p>
          <SegmentedControl
            :label="t('usage.tokenDisplay')"
            :options="tokenDisplayOptions"
            :model-value="preferences.token_display_mode"
            @update:model-value="updatePreference('token_display_mode', $event)"
          />
          <UsageModelDisplayModeToggle
            :model-value="usageModelDisplayMode"
            :disabled="updatingUsageModelDisplayMode || disabled"
            :label-text="t('usage.modelDisplay')"
            @update:modelValue="$emit('update-usage-model-display-mode', $event)"
          />
          <SegmentedControl
            :label="t('usage.tableDensity')"
            :options="densityOptions"
            :model-value="preferences.table_density"
            @update:model-value="updatePreference('table_density', $event)"
          />
          <SegmentedControl
            :label="t('usage.statsCardStyle')"
            :options="statsCardStyleOptions"
            :model-value="preferences.stats_card_style"
            @update:model-value="updatePreference('stats_card_style', $event)"
          />
        </section>

        <section class="space-y-2">
          <p class="text-xs font-semibold text-gray-500 dark:text-gray-400">
            {{ t("usage.displaySettingsColumns") }}
          </p>
          <div class="grid max-h-56 grid-cols-1 gap-1 overflow-y-auto pr-1">
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
        </section>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import Icon from "@/components/icons/Icon.vue";
import UsageModelDisplayModeToggle from "@/components/common/UsageModelDisplayModeToggle.vue";
import type {
  UsageModelDisplayMode,
  UsageViewPagePreferences,
} from "@/types";

type Option = {
  value: string;
  label: string;
};

const props = defineProps<{
  preferences: UsageViewPagePreferences;
  hiddenColumns: Set<string>;
  columns: Array<{ key: string; label: string; sortable?: boolean }>;
  alwaysVisibleColumns: string[];
  usageModelDisplayMode: UsageModelDisplayMode;
  updatingUsageModelDisplayMode: boolean;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  "update-preference": [key: keyof UsageViewPagePreferences, value: string];
  "toggle-column": [key: string];
  "update-usage-model-display-mode": [mode: UsageModelDisplayMode];
}>();

const { t } = useI18n();
const open = ref(false);
const menuRef = ref<HTMLElement | null>(null);

const tokenDisplayOptions = computed<Option[]>(() => [
  { value: "full", label: t("usage.tokenDisplayFull") },
  { value: "compact", label: t("usage.tokenDisplayCompact") },
]);

const densityOptions = computed<Option[]>(() => [
  { value: "comfortable", label: t("usage.tableDensityComfortable") },
  { value: "compact", label: t("usage.tableDensityCompact") },
]);

const statsCardStyleOptions = computed<Option[]>(() => [
  { value: "balanced", label: t("usage.statsCardStyleBalanced") },
  { value: "accent", label: t("usage.statsCardStyleAccent") },
]);

const alwaysVisibleSet = computed(() => new Set(props.alwaysVisibleColumns));
const toggleableColumns = computed(() =>
  props.columns.filter((column) => !alwaysVisibleSet.value.has(column.key)),
);

const updatePreference = (key: keyof UsageViewPagePreferences, value: string) => {
  emit("update-preference", key, value);
};

const SegmentedControl = defineComponent({
  name: "SegmentedControl",
  props: {
    label: { type: String, required: true },
    modelValue: { type: String, required: true },
    options: { type: Array as () => Option[], required: true },
  },
  emits: ["update:modelValue"],
  setup(controlProps, { emit: controlEmit }) {
    return () =>
      h("div", { class: "flex items-center justify-between gap-3" }, [
        h("span", { class: "text-xs font-medium text-gray-500 dark:text-gray-400" }, controlProps.label),
        h(
          "div",
          {
            class:
              "inline-flex shrink-0 rounded-lg border border-gray-200 bg-white p-0.5 shadow-sm dark:border-dark-600 dark:bg-dark-900",
          },
          controlProps.options.map((option) =>
            h(
              "button",
              {
                type: "button",
                class: [
                  "rounded-md px-2.5 py-1 text-xs font-medium transition-colors",
                  controlProps.modelValue === option.value
                    ? "bg-primary-500 text-white"
                    : "text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700",
                ],
                onClick: () => controlEmit("update:modelValue", option.value),
              },
              option.label,
            ),
          ),
        ),
      ]);
  },
});

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
