import { computed, ref, watch } from "vue";
import type { TimeAccessPolicy, TimeAccessWindow } from "@/types/api-key-groups";
import {
  buildPresetTimeAccessPolicy,
  dateTimeLocalToISOString,
  ensureEnabledTimeAccessPolicy,
  timeAccessAllDays,
  type TimeAccessPreset,
} from "@/utils/timeAccessPolicy";

export const timeAccessEditorPresets: Exclude<TimeAccessPreset, "custom">[] = [
  "daytime",
  "deep_night",
  "eight_hours",
  "twelve_hours",
  "business_days_daytime",
];

interface TimeAccessPolicyEditorOptions {
  modelValue: () => TimeAccessPolicy | null | undefined;
  onUpdate: (value: TimeAccessPolicy) => void;
}

export function useTimeAccessPolicyEditor(options: TimeAccessPolicyEditorOptions) {
  const activePreset = ref<TimeAccessPreset>("daytime");
  const policy = computed(() => ensureEnabledTimeAccessPolicy(options.modelValue()));
  const windows = computed(() => policy.value.weekly_windows?.length
    ? policy.value.weekly_windows
    : [defaultTimeAccessWindow()]);

  watch(
    options.modelValue,
    (value) => {
      activePreset.value = resolvePreset(value);
    },
    { immediate: true },
  );

  function updatePolicy(next: TimeAccessPolicy) {
    options.onUpdate(ensureEnabledTimeAccessPolicy(next));
  }

  function applyPreset(preset: TimeAccessPreset) {
    activePreset.value = preset;
    updatePolicy(buildPresetTimeAccessPolicy(preset));
  }

  function updateTimezone(value: string) {
    activePreset.value = "custom";
    updatePolicy({
      ...policy.value,
      timezone: value.trim() || "Asia/Singapore",
    });
  }

  function updateDailyAllowedMinutes(value: string) {
    activePreset.value = "custom";
    const parsed = value === "" ? null : Number(value);
    updatePolicy({
      ...policy.value,
      daily_allowed_minutes: Number.isFinite(parsed) ? parsed : null,
    });
  }

  function updateWindow(index: number, patch: Partial<TimeAccessWindow>) {
    activePreset.value = "custom";
    const nextWindows = cloneWindows(windows.value);
    nextWindows[index] = {
      ...(nextWindows[index] || defaultTimeAccessWindow()),
      ...patch,
    };
    if (!nextWindows[index].days?.length) {
      nextWindows[index].days = [...timeAccessAllDays];
    }
    updatePolicy({
      ...policy.value,
      weekly_windows: nextWindows,
    });
  }

  function addWindow() {
    activePreset.value = "custom";
    updatePolicy({
      ...policy.value,
      weekly_windows: [...cloneWindows(windows.value), defaultTimeAccessWindow()],
    });
  }

  function removeWindow(index: number) {
    if (windows.value.length <= 1) return;
    activePreset.value = "custom";
    updatePolicy({
      ...policy.value,
      weekly_windows: cloneWindows(windows.value)
        .filter((_, windowIndex) => windowIndex !== index),
    });
  }

  function toggleDay(index: number, day: number) {
    const current = windows.value[index] || defaultTimeAccessWindow();
    const days = new Set(current.days);
    if (days.has(day)) {
      if (days.size <= 1) return;
      days.delete(day);
    } else {
      days.add(day);
    }
    updateWindow(index, { days: Array.from(days).sort((a, b) => a - b) });
  }

  function updateBoundary(field: "not_before" | "not_after", value: string) {
    activePreset.value = "custom";
    updatePolicy({
      ...policy.value,
      [field]: dateTimeLocalToISOString(value) || null,
    });
  }

  return {
    activePreset,
    presets: timeAccessEditorPresets,
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
  };
}

function defaultTimeAccessWindow(): TimeAccessWindow {
  return { days: [...timeAccessAllDays], start: "08:00", end: "20:00" };
}

function cloneWindows(windows: TimeAccessWindow[]) {
  return windows.map((window) => ({ ...window, days: [...window.days] }));
}

function resolvePreset(value?: TimeAccessPolicy | null): TimeAccessPreset {
  const normalized = ensureEnabledTimeAccessPolicy(value);
  for (const preset of timeAccessEditorPresets) {
    const presetPolicy = buildPresetTimeAccessPolicy(preset);
    if (
      JSON.stringify(normalized.weekly_windows) === JSON.stringify(presetPolicy.weekly_windows) &&
      normalized.daily_allowed_minutes === presetPolicy.daily_allowed_minutes &&
      !normalized.not_before &&
      !normalized.not_after
    ) {
      return preset;
    }
  }
  return "custom";
}
