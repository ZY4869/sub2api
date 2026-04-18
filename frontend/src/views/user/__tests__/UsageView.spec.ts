import { describe, expect, it, vi, beforeEach } from "vitest";
import { flushPromises, mount } from "@vue/test-utils";
import { nextTick } from "vue";

import UsageView from "../UsageView.vue";

const {
  query,
  getStatsByDateRange,
  listFilterApiKeys,
  getRequestPreview,
  showError,
  showWarning,
  showSuccess,
  showInfo,
} = vi.hoisted(() => ({
  query: vi.fn(),
  getStatsByDateRange: vi.fn(),
  listFilterApiKeys: vi.fn(),
  getRequestPreview: vi.fn(),
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
  "common.actions": "Actions",
  "common.close": "Close",
  "common.loading": "Loading...",
  "common.refresh": "Refresh",
  "usage.requestPreview.action": "Request Details",
  "usage.requestPreview.title": "Request Details",
  "usage.requestPreview.description":
    "Review the captured preview for this usage request.",
  "usage.requestPreview.metaRequestId": "Request ID",
  "usage.requestPreview.metaCapturedAt": "Captured At",
  "usage.requestPreview.previewReady": "Preview is ready",
  "usage.requestPreview.empty": "No content available",
  "usage.requestPreview.unavailableTitle": "No request details captured",
  "usage.requestPreview.unavailableDescription":
    "This request did not capture a preview. Request detail capture may have been unavailable for that request.",
  "usage.requestPreview.failedToLoad": "Failed to load request details",
  "usage.requestPreview.sections.inbound": "Inbound Request",
  "usage.requestPreview.sections.normalized": "Normalized Request",
  "usage.requestPreview.sections.upstreamRequest": "Upstream Request",
  "usage.requestPreview.sections.upstreamResponse": "Upstream Response",
  "usage.requestPreview.sections.gatewayResponse": "Gateway Response",
  "usage.requestPreview.sections.tools": "Tools / Thinking",
  "usage.requestPreview.emptyStates.inbound":
    "No inbound request preview was captured for this request.",
  "usage.requestPreview.emptyStates.normalized":
    "No normalized request content is available for this request.",
  "usage.requestPreview.emptyStates.upstreamRequest":
    "No upstream request content is available for this request.",
  "usage.requestPreview.emptyStates.upstreamResponse":
    "No upstream response content is available for this request.",
  "usage.requestPreview.emptyStates.gatewayResponse":
    "No gateway response content is available for this request.",
  "usage.requestPreview.emptyStates.tools":
    "No tool or thinking trace was captured for this request.",
};

vi.mock("@/api", () => ({
  usageAPI: {
    query,
    getStatsByDateRange,
    listFilterApiKeys,
    getRequestPreview,
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
  template:
    '<div><slot name="actions" /><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>',
};

describe("user UsageView tooltip", () => {
  beforeEach(() => {
    query.mockReset();
    getStatsByDateRange.mockReset();
    listFilterApiKeys.mockReset();
    getRequestPreview.mockReset();
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
          id: 1,
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
    listFilterApiKeys.mockResolvedValue([]);

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
        id: 2,
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
    listFilterApiKeys.mockResolvedValue([]);

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
    expect(csvText.charCodeAt(0)).toBe(0xfeff);
    expect(csvText).toContain("Thinking Mode");
    expect(csvText).toContain("Enabled");
    expect(csvText).toContain("Billing Exempt Reason");

    window.URL.createObjectURL = originalCreateObjectURL;
    window.URL.revokeObjectURL = originalRevokeObjectURL;
    vi.stubGlobal("Blob", OriginalBlob);
    clickSpy.mockRestore();
  });

  it("keeps rendering and exporting rows when cost fields are undefined, null, or NaN", async () => {
    const exportedLogs = [
      {
        id: 3,
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
    listFilterApiKeys.mockResolvedValue([]);

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
    expect(columns.map((column: { key: string }) => column.key)).toContain(
      "request_protocol",
    );

    await setupState.exportToCSV();

    expect(exportedBlob).not.toBeNull();
    const csvText = await exportedBlob!.text();
    expect(csvText.charCodeAt(0)).toBe(0xfeff);
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
          id: 4,
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
    listFilterApiKeys.mockResolvedValue([]);

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
    expect(
      setupState.truncateUsageErrorMessage(
        "  Rate limit exceeded for this account  ",
      ),
    ).toBe("Rate limit exceeded for this account");
  });

  it("renders usage rows when query data is returned", async () => {
    query.mockResolvedValue({
      items: [
        {
          id: 5,
          request_id: "req-user-visible",
          model: "gpt-5.4",
          status: "succeeded",
          thinking_enabled: false,
          reasoning_effort: null,
          actual_cost: 0.01,
          total_cost: 0.01,
          input_cost: 0.004,
          output_cost: 0.006,
          cache_creation_cost: 0,
          cache_read_cost: 0,
          input_tokens: 100,
          output_tokens: 200,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          cache_creation_5m_tokens: 0,
          cache_creation_1h_tokens: 0,
          image_count: 0,
          image_size: null,
          first_token_ms: 20,
          duration_ms: 40,
          created_at: "2026-03-08T00:00:00Z",
          inbound_endpoint: "/v1/chat/completions",
          upstream_endpoint: "/v1/chat/completions",
          api_key: { name: "visible-key" },
        },
      ],
      total: 1,
      pages: 1,
    });
    getStatsByDateRange.mockResolvedValue({
      total_requests: 1,
      total_tokens: 300,
      total_cost: 0.01,
      avg_duration_ms: 40,
    });
    listFilterApiKeys.mockResolvedValue([
      { id: 1, name: "visible-key", deleted: false },
    ]);

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

    expect(wrapper.findAll("tbody tr[data-row-id]")).toHaveLength(1);
    expect(wrapper.text()).toContain("visible-key");
    expect(wrapper.text()).toContain("gpt-5.4");
  });

  it("retains the selected Gemini API key when date range refreshes remove it from the filter list", async () => {
    query.mockResolvedValue({
      items: [
        {
          id: 8,
          request_id: "req-gemini-visible",
          model: "gemini-2.5-pro",
          status: "succeeded",
          thinking_enabled: false,
          reasoning_effort: null,
          actual_cost: 0.02,
          total_cost: 0.02,
          input_cost: 0.01,
          output_cost: 0.01,
          cache_creation_cost: 0,
          cache_read_cost: 0,
          input_tokens: 120,
          output_tokens: 80,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          cache_creation_5m_tokens: 0,
          cache_creation_1h_tokens: 0,
          image_count: 0,
          image_size: null,
          first_token_ms: 15,
          duration_ms: 30,
          created_at: "2026-03-08T00:00:00Z",
          inbound_endpoint: "/v1beta/models/gemini-2.5-pro:generateContent",
          upstream_endpoint: "/v1beta/models/gemini-2.5-pro:generateContent",
          api_key: { name: "gemini-key" },
        },
      ],
      total: 1,
      pages: 1,
    });
    getStatsByDateRange.mockResolvedValue({
      total_requests: 1,
      total_tokens: 200,
      total_cost: 0.02,
      avg_duration_ms: 30,
    });
    listFilterApiKeys
      .mockResolvedValueOnce([{ id: 9, name: "gemini-key", deleted: false }])
      .mockResolvedValueOnce([]);

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
    setupState.filters.api_key_id = 9;

    await nextTick();

    setupState.onDateRangeChange({
      startDate: "2026-03-01",
      endDate: "2026-03-03",
      preset: null,
    });

    await flushPromises();
    await nextTick();

    expect(setupState.filters.api_key_id).toBe(9);
    expect(
      setupState.apiKeys.map((key: { id: number }) => key.id),
    ).toContain(9);
    expect(listFilterApiKeys).toHaveBeenLastCalledWith({
      start_date: "2026-03-01",
      end_date: "2026-03-03",
    });
    expect(wrapper.text()).toContain("gemini-key");
    expect(wrapper.text()).toContain("gemini-2.5-pro");
  });

  it("opens request preview modal and renders captured panels", async () => {
    query.mockResolvedValue({
      items: [
        {
          id: 6,
          request_id: "req-preview-visible",
          model: "gpt-5.4",
          status: "succeeded",
          thinking_enabled: false,
          reasoning_effort: null,
          actual_cost: 0.01,
          total_cost: 0.01,
          input_cost: 0.004,
          output_cost: 0.006,
          cache_creation_cost: 0,
          cache_read_cost: 0,
          input_tokens: 100,
          output_tokens: 200,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          cache_creation_5m_tokens: 0,
          cache_creation_1h_tokens: 0,
          image_count: 0,
          image_size: null,
          first_token_ms: 20,
          duration_ms: 40,
          created_at: "2026-03-08T00:00:00Z",
          api_key: { name: "preview-key" },
        },
      ],
      total: 1,
      pages: 1,
    });
    getStatsByDateRange.mockResolvedValue({
      total_requests: 1,
      total_tokens: 300,
      total_cost: 0.01,
      avg_duration_ms: 40,
    });
    listFilterApiKeys.mockResolvedValue([]);
    getRequestPreview.mockResolvedValue({
      available: true,
      request_id: "req-preview-visible",
      captured_at: "2026-03-08T00:00:10Z",
      inbound_request_json: '{"messages":[{"role":"user","content":"hello"}]}',
      normalized_request_json: '{"normalized":true}',
      upstream_request_json: '{"target":"upstream"}',
      upstream_response_json: '{"status":"ok"}',
      gateway_response_json: '{"gateway":"ok"}',
      tool_trace_json: '{"tools":["search"]}',
    });

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

    const previewButton = wrapper
      .findAll("button")
      .find((button) => button.text() === "Request Details");
    expect(previewButton).toBeDefined();

    await previewButton!.trigger("click");
    await flushPromises();
    await nextTick();

    expect(getRequestPreview).toHaveBeenCalledWith(6);
    expect(wrapper.text()).toContain("Inbound Request");
    expect(wrapper.text()).toContain("hello");
    expect(wrapper.text()).toContain("Tools / Thinking");
  });

  it("shows a clear empty state when request preview is unavailable", async () => {
    query.mockResolvedValue({
      items: [
        {
          id: 7,
          request_id: "req-preview-missing",
          model: "gpt-5.4",
          status: "succeeded",
          thinking_enabled: false,
          reasoning_effort: null,
          actual_cost: 0.01,
          total_cost: 0.01,
          input_cost: 0.004,
          output_cost: 0.006,
          cache_creation_cost: 0,
          cache_read_cost: 0,
          input_tokens: 100,
          output_tokens: 200,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          cache_creation_5m_tokens: 0,
          cache_creation_1h_tokens: 0,
          image_count: 0,
          image_size: null,
          first_token_ms: 20,
          duration_ms: 40,
          created_at: "2026-03-08T00:00:00Z",
          api_key: { name: "preview-key" },
        },
      ],
      total: 1,
      pages: 1,
    });
    getStatsByDateRange.mockResolvedValue({
      total_requests: 1,
      total_tokens: 300,
      total_cost: 0.01,
      avg_duration_ms: 40,
    });
    listFilterApiKeys.mockResolvedValue([]);
    getRequestPreview.mockResolvedValue({
      available: false,
      request_id: "req-preview-missing",
      captured_at: null,
      inbound_request_json: "",
      normalized_request_json: "",
      upstream_request_json: "",
      upstream_response_json: "",
      gateway_response_json: "",
      tool_trace_json: "",
    });

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

    const previewButton = wrapper
      .findAll("button")
      .find((button) => button.text() === "Request Details");
    await previewButton!.trigger("click");
    await flushPromises();
    await nextTick();

    expect(wrapper.text()).toContain("No request details captured");
  });

  it("shows a friendly error when request preview loading fails", async () => {
    query.mockResolvedValue({
      items: [
        {
          id: 8,
          request_id: "req-preview-error",
          model: "gpt-5.4",
          status: "succeeded",
          thinking_enabled: false,
          reasoning_effort: null,
          actual_cost: 0.01,
          total_cost: 0.01,
          input_cost: 0.004,
          output_cost: 0.006,
          cache_creation_cost: 0,
          cache_read_cost: 0,
          input_tokens: 100,
          output_tokens: 200,
          cache_creation_tokens: 0,
          cache_read_tokens: 0,
          cache_creation_5m_tokens: 0,
          cache_creation_1h_tokens: 0,
          image_count: 0,
          image_size: null,
          first_token_ms: 20,
          duration_ms: 40,
          created_at: "2026-03-08T00:00:00Z",
          api_key: { name: "preview-key" },
        },
      ],
      total: 1,
      pages: 1,
    });
    getStatsByDateRange.mockResolvedValue({
      total_requests: 1,
      total_tokens: 300,
      total_cost: 0.01,
      avg_duration_ms: 40,
    });
    listFilterApiKeys.mockResolvedValue([]);
    getRequestPreview.mockRejectedValue(new Error("network error"));

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

    const previewButton = wrapper
      .findAll("button")
      .find((button) => button.text() === "Request Details");
    await previewButton!.trigger("click");
    await flushPromises();
    await nextTick();

    expect(wrapper.text()).toContain("Failed to load request details");
  });
});
