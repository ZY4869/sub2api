import { mount } from "@vue/test-utils";
import type { VueWrapper } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";

import APIKeyGroupBindingsEditor from "../APIKeyGroupBindingsEditor.vue";
import type { EditableApiKeyGroupBinding } from "../apiKeyGroupBindings";

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params?.count !== undefined ? `${key}:${params.count}` : key,
    }),
  };
});

const groups = [
  {
    id: 1,
    name: "OpenAI",
    description: "",
    platform: "openai",
    priority: 1,
    rate_multiplier: 1,
    subscription_type: "standard",
  },
  {
    id: 2,
    name: "Gemini Backup",
    description: "",
    platform: "gemini",
    priority: 2,
    rate_multiplier: 0.055,
    subscription_type: "standard",
  },
];

const modelValue: EditableApiKeyGroupBinding[] = [
  {
    group_id: 1,
    quota: 0,
    model_patterns_text: "",
    selected_models: [],
    model_selection_dirty: false,
  },
];

const groupModelOptions = {
  1: [
    { public_id: "gpt-image-2", display_name: "GPT Image 2" },
    { public_id: "gpt-5.4", display_name: "GPT 5.4" },
    { public_id: "custom-image", display_name: "Custom Image", request_protocols: ["images"] },
  ],
};

const groupModelCatalogItems = {
  1: [
    { model: "gpt-image-2", display_name: "GPT Image 2", provider: "openai", mode: "image" },
    { model: "gpt-5.4", display_name: "GPT 5.4", provider: "openai", mode: "chat" },
  ],
};

const mountedWrappers: VueWrapper[] = [];

const mountEditor = (
  imageOnly: boolean,
  modelSelectionRequired = false,
  overrides: Partial<InstanceType<typeof APIKeyGroupBindingsEditor>["$props"]> = {},
) => {
  const wrapper = mount(APIKeyGroupBindingsEditor, {
    props: {
      modelValue,
      groups,
      groupModelOptions,
      groupModelCatalogItems,
      imageOnly,
      modelSelectionRequired,
      ...overrides,
    },
    attachTo: document.body,
    global: {
      stubs: {
        ModelIcon: { template: "<span />" },
      },
    },
  });
  mountedWrappers.push(wrapper);
  return wrapper;
};

describe("APIKeyGroupBindingsEditor", () => {
  afterEach(() => {
    for (const wrapper of mountedWrappers.splice(0)) {
      wrapper.unmount();
    }
    document.body.innerHTML = "";
  });

  it("renders searchable group options as icon-name-rate pills without platform text or priority", async () => {
    const wrapper = mountEditor(false, false, {
      modelValue: [{ ...modelValue[0], group_id: 0 }],
      userGroupRates: { 2: 0.05 },
    });

    await wrapper.find(".select-trigger").trigger("click");

    expect(document.body.textContent).toContain("OpenAI");
    expect(document.body.textContent).toContain("1x");
    expect(document.body.textContent).toContain("Gemini Backup");
    expect(document.body.textContent).toContain("0.05x");
    expect(document.body.textContent).not.toContain("openai");
    expect(document.body.textContent).not.toContain("P1");
  });

  it("emits the existing group_id payload when selecting a group", async () => {
    const wrapper = mountEditor(false, false, {
      modelValue: [{ ...modelValue[0], group_id: 0 }],
    });

    await wrapper.find(".select-trigger").trigger("click");
    const options = Array.from(document.body.querySelectorAll('[role="option"]'));
    const geminiOption = options.find((item) => item.textContent?.includes("Gemini Backup"));
    expect(geminiOption).toBeTruthy();
    geminiOption?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    await wrapper.vm.$nextTick();

    const emitted = wrapper.emitted("update:modelValue")?.[0]?.[0] as EditableApiKeyGroupBinding[];
    expect(emitted[0].group_id).toBe(2);
    expect(emitted[0].selected_models).toEqual([]);
    expect(emitted[0].model_selection_dirty).toBe(true);
  });

  it("disables groups already selected in another binding row", async () => {
    const wrapper = mountEditor(false, false, {
      modelValue: [
        { ...modelValue[0], group_id: 1 },
        { ...modelValue[0], group_id: 0 },
      ],
    });

    await wrapper.findAll(".select-trigger")[1].trigger("click");
    const options = Array.from(document.body.querySelectorAll('[role="option"]'));
    const openAiOption = options.find((item) => item.textContent?.includes("OpenAI"));

    expect(openAiOption?.getAttribute("aria-disabled")).toBe("true");
    openAiOption?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
    await wrapper.vm.$nextTick();

    expect(wrapper.emitted("update:modelValue")).toBeUndefined();
  });

  it("filters selectable models to image models when image-only is enabled", () => {
    const wrapper = mountEditor(true);

    expect(wrapper.text()).toContain("GPT Image 2");
    expect(wrapper.text()).toContain("Custom Image");
    expect(wrapper.text()).not.toContain("GPT 5.4");
    expect(wrapper.findAll('input[type="checkbox"]')).toHaveLength(2);
  });

  it("emits only image model selections in image-only mode", async () => {
    const wrapper = mountEditor(true);

    await wrapper.find('input[type="checkbox"]').setValue(true);

    const emitted = wrapper.emitted("update:modelValue")?.[0]?.[0] as EditableApiKeyGroupBinding[];
    expect(emitted[0].selected_models).toEqual(["gpt-image-2"]);
    expect(emitted[0].model_selection_dirty).toBe(true);
  });

  it("hides whole-group shortcut and shows required hint when model selection is required", () => {
    const wrapper = mountEditor(false, true);

    expect(wrapper.text()).not.toContain("keys.modelScopeAll");
    expect(wrapper.text()).toContain("keys.modelScopeRequiredHint");
    expect(wrapper.text()).toContain("keys.modelSelectionRequired");
  });
});
