import { describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";

import UsageTable from "../UsageTable.vue";

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
        <slot name="cell-thinking_enabled" :row="row" />
      </div>
    </div>
  `,
};

const IconStub = {
  props: ["name"],
  template: '<span :data-icon="name"><slot /></span>',
};

function mountUsageTable(row: Record<string, unknown>) {
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
      formatUsageEndpoints: () => [],
      getRequestTypeBadgeClass: () => "",
      getRequestTypeLabel: () => "Sync",
      getChargeLabel: () => null,
      getChargeBadgeClass: () => "",
    },
    global: {
      stubs: {
        DataTable: DataTableStub,
        EmptyState: true,
        UsageModelCell: true,
        UsageSuccessRateCell: true,
        UsageProtocolCell: true,
        Icon: IconStub,
      },
    },
  });
}

describe("user usage UsageTable", () => {
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
});
