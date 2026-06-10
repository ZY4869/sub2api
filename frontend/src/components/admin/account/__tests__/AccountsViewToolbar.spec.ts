import { mount } from "@vue/test-utils";
import { nextTick } from "vue";
import { describe, expect, it, vi } from "vitest";
import AccountsViewToolbar from "../AccountsViewToolbar.vue";

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) =>
        key === "admin.accounts.autoRefreshCountdown" && params?.seconds
          ? `countdown-${params.seconds}`
          : key,
    }),
  };
});

function mountToolbar(overrides: Record<string, unknown> = {}) {
  return mount(AccountsViewToolbar, {
    props: {
      loading: false,
      usageRefreshing: false,
      searchQuery: "",
      filters: { platform: "", type: "", status: "", group: "", search: "" },
      groups: [{ id: 1, name: "Default" }],
      hasPendingListSync: true,
      selectedCount: 2,
      autoRefreshEnabled: true,
      autoRefreshCountdown: 15,
      autoRefreshIntervals: [5, 10, 15, 30],
      autoRefreshIntervalSeconds: 10,
      viewMode: "table",
      groupViewEnabled: false,
      platformCountSortOrder: "count_asc",
      actualUsageRefreshSummary: {
        total: 6,
        live: 4,
        fallback: 2,
      },
      accountVisualPresetOverride: 'inherit',
      visualStyle: 'classic',
      accountVisualStyleUpdating: false,
      accountTodayStatsWindows: ['today', 'weekly', 'monthly', 'total'],
      accountTodayStatsCycleMode: 'calendar',
      accountGroupDisplayMode: 'full',
      accountDisplayPreferencesUpdating: false,
      filteredBulkEditTotal: 6,
      filteredBulkEditExcludeGrouped: false,
      filteredBulkEditExcludeGroupedDisabled: false,
      daily5HTriggerEnabled: true,
      toggleableColumns: [
        { key: "proxy", label: "Proxy", visible: true },
        { key: "notes", label: "Notes", visible: false },
      ],
      ...overrides,
    },
    global: {
      stubs: {
        Icon: true,
        AccountViewModeToggle: {
          props: ["modelValue"],
          emits: ["update:modelValue"],
          template:
            "<button class=\"view-mode-toggle\" @click=\"$emit('update:modelValue', 'card')\" />",
        },
        AccountTableFilters: {
          emits: ["update:filters", "update:searchQuery", "change"],
          template: `
            <div>
              <button class="filters-update" @click="$emit('update:filters', { platform: 'openai' })" />
              <button class="filters-search" @click="$emit('update:searchQuery', 'claude')" />
              <button class="filters-change" @click="$emit('change')" />
            </div>
          `,
        },
        AccountTableActions: {
          props: ["loading"],
          emits: ["refresh", "sync", "create"],
          template: `
            <div>
              <button class="refresh" @click="$emit('refresh')" />
              <slot name="after" />
              <slot name="beforeCreate" />
              <button class="sync" @click="$emit('sync')" />
              <button class="create" @click="$emit('create')" />
            </div>
          `,
        },
        Teleport: true,
      },
    },
  });
}

