import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";

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
    subscription_type: "none",
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

const mountEditor = (imageOnly: boolean) =>
  mount(APIKeyGroupBindingsEditor, {
    props: {
      modelValue,
      groups,
      groupModelOptions,
      groupModelCatalogItems,
      imageOnly,
    },
    global: {
      stubs: {
        ModelIcon: { template: "<span />" },
      },
    },
  });

describe("APIKeyGroupBindingsEditor", () => {
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
});
