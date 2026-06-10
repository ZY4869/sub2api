import type {
  AccountGroupDisplayMode,
  AccountStatusDisplayMode,
  AccountTodayStatsCycleMode,
  AccountTodayStatsWindow,
} from "@/types";

export const ACCOUNT_TODAY_STATS_WINDOWS: AccountTodayStatsWindow[] = [
  "today",
  "weekly",
  "monthly",
  "total",
];

export const DEFAULT_ACCOUNT_TODAY_STATS_WINDOWS: AccountTodayStatsWindow[] = [
  ...ACCOUNT_TODAY_STATS_WINDOWS,
];

export const normalizeAccountTodayStatsWindows = (
  values?: readonly string[] | null,
): AccountTodayStatsWindow[] => {
  const seen = new Set<AccountTodayStatsWindow>();
  for (const value of values || []) {
    if (ACCOUNT_TODAY_STATS_WINDOWS.includes(value as AccountTodayStatsWindow)) {
      seen.add(value as AccountTodayStatsWindow);
    }
  }
  return seen.size > 0
    ? ACCOUNT_TODAY_STATS_WINDOWS.filter((value) => seen.has(value))
    : [...DEFAULT_ACCOUNT_TODAY_STATS_WINDOWS];
};

export const normalizeAccountGroupDisplayMode = (
  value?: string | null,
): AccountGroupDisplayMode => (value === "icon" ? "icon" : "full");

export const normalizeAccountStatusDisplayMode = (
  value?: string | null,
): AccountStatusDisplayMode => (value === "simple" ? "simple" : "detailed");

export const normalizeAccountTodayStatsCycleMode = (
  value?: string | null,
): AccountTodayStatsCycleMode => (value === "fixed" ? "fixed" : "calendar");
