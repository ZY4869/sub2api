import { flushPromises, mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from "vitest";
import PublicModelCatalogContent from "../PublicModelCatalogContent.vue";
import { resetPublicModelCatalogStoreForTests } from '@/stores/publicModelCatalog'

const apiMocks = vi.hoisted(() => ({
  getModelCatalog: vi.fn(),
  getUSDCNYExchangeRate: vi.fn(),
}));

const appStoreMocks = vi.hoisted(() => ({
  showSuccess: vi.fn(),
  showError: vi.fn(),
}));

const clipboardMocks = vi.hoisted(() => ({
  writeText: vi.fn(),
}));

vi.mock("@/api/meta", () => ({
  getModelCatalog: apiMocks.getModelCatalog,
  getUSDCNYExchangeRate: apiMocks.getUSDCNYExchangeRate,
}));

vi.mock("@/stores/app", () => ({
  useAppStore: () => ({
    showSuccess: appStoreMocks.showSuccess,
    showError: appStoreMocks.showError,
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

describe("PublicModelCatalogContent", () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    resetPublicModelCatalogStoreForTests()
    apiMocks.getModelCatalog.mockReset();
    apiMocks.getUSDCNYExchangeRate.mockReset();
    appStoreMocks.showSuccess.mockReset();
    appStoreMocks.showError.mockReset();
    clipboardMocks.writeText.mockReset();
    localStorage.clear();

    Object.defineProperty(window.navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: clipboardMocks.writeText,
      },
    });

    apiMocks.getModelCatalog.mockResolvedValue({
      notModified: false,
      etag: 'W/"catalog"',
      data: {
        etag: 'W/"catalog"',
        updated_at: "2026-04-18T00:00:00Z",
        page_size: 10,
        catalog_source: 'published',
        items: [
          {
            model: "gpt-5.4",
            display_name: "GPT-5.4",
            provider: "openai",
            provider_icon_key: "openai",
            status: "ok",
            request_protocols: ["openai"],
            source_ids: ["source-alpha"],
            mode: "chat",
            currency: "USD",
            price_display: {
              primary: [{ id: "input_price", unit: "input_token", value: 0.000001 }],
            },
            multiplier_summary: {
              enabled: true,
              kind: "uniform",
              mode: "shared",
              value: 0.12,
            },
          },
          {
            model: "claude-sonnet-4.5",
            display_name: "Claude Sonnet 4.5",
            provider: "anthropic",
            provider_icon_key: "anthropic",
            status: "warning",
            request_protocols: ["anthropic"],
            source_ids: ["claude-source-id"],
            mode: "chat",
            currency: "USD",
            price_display: {
              primary: [{ id: "input_price", unit: "input_token", value: 0.000002 }],
            },
            multiplier_summary: {
              enabled: false,
              kind: "disabled",
            },
          },
          {
            model: "gpt-5.4-compat",
            display_name: "GPT 5.4 Compatible",
            provider: "openai",
            provider_icon_key: "openai",
            status: "maintenance",
            request_protocols: ["gemini"],
            source_ids: ["compat-source-id"],
            mode: "chat",
            currency: "USD",
            price_display: {
              primary: [{ id: "output_price", unit: "output_token", value: 0.000004 }],
              secondary: [{ id: "cache_price", unit: "input_token", value: 0.000001 }],
            },
            multiplier_summary: {
              enabled: true,
              kind: "mixed",
            },
          },
        ],
      },
    });
    apiMocks.getUSDCNYExchangeRate.mockResolvedValue({
      base: "USD",
      quote: "CNY",
      rate: 7.2,
      date: "2026-04-18",
      updated_at: "2026-04-18T00:00:00Z",
      cached: true,
    });
  });

  it("supports card filters, search, view persistence, copy, and detail actions", async () => {
    localStorage.setItem("public-model-catalog:view-mode", "list");
    const pinia = createPinia()

    const wrapper = mount(PublicModelCatalogContent, {
      global: {
        plugins: [pinia],
        stubs: {
          PublicModelCatalogDetailDialog: {
            props: ["show", "model", "catalogItem"],
            template: '<div v-if="show" data-testid="detail-dialog">{{ model }}</div>',
          },
          ModelIcon: { template: '<span data-testid="model-icon" />' },
          ModelPlatformIcon: { template: '<span data-testid="provider-icon" />' },
        },
      },
    });

    await flushPromises();

    expect(wrapper.get('[data-testid="public-model-results"]').attributes("data-view-mode")).toBe("list");
    expect(wrapper.text()).toContain("GPT 5.4");
    expect(wrapper.text()).toContain("Claude Sonnet 4.5");
    expect(wrapper.find('[data-testid="public-model-primary-price-gpt-5.4-compat-cache_price"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="public-model-secondary-price-gpt-5.4-compat-cache_price"]').exists()).toBe(false);
    expect(wrapper.text()).toContain("ui.modelCatalog.status.maintenance");
    expect(wrapper.findAll('[aria-label="ui.modelCatalog.status.maintenance"]').length).toBeGreaterThan(0);
    expect(wrapper.find('[data-testid="models-filter-provider-gemini"]').exists()).toBe(false);

    await wrapper.get('[data-testid="models-filter-provider-openai"]').trigger("click");
    expect(wrapper.text()).toContain("GPT 5.4");
    expect(wrapper.text()).toContain("GPT 5.4 Compatible");
    expect(wrapper.text()).not.toContain("Claude Sonnet 4.5");
    expect(wrapper.get('[data-testid="public-model-card-gpt-5.4"]').text()).not.toContain("gpt-5.4");

    await wrapper.get('[data-testid="models-filter-provider-all"]').trigger("click");
    await wrapper.get('[data-testid="models-filter-protocol-gemini"]').trigger("click");
    expect(wrapper.text()).toContain("GPT 5.4 Compatible");
    expect(wrapper.text()).not.toContain("Claude Sonnet 4.5");
    expect(wrapper.text()).not.toContain("GPT-5.4");

    await wrapper.get('[data-testid="public-models-search"]').setValue("compat-source-id");
    expect(wrapper.text()).toContain("GPT 5.4 Compatible");

    await wrapper.get('[data-testid="public-model-copy-gpt-5.4-compat"]').trigger("click");
    expect(clipboardMocks.writeText).toHaveBeenCalledWith("gpt-5.4-compat");
    expect(appStoreMocks.showSuccess).toHaveBeenCalledWith("ui.modelCatalog.copySuccess");
    expect(wrapper.find('[data-testid="detail-dialog"]').exists()).toBe(false);

    await wrapper.get('[data-testid="public-model-detail-gpt-5.4-compat"]').trigger("click");
    expect(wrapper.get('[data-testid="detail-dialog"]').text()).toContain("gpt-5.4-compat");

    await wrapper.get('[data-testid="public-models-search"]').setValue("");
    await wrapper.get('[data-testid="models-filter-protocol-all"]').trigger("click");
    await wrapper.get('[data-testid="models-filter-multiplier-disabled"]').trigger("click");
    expect(wrapper.text()).toContain("Claude Sonnet 4.5");
    expect(wrapper.text()).not.toContain("GPT 5.4 Compatible");
    expect(wrapper.text()).not.toContain("GPT-5.4");

    await wrapper.get('[data-testid="public-models-view-grid"]').trigger("click");
    expect(wrapper.get('[data-testid="public-model-results"]').attributes("data-view-mode")).toBe("grid");
    expect(localStorage.getItem("public-model-catalog:view-mode")).toBe("grid");
  });

  it("does not auto-refresh on focus or visibility changes", async () => {
    const pinia = createPinia()
    mount(PublicModelCatalogContent, {
      global: {
        plugins: [pinia],
        stubs: {
          PublicModelCatalogDetailDialog: true,
          ModelIcon: true,
          ModelPlatformIcon: true,
        },
      },
    });

    await flushPromises();
    expect(apiMocks.getModelCatalog).toHaveBeenCalledTimes(1);

    window.dispatchEvent(new Event("focus"));
    document.dispatchEvent(new Event("visibilitychange"));
    await flushPromises();

    expect(apiMocks.getModelCatalog).toHaveBeenCalledTimes(1);
  });

  it("paginates with the published page size", async () => {
    apiMocks.getModelCatalog.mockResolvedValueOnce({
      notModified: false,
      etag: 'W/"paged"',
      data: {
        etag: 'W/"paged"',
        updated_at: "2026-04-18T00:00:00Z",
        page_size: 2,
        items: [
          {
            model: "model-a",
            display_name: "Model A",
            provider: "openai",
            provider_icon_key: "openai",
            request_protocols: ["openai"],
            currency: "USD",
            price_display: {
              primary: [{ id: "input_price", unit: "input_token", value: 0.000001 }],
            },
            multiplier_summary: {
              enabled: false,
              kind: "disabled",
            },
          },
          {
            model: "model-b",
            display_name: "Model B",
            provider: "openai",
            provider_icon_key: "openai",
            request_protocols: ["openai"],
            currency: "USD",
            price_display: {
              primary: [{ id: "input_price", unit: "input_token", value: 0.000001 }],
            },
            multiplier_summary: {
              enabled: false,
              kind: "disabled",
            },
          },
          {
            model: "model-c",
            display_name: "Model C",
            provider: "openai",
            provider_icon_key: "openai",
            request_protocols: ["openai"],
            currency: "USD",
            price_display: {
              primary: [{ id: "input_price", unit: "input_token", value: 0.000001 }],
            },
            multiplier_summary: {
              enabled: false,
              kind: "disabled",
            },
          },
        ],
      },
    });

    const pinia = createPinia()
    const wrapper = mount(PublicModelCatalogContent, {
      global: {
        plugins: [pinia],
        stubs: {
          PublicModelCatalogDetailDialog: true,
          ModelIcon: true,
          ModelPlatformIcon: true,
        },
      },
    });

    await flushPromises();

    expect(wrapper.text()).toContain("Model A");
    expect(wrapper.text()).toContain("Model B");
    expect(wrapper.text()).not.toContain("Model C");

    await wrapper.get('[data-testid="public-models-page-next"]').trigger("click");

    expect(wrapper.text()).toContain("Model C");
    expect(wrapper.text()).not.toContain("Model A");
  });

  it("restores a cached snapshot and shows a soft stale notice when revalidation fails", async () => {
    localStorage.setItem(
      "public-model-catalog:snapshot",
      JSON.stringify({
        snapshot: {
          etag: 'W/"cached"',
          updated_at: "2026-04-17T00:00:00Z",
          items: [
            {
              model: "cached-model",
              display_name: "Cached Model",
              provider: "openai",
              provider_icon_key: "openai",
              request_protocols: ["openai"],
              currency: "USD",
              price_display: {
                primary: [{ id: "input_price", unit: "input_token", value: 0.000001 }],
              },
              multiplier_summary: {
                enabled: false,
                kind: "disabled",
              },
            },
          ],
        },
        etag: 'W/"cached"',
        loadedAt: 0,
        usdToCnyRate: null,
        exchangeRateLoadedAt: 0,
      }),
    );
    apiMocks.getModelCatalog.mockRejectedValueOnce(new Error("Network error. Please check your connection."));
    const pinia = createPinia()

    const wrapper = mount(PublicModelCatalogContent, {
      global: {
        plugins: [pinia],
        stubs: {
          PublicModelCatalogDetailDialog: true,
          ModelIcon: { template: '<span data-testid="model-icon" />' },
          ModelPlatformIcon: { template: '<span data-testid="provider-icon" />' },
        },
      },
    });

    await flushPromises();

    expect(wrapper.text()).toContain("Cached Model");
    expect(wrapper.text()).toContain("ui.modelCatalog.staleNotice");
    expect(wrapper.text()).not.toContain("Network error. Please check your connection.");
  });

  it("keeps catalog content visible when exchange-rate loading fails", async () => {
    apiMocks.getModelCatalog.mockResolvedValueOnce({
      notModified: false,
      etag: 'W/"catalog-cny"',
      data: {
        etag: 'W/"catalog-cny"',
        updated_at: "2026-04-18T00:00:00Z",
        catalog_source: 'published',
        items: [
          {
            model: "gemini-2.5-pro",
            display_name: "Gemini 2.5 Pro",
            provider: "gemini",
            provider_icon_key: "gemini",
            request_protocols: ["gemini"],
            currency: "CNY",
            price_display: {
              primary: [{ id: "output_price", unit: "output_token", value: 0.000004 }],
            },
            multiplier_summary: {
              enabled: false,
              kind: "disabled",
            },
          },
        ],
      },
    });
    apiMocks.getUSDCNYExchangeRate.mockRejectedValueOnce(new Error("Network error. Please check your connection."));
    const pinia = createPinia()

    const wrapper = mount(PublicModelCatalogContent, {
      global: {
        plugins: [pinia],
        stubs: {
          PublicModelCatalogDetailDialog: true,
          ModelIcon: { template: '<span data-testid="model-icon" />' },
          ModelPlatformIcon: { template: '<span data-testid="provider-icon" />' },
        },
      },
    });

    await flushPromises();

    expect(wrapper.text()).toContain("Gemini 2.5 Pro");
    expect(wrapper.text()).toContain("ui.modelCatalog.exchangeRateSoftWarning");
    expect(wrapper.text()).not.toContain("Network error. Please check your connection.");
  });

  it("shows a live fallback notice and empty-state copy when the catalog is unpublished", async () => {
    apiMocks.getModelCatalog.mockResolvedValueOnce({
      notModified: false,
      etag: 'W/"live-fallback"',
      data: {
        etag: 'W/"live-fallback"',
        updated_at: "2026-04-18T00:00:00Z",
        page_size: 10,
        catalog_source: 'live_fallback',
        items: [],
      },
    });

    const pinia = createPinia()
    const wrapper = mount(PublicModelCatalogContent, {
      global: {
        plugins: [pinia],
        stubs: {
          PublicModelCatalogDetailDialog: true,
          ModelIcon: true,
          ModelPlatformIcon: true,
        },
      },
    });

    await flushPromises();

    expect(wrapper.text()).toContain('ui.modelCatalog.liveFallbackNotice')
    expect(wrapper.text()).toContain('ui.modelCatalog.emptyLiveFallback')
  })
});
