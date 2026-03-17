import { describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";

import AccountModelScopeWhitelistEditor from "../AccountModelScopeWhitelistEditor.vue";

const mockModels = [
  {
    id: "claude-sonnet-legacy",
    display_name: "Claude Sonnet Legacy",
    provider: "anthropic",
    platforms: ["anthropic"],
    protocol_ids: [],
    aliases: [],
    pricing_lookup_ids: [],
    modalities: ["text"],
    capabilities: ["text"],
    ui_priority: 1,
    exposed_in: ["runtime"],
    status: "deprecated",
  },
  {
    id: "claude-sonnet-beta",
    display_name: "Claude Sonnet Beta",
    provider: "anthropic",
    platforms: ["anthropic"],
    protocol_ids: [],
    aliases: [],
    pricing_lookup_ids: [],
    modalities: ["text"],
    capabilities: ["text"],
    ui_priority: 2,
    exposed_in: ["runtime"],
    status: "beta",
  },
];

vi.mock("@/stores/modelRegistry", () => ({
  getModelRegistrySnapshot: () => ({ models: mockModels }),
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

describe("AccountModelScopeWhitelistEditor", () => {
  it("hides deprecated badges while keeping other statuses visible", () => {
    const wrapper = mount(AccountModelScopeWhitelistEditor, {
      props: {
        platform: "anthropic",
        allowedModels: [],
      },
      global: {
        stubs: {
          ModelIcon: { template: "<span />" },
        },
      },
    });

    expect(wrapper.text()).toContain("Claude Sonnet Legacy");
    expect(wrapper.text()).toContain("Claude Sonnet Beta");
    expect(wrapper.text()).toContain("beta");
    expect(wrapper.text().toLowerCase()).not.toContain("deprecated");
  });
});
