import { computed, ref } from "vue";
import type { Column } from "@/components/common/types";

export const KEYS_COLUMN_VISIBILITY_STORAGE_KEY = "sub2api.user.keys.hiddenColumns";
export const KEYS_ALWAYS_VISIBLE_COLUMNS = ["key", "actions"];

function readHiddenColumns(storageKey: string, alwaysVisible: Set<string>): Set<string> {
  try {
    const raw = localStorage.getItem(storageKey);
    if (!raw) return new Set();
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return new Set();
    return new Set(
      parsed
        .filter((key): key is string => typeof key === "string")
        .filter((key) => !alwaysVisible.has(key)),
    );
  } catch {
    return new Set();
  }
}

function persistHiddenColumns(storageKey: string, hiddenColumns: Set<string>) {
  localStorage.setItem(storageKey, JSON.stringify([...hiddenColumns].sort()));
}

export function useKeysColumnVisibility(
  columns: { value: Column[] },
  storageKey = KEYS_COLUMN_VISIBILITY_STORAGE_KEY,
) {
  const alwaysVisible = new Set(KEYS_ALWAYS_VISIBLE_COLUMNS);
  const hiddenColumns = ref(readHiddenColumns(storageKey, alwaysVisible));

  const visibleColumns = computed(() =>
    columns.value.filter(
      (column) => alwaysVisible.has(column.key) || !hiddenColumns.value.has(column.key),
    ),
  );

  function toggleColumn(key: string) {
    if (alwaysVisible.has(key)) {
      hiddenColumns.value.delete(key);
      persistHiddenColumns(storageKey, hiddenColumns.value);
      return;
    }
    const next = new Set(hiddenColumns.value);
    if (next.has(key)) {
      next.delete(key);
    } else {
      next.add(key);
    }
    hiddenColumns.value = next;
    persistHiddenColumns(storageKey, next);
  }

  return {
    alwaysVisibleColumns: KEYS_ALWAYS_VISIBLE_COLUMNS,
    hiddenColumns,
    visibleColumns,
    toggleColumn,
  };
}

