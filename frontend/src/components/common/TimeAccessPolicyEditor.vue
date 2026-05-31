<template>
  <section class="space-y-3">
    <div class="flex flex-wrap gap-2">
      <button
        v-for="preset in presets"
        :key="preset"
        type="button"
        @click="applyPreset(preset)"
        :class="[
          'rounded-lg px-3 py-1.5 text-sm transition-colors',
          activePreset === preset
            ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
            : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600',
        ]"
      >
        {{ t(`common.timeAccess.presets.${preset}`) }}
      </button>
    </div>

    <div class="grid gap-3 md:grid-cols-4">
      <label class="space-y-1">
        <span class="input-label">{{ t("common.timeAccess.timezone") }}</span>
        <input
          :value="policy.timezone"
          type="text"
          class="input"
          placeholder="Asia/Singapore"
          data-testid="time-access-timezone"
          @input="updateTimezone(($event.target as HTMLInputElement).value)"
        />
      </label>
      <label class="space-y-1">
        <span class="input-label">{{ t("common.timeAccess.dailyAllowedMinutes") }}</span>
        <input
          :value="policy.daily_allowed_minutes"
          type="number"
          min="0"
          max="1440"
          step="15"
          class="input"
          data-testid="time-access-daily-allowed-minutes"
          @input="updateDailyAllowedMinutes(($event.target as HTMLInputElement).value)"
        />
      </label>
      <label class="space-y-1">
        <span class="input-label">{{ t("common.timeAccess.notBefore") }}</span>
        <input
          :value="formatDateTimeLocal(policy.not_before)"
          type="datetime-local"
          class="input"
          data-testid="time-access-not-before"
          @input="updateBoundary('not_before', ($event.target as HTMLInputElement).value)"
        />
      </label>
      <label class="space-y-1">
        <span class="input-label">{{ t("common.timeAccess.notAfter") }}</span>
        <input
          :value="formatDateTimeLocal(policy.not_after)"
          type="datetime-local"
          class="input"
          data-testid="time-access-not-after"
          @input="updateBoundary('not_after', ($event.target as HTMLInputElement).value)"
        />
      </label>
    </div>

    <TimeAccessWindowList
      :windows="windows"
      @add="addWindow"
      @remove="removeWindow"
      @toggle-day="toggleDay"
      @update-window="updateWindow"
    />

    <p v-if="hint" class="input-hint">{{ hint }}</p>
    <p v-else class="input-hint">{{ t("common.timeAccess.policyHint") }}</p>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import TimeAccessWindowList from "@/components/common/TimeAccessWindowList.vue";
import type { TimeAccessPolicy } from "@/types/api-key-groups";
import { formatDateTimeLocal } from "@/utils/timeAccessPolicy";
import { useTimeAccessPolicyEditor } from "./useTimeAccessPolicyEditor";

const props = defineProps<{
  modelValue: TimeAccessPolicy | null | undefined;
  hint?: string;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: TimeAccessPolicy];
}>();

const { t } = useI18n();
const {
  activePreset,
  presets,
  policy,
  windows,
  addWindow,
  applyPreset,
  removeWindow,
  toggleDay,
  updateBoundary,
  updateDailyAllowedMinutes,
  updateTimezone,
  updateWindow,
} = useTimeAccessPolicyEditor({
  modelValue: () => props.modelValue,
  onUpdate: (value) => emit("update:modelValue", value),
});
</script>
