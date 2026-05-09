import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";

import UsageView from "../UsageView.vue";

const { list, getStats, getSnapshotV2, getModelStats, getById } = vi.hoisted(() => {
  vi.stubGlobal("localStorage", {
    getItem: vi.fn(() => null),
    setItem: vi.fn(),
    removeItem: vi.fn(),
  });

  return {
    list: vi.fn(),
    getStats: vi.fn(),
    getSnapshotV2: vi.fn(),
    getModelStats: vi.fn(),
    getById: vi.fn(),
  };
});

const authState = vi.hoisted(() => ({
  isAdmin: true,
  canReviewRequestDetails: true,
}));

const routeState = vi.hoisted(() => ({
  query: {} as Record<string, string | undefined>,
  replace: vi.fn(),
}));

const usageModelDisplayModeState = vi.hoisted(() => ({
  usageModelDisplayMode: "display_and_model" as const,
  updatingUsageModelDisplayMode: false,
  setUsageModelDisplayMode: vi.fn(),
}));

const usageContextBadgeDisplayModeState = vi.hoisted(() => ({
  usageContextBadgeDisplayMode: "both" as const,
  updatingUsageContextBadgeDisplayMode: false,
  setUsageContextBadgeDisplayMode: vi.fn(),
}));

const messages: Record<string, string> = {
  "admin.dashboard.day": "Day",
  "admin.dashboard.hour": "Hour",
  "admin.usage.failedToLoadUser": "Failed to load user",
  "usage.thinkingMode": "Thinking Mode",
  "usage.reasoningEffort": "Reasoning Effort",
  "usage.endpoint": "Endpoint",
  "usage.model": "Model",
  "usage.apiKeyFilter": "API Key",
  "admin.usage.account": "Account",
  "admin.usage.group": "Group",
  "usage.type": "Type",
  "usage.tokens": "Tokens",
  "usage.cost": "Cost",
  "usage.firstToken": "First Token",
  "usage.duration": "Duration",
  "usage.time": "Time",
  "usage.userAgent": "User Agent",
  "admin.usage.ipAddress": "IP Address",
  "admin.usage.user": "User",
};

vi.mock("@/api/admin", () => ({
  adminAPI: {
    usage: {
      list,
      getStats,
    },
    dashboard: {
      getSnapshotV2,
      getModelStats,
    },
    users: {
      getById,
    },
  },
}));

vi.mock("@/api/admin/usage", () => ({
  adminUsageAPI: {
    list: vi.fn(),
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showWarning: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn(),
  }),
}));

vi.mock("@/stores/auth", () => ({
  useAuthStore: () => authState,
}));

vi.mock("@/composables/useUsageModelDisplayModePreference", () => ({
  useUsageModelDisplayModePreference: () => usageModelDisplayModeState,
}));

vi.mock("@/composables/useUsageContextBadgeDisplayModePreference", () => ({
  useUsageContextBadgeDisplayModePreference: () =>
    usageContextBadgeDisplayModeState,
}));

vi.mock("vue-router", () => ({
  useRoute: () => routeState,
  useRouter: () => ({
    replace: routeState.replace,
  }),
}));

vi.mock("@/utils/format", () => ({
  formatReasoningEffort: (value: string | null | undefined) => value ?? "-",
  formatThinkingEnabled: (value: boolean | null | undefined) =>
    value === true ? "Enabled" : value === false ? "Disabled" : "-",
}));

vi.mock("vue-i18n", async () => {
  return {
    createI18n: () => ({
      global: {
        locale: {
          value: "zh-CN",
        },
      },
    }),
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  };
});

