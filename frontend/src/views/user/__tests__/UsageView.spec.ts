import { describe, expect, it, vi, beforeEach } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { nextTick } from "vue";

import UsageView from "../UsageView.vue";

const {
  query,
  getStatsByDateRange,
  list,
  showError,
  showWarning,
  showSuccess,
  showInfo,
} = vi.hoisted(() => ({
  query: vi.fn(),
  getStatsByDateRange: vi.fn(),
  list: vi.fn(),
  showError: vi.fn(),
  showWarning: vi.fn(),
  showSuccess: vi.fn(),
  showInfo: vi.fn(),
}));

const messages: Record<string, string> = {
  "usage.costDetails": "Cost Breakdown",
  "usage.status": "Status",
  "usage.statusFailed": "Failed",
  "usage.statusSucceeded": "Succeeded",
  "usage.httpStatus": "HTTP Status",
  "usage.errorCode": "Error Code",
  "usage.errorMessage": "Error Message",
  "usage.simulatedClientCodex": "Codex",
  "usage.simulatedClientGeminiCli": "Gemini CLI",
  "admin.usage.inputCost": "Input Cost",
  "admin.usage.outputCost": "Output Cost",
  "admin.usage.cacheCreationCost": "Cache Creation Cost",
  "admin.usage.cacheReadCost": "Cache Read Cost",
  "usage.inputTokenPrice": "Input price",
  "usage.outputTokenPrice": "Output price",
  "usage.perMillionTokens": "/ 1M tokens",
  "usage.serviceTier": "Service tier",
  "usage.serviceTierPriority": "Fast",
  "usage.serviceTierFlex": "Flex",
  "usage.serviceTierStandard": "Standard",
  "usage.rate": "Rate",
  "usage.original": "Original",
  "usage.billed": "Billed",
  "usage.allApiKeys": "All API Keys",
  "usage.apiKeyFilter": "API Key",
  "usage.model": "Model",
  "usage.thinkingMode": "Thinking Mode",
  "usage.thinkingEnabled": "Enabled",
  "usage.thinkingDisabled": "Disabled",
  "usage.reasoningEffort": "Reasoning Effort",
  "usage.type": "Type",
  "usage.tokens": "Tokens",
  "usage.cost": "Cost",
  "usage.firstToken": "First Token",
  "usage.duration": "Duration",
  "usage.time": "Time",
  "usage.userAgent": "User Agent",
  "usage.requestInfo": "Request Info",
};

