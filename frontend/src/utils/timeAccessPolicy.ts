import type { TimeAccessPolicy, TimeAccessWindow } from "@/types/api-key-groups";

export type TimeAccessPreset =
  | "daytime"
  | "deep_night"
  | "eight_hours"
  | "twelve_hours"
  | "business_days_daytime"
  | "custom";

export const timeAccessAllDays = [0, 1, 2, 3, 4, 5, 6];
export const timeAccessWeekdays = [1, 2, 3, 4, 5];

const presetWindows: Record<Exclude<TimeAccessPreset, "custom">, TimeAccessWindow[]> = {
  daytime: [{ days: timeAccessAllDays, start: "08:00", end: "20:00" }],
  deep_night: [{ days: timeAccessAllDays, start: "00:00", end: "06:00" }],
  eight_hours: [{ days: timeAccessAllDays, start: "09:00", end: "17:00" }],
  twelve_hours: [{ days: timeAccessAllDays, start: "08:00", end: "20:00" }],
  business_days_daytime: [{ days: timeAccessWeekdays, start: "08:00", end: "20:00" }],
};

export function createDefaultTimeAccessPolicy(): TimeAccessPolicy {
  return {
    enabled: false,
    timezone: "Asia/Singapore",
    weekly_windows: [],
    daily_allowed_minutes: null,
  };
}

export function buildPresetTimeAccessPolicy(preset: TimeAccessPreset): TimeAccessPolicy {
  if (preset === "custom") {
    return {
      enabled: true,
      timezone: "Asia/Singapore",
      weekly_windows: [{ days: [...timeAccessAllDays], start: "08:00", end: "20:00" }],
      daily_allowed_minutes: null,
    };
  }
  return {
    enabled: true,
    timezone: "Asia/Singapore",
    weekly_windows: presetWindows[preset].map((window) => ({
      days: [...window.days],
      start: window.start,
      end: window.end,
    })),
    daily_allowed_minutes:
      preset === "eight_hours" ? 480 : preset === "deep_night" ? 360 : 720,
  };
}

export function normalizeTimeAccessPolicy(policy?: TimeAccessPolicy | null): TimeAccessPolicy {
  if (!policy) return createDefaultTimeAccessPolicy();
  return {
    enabled: !!policy.enabled,
    timezone: policy.timezone || "Asia/Singapore",
    not_before: policy.not_before ?? null,
    not_after: policy.not_after ?? null,
    weekly_windows: (policy.weekly_windows || []).map((window) => ({
      days: Array.isArray(window.days) && window.days.length > 0
        ? [...window.days].filter((day) => day >= 0 && day <= 6).sort()
        : [...timeAccessAllDays],
      start: window.start || "08:00",
      end: window.end || "20:00",
    })),
    daily_allowed_minutes: policy.daily_allowed_minutes ?? null,
  };
}

export function ensureEnabledTimeAccessPolicy(policy?: TimeAccessPolicy | null): TimeAccessPolicy {
  const normalized = normalizeTimeAccessPolicy(policy);
  return {
    ...normalized,
    enabled: true,
    timezone: normalized.timezone || "Asia/Singapore",
    weekly_windows: normalized.weekly_windows?.length
      ? normalized.weekly_windows
      : [{ days: [...timeAccessAllDays], start: "08:00", end: "20:00" }],
  };
}

export function policyToPayload(policy?: TimeAccessPolicy | null): TimeAccessPolicy | undefined {
  if (!policy?.enabled) return undefined;
  const normalized = ensureEnabledTimeAccessPolicy(policy);
  return {
    ...normalized,
    weekly_windows: normalized.weekly_windows?.filter((window) => window.start && window.end) || [],
  };
}

export function formatDateTimeLocal(value?: string | null): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  const pad = (part: number) => String(part).padStart(2, "0");
  return [
    date.getFullYear(),
    "-",
    pad(date.getMonth() + 1),
    "-",
    pad(date.getDate()),
    "T",
    pad(date.getHours()),
    ":",
    pad(date.getMinutes()),
  ].join("");
}

export function dateTimeLocalToISOString(value?: string | null): string | undefined {
  const trimmed = String(value || "").trim();
  if (!trimmed) return undefined;
  const date = new Date(trimmed);
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toISOString();
}
