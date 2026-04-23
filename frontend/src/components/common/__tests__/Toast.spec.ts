import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  };
});

import Toast from "../Toast.vue";
import { useAppStore } from "@/stores/app";

describe("Toast", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
  });

  it("renders legacy string details via the app store normalization", async () => {
    const appStore = useAppStore();
    appStore.showSuccess("done", {
      details: ["legacy one", "legacy two"],
      persistent: true,
    });

    const wrapper = mount(Toast, {
      global: {
        stubs: {
          Icon: true,
          Teleport: true,
        },
      },
    });

    await wrapper.vm.$nextTick();

    const items = wrapper.findAll("li");
    expect(items).toHaveLength(2);
    expect(items[0].text()).toContain("legacy one");
    expect(items[1].text()).toContain("legacy two");
  });

  it("renders structured detail tones", async () => {
    const appStore = useAppStore();
    appStore.showWarning("refresh", {
      details: [
        { text: "live", tone: "success" },
        { text: "fallback", tone: "warning" },
        { text: "failed", tone: "error" },
      ],
      persistent: true,
    });

    const wrapper = mount(Toast, {
      global: {
        stubs: {
          Icon: true,
          Teleport: true,
        },
      },
    });

    await wrapper.vm.$nextTick();

    const items = wrapper.findAll("li");
    expect(items).toHaveLength(3);
    expect(items[0].classes().join(" ")).toContain("text-green-700");
    expect(items[1].classes().join(" ")).toContain("text-amber-700");
    expect(items[2].classes().join(" ")).toContain("text-red-700");
  });
});
