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

vi.mock("@/utils/format", () => ({
  formatReasoningEffort: (value: string | null | undefined) => value ?? "-",
  formatThinkingEnabled: (value: boolean | null | undefined) =>
    value === true ? "Enabled" : value === false ? "Disabled" : "-",
}));

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  };
});

const AppLayoutStub = { template: "<div><slot /></div>" };
const UsageFiltersStub = { template: '<div><slot name="after-reset" /></div>' };
const UsageTableStub = {
  props: ["columns"],
  template:
    '<div data-test="usage-table-columns">{{ columns.map(column => column.key).join(",") }}</div>',
};
const ModelDistributionChartStub = {
  props: ["metric"],
  emits: ["update:metric"],
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

describe("admin UsageView distribution metric toggles", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    list.mockReset();
    getStats.mockReset();
    getSnapshotV2.mockReset();
    getModelStats.mockReset();
    getById.mockReset();

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
    const wrapper = mount(UsageView, {
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
          ModelDistributionChart: ModelDistributionChartStub,
          GroupDistributionChart: GroupDistributionChartStub,
        },
      },
    });

    vi.advanceTimersByTime(120);
    await flushPromises();

    expect(getSnapshotV2).toHaveBeenCalledTimes(1);

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

  it("keeps thinking mode and reasoning effort visible by default", async () => {
    const wrapper = mount(UsageView, {
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
          Icon: true,
          TokenDisplayModeToggle: true,
          TokenUsageTrend: true,
          ModelDistributionChart: ModelDistributionChartStub,
          GroupDistributionChart: GroupDistributionChartStub,
        },
      },
    });

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
});