vi.mock("@/api", () => ({
  usageAPI: {
    query,
    getStatsByDateRange,
  },
  keysAPI: {
    list,
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: () => ({ showError, showWarning, showSuccess, showInfo }),
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
const TablePageLayoutStub = {
  template: '<div><slot name="actions" /><slot name="filters" /><slot /></div>',
};

describe("user UsageView tooltip", () => {
  beforeEach(() => {
    query.mockReset();
    getStatsByDateRange.mockReset();
    list.mockReset();
    showError.mockReset();
    showWarning.mockReset();
    showSuccess.mockReset();
    showInfo.mockReset();

    vi.spyOn(HTMLElement.prototype, "getBoundingClientRect").mockReturnValue({
      x: 0,
      y: 0,
      top: 20,
      left: 20,
      right: 120,
      bottom: 40,
      width: 100,
      height: 20,
      toJSON: () => ({}),
    } as DOMRect);
    (globalThis as any).ResizeObserver = class {
      observe() {}
      disconnect() {}
    };
  });

  it("shows fast service tier and unit prices in user tooltip", async () => {
    query.mockResolvedValue({
      items: [
        {
          request_id: "req-user-1",
          model: "gpt-5.4",
          thinking_enabled: true,
          reasoning_effort: "high",
          actual_cost: 0.092883,
          total_cost: 0.092883,
          rate_multiplier: 1,
          service_tier: "priority",
          input_cost: 0.020285,
          output_cost: 0.00303,
          cache_creation_cost: 0,
          cache_read_cost: 0.069568,
          input_tokens: 4057,
          output_tokens: 101,
          cache_creation_tokens: 0,
          cache_read_tokens: 278272,
          cache_creation_5m_tokens: 0,
          cache_creation_1h_tokens: 0,
          image_count: 0,
          image_size: null,
          first_token_ms: null,
          duration_ms: 1,
          created_at: "2026-03-08T00:00:00Z",
          api_key: { name: "demo-key" },
        },
      ],
      total: 1,
      pages: 1,
    });
    getStatsByDateRange.mockResolvedValue({
      total_requests: 1,
      total_tokens: 100,
      total_cost: 0.1,
      avg_duration_ms: 1,
    });
    list.mockResolvedValue({ items: [] });

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          TablePageLayout: TablePageLayoutStub,
          Pagination: true,
          EmptyState: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenDisplayModeToggle: true,
          Teleport: true,
        },
      },
    });

    await flushPromises();
    await nextTick();

    const setupState = (wrapper.vm as any).$?.setupState;
    const columns = setupState.columns.value ?? setupState.columns;
    expect(columns.map((column: { key: string }) => column.key)).toContain(
      "thinking_enabled",
    );
    expect(
      columns.find(
        (column: { key: string }) => column.key === "thinking_enabled",
      )?.label,
    ).toBe("Thinking Mode");
    setupState.tooltipData = {
      request_id: "req-user-1",
      actual_cost: 0.092883,
      total_cost: 0.092883,
      rate_multiplier: 1,
      service_tier: "priority",
      input_cost: 0.020285,
      output_cost: 0.00303,
      cache_creation_cost: 0,
      cache_read_cost: 0.069568,
      input_tokens: 4057,
      output_tokens: 101,
    };
    setupState.tooltipVisible = true;
    await nextTick();

    const text = wrapper.text();
    expect(text).toContain("Service tier");
    expect(text).toContain("Fast");
    expect(text).toContain("Rate");
    expect(text).toContain("1.00x");
    expect(text).toContain("Billed");
    expect(text).toContain("$0.092883");
    expect(text).toContain("$5.0000 / 1M tokens");
    expect(text).toContain("$30.0000 / 1M tokens");
  });

  it("exports csv with input and output unit price columns", async () => {
    const exportedLogs = [
      {
        request_id: "req-user-export",
        actual_cost: 0.092883,
        total_cost: 0.092883,
        rate_multiplier: 1,
        service_tier: "priority",
        input_cost: 0.020285,
        output_cost: 0.00303,
        cache_creation_cost: 0.000001,
        cache_read_cost: 0.069568,
        input_tokens: 4057,
        output_tokens: 101,
        cache_creation_tokens: 4,
        cache_read_tokens: 278272,
        cache_creation_5m_tokens: 0,
        cache_creation_1h_tokens: 0,
        image_count: 0,
        image_size: null,
        first_token_ms: 12,
        duration_ms: 345,
        created_at: "2026-03-08T00:00:00Z",
        model: "gpt-5.4",
        thinking_enabled: true,
        reasoning_effort: null,
        api_key: { name: "demo-key" },
      },
    ];

    query.mockResolvedValue({
      items: exportedLogs,
      total: 1,
      pages: 1,
    });
    getStatsByDateRange.mockResolvedValue({
      total_requests: 1,
      total_tokens: 100,
      total_cost: 0.1,
      avg_duration_ms: 1,
    });
    list.mockResolvedValue({ items: [] });

    let exportedBlob: Blob | null = null;
    const originalCreateObjectURL = window.URL.createObjectURL;
    const originalRevokeObjectURL = window.URL.revokeObjectURL;
    const OriginalBlob = globalThis.Blob;
    class MockBlob {
      private readonly content: string;

      constructor(parts: Array<BlobPart>) {
        this.content = parts.map((part) => String(part)).join("");
      }

      text() {
        return Promise.resolve(this.content);
      }
    }
    vi.stubGlobal("Blob", MockBlob as typeof Blob);
    window.URL.createObjectURL = vi.fn((blob: Blob | MediaSource) => {
      exportedBlob = blob as Blob;
      return "blob:usage-export";
    }) as typeof window.URL.createObjectURL;
    window.URL.revokeObjectURL = vi.fn(
      () => {},
    ) as typeof window.URL.revokeObjectURL;
    const clickSpy = vi
      .spyOn(HTMLAnchorElement.prototype, "click")
      .mockImplementation(() => {});

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          TablePageLayout: TablePageLayoutStub,
          Pagination: true,
          EmptyState: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenDisplayModeToggle: true,
          Teleport: true,
        },
      },
    });

    await flushPromises();

    const setupState = (wrapper.vm as any).$?.setupState;
    await setupState.exportToCSV();

    expect(exportedBlob).not.toBeNull();
    expect(clickSpy).toHaveBeenCalled();
    expect(showSuccess).toHaveBeenCalled();
    const csvText = await exportedBlob!.text();
    expect(csvText).toContain("Thinking Mode");
    expect(csvText).toContain("Enabled");

    window.URL.createObjectURL = originalCreateObjectURL;
    window.URL.revokeObjectURL = originalRevokeObjectURL;
    vi.stubGlobal("Blob", OriginalBlob);
    clickSpy.mockRestore();
  });

  it("keeps rendering and exporting rows when cost fields are undefined, null, or NaN", async () => {
    const exportedLogs = [
      {
        request_id: "req-user-unpriced",
        actual_cost: undefined,
        total_cost: Number.NaN,
        rate_multiplier: undefined,
        service_tier: "standard",
        input_cost: null,
        output_cost: undefined,
        cache_creation_cost: Number.NaN,
        cache_read_cost: null,
        input_tokens: 0,
        output_tokens: 0,
        cache_creation_tokens: 0,
        cache_read_tokens: 0,
        cache_creation_5m_tokens: 0,
        cache_creation_1h_tokens: 0,
        image_count: 0,
        image_size: null,
        first_token_ms: null,
        duration_ms: 12,
        created_at: "2026-03-08T00:00:00Z",
        model: "gpt-5.4",
        thinking_enabled: false,
        reasoning_effort: null,
        inbound_endpoint: "/v1/chat/completions",
        upstream_endpoint: "/v1/chat/completions",
        api_key: { name: "demo-key" },
      },
    ];

    query.mockResolvedValue({
      items: exportedLogs,
      total: 1,
      pages: 1,
    });
    getStatsByDateRange.mockResolvedValue({
      total_requests: 1,
      total_tokens: 0,
      total_cost: 0,
      avg_duration_ms: 12,
    });
    list.mockResolvedValue({ items: [] });

    let exportedBlob: Blob | null = null;
    const originalCreateObjectURL = window.URL.createObjectURL;
    const originalRevokeObjectURL = window.URL.revokeObjectURL;
    const OriginalBlob = globalThis.Blob;
    class MockBlob {
      private readonly content: string;

      constructor(parts: Array<BlobPart>) {
        this.content = parts.map((part) => String(part)).join("");
      }

      text() {
        return Promise.resolve(this.content);
      }
    }
    vi.stubGlobal("Blob", MockBlob as typeof Blob);
    window.URL.createObjectURL = vi.fn((blob: Blob | MediaSource) => {
      exportedBlob = blob as Blob;
      return "blob:usage-export-invalid";
    }) as typeof window.URL.createObjectURL;
    window.URL.revokeObjectURL = vi.fn(
      () => {},
    ) as typeof window.URL.revokeObjectURL;
    const clickSpy = vi
      .spyOn(HTMLAnchorElement.prototype, "click")
      .mockImplementation(() => {});

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          TablePageLayout: TablePageLayoutStub,
          Pagination: true,
          EmptyState: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenDisplayModeToggle: true,
          Teleport: true,
        },
      },
    });

    await flushPromises();

    const setupState = (wrapper.vm as any).$?.setupState;
    const columns = setupState.columns.value ?? setupState.columns;
    expect(columns.map((column: { key: string }) => column.key)).toContain("request_protocol");

    await setupState.exportToCSV();

    expect(exportedBlob).not.toBeNull();
    const csvText = await exportedBlob!.text();
    expect(csvText).toContain("demo-key");
    expect(csvText).toContain("gpt-5.4");
    expect(csvText).toContain("OpenAI /v1/chat/completions Native");
    expect(showError).not.toHaveBeenCalled();

    window.URL.createObjectURL = originalCreateObjectURL;
    window.URL.revokeObjectURL = originalRevokeObjectURL;
    vi.stubGlobal("Blob", OriginalBlob);
    clickSpy.mockRestore();
  });

  it("formats failed status labels and simulated client tags for failed rows", async () => {
    query.mockResolvedValue({
      items: [
        {
          request_id: "req-user-failed",
          model: "gpt-5.4",
          status: "failed",
          simulated_client: "codex",
          http_status: 429,
          error_code: "rate_limited",
          error_message: "Rate limit exceeded for this account",
          actual_cost: 0,
          total_cost: 0,
          input_cost: 0,
          output_cost: 0,
          cache_creation_cost: 0,
          cache_read_cost: 0,
          input_tokens: 0,
          output_tokens: 0,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          cache_creation_5m_tokens: 0,
          cache_creation_1h_tokens: 0,
          image_count: 0,
          image_size: null,
          first_token_ms: null,
          duration_ms: 10,
          created_at: "2026-03-08T00:00:00Z",
          api_key: { name: "demo-key" },
        },
      ],
      total: 1,
      pages: 1,
    });
    getStatsByDateRange.mockResolvedValue({
      total_requests: 1,
      total_tokens: 0,
      total_cost: 0,
      avg_duration_ms: 10,
    });
    list.mockResolvedValue({ items: [] });

    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          TablePageLayout: TablePageLayoutStub,
          Pagination: true,
          EmptyState: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenDisplayModeToggle: true,
          Teleport: true,
        },
      },
    });

    await flushPromises();
    await nextTick();

    const setupState = (wrapper.vm as any).$?.setupState;
    expect(setupState.getStatusLabel("failed")).toBe("Failed");
    expect(setupState.getStatusLabel("succeeded")).toBe("Succeeded");
    expect(setupState.getSimulatedClientLabel("codex")).toBe("Codex");
    expect(setupState.getSimulatedClientLabel("gemini_cli")).toBe("Gemini CLI");
    expect(setupState.truncateUsageErrorMessage("  Rate limit exceeded for this account  ")).toBe(
      "Rate limit exceeded for this account"
    );
  });
});
