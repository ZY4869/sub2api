<template>
  <div class="space-y-3">
    <div class="flex items-center justify-between gap-3">
      <div>
        <div class="input-label">{{ t("common.timeAccess.weeklyWindows") }}</div>
        <p class="input-hint">{{ t("common.timeAccess.windowHint") }}</p>
      </div>
      <button
        type="button"
        class="btn btn-secondary inline-flex items-center gap-1.5 px-3 py-2 text-sm"
        data-testid="time-access-add-window"
        :aria-label="t('common.timeAccess.addWindow')"
        @click="emit('add')"
      >
        <Icon name="plus" size="sm" />
        <span>{{ t("common.timeAccess.addWindow") }}</span>
      </button>
    </div>

    <div
      v-for="(window, index) in windows"
      :key="index"
      class="space-y-3 rounded-lg border border-gray-200 bg-gray-50/60 p-3 dark:border-dark-700 dark:bg-dark-900/30"
      data-testid="time-access-window"
    >
      <div class="flex items-center justify-between gap-3">
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t("common.timeAccess.windowTitle", { index: index + 1 }) }}
        </div>
        <button
          type="button"
          class="inline-flex items-center gap-1.5 rounded-lg px-2 py-1 text-sm text-red-600 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
          :class="{ 'cursor-not-allowed opacity-50': windows.length <= 1 }"
          :disabled="windows.length <= 1"
          :data-testid="`time-access-remove-window-${index}`"
          :aria-label="t('common.timeAccess.removeWindow')"
          @click="emit('remove', index)"
        >
          <Icon name="trash" size="sm" />
          <span>{{ t("common.remove") }}</span>
        </button>
      </div>

      <div>
        <div class="mb-2 text-xs font-medium text-gray-500 dark:text-gray-400">
          {{ t("common.timeAccess.days") }}
        </div>
        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="day in dayOptions"
            :key="day.value"
            type="button"
            class="min-w-10 rounded-lg px-2.5 py-1.5 text-xs font-medium transition-colors"
            :class="isDaySelected(window, day.value)
              ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
              : 'bg-white text-gray-500 ring-1 ring-gray-200 hover:bg-gray-100 dark:bg-dark-800 dark:text-gray-400 dark:ring-dark-700 dark:hover:bg-dark-700'"
            :aria-pressed="isDaySelected(window, day.value)"
            :data-testid="`time-access-day-${index}-${day.value}`"
            @click="emit('toggle-day', index, day.value)"
          >
            {{ t(`common.timeAccess.daysShort.${day.key}`) }}
          </button>
        </div>
      </div>

      <div class="grid gap-3 md:grid-cols-2">
        <label class="space-y-1">
          <span class="input-label">{{ t("common.timeAccess.start") }}</span>
          <input
            :value="window.start"
            type="time"
            class="input"
            :data-testid="`time-access-start-${index}`"
            @input="emit('update-window', index, { start: ($event.target as HTMLInputElement).value })"
          />
        </label>
        <label class="space-y-1">
          <span class="input-label">{{ t("common.timeAccess.end") }}</span>
          <input
            :value="window.end"
            type="time"
            class="input"
            :data-testid="`time-access-end-${index}`"
            @input="emit('update-window', index, { end: ($event.target as HTMLInputElement).value })"
          />
        </label>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import Icon from "@/components/icons/Icon.vue";
import type { TimeAccessWindow } from "@/types/api-key-groups";

defineProps<{
  windows: TimeAccessWindow[];
}>();

const emit = defineEmits<{
  add: [];
  remove: [index: number];
  "toggle-day": [index: number, day: number];
  "update-window": [index: number, patch: Partial<TimeAccessWindow>];
}>();

const { t } = useI18n();
const dayOptions = [
  { value: 0, key: "sun" },
  { value: 1, key: "mon" },
  { value: 2, key: "tue" },
  { value: 3, key: "wed" },
  { value: 4, key: "thu" },
  { value: 5, key: "fri" },
  { value: 6, key: "sat" },
] as const;

function isDaySelected(window: TimeAccessWindow, day: number) {
  return window.days.includes(day);
}
</script>
