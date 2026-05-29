import { computed, reactive, ref } from 'vue'
import type { AccountPlatformCountSortOrder } from '@/types'

export function useAccountsColumnPreferences(ctx: any) {
  const {
    getLimitedView,
    refreshTodayStats
  } = ctx

// Column settings
const hiddenColumns = reactive<Set<string>>(new Set());
const DEFAULT_HIDDEN_COLUMNS = [
  "today_stats",
  "proxy",
  "notes",
  "priority",
  "rate_multiplier",
];
const HIDDEN_COLUMNS_KEY = "account-hidden-columns";

// Sorting settings
const ACCOUNT_SORT_STORAGE_KEY = "account-table-sort";
const HIDE_LIMITED_ACCOUNTS_STORAGE_KEY =
  "account-always-hide-limited-accounts";
const PLATFORM_COUNT_SORT_ORDER_STORAGE_KEY =
  "account-platform-count-sort-order";
const loadHideLimitedPreference = () => {
  if (typeof window === "undefined") {
    return true;
  }
  try {
    const saved = localStorage.getItem(HIDE_LIMITED_ACCOUNTS_STORAGE_KEY);
    return saved === "true";
  } catch (error) {
    console.error("Failed to load limited accounts visibility:", error);
    return false;
  }
};

const saveHideLimitedPreference = (value: boolean) => {
  if (typeof window === "undefined") {
    return;
  }
  try {
    localStorage.setItem(HIDE_LIMITED_ACCOUNTS_STORAGE_KEY, String(value));
  } catch (error) {
    console.error("Failed to save limited accounts visibility:", error);
  }
};

const loadPlatformCountSortOrderPreference =
  (): AccountPlatformCountSortOrder => {
    if (typeof window === "undefined") {
      return "count_asc";
    }
    try {
      const saved = localStorage.getItem(PLATFORM_COUNT_SORT_ORDER_STORAGE_KEY);
      return saved === "count_desc" ? "count_desc" : "count_asc";
    } catch (error) {
      console.error("Failed to load platform count sort order:", error);
      return "count_asc";
    }
  };

const savePlatformCountSortOrderPreference = (
  value: AccountPlatformCountSortOrder,
) => {
  if (typeof window === "undefined") {
    return;
  }
  try {
    localStorage.setItem(PLATFORM_COUNT_SORT_ORDER_STORAGE_KEY, value);
  } catch (error) {
    console.error("Failed to save platform count sort order:", error);
  }
};

const loadSavedColumns = () => {
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY);
    if (saved) {
      const parsed = JSON.parse(saved) as string[];
      parsed.forEach((key) => {
        hiddenColumns.add(key);
      });
    } else {
      DEFAULT_HIDDEN_COLUMNS.forEach((key) => {
        hiddenColumns.add(key);
      });
    }
  } catch (e) {
    console.error("Failed to load saved columns:", e);
    DEFAULT_HIDDEN_COLUMNS.forEach((key) => {
      hiddenColumns.add(key);
    });
  }
};

const saveColumnsToStorage = () => {
  try {
    localStorage.setItem(
      HIDDEN_COLUMNS_KEY,
      JSON.stringify([...hiddenColumns]),
    );
  } catch (e) {
    console.error("Failed to save columns:", e);
  }
};

if (typeof window !== "undefined") {
  loadSavedColumns();
}

const hideLimitedAccounts = computed(
  () => String(getLimitedView?.() || "") === "normal_only",
);
const platformCountSortOrder = ref<AccountPlatformCountSortOrder>(
  loadPlatformCountSortOrderPreference(),
);

const handlePlatformCountSortOrderUpdate = (
  value: AccountPlatformCountSortOrder,
) => {
  platformCountSortOrder.value = value;
  savePlatformCountSortOrderPreference(value);
};

const toggleColumn = (key: string) => {
  const wasHidden = hiddenColumns.has(key);
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key);
  } else {
    hiddenColumns.add(key);
  }
  saveColumnsToStorage();
  if ((key === "today_stats" || key === "usage") && wasHidden) {
    refreshTodayStats?.().catch((error: unknown) => {
      console.error(
        "Failed to load account today stats after showing column:",
        error,
      );
    });
  }
};


  return {
    hiddenColumns,
    DEFAULT_HIDDEN_COLUMNS,
    HIDDEN_COLUMNS_KEY,
    ACCOUNT_SORT_STORAGE_KEY,
    HIDE_LIMITED_ACCOUNTS_STORAGE_KEY,
    PLATFORM_COUNT_SORT_ORDER_STORAGE_KEY,
    loadHideLimitedPreference,
    saveHideLimitedPreference,
    loadPlatformCountSortOrderPreference,
    savePlatformCountSortOrderPreference,
    loadSavedColumns,
    saveColumnsToStorage,
    hideLimitedAccounts,
    platformCountSortOrder,
    handlePlatformCountSortOrderUpdate,
    toggleColumn
  }
}
