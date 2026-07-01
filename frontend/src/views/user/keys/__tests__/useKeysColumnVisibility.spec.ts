import { describe, expect, it, beforeEach } from "vitest";
import { ref } from "vue";
import type { Column } from "@/components/common/types";
import {
  KEYS_COLUMN_VISIBILITY_STORAGE_KEY,
  useKeysColumnVisibility,
} from "../useKeysColumnVisibility";

const columns = ref<Column[]>([
  { key: "name", label: "Name" },
  { key: "key", label: "API Key" },
  { key: "usage", label: "Usage" },
  { key: "actions", label: "Actions" },
]);

describe("useKeysColumnVisibility", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("hides toggleable columns and persists the preference", () => {
    const visibility = useKeysColumnVisibility(columns);

    visibility.toggleColumn("usage");

    expect(visibility.visibleColumns.value.map((column) => column.key)).toEqual([
      "name",
      "key",
      "actions",
    ]);
    expect(localStorage.getItem(KEYS_COLUMN_VISIBILITY_STORAGE_KEY)).toBe('["usage"]');
  });

  it("keeps key and actions columns visible even when persisted as hidden", () => {
    localStorage.setItem(
      KEYS_COLUMN_VISIBILITY_STORAGE_KEY,
      JSON.stringify(["key", "actions", "name"]),
    );

    const visibility = useKeysColumnVisibility(columns);

    expect(visibility.visibleColumns.value.map((column) => column.key)).toEqual([
      "key",
      "usage",
      "actions",
    ]);
  });

  it("does not allow toggling always-visible columns off", () => {
    const visibility = useKeysColumnVisibility(columns);

    visibility.toggleColumn("key");
    visibility.toggleColumn("actions");

    expect(visibility.hiddenColumns.value.size).toBe(0);
    expect(visibility.visibleColumns.value.map((column) => column.key)).toEqual([
      "name",
      "key",
      "usage",
      "actions",
    ]);
  });
});

