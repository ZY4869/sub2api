import { flushPromises, mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import PublicModelCatalogDetailDialog from "../PublicModelCatalogDetailDialog.vue";

const apiMocks = vi.hoisted(() => ({
  getModelCatalogDetail: vi.fn(),
  keysList: vi.fn(),
  getModelOptions: vi.fn(),
  authStore: {
    isAuthenticated: false,
  },
  appStore: {
    apiBaseUrl: "https://api.example.com",
  },
}));

vi.mock("@/api/meta", () => ({
  getModelCatalogDetail: apiMocks.getModelCatalogDetail,
}));

vi.mock("@/api/keys", () => ({
  default: {
    list: apiMocks.keysList,
  },
}));

vi.mock("@/api/groups", () => ({
  default: {
    getModelOptions: apiMocks.getModelOptions,
  },
}));

vi.mock("@/stores/auth", () => ({
  useAuthStore: () => apiMocks.authStore,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: () => apiMocks.appStore,
}));

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params?.protocol
          ? `${key}:${String(params.protocol)}`
          : params?.name
            ? `${key}:${String(params.name)}`
            : key,
    }),
  };
});

describe("PublicModelCatalogDetailDialog", () => {
  beforeEach(() => {
    document.body.innerHTML = "";
    apiMocks.authStore.isAuthenticated = false;
    apiMocks.getModelCatalogDetail.mockReset();
    apiMocks.keysList.mockReset();
    apiMocks.getModelOptions.mockReset();
    apiMocks.getModelCatalogDetail.mockResolvedValue({
      item: {
        model: "text-embedding-3-large",
        display_name: "Text Embedding 3 Large",
        provider: "openai",
        provider_icon_key: "openai",
        status: "warning",
        context_window_tokens: 8191,
        modalities: ["text"],
        capabilities: ["text"],
        request_protocols: ["openai"],
        currency: "USD",
        price_display: {
          primary: [{ id: "input_price", unit: "input_token", value: 0.00000013 }],
          secondary: [{ id: "cache_price", unit: "input_token", value: 0.00000005 }],
        },
        multiplier_summary: {
          enabled: false,
          kind: "disabled",
        },
      },
      example_source: "override_template",
      example_protocol: "openai",
      example_override_id: "embeddings",
    });
    apiMocks.keysList.mockResolvedValue({ items: [] });
    apiMocks.getModelOptions.mockResolvedValue([]);
  });

  function mountDialog() {
    return mount(PublicModelCatalogDetailDialog, {
      attachTo: document.body,
      props: {
        show: true,
        model: "text-embedding-3-large",
        health: {
          model: "text-embedding-3-large",
          status: "healthy",
          success_rate_today: 1,
          success_rate_7d: 0.998,
          latency_ms: 120,
          daily: [{ date: "2026-04-18", status: "healthy", success_rate: 1, latency_ms: 120 }],
          trend: [{ timestamp: "2026-04-18", success_rate: 1, latency_ms: 120 }],
          rate_limit: { rpm: 60 },
        },
        catalogItem: {
          model: "text-embedding-3-large",
          display_name: "Text Embedding 3 Large",
          provider: "openai",
          provider_icon_key: "openai",
          status: "warning",
          context_window_tokens: 8191,
          modalities: ["text"],
          capabilities: ["text"],
          request_protocols: ["openai"],
          currency: "USD",
          price_display: {
            primary: [{ id: "input_price", unit: "input_token", value: 0.00000013 }],
            secondary: [{ id: "cache_price", unit: "input_token", value: 0.00000005 }],
          },
          multiplier_summary: {
            enabled: false,
            kind: "disabled",
          },
        },
      },
      global: {
        stubs: {
          ModelIcon: { template: '<span data-test="model-icon" />' },
          ModelPlatformIcon: { template: '<span data-test="platform-icon" />' },
          DocsCodeTabs: {
            props: ["group"],
            template: '<pre data-test="example-code">{{ group.tabs[0]?.code }}</pre>',
          },
        },
      },
    });
  }

  it("keeps the placeholder key for guests", async () => {
    const wrapper = mountDialog();

    await flushPromises();

    expect(apiMocks.getModelCatalogDetail).toHaveBeenCalledWith("text-embedding-3-large");
    expect(document.body.textContent).toContain("120ms");
    expect(document.body.textContent).toContain("99.8%");
    expect(document.body.querySelector('[data-testid="detail-primary-price-cache_price"]')).toBeTruthy();
    await document.body.querySelector<HTMLElement>('[data-testid="public-model-detail-tab-routing"]')?.click();
    await flushPromises();
    expect(document.body.querySelector("[data-test='example-code']")?.textContent).toContain("sk-your-key");
    wrapper.unmount();
  });

  it("renders overview, monitor, and routing tabs", async () => {
    const wrapper = mountDialog();

    await flushPromises();

    expect(document.body.textContent).toContain("text-embedding-3-large");
    expect(document.body.textContent).toContain("RPM 60");

    await document.body.querySelector<HTMLElement>('[data-testid="public-model-detail-tab-monitor"]')?.click();
    await flushPromises();
    expect(document.body.textContent).toContain("ui.modelCatalog.detail.dailyMatrix");
    expect(document.body.textContent).toContain("2026-04-18");

    await document.body.querySelector<HTMLElement>('[data-testid="public-model-detail-tab-routing"]')?.click();
    await flushPromises();
    expect(document.body.textContent).toContain("Authorization: Bearer <TOKEN>");
    expect(document.body.textContent).toContain("sk-your-key");
    wrapper.unmount();
  });

  it("injects the first matching user key and shows a switcher for multiple matches", async () => {
    apiMocks.authStore.isAuthenticated = true;
    apiMocks.keysList.mockResolvedValue({
      items: [
        {
          id: 3,
          key: "sk-user-b",
          name: "Beta Key",
          group_id: 10,
          api_key_groups: [{ group_id: 10, group_name: "OpenAI", platform: "openai", priority: 1, quota: 0, quota_used: 0, model_patterns: [] }],
        },
        {
          id: 2,
          key: "sk-user-a",
          name: "Alpha Key",
          group_id: 10,
          api_key_groups: [{ group_id: 10, group_name: "OpenAI", platform: "openai", priority: 1, quota: 0, quota_used: 0, model_patterns: [] }],
        },
      ],
    });
    apiMocks.getModelOptions.mockResolvedValue([
      {
        group_id: 10,
        name: "OpenAI",
        platform: "openai",
        priority: 1,
        model_count: 1,
        models: [
          {
            public_id: "text-embedding-3-large",
            display_name: "Text Embedding 3 Large",
            request_protocols: ["openai"],
          },
        ],
      },
    ]);

    const wrapper = mountDialog();

    await flushPromises();
    await document.body.querySelector<HTMLElement>('[data-testid="public-model-detail-tab-routing"]')?.click();
    await flushPromises();

    expect(document.body.querySelector("[data-test='example-code']")?.textContent).toContain("sk-user-a");
    const options = document.body.querySelectorAll("select option");
    expect(options).toHaveLength(2);
    wrapper.unmount();
  });
});
