import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ModelDebugView from "../ModelDebugView.vue";
import { resetPublicModelCatalogStoreForTests } from "@/stores/publicModelCatalog";

const metaMocks = vi.hoisted(() => ({
  getModelCatalog: vi.fn(),
  getUSDCNYExchangeRate: vi.fn(),
}));

const debugMocks = vi.hoisted(() => ({
  runModelDebugStream: vi.fn(),
}));

const contextMocks = vi.hoisted(() => ({
  keysList: vi.fn(),
  getModelOptions: vi.fn(),
  showError: vi.fn(),
}));

vi.mock("@/api/meta", () => ({
  getModelCatalog: metaMocks.getModelCatalog,
  getUSDCNYExchangeRate: metaMocks.getUSDCNYExchangeRate,
}));

vi.mock("@/api/admin/modelDebug", () => ({
  runModelDebugStream: debugMocks.runModelDebugStream,
}));

vi.mock("@/api/keys", () => ({
  default: {
    list: contextMocks.keysList,
  },
}));

vi.mock("@/api/groups", () => ({
  default: {
    getModelOptions: contextMocks.getModelOptions,
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: () => ({
    showError: contextMocks.showError,
  }),
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

describe("ModelDebugView", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    resetPublicModelCatalogStoreForTests();
    localStorage.clear();
    metaMocks.getModelCatalog.mockReset();
    metaMocks.getUSDCNYExchangeRate.mockReset();
    debugMocks.runModelDebugStream.mockReset();
    contextMocks.keysList.mockReset();
    contextMocks.getModelOptions.mockReset();
    contextMocks.showError.mockReset();

    metaMocks.getModelCatalog.mockResolvedValue({
      notModified: false,
      etag: 'W/"debug-catalog"',
      data: {
        etag: 'W/"debug-catalog"',
        updated_at: "2026-04-29T12:00:00Z",
        items: [
          {
            model: "gpt-5.4",
            display_name: "GPT-5.4",
            provider: "openai",
            provider_icon_key: "openai",
            request_protocols: ["openai"],
            currency: "USD",
            price_display: { primary: [] },
            multiplier_summary: { enabled: false, kind: "disabled" },
          },
          {
            model: "gemini-2.5-pro",
            display_name: "Gemini 2.5 Pro",
            provider: "gemini",
            provider_icon_key: "gemini",
            request_protocols: ["gemini"],
            currency: "USD",
            price_display: { primary: [] },
            multiplier_summary: { enabled: false, kind: "disabled" },
          },
        ],
      },
    });
    metaMocks.getUSDCNYExchangeRate.mockResolvedValue({
      base: "USD",
      quote: "CNY",
      rate: 7.2,
      date: "2026-04-29",
      updated_at: "2026-04-29T12:00:00Z",
      cached: true,
    });
    contextMocks.keysList.mockResolvedValue({
      items: [
        {
          id: 11,
          key: "sk-saved",
          name: "Admin Key",
          user_id: 1,
          group_id: 10,
          status: "active",
          ip_whitelist: [],
          ip_blacklist: [],
          last_used_at: null,
          quota: 0,
          quota_used: 0,
          image_only_enabled: false,
          image_count_billing_enabled: false,
          image_max_count: 0,
          image_count_used: 0,
          image_count_weights: {},
          expires_at: null,
          created_at: "",
          updated_at: "",
          rate_limit_5h: 0,
          rate_limit_1d: 0,
          rate_limit_7d: 0,
          usage_5h: 0,
          usage_1d: 0,
          usage_7d: 0,
          window_5h_start: null,
          window_1d_start: null,
          window_7d_start: null,
          reset_5h_at: null,
          reset_1d_at: null,
          reset_7d_at: null,
          api_key_groups: [
            {
              group_id: 10,
              group_name: "OpenAI",
              platform: "openai",
              priority: 1,
              quota: 0,
              quota_used: 0,
              model_patterns: [],
            },
          ],
        },
      ],
    });
    contextMocks.getModelOptions.mockResolvedValue([
      {
        group_id: 10,
        name: "OpenAI",
        platform: "openai",
        priority: 1,
        model_count: 1,
        models: [
          {
            public_id: "gpt-5.4",
            display_name: "GPT-5.4",
            request_protocols: ["openai"],
            source_ids: ["gpt-5.4"],
          },
        ],
      },
    ]);
  });

  it("filters saved-key models and restores the full catalog in manual mode", async () => {
    const wrapper = mount(ModelDebugView, {
      global: {
        plugins: [createPinia()],
      },
    });

    await flushPromises();

    const select = wrapper.get('[data-testid="debug-model-select"]');
    expect(select.findAll("option")).toHaveLength(2);

    await wrapper.get('[data-testid="debug-key-mode"]').setValue("manual");
    await flushPromises();

    expect(wrapper.get('[data-testid="debug-model-select"]').findAll("option")).toHaveLength(3);
  });

  it("merges advanced JSON into the generated request body and renders stream output", async () => {
    debugMocks.runModelDebugStream.mockImplementation(async (_payload, options) => {
      options.onEvent({ type: "start", debug_run_id: "run-1" });
      options.onEvent({ type: "content", chunk: "debug-ok" });
      options.onEvent({ type: "final", status_code: 200, bytes_received: 24 });
    });

    const wrapper = mount(ModelDebugView, {
      global: {
        plugins: [createPinia()],
      },
    });

    await flushPromises();
    await wrapper.find('[data-testid="debug-advanced-json"]').find("textarea").setValue(
      '{"temperature":0.6,"metadata":{"source":"spec"}}',
    );
    await wrapper.get('[data-testid="debug-run"]').trigger("click");
    await flushPromises();

    expect(debugMocks.runModelDebugStream).toHaveBeenCalledTimes(1);
    expect(debugMocks.runModelDebugStream.mock.calls[0][0]).toMatchObject({
      key_mode: "saved",
      api_key_id: 11,
      model: "gpt-5.4",
      endpoint_kind: "responses",
      request_body: {
        temperature: 0.6,
        metadata: {
          source: "spec",
        },
      },
    });
    expect(wrapper.text()).toContain("debug-ok");
    expect(wrapper.text()).toContain("admin.models.pages.debug.events.final");
  });

  it("normalizes the endpoint when switching to a protocol that does not support the current selection", async () => {
    const wrapper = mount(ModelDebugView, {
      global: {
        plugins: [createPinia()],
      },
    });

    await flushPromises();
    await wrapper.get('[data-testid="debug-endpoint-select"]').setValue("chat_completions");
    await wrapper.get('[data-testid="debug-protocol-anthropic"]').trigger("click");
    await flushPromises();

    expect((wrapper.get('[data-testid="debug-endpoint-select"]').element as HTMLSelectElement).value).toBe("messages");

    await wrapper.get('[data-testid="debug-protocol-gemini"]').trigger("click");
    await flushPromises();

    expect((wrapper.get('[data-testid="debug-endpoint-select"]').element as HTMLSelectElement).value).toBe("generate_content");
  });

  it("blocks runs when advanced JSON is invalid or not an object", async () => {
    const wrapper = mount(ModelDebugView, {
      global: {
        plugins: [createPinia()],
      },
    });

    await flushPromises();
    const textarea = wrapper.find('[data-testid="debug-advanced-json"]').find("textarea");

    await textarea.setValue("{");
    await flushPromises();
    expect(wrapper.get('[data-testid="debug-run"]').attributes("disabled")).toBeDefined();
    expect(wrapper.text()).toContain("admin.models.pages.debug.advancedJsonInvalidError");
    await wrapper.get('[data-testid="debug-run"]').trigger("click");
    expect(debugMocks.runModelDebugStream).not.toHaveBeenCalled();

    await textarea.setValue("[]");
    await flushPromises();
    expect(wrapper.get('[data-testid="debug-run"]').attributes("disabled")).toBeDefined();
    expect(wrapper.text()).toContain("admin.models.pages.debug.advancedJsonObjectError");
    await wrapper.get('[data-testid="debug-run"]').trigger("click");
    expect(debugMocks.runModelDebugStream).not.toHaveBeenCalled();
  });

  it("aborts an in-flight debug run and renders the cancelled state", async () => {
    debugMocks.runModelDebugStream.mockImplementation(
      (_payload, options) =>
        new Promise<void>((_resolve, reject) => {
          options.signal?.addEventListener("abort", () => {
            const abortError = new Error("aborted");
            abortError.name = "AbortError";
            reject(abortError);
          });
        }),
    );

    const wrapper = mount(ModelDebugView, {
      global: {
        plugins: [createPinia()],
      },
    });

    await flushPromises();
    await wrapper.get('[data-testid="debug-run"]').trigger("click");
    await flushPromises();

    expect(debugMocks.runModelDebugStream).toHaveBeenCalledTimes(1);
    expect(wrapper.find('[data-testid="debug-cancel"]').exists()).toBe(true);

    await wrapper.get('[data-testid="debug-cancel"]').trigger("click");
    await flushPromises();

    expect(wrapper.find('[data-testid="debug-cancel"]').exists()).toBe(false);
    expect(wrapper.text()).toContain("admin.models.pages.debug.cancelled");
  });
});
