import { defineComponent } from "vue";
import { mount, flushPromises } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import RequestDetailsTraceTab from "../RequestDetailsTraceTab.vue";

const {
  replace,
  copyToClipboard,
  showError,
  routeState,
  listRequestTraces,
  getRequestTraceSummary,
  getRequestTraceDetail,
  getRequestTraceRawDetail,
  exportRequestTracesCSV,
} = vi.hoisted(() => ({
  replace: vi.fn(),
  copyToClipboard: vi.fn(),
  showError: vi.fn(),
  routeState: {
    query: { tab: "trace", account_id: "12" } as Record<string, string>,
  },
  listRequestTraces: vi.fn(),
  getRequestTraceSummary: vi.fn(),
  getRequestTraceDetail: vi.fn(),
  getRequestTraceRawDetail: vi.fn(),
  exportRequestTracesCSV: vi.fn(),
}));

vi.mock("vue-router", () => ({
  useRoute: () => routeState,
  useRouter: () => ({
    replace,
  }),
}));

vi.mock("@/api/admin/ops", () => ({
  opsAPI: {
    listRequestTraces,
    getRequestTraceSummary,
    getRequestTraceDetail,
    getRequestTraceRawDetail,
    exportRequestTracesCSV,
  },
}));

vi.mock("@/composables/useClipboard", () => ({
  useClipboard: () => ({
    copyToClipboard,
  }),
}));

vi.mock("@/stores", () => ({
  useAppStore: () => ({
    showError,
  }),
}));

vi.mock("@/stores/modelRegistry", () => ({
  getModelRegistrySnapshot: () => ({
    etag: "test",
    updated_at: "2026-04-04T00:00:00Z",
    models: [],
    presets: [],
  }),
}));

vi.mock("@/utils/requestPreview", () => ({
  parseRequestPreviewContent: (value: string) => value,
}));

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  };
});

const RequestDetailsFilterPanelStub = defineComponent({
  props: {
    modelValue: {
      type: Object,
      required: true,
    },
  },
  emits: ["update:modelValue", "apply", "reset", "export"],
  template: `
    <button
      data-test="apply-filter"
      type="button"
      @click="$emit('update:modelValue', { ...modelValue, request_id: 'req-123' })"
    >
      apply filter
    </button>
  `,
});

function mountTraceTab(props?: { routeTabValue?: string }) {
  return mount(RequestDetailsTraceTab, {
    props,
    global: {
      stubs: {
        RequestDetailsFilterPanel: RequestDetailsFilterPanelStub,
        RequestDetailsSummaryCards: true,
        RequestDetailsTrendChart: true,
        RequestDetailsBreakdownChart: true,
        RequestDetailsTable: true,
        RequestDetailsDrawer: true,
      },
    },
  });
}

describe("RequestDetailsTraceTab", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    replace.mockReset();
    copyToClipboard.mockReset();
    showError.mockReset();
    routeState.query = { tab: "trace", account_id: "12" };
    listRequestTraces.mockReset();
    getRequestTraceSummary.mockReset();
    getRequestTraceDetail.mockReset();
    getRequestTraceRawDetail.mockReset();
    exportRequestTracesCSV.mockReset();

    listRequestTraces.mockResolvedValue({
      items: [],
      total: 0,
    });
    getRequestTraceSummary.mockResolvedValue({
      raw_access_allowed: false,
      trend: [],
      status_distribution: [],
      protocol_pair_distribution: [],
      finish_reason_distribution: [],
      model_distribution: [],
      capability_distribution: [],
      totals: {
        request_count: 0,
      },
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("keeps the default trace tab value when syncing route filters", async () => {
    const wrapper = mountTraceTab();

    await flushPromises();
    await wrapper.get('[data-test="apply-filter"]').trigger("click");
    vi.advanceTimersByTime(300);
    await flushPromises();

    expect(replace).toHaveBeenLastCalledWith({
      query: expect.objectContaining({
        tab: "trace",
        account_id: "12",
        request_id: "req-123",
      }),
    });
  });

  it("uses the provided route tab value when embedded in the admin usage view", async () => {
    const wrapper = mountTraceTab({ routeTabValue: "request_details" });

    await flushPromises();
    await wrapper.get('[data-test="apply-filter"]').trigger("click");
    vi.advanceTimersByTime(300);
    await flushPromises();

    expect(replace).toHaveBeenLastCalledWith({
      query: expect.objectContaining({
        tab: "request_details",
        account_id: "12",
        request_id: "req-123",
      }),
    });
  });
});