const AppLayoutStub = { template: "<div><slot /></div>" };
const UsageFiltersStub = {
  props: ["modelValue"],
  template: `
    <div>
      <div data-test="toolbar-left"><slot name="toolbar-left" /></div>
      <button
        data-test="apply-channel-filter"
        @click="$emit('update:modelValue', { ...modelValue, channel_id: 11 }); $emit('change')"
      >
        apply channel
      </button>
      <div data-test="after-reset"><slot name="after-reset" /></div>
    </div>
  `,
};
const UsageTableStub = {
  props: ["columns", "usageModelDisplayMode", "usageContextBadgeDisplayMode"],
  template: `
    <div>
      <div data-test="usage-table-columns">{{ columns.map(column => column.key).join(",") }}</div>
      <div data-test="usage-table-display-mode">{{ usageModelDisplayMode }}</div>
      <div data-test="usage-table-badge-mode">{{ usageContextBadgeDisplayMode }}</div>
    </div>
  `,
};
const UsageModelDisplayModeToggleStub = {
  props: ["modelValue", "disabled"],
  emits: ["update:modelValue"],
  template: `
    <button
      data-test="usage-model-display-toggle"
      @click="$emit('update:modelValue', 'model_only')"
    >
      {{ modelValue }}
    </button>
  `,
};
const UsageContextBadgeDisplayModeToggleStub = {
  props: ["modelValue", "disabled"],
  emits: ["update:modelValue"],
  template: `
    <button
      data-test="usage-context-badge-toggle"
      @click="$emit('update:modelValue', 'native_only')"
    >
      {{ modelValue }}
    </button>
  `,
};
const ModelDistributionChartStub = {
  props: ["metric"],
  emits: ["update:metric", "ranking-click"],
  template: `
    <div data-test="model-chart">
      <span class="metric">{{ metric }}</span>
      <button class="switch-metric" @click="$emit('update:metric', 'actual_cost')">switch</button>
    </div>
  `,
};
const GroupDistributionChartStub = {
  props: ["metric"],
  emits: ["update:metric"],
  template: `
    <div data-test="group-chart">
      <span class="metric">{{ metric }}</span>
      <button class="switch-metric" @click="$emit('update:metric', 'actual_cost')">switch</button>
    </div>
  `,
};

function mountUsageView(extraStubs: Record<string, unknown> = {}) {
  return mount(UsageView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        UsageStatsCards: true,
        UsageFilters: UsageFiltersStub,
        UsageTable: UsageTableStub,
        UsageExportProgress: true,
        UsageCleanupDialog: true,
        UserBalanceHistoryModal: true,
        Pagination: true,
        Select: true,
        DateRangePicker: true,
        Icon: true,
        TokenDisplayModeToggle: true,
        TokenUsageTrend: true,
        UsageModelDisplayModeToggle: UsageModelDisplayModeToggleStub,
        UsageContextBadgeDisplayModeToggle:
          UsageContextBadgeDisplayModeToggleStub,
        ModelDistributionChart: ModelDistributionChartStub,
        GroupDistributionChart: GroupDistributionChartStub,
        ...extraStubs,
      },
    },
  });
}

