import { beforeEach, describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";

import UsageTable from "../UsageTable.vue";

const clipboardState = vi.hoisted(() => ({
  copyToClipboard: vi.fn(),
}));

vi.mock("@/composables/useClipboard", () => ({
  useClipboard: () => ({
    copyToClipboard: clipboardState.copyToClipboard,
  }),
}));

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => {
        const messages: Record<string, string> = {
          "usage.statusSucceeded": "Succeeded",
          "usage.statusFailed": "Failed",
          "usage.stream": "Stream",
          "usage.sync": "Sync",
          "usage.requestPreview.action": "Preview",
          "usage.cacheTtlOverriddenHint": "TTL overridden",
          "usage.inbound": "Inbound",
          "usage.upstream": "Upstream",
          "usage.userAgentCopied": "User-Agent copied",
          "usage.millionContextRequested": "1M Requested",
          "common.copy": "Copy",
        };
        return messages[key] ?? key;
      },
    }),
  };
});

const DataTableStub = {
  props: ["data", "rowKey"],
  template: `
    <div>
      <span data-test="row-key">{{ rowKey }}</span>
      <div v-for="(row, index) in data" :key="row.id || index">
        <div data-test="token-cell"><slot name="cell-tokens" :row="row" /></div>
        <div data-test="cache-hit-cell"><slot name="cell-cache_hit" :row="row" /></div>
        <div data-test="status-cell"><slot name="cell-status" :row="row" /></div>
        <div data-test="reasoning-cell"><slot name="cell-reasoning_effort" :row="row" /></div>
        <div data-test="endpoint-cell"><slot name="cell-endpoint" :row="row" /></div>
        <div data-test="group-cell"><slot name="cell-group" :row="row" /></div>
        <div data-test="user-agent-cell"><slot name="cell-user_agent" :row="row" /></div>
        <slot name="cell-thinking_enabled" :row="row" />
        <div data-test="actions-cell"><slot name="cell-actions" :row="row" /></div>
      </div>
    </div>
  `,
};

const IconStub = {
  props: ["name"],
  template: '<span :data-icon="name"><slot /></span>',
};

function mountUsageTable(
  row: Record<string, unknown>,
  overrideProps: Record<string, unknown> = {},
) {
  return mount(UsageTable, {
    props: {
      columns: [],
      usageLogs: [row],
      loading: false,
      usageModelDisplayMode: "model_only",
      formatCurrencyBreakdown: () => "$0.0000",
      formatTokens: (value: number) => String(value),
      formatCacheTokens: (value: number) => String(value),
      formatDuration: () => "0ms",
      formatUserAgent: (value: string) => value,
      getStatusBadgeClass: () => "",
      getStatusLabel: () => "Succeeded",
      getSimulatedClientLabel: () => "Codex",
      truncateUsageErrorMessage: (value: string) => value,
      formatReasoningEffortPair: () => "-",
      formatUsageMillionContextLines: () => [],
      formatThinkingEnabled: (value: boolean | null | undefined) =>
        value === true ? "Enabled" : "Disabled",
      getRequestTypeBadgeClass: () => "",
      getRequestTypeLabel: () => "Sync",
      getChargeLabel: () => null,
      getChargeBadgeClass: () => "",
      ...overrideProps,
    },
    global: {
      stubs: {
        DataTable: DataTableStub,
        EmptyState: true,
        UsageModelCell: true,
        UsageSuccessRateCell: true,
        UsageProtocolCell: true,
        PlatformIcon: {
          props: ["platform"],
          template: '<span data-test="platform-icon" :data-platform="platform" />',
        },
        Icon: IconStub,
        Teleport: true,
      },
    },
  });
}

