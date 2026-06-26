import { describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";

import CacheStatsCard from "../CacheStatsCard.vue";

const messages: Record<string, string> = {
  "usage.cacheHitRate": "命中率",
  "usage.cacheWrite": "写入",
  "usage.cacheRead": "读取",
  "common.total": "总计",
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
    expect(wrapper.text()).toContain("写入");
    expect(wrapper.text()).toContain("12.5K");
    expect(wrapper.text()).toContain("读取");
    expect(wrapper.text()).toContain("34K");
    expect(wrapper.text()).toContain("46.5K");
  });

  it("keeps whole-number cache hit rates as percentages", () => {
    const wrapper = mountCard({
      cacheHitRate: 75,
      cacheCreationTokens: 200,
      cacheReadTokens: 700,
    });

    expect(wrapper.text()).toContain("75.0%");
    expect(wrapper.text()).toContain("200");
    expect(wrapper.text()).toContain("700");
    expect(wrapper.text()).toContain("900");
  });

  it("falls back to zero when cache fields are missing", () => {
    const wrapper = mountCard({
      cacheHitRate: null,
      cacheCreationTokens: undefined,
      cacheReadTokens: null,
    });

    expect(wrapper.text()).toContain("0.0%");
    expect(wrapper.text()).toContain("写入");
    expect(wrapper.text()).toContain("读取");
  });
});
