import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";
import ModelDebugView from "../ModelDebugView.vue";
import { resetPublicModelCatalogStoreForTests } from "@/stores/publicModelCatalog";

const localeMessages = vi.hoisted<Record<string, string>>(() => ({
  "admin.models.pages.debug.requestTitle": "调试请求",
  "admin.models.pages.debug.requestDescription": "先用常用表单生成基础请求体，再用高级 JSON 覆盖或补充字段。",
  "admin.models.pages.debug.cancel": "取消调试",
  "admin.models.pages.debug.run": "开始调试",
  "admin.models.pages.debug.keyModeLabel": "密钥模式",
  "admin.models.pages.debug.keyModes.saved": "已保存密钥",
  "admin.models.pages.debug.keyModes.manual": "手动输入密钥",
  "admin.models.pages.debug.endpointLabel": "调试接口",
  "admin.models.pages.debug.savedKeyLabel": "已保存密钥",
  "admin.models.pages.debug.savedKeyPlaceholder": "请选择当前管理员名下的密钥",
  "admin.models.pages.debug.manualKeyLabel": "临时 API 密钥",
  "admin.models.pages.debug.manualKeyPlaceholder": "仅本次调试内存使用，不会保存",
  "admin.models.pages.debug.modelLabel": "调试模型",
  "admin.models.pages.debug.modelPlaceholder": "请选择模型",
  "admin.models.pages.debug.systemPromptLabel": "系统指令",
  "admin.models.pages.debug.systemPromptPlaceholder": "可选，用来覆盖系统提示词或角色设定",
  "admin.models.pages.debug.userPromptLabel": "用户输入",
  "admin.models.pages.debug.userPromptPlaceholder": "填写要发送给模型的用户消息",
  "admin.models.pages.debug.temperatureLabel": "采样温度",
  "admin.models.pages.debug.temperaturePlaceholder": "例如 0.2",
  "admin.models.pages.debug.maxTokensLabel": "最大输出 Token",
  "admin.models.pages.debug.maxTokensPlaceholder": "例如 256",
  "admin.models.pages.debug.reasoningLabel": "推理强度",
  "admin.models.pages.debug.reasoningPlaceholder": "例如 low / medium / high（低 / 中 / 高）",
  "admin.models.pages.debug.streamLabel": "使用流式返回",
  "admin.models.pages.debug.advancedJsonLabel": "高级 JSON 覆盖",
  "admin.models.pages.debug.advancedJsonPlaceholder": "输入 JSON 对象，对上方生成的请求体做覆盖或补充。",
  "admin.models.pages.debug.requestPreviewLabel": "最终请求体预览",
  "admin.models.pages.debug.outputTitle": "调试输出",
  "admin.models.pages.debug.outputIdle": "运行后会按事件流记录开始、请求预览、响应头与最终结果。",
  "admin.models.pages.debug.outputRunning": "正在等待上游返回，输出区会持续追加服务端推送事件（SSE）。",
  "admin.models.pages.debug.outputEmpty": "还没有调试输出，点击右上角开始调试。",
  "admin.models.pages.debug.running": "运行中",
  "admin.models.pages.debug.ready": "待命",
  "admin.models.pages.debug.defaults.systemPrompt": "你是一名简洁的诊断助手。",
  "admin.models.pages.debug.defaults.userPrompt": "请返回一条简短确认消息，并带上当前使用的模型名称。",
  "admin.models.pages.debug.endpointNames.responses": "响应接口",
  "admin.models.pages.debug.endpointNames.chatCompletions": "聊天补全",
  "admin.models.pages.debug.endpointNames.messages": "消息接口",
  "admin.models.pages.debug.endpointNames.generateContent": "内容生成",
  "admin.models.pages.debug.advancedJsonInvalidError": "高级 JSON 解析失败，请检查格式。",
  "admin.models.pages.debug.advancedJsonObjectError": "高级 JSON 必须是对象。",
  "admin.models.pages.debug.contextLoadFailed": "加载调试所需的密钥或模型分组失败",
  "admin.models.pages.debug.runFailed": "模型调试失败",
  "admin.models.pages.debug.cancelled": "调试已取消",
  "admin.models.pages.debug.protocolHints.openai": "响应接口（Responses）与聊天补全接口（Chat Completions）",
  "admin.models.pages.debug.protocolHints.anthropic": "消息接口（Messages）JSON 端点",
  "admin.models.pages.debug.protocolHints.gemini": "内容生成接口（Generate Content）与 SSE 变体",
  "admin.models.pages.debug.events.start": "调试开始",
  "admin.models.pages.debug.events.request": "请求预览",
  "admin.models.pages.debug.events.headers": "响应头",
  "admin.models.pages.debug.events.content": "内容分片",
  "admin.models.pages.debug.events.final": "最终结果",
  "admin.models.pages.debug.events.error": "错误事件",
}));

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
      t: (key: string) => localeMessages[key] ?? key,
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
        input: [
          {
            role: "system",
            content: "你是一名简洁的诊断助手。",
          },
          {
            role: "user",
            content: "请返回一条简短确认消息，并带上当前使用的模型名称。",
          },
        ],
        temperature: 0.6,
        metadata: {
          source: "spec",
        },
      },
    });
    expect(wrapper.text()).toContain("debug-ok");
    expect(wrapper.text()).toContain("最终结果");
  });

  it("renders chinese endpoint names and localized fallback prompt text", async () => {
    const wrapper = mount(ModelDebugView, {
      global: {
        plugins: [createPinia()],
      },
    });

    await flushPromises();

    const endpointSelect = wrapper.get('[data-testid="debug-endpoint-select"]');
    expect(endpointSelect.text()).toContain("响应接口");
    expect(endpointSelect.text()).toContain("聊天补全");

    const userPrompt = wrapper.get('[data-testid="debug-user-prompt"]').find("textarea");
    expect((userPrompt.element as HTMLTextAreaElement).value).toBe(
      "请返回一条简短确认消息，并带上当前使用的模型名称。",
    );

    await userPrompt.setValue("");
    await wrapper.get('[data-testid="debug-run"]').trigger("click");
    await flushPromises();

    expect(debugMocks.runModelDebugStream.mock.calls.at(-1)?.[0]).toMatchObject({
      request_body: {
        input: [
          {
            role: "system",
            content: "你是一名简洁的诊断助手。",
          },
          {
            role: "user",
            content: "请返回一条简短确认消息，并带上当前使用的模型名称。",
          },
        ],
      },
    });
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
    expect(wrapper.text()).toContain("高级 JSON 解析失败，请检查格式。");
    await wrapper.get('[data-testid="debug-run"]').trigger("click");
    expect(debugMocks.runModelDebugStream).not.toHaveBeenCalled();

    await textarea.setValue("[]");
    await flushPromises();
    expect(wrapper.get('[data-testid="debug-run"]').attributes("disabled")).toBeDefined();
    expect(wrapper.text()).toContain("高级 JSON 必须是对象。");
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
    expect(wrapper.text()).toContain("调试已取消");
  });
});