describe("AccountsViewToolbar", () => {
  it("forwards filter, search and toolbar actions", async () => {
    const wrapper = mountToolbar();
    const refreshUsageButton = wrapper.get('[data-actual-usage-button="true"]');

    await wrapper.get(".view-mode-toggle").trigger("click");
    await wrapper.get(".filters-update").trigger("click");
    await wrapper.get(".filters-search").trigger("click");
    await wrapper.get(".filters-change").trigger("click");
    await wrapper.get(".refresh").trigger("click");
    await refreshUsageButton.trigger("click");
    await wrapper.get(".sync").trigger("click");
    await wrapper.get(".create").trigger("click");

    expect(wrapper.text()).not.toContain("admin.accounts.viewArchived");
    expect(wrapper.text()).not.toContain("admin.accounts.batchCreate");
    expect(refreshUsageButton.attributes("title")).toBe(
      "admin.accounts.refreshActualUsageTitle",
    );
    expect(wrapper.find('[data-actual-usage-help="true"]').exists()).toBe(
      false,
    );
    expect(wrapper.emitted("update:view-mode")).toEqual([["card"]]);
    expect(wrapper.emitted("update:filters")).toEqual([
      [{ platform: "openai" }],
    ]);
    expect(wrapper.emitted("update:searchQuery")).toEqual([["claude"]]);
    expect(wrapper.emitted("change")).toEqual([[]]);
    expect(wrapper.emitted("refresh")).toEqual([[]]);
    expect(wrapper.emitted("refresh-usage")).toEqual([[]]);
    expect(wrapper.emitted("sync")).toEqual([[]]);
    expect(wrapper.emitted("create")).toEqual([[]]);
  });

  it("does not render the archive current group action", () => {
    const wrapper = mountToolbar({
      filters: { platform: "", type: "", status: "", group: "1", search: "" },
      groups: [{ id: 1, name: "Default", platform: "openai" }],
    });

    expect(wrapper.text()).not.toContain(
      "admin.accounts.bulkActions.archiveCurrentGroup",
    );
  });

  it("emits dropdown and pending sync actions", async () => {
    const wrapper = mountToolbar();

    await wrapper
      .get('button[title="admin.accounts.autoRefresh"]')
      .trigger("click");
    expect(wrapper.text()).toContain("admin.accounts.enableAutoRefresh");
    await wrapper
      .findAll("button")
      .find((button) =>
        button.text().includes("admin.accounts.enableAutoRefresh"),
      )
      ?.trigger("click");
    await wrapper
      .findAll("button")
      .find((button) =>
        button.text().includes("admin.accounts.refreshInterval5s"),
      )
      ?.trigger("click");

    await wrapper.get('[data-more-actions-button="true"]').trigger("click");
    await nextTick();
    expect(wrapper.text()).toContain("admin.users.columnSettings");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("admin.users.columnSettings"))
      ?.trigger("click");
    await nextTick();
    expect(wrapper.text()).toContain("Proxy");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Proxy"))
      ?.trigger("click");
    await wrapper.get('[data-daily5h-toggle="true"]').trigger("click");
    await wrapper.get('[data-daily5h-settings="true"]').trigger("click");
    await wrapper.get('[data-account-import-button="true"]').trigger("click");
    await wrapper.get('[data-account-export-button="true"]').trigger("click");
    await wrapper.get('[data-more-actions-button="true"]').trigger("click");
    await nextTick();
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("admin.errorPassthrough.title"))
      ?.trigger("click");
    await wrapper.get('[data-more-actions-button="true"]').trigger("click");
    await nextTick();
    await wrapper
      .findAll("button")
      .find((button) =>
        button.text().includes("admin.tlsFingerprintProfiles.title"),
      )
      ?.trigger("click");
    await wrapper
      .findAll("button")
      .find((button) =>
        button.text().includes("admin.accounts.groupView.enable"),
      )
      ?.trigger("click");
    await wrapper
      .findAll("button")
      .find((button) =>
        button.text().includes("admin.accounts.listPendingSyncAction"),
      )
      ?.trigger("click");

    expect(wrapper.emitted("set-auto-refresh-enabled")).toEqual([[false]]);
    expect(wrapper.emitted("set-auto-refresh-interval")).toEqual([[5]]);
    expect(wrapper.emitted("toggle-column")).toEqual([["proxy"]]);
    expect(wrapper.emitted("toggle-daily-5h-trigger")).toEqual([[]]);
    expect(wrapper.emitted("open-daily-5h-settings")).toEqual([[]]);
    expect(wrapper.emitted("import-data")).toEqual([[]]);
    expect(wrapper.emitted("export-data")).toEqual([[]]);
    expect(wrapper.emitted("show-error-passthrough")).toEqual([[]]);
    expect(wrapper.emitted("show-tls-fingerprint-profiles")).toEqual([[]]);
    expect(wrapper.emitted("toggle-group-view")).toEqual([[]]);
    expect(wrapper.emitted("sync-pending-list")).toEqual([[]]);
  });

  it("renders the more actions and column settings panels as floating overlays", async () => {
    const wrapper = mountToolbar();

    await wrapper.get('[data-more-actions-button="true"]').trigger("click");
    await nextTick();

    expect(wrapper.get('[data-account-import-button="true"]').text()).toContain(
      "admin.accounts.dataImport",
    );
    expect(wrapper.get('[data-account-export-button="true"]').text()).toContain(
      "admin.accounts.dataExportSelected",
    );
    expect(wrapper.text()).toContain("admin.users.columnSettings");

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("admin.users.columnSettings"))
      ?.trigger("click");
    await nextTick();

    expect(wrapper.text()).toContain("Proxy");
    expect(wrapper.text()).toContain("Notes");
  });

  it("renders and emits the account realtime countdown toggle from More", async () => {
    const wrapper = mountToolbar({
      accountRealtimeCountdownEnabled: true,
    });

    await wrapper.get('[data-more-actions-button="true"]').trigger("click");
    await nextTick();

    const toggleButton = wrapper.get('[data-account-realtime-toggle="true"]');
    expect(toggleButton.text()).toContain(
      "admin.accounts.accountRealtimeCountdown",
    );

    await toggleButton.trigger("click");
    await nextTick();

    expect(wrapper.emitted("toggle-account-realtime-countdown")).toEqual([[]]);
    expect(wrapper.text()).not.toContain(
      "admin.accounts.accountRealtimeCountdown",
    );
  });

  it("renders and emits the account visual style toggle", async () => {
    const wrapper = mountToolbar({
      accountVisualPresetOverride: "inherit",
      visualStyle: "classic",
    });

    const visualStyleToggle = wrapper.get('[data-account-visual-style-toggle="true"]');
    expect(visualStyleToggle.text()).toContain(
      "admin.accounts.accountVisualStyleInherit",
    );
    expect(visualStyleToggle.text()).toContain(
      "admin.accounts.accountVisualStyleClassic",
    );
    expect(visualStyleToggle.text()).toContain(
      "admin.accounts.accountVisualStyleAiry",
    );

    await wrapper
      .findAll("button")
      .find((button) =>
        button.text().includes("admin.accounts.accountVisualStyleAiry"),
      )
      ?.trigger("click");

    expect(wrapper.emitted("set-account-visual-preset-override")).toEqual([["airy"]]);
  });

  it("renders and emits display optimization preferences", async () => {
    const wrapper = mountToolbar();

    await wrapper.get('[data-account-display-optimization-button="true"]').trigger("click");
    await nextTick();

    expect(wrapper.text()).toContain("admin.accounts.displayOptimization.todayStats");
    expect(wrapper.text()).toContain("admin.accounts.displayOptimization.todayStatsCycleMode");
    expect(wrapper.text()).toContain("admin.accounts.displayOptimization.groupDisplay");
    expect(wrapper.text()).toContain("admin.accounts.displayOptimization.statusDisplay");

    const weeklyCheckbox = wrapper
      .findAll('input[type="checkbox"]')
      .find((input) =>
        input.element.parentElement?.textContent?.includes(
          "admin.accounts.displayOptimization.windows.weekly",
        ),
      );
    await weeklyCheckbox?.trigger("change");
    const monthlyCheckbox = wrapper
      .findAll('input[type="checkbox"]')
      .find((input) =>
        input.element.parentElement?.textContent?.includes(
          "admin.accounts.displayOptimization.windows.monthly",
        ),
      );
    await monthlyCheckbox?.trigger("change");

    await wrapper
      .findAll("button")
      .find((button) =>
        button.text().includes("admin.accounts.displayOptimization.cycleModes.fixed"),
      )
      ?.trigger("click");
    await wrapper
      .findAll("button")
      .find((button) =>
        button.text().includes("admin.accounts.displayOptimization.groupModes.icon"),
      )
      ?.trigger("click");
    await wrapper
      .findAll("button")
      .find((button) =>
        button.text().includes("admin.accounts.displayOptimization.statusModes.simple"),
      )
      ?.trigger("click");

    await wrapper.get('[data-account-display-optimization-save="true"]').trigger("click");

    expect(wrapper.emitted("save-account-display-preferences")).toEqual([
      [
        {
          todayStatsWindows: ["today", "total"],
          todayStatsCycleMode: "fixed",
          groupDisplayMode: "icon",
          statusDisplayMode: "simple",
        },
      ],
    ]);
  });

  it("renders limited account controls and forwards their actions", async () => {
    const wrapper = mountToolbar({
      showLimitedControls: true,
      hideLimitedAccounts: true,
      limitedAccountsCount: 7,
    });

    expect(wrapper.text()).toContain("admin.accounts.limited.entry");

    await wrapper.get('[data-more-actions-button="true"]').trigger("click");
    await nextTick();

    expect(wrapper.text()).toContain("admin.accounts.limited.hideToggleOn");

    await wrapper
      .findAll("button")
      .find((button) =>
        button.text().includes("admin.accounts.limited.hideToggleOn"),
      )
      ?.trigger("click");
    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("admin.accounts.limited.entry"))
      ?.trigger("click");

    expect(wrapper.emitted("toggle-hide-limited")).toEqual([[]]);
    expect(wrapper.emitted("open-limited-page")).toEqual([[]]);
  });

  it("renders the platform count sort toggle and emits the next mode", async () => {
    const wrapper = mountToolbar({
      platformCountSortOrder: "count_asc",
    });

    await wrapper.get('[data-more-actions-button="true"]').trigger("click");
    await nextTick();

    const button = wrapper.get('[data-platform-sort-button="true"]');
    expect(button.text()).toContain("admin.accounts.platformSort.countAsc");
    expect(button.attributes("title")).toBe(
      "admin.accounts.platformSort.toggleDesc",
    );

    await button.trigger("click");

    expect(wrapper.emitted("update:platform-count-sort-order")).toEqual([
      ["count_desc"],
    ]);
  });

  it("moves filtered bulk edit controls into More", async () => {
    const wrapper = mountToolbar({
      selectedCount: 0,
      filteredBulkEditTotal: 12,
    });

    await wrapper.get('[data-more-actions-button="true"]').trigger("click");
    await nextTick();

    expect(wrapper.text()).toContain("admin.accounts.bulkEdit.editCurrentCategory");
    expect(wrapper.text()).toContain("12");

    await wrapper
      .findAll("button")
      .find((button) =>
        button.text().includes("admin.accounts.bulkEdit.editCurrentCategory"),
      )
      ?.trigger("click");
    await wrapper.get('[data-more-actions-button="true"]').trigger("click");
    await nextTick();
    await wrapper
      .get('[data-account-filtered-bulk-edit-exclude-grouped="true"]')
      .setValue(true);

    expect(wrapper.emitted("bulk-edit-filtered")).toEqual([[]]);
    expect(wrapper.emitted("update:filtered-bulk-edit-exclude-grouped")).toEqual([[true]]);
  });
});
