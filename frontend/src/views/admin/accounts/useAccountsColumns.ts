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
      key: "id",
      label: t("admin.accounts.columns.accountId"),
      sortable: true,
      class: "w-[88px] min-w-[80px] max-w-[96px] whitespace-nowrap",
    },
    {
      key: "name",
      label: t("admin.accounts.columns.name"),
      sortable: true,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[clamp(184px,18vw,220px)] min-w-[184px] max-w-[220px]"
          : "w-[clamp(192px,20vw,240px)] min-w-[192px] max-w-[240px]",
    },
    {
      key: "platform_type",
      label: t("admin.accounts.columns.platformType"),
      sortable: false,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[clamp(144px,13vw,168px)] min-w-[140px] max-w-[172px]"
          : "w-[clamp(128px,12vw,148px)] min-w-[124px] max-w-[152px]",
    },
    {
      key: "capacity",
      label: t("admin.accounts.columns.capacity"),
      sortable: false,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[clamp(164px,14vw,184px)] min-w-[156px] max-w-[188px]"
          : "w-[clamp(132px,12vw,148px)] min-w-[128px] max-w-[152px]",
    },
    {
      key: "status",
      label: t("admin.accounts.columns.status"),
      sortable: true,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[clamp(176px,14vw,192px)] min-w-[168px] max-w-[196px]"
          : "w-[clamp(184px,16vw,216px)] min-w-[176px] max-w-[224px]",
    },
    {
      key: "schedulable",
      label: t("admin.accounts.columns.schedulable"),
      sortable: true,
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
            ? "w-[88px] min-w-[80px] max-w-[88px]"
            : "w-[clamp(144px,12vw,176px)] min-w-[136px] max-w-[180px]"
          : accountGroupDisplayMode?.value === "icon"
            ? "w-[88px] max-w-[88px]"
          : "w-[clamp(156px,14vw,196px)] min-w-[148px] max-w-[200px]",
    });
  }
  c.push(
    {
      key: "today_stats",
      label: t("admin.accounts.columns.todayStats"),
      sortable: false,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[clamp(156px,13vw,176px)] min-w-[148px] max-w-[180px]"
          : "w-[clamp(156px,13vw,180px)] min-w-[148px] max-w-[184px]",
    },
    {
      key: "usage",
      label: t("admin.accounts.columns.usageWindows"),
      sortable: false,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[clamp(148px,12vw,168px)] min-w-[140px] max-w-[172px]"
          : undefined,
    },
    {
      key: "usage_reset_dates",
      label: t("admin.accounts.columns.usageResetDates"),
      sortable: false,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[clamp(216px,18vw,248px)] min-w-[204px] max-w-[252px]"
          : "w-[clamp(228px,18vw,260px)] min-w-[216px] max-w-[264px]",
    },
    {
      key: "proxy",
      label: t("admin.accounts.columns.proxy"),
      sortable: false,
      class: "w-[clamp(112px,10vw,160px)] min-w-[104px] max-w-[176px]",
    },
    {
      key: "priority",
      label: t("admin.accounts.columns.priority"),
      sortable: true,
      class: "w-[72px] min-w-[64px] max-w-[80px]",
    },
    {
      key: "rate_multiplier",
      label: t("admin.accounts.columns.billingRateMultiplier"),
      sortable: true,
      class: "w-[96px] min-w-[84px] max-w-[108px]",
    },
    {
      key: "last_used_at",
      label: t("admin.accounts.columns.lastUsed"),
      sortable: true,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[112px] min-w-[104px] max-w-[120px] whitespace-nowrap"
          : "w-[120px] min-w-[112px] max-w-[128px] whitespace-nowrap",
    },
    {
      key: "created_at",
      label: t("admin.accounts.columns.createdAt"),
      sortable: true,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[clamp(136px,12vw,152px)] min-w-[132px] max-w-[156px] whitespace-nowrap"
          : "w-[clamp(148px,12vw,164px)] min-w-[144px] max-w-[168px] whitespace-nowrap",
    },
    {
      key: "expires_at",
      label: t("admin.accounts.columns.expiresAt"),
      sortable: true,
      class:
        resolvedAccountVisualPreset.value === "airy"
          ? "w-[clamp(164px,13vw,184px)] min-w-[156px] max-w-[188px] whitespace-nowrap"
          : "w-[clamp(176px,14vw,204px)] min-w-[168px] max-w-[208px] whitespace-nowrap",
    },
    {
      key: "notes",
      label: t("admin.accounts.columns.notes"),
      sortable: false,
      class: "w-[clamp(128px,14vw,220px)] min-w-[112px] max-w-[240px]",
    },
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
