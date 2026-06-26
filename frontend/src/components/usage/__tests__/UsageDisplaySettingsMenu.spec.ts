import { describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";

import UsageColumnSettingsMenu from "../UsageColumnSettingsMenu.vue";
import UsageDisplaySettingsMenu from "../UsageDisplaySettingsMenu.vue";

const messages: Record<string, string> = {
  "usage.displaySettings": "Display settings",
  "usage.displaySettingsAppearance": "Appearance",
  "usage.displaySettingsColumns": "Columns",
  "usage.tokenDisplay": "Token display",
  "usage.tokenDisplayNatural": "Natural",
  "usage.tokenDisplayK": "K",
  "usage.tokenDisplayM": "M",
  "usage.modelDisplay": "Model display",
  "usage.tableDensity": "Density",
  "usage.tableDensityComfortable": "Comfortable",
  "usage.tableDensityCompact": "Compact density",
  "usage.statsCardStyle": "Stats cards",
  "usage.statsCardStyleBalanced": "Balanced",
  "usage.statsCardStyleAccent": "Accent",
  "usage.showMillionContextLines": "1M details",
  "usage.userAgentDisplay": "User-Agent",
  "usage.userAgentDisplayCompact": "Compact UA",
  "usage.userAgentDisplayFull": "Full UA",
  "usage.columnSettings": "Columns",
};

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  };
});

const IconStub = {
  props: ["name"],
  template: '<span :data-icon="name"><slot /></span>',
};

const UsageModelDisplayModeToggleStub = {
  props: ["modelValue", "disabled", "labelText"],
  emits: ["update:modelValue"],
  template: `
    <button
      type="button"
      data-test="model-toggle"
      :disabled="disabled"
      @click="$emit('update:modelValue', 'display_and_model')"
    >
      {{ labelText }}|{{ modelValue }}
    </button>
  `,
};

function mountMenu(overrides: Record<string, unknown> = {}) {
  return mount(UsageDisplaySettingsMenu, {
    props: {
      preferences: {
        hidden_columns: ["user_agent"],
        token_display_mode: "m",
        table_density: "comfortable",
        stats_card_style: "balanced",
        show_million_context_lines: true,
        user_agent_display_mode: "compact",
      },
      usageModelDisplayMode: "model_only",
      updatingUsageModelDisplayMode: false,
      ...overrides,
    },
    global: {
      stubs: {
        Icon: IconStub,
        UsageModelDisplayModeToggle: UsageModelDisplayModeToggleStub,
      },
    },
    attachTo: document.body,
  });
}

describe("UsageDisplaySettingsMenu", () => {
  it("renders display controls and emits preference updates", async () => {
    const wrapper = mountMenu();

    await wrapper.get("button").trigger("click");

    expect(wrapper.text()).toContain("Display settings");
    expect(wrapper.text()).toContain("Token display");
    expect(wrapper.text()).toContain("Model display|model_only");
    expect(wrapper.text()).toContain("Density");
    expect(wrapper.text()).toContain("Stats cards");
    expect(wrapper.text()).toContain("1M details");
    expect(wrapper.text()).toContain("User-Agent");
    expect(wrapper.text()).toContain("Compact UA");
    expect(wrapper.text()).not.toContain("Cache hit");
    expect(wrapper.text()).not.toContain("User agent");

    await wrapper.get('[data-test="model-toggle"]').trigger("click");
    expect(wrapper.emitted("update-usage-model-display-mode")?.[0]).toEqual([
      "display_and_model",
    ]);

    const buttons = wrapper.findAll("button");
    await buttons.find((button) => button.text() === "K")!.trigger("click");
    await buttons.find((button) => button.text() === "Compact density")!.trigger("click");
    await buttons.find((button) => button.text() === "Accent")!.trigger("click");
    await buttons.find((button) => button.text() === "Full UA")!.trigger("click");
    await buttons
      .find((button) => button.attributes("aria-pressed") === "true")!
      .trigger("click");

    expect(wrapper.emitted("update-preference")).toContainEqual([
      "token_display_mode",
      "k",
    ]);
    expect(wrapper.emitted("update-preference")).toContainEqual([
      "table_density",
      "compact",
    ]);
    expect(wrapper.emitted("update-preference")).toContainEqual([
      "stats_card_style",
      "accent",
    ]);
    expect(wrapper.emitted("update-preference")).toContainEqual([
      "user_agent_display_mode",
      "full",
    ]);
    expect(wrapper.emitted("update-preference")).toContainEqual([
      "show_million_context_lines",
      false,
    ]);
    expect(wrapper.emitted("toggle-column")).toBeUndefined();
  });

  it("disables the trigger and nested model toggle while saving", async () => {
    const wrapper = mountMenu({ disabled: true });

    expect(wrapper.get("button").attributes("disabled")).toBeDefined();

    await wrapper.setProps({ disabled: false, updatingUsageModelDisplayMode: true });
    await wrapper.get("button").trigger("click");

    expect(wrapper.get('[data-test="model-toggle"]').attributes("disabled")).toBeDefined();
  });

  it("opens the separate column settings menu and emits column toggles", async () => {
    const wrapper = mount(UsageColumnSettingsMenu, {
      props: {
        hiddenColumns: new Set(["user_agent"]),
        columns: [
          { key: "created_at", label: "Time" },
          { key: "tokens", label: "Tokens" },
          { key: "cache_hit", label: "Cache hit" },
          { key: "user_agent", label: "User agent" },
        ],
        alwaysVisibleColumns: ["created_at"],
      },
      global: {
        stubs: {
          Icon: IconStub,
        },
      },
      attachTo: document.body,
    });

    await wrapper.get("button").trigger("click");

    expect(wrapper.text()).toContain("Columns");
    expect(wrapper.text()).toContain("Cache hit");
    expect(wrapper.text()).toContain("User agent");
    expect(wrapper.text()).not.toContain("Time");

    await wrapper
      .findAll("button")
      .find((button) => button.text().includes("Cache hit"))!
      .trigger("click");

    expect(wrapper.emitted("toggle-column")?.[0]).toEqual(["cache_hit"]);
  });
});
