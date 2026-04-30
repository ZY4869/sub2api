import { defineComponent } from "vue";
import { mount, flushPromises } from "@vue/test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import RequestDetailsTraceTab from "../RequestDetailsTraceTab.vue";

const {
  replace,
  copyToClipboard,
  showError,
  showSuccess,
  routeState,
  listRequestTraces,
  getRequestTraceSummary,
  getRequestTraceDetail,
  getRequestTraceRawDetail,
  exportRequestTracesCSV,
  cleanupRequestTraces,
} = vi.hoisted(() => ({
  replace: vi.fn(),
  copyToClipboard: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  routeState: {
    query: { tab: "trace", account_id: "12" } as Record<string, string>,
  },
  listRequestTraces: vi.fn(),
  getRequestTraceSummary: vi.fn(),
  getRequestTraceDetail: vi.fn(),
  getRequestTraceRawDetail: vi.fn(),
  exportRequestTracesCSV: vi.fn(),
  cleanupRequestTraces: vi.fn(),
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
    cleanupRequestTraces,
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
    showSuccess,
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
    loading: {
      type: Boolean,
      default: false,
    },
    rawExportAllowed: {
      type: Boolean,
      default: false,
    },
    cleanupLoading: {
      type: Boolean,
      default: false,
    },
  },
  emits: ["update:modelValue", "apply", "reset", "export", "cleanup-filter", "cleanup-expired"],
  template: `
    <div>
      <button
        data-test="apply-filter"
        type="button"
        @click="$emit('update:modelValue', { ...modelValue, request_id: 'req-123' })"
      >
        apply filter
      </button>
      <button data-test="cleanup-filter" type="button" @click="$emit('cleanup-filter')">cleanup filter</button>
      <button data-test="cleanup-expired" type="button" @click="$emit('cleanup-expired')">cleanup expired</button>
    </div>
  `,
});

const RequestDetailsTableStub = defineComponent({
  props: {
    items: {
      type: Array,
      default: () => [],
    },
  },
  emits: ["select"],
  template: `
    <button
      data-test="select-first"
      type="button"
      @click="$emit('select', items[0])"
    >
      select first
    </button>
  `,
});

const RequestDetailsDrawerStub = defineComponent({
  props: {
    open: {
      type: Boolean,
      default: false,
    },
  },
  template: `<div data-test="drawer" :data-open="open ? '1' : '0'"></div>`,
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
        RequestDetailsTable: RequestDetailsTableStub,
        RequestDetailsDrawer: RequestDetailsDrawerStub,
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
    showSuccess.mockReset();
    routeState.query = { tab: "trace", account_id: "12" };
    listRequestTraces.mockReset();
    getRequestTraceSummary.mockReset();
    getRequestTraceDetail.mockReset();
    getRequestTraceRawDetail.mockReset();
    exportRequestTracesCSV.mockReset();
    cleanupRequestTraces.mockReset();

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
    cleanupRequestTraces.mockResolvedValue({
      mode: "filter",
      deleted_traces: 0,
      deleted_audits: 0,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
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

  it("runs cleanup with the current filter", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    cleanupRequestTraces.mockResolvedValueOnce({
      mode: "filter",
      deleted_traces: 2,
      deleted_audits: 3,
    });

    const wrapper = mountTraceTab();
    await flushPromises();

    await wrapper.get('[data-test="cleanup-filter"]').trigger("click");
    await flushPromises();

    expect(cleanupRequestTraces).toHaveBeenCalledWith(
      expect.objectContaining({
        mode: "filter",
        filter: expect.objectContaining({
          account_id: 12,
        }),
      })
    );

    const payload = cleanupRequestTraces.mock.calls[0]?.[0] as any;
    expect(payload.filter).not.toHaveProperty("page");
    expect(payload.filter).not.toHaveProperty("page_size");
    expect(payload.filter).not.toHaveProperty("sort");

    expect(showSuccess).toHaveBeenCalled();
  });

  it("runs expired cleanup without filter", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    cleanupRequestTraces.mockResolvedValueOnce({
      mode: "expired",
      deleted_traces: 5,
      deleted_audits: 6,
      cutoff: "2026-04-01T00:00:00Z",
    });

    const wrapper = mountTraceTab();
    await flushPromises();

    await wrapper.get('[data-test="cleanup-expired"]').trigger("click");
    await flushPromises();

    expect(cleanupRequestTraces).toHaveBeenCalledWith({ mode: "expired" });
    expect(showSuccess).toHaveBeenCalled();
  });

  it("closes the drawer when the selected trace is deleted by cleanup", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);

    listRequestTraces.mockResolvedValueOnce({
      items: [{ id: 1 }],
      total: 1,
    });
    getRequestTraceDetail
      .mockResolvedValueOnce({ id: 1 })
      .mockRejectedValueOnce({ response: { status: 404 } });

    cleanupRequestTraces.mockResolvedValueOnce({
      mode: "filter",
      deleted_traces: 1,
      deleted_audits: 1,
    });

    const wrapper = mountTraceTab();
    await flushPromises();

    // Select a trace to open the drawer (watcher triggers detail fetch).
    await wrapper.get('[data-test="select-first"]').trigger("click");
    await flushPromises();

    expect(wrapper.get('[data-test="drawer"]').attributes("data-open")).toBe("1");

    // Cleanup deletes the selected trace -> detail reload returns 404 -> drawer closes.
    await wrapper.get('[data-test="cleanup-filter"]').trigger("click");
    await flushPromises();

    expect(wrapper.get('[data-test="drawer"]').attributes("data-open")).toBe("0");
  });
});
