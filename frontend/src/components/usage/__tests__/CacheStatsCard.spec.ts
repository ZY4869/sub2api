import { describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";

import CacheStatsCard from "../CacheStatsCard.vue";

const messages: Record<string, string> = {
  "usage.cacheHitRate": "命中率",
  "usage.cacheSplit": "写入 {write} / 读取 {read}",
};

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        let message = messages[key] ?? key;
        Object.entries(params || {}).forEach(([name, value]) => {
          message = message.replace(`{${name}}`, String(value));
        });
        return message;
      },
    }),
  };
});

describe("CacheStatsCard", () => {
  const mountCard = (props?: Partial<InstanceType<typeof CacheStatsCard>["$props"]>) =>
    mount(CacheStatsCard, {
      props,
      global: {
        stubs: {
          Icon: true,
        },
      },
    });

  it("normalizes fractional cache hit rates", () => {
    const wrapper = mountCard({
      cacheHitRate: 0.896,
      cacheCreationTokens: 12500,
      cacheReadTokens: 34000,
    });

    expect(wrapper.text()).toContain("命中率");
    expect(wrapper.text()).toContain("89.6%");
    expect(wrapper.text()).toContain("写入 12,500 / 读取 34,000");
  });

  it("keeps whole-number cache hit rates as percentages", () => {
    const wrapper = mountCard({
      cacheHitRate: 75,
      cacheCreationTokens: 200,
      cacheReadTokens: 700,
    });

    expect(wrapper.text()).toContain("75.0%");
    expect(wrapper.text()).toContain("写入 200 / 读取 700");
  });

  it("falls back to zero when cache fields are missing", () => {
    const wrapper = mountCard({
      cacheHitRate: null,
      cacheCreationTokens: undefined,
      cacheReadTokens: null,
    });

    expect(wrapper.text()).toContain("0.0%");
    expect(wrapper.text()).toContain("写入 0 / 读取 0");
  });
});