describe("admin UsageView distribution metric toggles", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    list.mockReset();
    getStats.mockReset();
    getSnapshotV2.mockReset();
    getModelStats.mockReset();
    getById.mockReset();
    routeState.query = {};
    routeState.replace.mockReset();
    authState.isAdmin = true;
    authState.canReviewRequestDetails = true;
    usageModelDisplayModeState.usageModelDisplayMode = "display_and_model";
    usageModelDisplayModeState.updatingUsageModelDisplayMode = false;
    usageModelDisplayModeState.setUsageModelDisplayMode.mockReset();
    usageContextBadgeDisplayModeState.usageContextBadgeDisplayMode = "both";
    usageContextBadgeDisplayModeState.updatingUsageContextBadgeDisplayMode =
      false;
    usageContextBadgeDisplayModeState.setUsageContextBadgeDisplayMode.mockReset();

    list.mockResolvedValue({
      items: [],
      total: 0,
      pages: 0,
    });
    getStats.mockResolvedValue({
      total_requests: 0,
      total_input_tokens: 0,
      total_output_tokens: 0,
      total_cache_tokens: 0,
      total_tokens: 0,
      total_cost: 0,
      total_actual_cost: 0,
      average_duration_ms: 0,
    });
    getSnapshotV2.mockResolvedValue({
      trend: [],
      models: [],
      groups: [],
    });
    getModelStats.mockResolvedValue({
      models: [],
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("keeps model and group metric toggles independent without refetching chart data", async () => {
    const wrapper = mountUsageView();

    vi.advanceTimersByTime(120);
    await flushPromises();

    expect(getSnapshotV2).toHaveBeenCalledTimes(1);

    await wrapper.get('[data-test="admin-usage-tab-leaderboard"]').trigger("click");
    await flushPromises();

    const modelChart = wrapper.find('[data-test="model-chart"]');
    const groupChart = wrapper.find('[data-test="group-chart"]');

    expect(modelChart.find(".metric").text()).toBe("tokens");
    expect(groupChart.find(".metric").text()).toBe("tokens");

    await modelChart.find(".switch-metric").trigger("click");
    await flushPromises();

    expect(modelChart.find(".metric").text()).toBe("actual_cost");
    expect(groupChart.find(".metric").text()).toBe("tokens");
    expect(getSnapshotV2).toHaveBeenCalledTimes(1);

    await groupChart.find(".switch-metric").trigger("click");
    await flushPromises();

    expect(modelChart.find(".metric").text()).toBe("actual_cost");
    expect(groupChart.find(".metric").text()).toBe("actual_cost");
    expect(getSnapshotV2).toHaveBeenCalledTimes(1);
  });

  it("renders the display toggles in toolbar-left and forwards the selected modes", async () => {
    const wrapper = mountUsageView();

    vi.advanceTimersByTime(120);
    await flushPromises();

    const toolbarLeft = wrapper.get('[data-test="toolbar-left"]');

    expect(toolbarLeft.find('[data-test="usage-model-display-toggle"]').text()).toBe(
      "display_and_model",
    );
    expect(
      toolbarLeft.find('[data-test="usage-context-badge-toggle"]').text(),
    ).toBe("both");
    expect(wrapper.findAll('[data-test="usage-model-display-toggle"]')).toHaveLength(1);
    expect(wrapper.findAll('[data-test="usage-context-badge-toggle"]')).toHaveLength(1);
    expect(wrapper.get('[data-test="usage-table-display-mode"]').text()).toBe(
      "display_and_model",
    );
    expect(wrapper.get('[data-test="usage-table-badge-mode"]').text()).toBe(
      "both",
    );

    await toolbarLeft.get('[data-test="usage-model-display-toggle"]').trigger("click");
    await toolbarLeft.get('[data-test="usage-context-badge-toggle"]').trigger("click");
    await flushPromises();

    expect(
      usageModelDisplayModeState.setUsageModelDisplayMode,
    ).toHaveBeenCalledWith("model_only");
    expect(
      usageContextBadgeDisplayModeState.setUsageContextBadgeDisplayMode,
    ).toHaveBeenCalledWith("native_only");
  });

  it("keeps thinking mode and reasoning effort visible by default", async () => {
    const wrapper = mountUsageView();

    vi.advanceTimersByTime(120);
    await flushPromises();

    const renderedColumns = wrapper
      .get('[data-test="usage-table-columns"]')
      .text()
      .split(",");
    expect(renderedColumns).toContain("thinking_enabled");
    expect(renderedColumns).toContain("reasoning_effort");
    expect(renderedColumns).toContain("endpoint");
    expect(renderedColumns).not.toContain("user_agent");
  });

  it("passes channel_id to usage list, stats, and charts when filters change", async () => {
    const wrapper = mountUsageView();

    vi.advanceTimersByTime(120);
    await flushPromises();

    list.mockClear();
    getStats.mockClear();
    getSnapshotV2.mockClear();
    getModelStats.mockClear();

    await wrapper.get('[data-test="apply-channel-filter"]').trigger("click");
    await flushPromises();

    expect(list).toHaveBeenCalledWith(
      expect.objectContaining({ channel_id: 11 }),
      expect.any(Object),
    );
    expect(getStats).toHaveBeenCalledWith(
      expect.objectContaining({ channel_id: 11 }),
    );
    expect(getSnapshotV2).toHaveBeenCalledWith(
      expect.objectContaining({ channel_id: 11 }),
    );
    expect(getModelStats).toHaveBeenCalledWith(
      expect.objectContaining({ channel_id: 11, model_source: "requested" }),
    );
  });

  it("hides request details tab when permission is missing", async () => {
    authState.isAdmin = false;
    authState.canReviewRequestDetails = false;

    const wrapper = mountUsageView();

    expect(wrapper.find('[data-test=\"admin-usage-tab-request-details\"]').exists()).toBe(false);
  });

  it("opens request details from the route tab query and keeps tab changes in the URL", async () => {
    routeState.query = { tab: "request_details" };

    const wrapper = mountUsageView({
      RequestDetailsTraceTab: {
        props: ["routeTabValue"],
        template:
          '<div data-test="request-details-trace">{{ routeTabValue }}</div>',
      },
    });

    expect(wrapper.find('[data-test="request-details-trace"]').exists()).toBe(true);
    expect(wrapper.find('[data-test="request-details-trace"]').text()).toBe(
      "request_details",
    );

    await wrapper.get('[data-test="admin-usage-tab-records"]').trigger("click");
    await flushPromises();

    expect(routeState.replace).toHaveBeenCalledWith({ query: {} });
  });
});
