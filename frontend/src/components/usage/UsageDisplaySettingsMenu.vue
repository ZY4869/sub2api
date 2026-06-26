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
          <ToggleControl
            :label="t('usage.showMillionContextLines')"
            :model-value="preferences.show_million_context_lines"
            @update:model-value="updatePreference('show_million_context_lines', $event)"
          />
          <SegmentedControl
            :label="t('usage.userAgentDisplay')"
            :options="userAgentDisplayOptions"
            :model-value="preferences.user_agent_display_mode"
            @update:model-value="updatePreference('user_agent_display_mode', $event)"
          />
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, onUnmounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import Icon from "@/components/icons/Icon.vue";
import UsageModelDisplayModeToggle from "@/components/common/UsageModelDisplayModeToggle.vue";
import type { UsageModelDisplayMode, UsageViewPagePreferences } from "@/types";

type Option = { value: string; label: string };

defineProps<{
  preferences: UsageViewPagePreferences;
  usageModelDisplayMode: UsageModelDisplayMode;
  updatingUsageModelDisplayMode: boolean;
  disabled?: boolean;
}>();

const emit = defineEmits<{
  "update-preference": [key: keyof UsageViewPagePreferences, value: string | boolean];
  "update-usage-model-display-mode": [mode: UsageModelDisplayMode];
}>();

const { t } = useI18n();
const open = ref(false);
const menuRef = ref<HTMLElement | null>(null);

const tokenDisplayOptions = computed<Option[]>(() => [
  { value: "natural", label: t("usage.tokenDisplayNatural") },
  { value: "k", label: t("usage.tokenDisplayK") },
  { value: "m", label: t("usage.tokenDisplayM") },
]);

const densityOptions = computed<Option[]>(() => [
  { value: "comfortable", label: t("usage.tableDensityComfortable") },
  { value: "compact", label: t("usage.tableDensityCompact") },
]);

const statsCardStyleOptions = computed<Option[]>(() => [
  { value: "balanced", label: t("usage.statsCardStyleBalanced") },
  { value: "accent", label: t("usage.statsCardStyleAccent") },
]);

const userAgentDisplayOptions = computed<Option[]>(() => [
  { value: "compact", label: t("usage.userAgentDisplayCompact") },
  { value: "full", label: t("usage.userAgentDisplayFull") },
]);

const updatePreference = (key: keyof UsageViewPagePreferences, value: string | boolean) => {
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

const ToggleControl = defineComponent({
  name: "ToggleControl",
  props: {
    label: { type: String, required: true },
    modelValue: { type: Boolean, required: true },
  },
  emits: ["update:modelValue"],
  setup(controlProps, { emit: controlEmit }) {
    return () =>
      h("div", { class: "flex items-center justify-between gap-3" }, [
        h("span", { class: "text-xs font-medium text-gray-500 dark:text-gray-400" }, controlProps.label),
        h(
          "button",
          {
            type: "button",
            class: [
              "inline-flex h-6 w-11 shrink-0 items-center rounded-full p-0.5 transition-colors",
              controlProps.modelValue ? "bg-primary-500" : "bg-gray-200 dark:bg-dark-600",
            ],
            "aria-pressed": controlProps.modelValue,
            onClick: () => controlEmit("update:modelValue", !controlProps.modelValue),
          },
          [
            h("span", {
              class: [
                "h-5 w-5 rounded-full bg-white shadow-sm transition-transform",
                controlProps.modelValue ? "translate-x-5" : "translate-x-0",
              ],
            }),
          ],
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

onMounted(() => document.addEventListener("click", handleClickOutside));
onUnmounted(() => document.removeEventListener("click", handleClickOutside));
</script>
