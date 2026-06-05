import { computed } from 'vue'

export function useAccountsColumns(ctx: any) {
  const {
    authStore,
    accountGroupDisplayMode,
    hiddenColumns,
    resolvedAccountVisualPreset,
    t
  } = ctx

// All available columns
const allColumns = computed(() => {
  const c = [
    { key: "select", label: "", sortable: false, class: "w-[36px] min-w-[36px] max-w-[36px]" },
    {
      key: "name",
      label: t("admin.accounts.columns.name"),
      sortable: true,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[236px] min-w-[196px] max-w-[236px]"
          : "w-[240px] min-w-[192px] max-w-[240px]",
    },
    {
      key: "platform_type",
      label: t("admin.accounts.columns.platformType"),
      sortable: false,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[160px] min-w-[140px] max-w-[160px]"
          : "w-[140px] max-w-[140px]",
    },
    {
      key: "capacity",
      label: t("admin.accounts.columns.capacity"),
      sortable: false,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[184px] min-w-[172px] max-w-[184px]"
          : "w-[148px] max-w-[148px]",
    },
    {
      key: "status",
      label: t("admin.accounts.columns.status"),
      sortable: true,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[244px] min-w-[220px] max-w-[244px]"
          : "w-[240px] max-w-[240px]",
    },
    {
      key: "schedulable",
      label: t("admin.accounts.columns.schedulable"),
      sortable: true,
    },
    {
      key: "today_stats",
      label: t("admin.accounts.columns.todayStats"),
      sortable: false,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[152px] min-w-[144px] max-w-[160px]"
          : "w-[212px] max-w-[212px]",
    },
  ];
  if (!authStore.isSimpleMode) {
    c.push({
      key: "groups",
      label: t("admin.accounts.columns.groups"),
      sortable: false,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? accountGroupDisplayMode?.value === "icon"
            ? "w-[132px] min-w-[120px] max-w-[132px]"
            : "w-[196px] min-w-[172px] max-w-[196px]"
          : accountGroupDisplayMode?.value === "icon"
            ? "w-[132px] max-w-[132px]"
            : "w-[196px] max-w-[196px]",
    });
  }
  c.push(
    {
      key: "usage",
      label: t("admin.accounts.columns.usageWindows"),
      sortable: false,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[184px] min-w-[168px] max-w-[184px]"
          : undefined,
    },
    {
      key: "usage_reset_dates",
      label: t("admin.accounts.columns.usageResetDates"),
      sortable: false,
      class: "w-[260px] min-w-[240px] max-w-[260px]",
    },
    { key: "proxy", label: t("admin.accounts.columns.proxy"), sortable: false },
    {
      key: "priority",
      label: t("admin.accounts.columns.priority"),
      sortable: true,
    },
    {
      key: "rate_multiplier",
      label: t("admin.accounts.columns.billingRateMultiplier"),
      sortable: true,
    },
    {
      key: "last_used_at",
      label: t("admin.accounts.columns.lastUsed"),
      sortable: true,
      class: "w-[120px] min-w-[120px] max-w-[128px] whitespace-nowrap",
    },
    {
      key: "created_at",
      label: t("admin.accounts.columns.createdAt"),
      sortable: true,
      class: "w-[156px] min-w-[156px] max-w-[164px] whitespace-nowrap",
    },
    {
      key: "expires_at",
      label: t("admin.accounts.columns.expiresAt"),
      sortable: true,
      class: "w-[192px] min-w-[192px] max-w-[204px] whitespace-nowrap",
    },
    { key: "notes", label: t("admin.accounts.columns.notes"), sortable: false },
    {
      key: "actions",
      label: t("admin.accounts.columns.actions"),
      sortable: false,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[156px] min-w-[156px] max-w-[156px]"
          : undefined,
    },
  );
  return c;
});

// Columns that can be toggled (exclude select, name, and actions)
const toggleableColumns = computed(() =>
  allColumns.value
    .filter(
      (col) =>
        col.key !== "select" && col.key !== "name" && col.key !== "actions",
    )
    .map((col) => ({
      key: col.key,
      label: col.label,
      visible: !hiddenColumns.has(col.key),
    })),
);

// Filtered columns based on visibility
const cols = computed(() =>
  allColumns.value.filter(
    (col) =>
      resolvedAccountVisualPreset.value !== "airy" ||
      col.key !== "schedulable",
  ).filter(
    (col) =>
      col.key === "select" ||
      col.key === "name" ||
      col.key === "actions" ||
      !hiddenColumns.has(col.key),
  ),
);

  return {
    allColumns,
    toggleableColumns,
    cols
  }
}