describe("user usage UsageTable", () => {
  beforeEach(() => {
    clipboardState.copyToClipboard.mockReset();
    clipboardState.copyToClipboard.mockResolvedValue(true);
  });

  it("renders cache hit separately from token groups", () => {
    const wrapper = mountUsageTable({
      id: 1,
      input_tokens: 100,
      output_tokens: 200,
      cache_creation_tokens: 10,
      cache_creation_5m_tokens: 20,
      cache_creation_1h_tokens: 30,
      cache_read_tokens: 70,
      cache_ttl_overridden: false,
      image_count: 0,
      thinking_enabled: true,
    });

    const tokenText = wrapper.get('[data-test="token-cell"]').text();
    const cacheHitText = wrapper.get('[data-test="cache-hit-cell"]').text();
    expect(tokenText).toContain("100");
    expect(tokenText).toContain("200");
    expect(tokenText).toContain("60");
    expect(tokenText).toContain("70");
    expect(cacheHitText).toContain("70");
    expect(cacheHitText).toContain("30.4%");
  });

  it("renders thinking mode as accessible icons", () => {
    const enabled = mountUsageTable({
      id: 2,
      input_tokens: 0,
      output_tokens: 0,
      cache_creation_tokens: 0,
      cache_creation_5m_tokens: 0,
      cache_creation_1h_tokens: 0,
      cache_read_tokens: 0,
      image_count: 0,
      thinking_enabled: true,
    });
    const disabled = mountUsageTable({
      id: 3,
      input_tokens: 0,
      output_tokens: 0,
      cache_creation_tokens: 0,
      cache_creation_5m_tokens: 0,
      cache_creation_1h_tokens: 0,
      cache_read_tokens: 0,
      image_count: 0,
      thinking_enabled: false,
    });

    expect(enabled.find('[data-icon="checkCircle"]').exists()).toBe(true);
    expect(enabled.find('[aria-label="Enabled"]').exists()).toBe(true);
    expect(enabled.find('[title="Enabled"]').exists()).toBe(true);
    expect(disabled.find('[data-icon="xCircle"]').exists()).toBe(true);
    expect(disabled.find('[aria-label="Disabled"]').exists()).toBe(true);
    expect(disabled.find('[title="Disabled"]').exists()).toBe(true);
  });

  it("renders group, endpoint icons, compact user-agent, and no actions slot", async () => {
    const wrapper = mountUsageTable({
      id: 4,
      input_tokens: 0,
      output_tokens: 0,
      cache_creation_tokens: 0,
      cache_creation_5m_tokens: 0,
      cache_creation_1h_tokens: 0,
      cache_read_tokens: 0,
      image_count: 0,
      thinking_enabled: false,
      inbound_endpoint: "/v1/chat/completions",
      upstream_endpoint: "/v1beta/models/gemini-2.5-pro:generateContent",
      group: { id: 9, name: "Paid Group" },
      user_agent: "curl/8.6.0",
    });

    expect(wrapper.get('[data-test="group-cell"]').text()).toContain("Paid Group");
    expect(wrapper.findAll('[data-test="platform-icon"]')).toHaveLength(2);
    expect(wrapper.get('[title="Inbound: Chat Completions (/v1/chat/completions)"]').exists()).toBe(true);
    expect(wrapper.get('[title="Upstream: Gemini Models API (/v1beta/models/gemini-2.5-pro:generateContent)"]').exists()).toBe(true);
    expect(wrapper.get('[data-test="user-agent-cell"]').text()).toContain("curl/8.6.0");
    expect(wrapper.get('[data-test="actions-cell"]').text()).toBe("");

    await wrapper.get('[data-test="user-agent-cell"] button').trigger("click");

    expect(clipboardState.copyToClipboard).toHaveBeenCalledWith(
      "curl/8.6.0",
      "User-Agent copied",
    );
  });

  it("hides 1M detail lines when the display preference is disabled", () => {
    const row = {
      id: 5,
      input_tokens: 0,
      output_tokens: 0,
      cache_creation_tokens: 0,
      cache_creation_5m_tokens: 0,
      cache_creation_1h_tokens: 0,
      cache_read_tokens: 0,
      image_count: 0,
      thinking_enabled: true,
    };
    const wrapper = mountUsageTable(row, {
      showMillionContextLines: false,
      formatUsageMillionContextLines: () => [
        {
          key: "requested",
          labelKey: "usage.millionContextRequested",
          raw: "true",
          display: "Yes",
        },
      ],
    });

    expect(wrapper.get('[data-test="reasoning-cell"]').text()).not.toContain("1M Requested");
  });

  it("keeps failed status compact and copies full failure detail from the tooltip trigger", async () => {
    const wrapper = mountUsageTable({
      id: 6,
      request_id: "req-user-failed",
      status: "failed",
      simulated_client: "codex",
      http_status: 503,
      error_code: "upstream_unavailable",
      error_message: "upstream error: 503 Service temporarily unavailable",
      input_tokens: 0,
      output_tokens: 0,
      cache_creation_tokens: 0,
      cache_creation_5m_tokens: 0,
      cache_creation_1h_tokens: 0,
      cache_read_tokens: 0,
      image_count: 0,
      thinking_enabled: false,
    }, {
      getStatusLabel: () => "Failed",
    });

    const statusCell = wrapper.get('[data-test="status-cell"]');
    expect(statusCell.text()).toContain("Failed");
    expect(statusCell.text()).toContain("Codex");
    expect(statusCell.text()).not.toContain("HTTP Status");
    expect(statusCell.text()).not.toContain("upstream_unavailable");
    expect(statusCell.text()).not.toContain("upstream error: 503");

    const trigger = statusCell.get(".error-info-trigger");
    await trigger.trigger("mouseenter");

    expect(wrapper.text()).toContain("http_status: 503");
    expect(wrapper.text()).toContain("error_code: upstream_unavailable");
    expect(wrapper.text()).toContain("error_message: upstream error: 503 Service temporarily unavailable");

    await trigger.trigger("click");

    expect(clipboardState.copyToClipboard).toHaveBeenCalledWith(
      [
        "request_id: req-user-failed",
        "http_status: 503",
        "error_code: upstream_unavailable",
        "error_message: upstream error: 503 Service temporarily unavailable",
      ].join("\n"),
    );
  });
});
