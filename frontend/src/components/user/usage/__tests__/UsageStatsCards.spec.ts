import { describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";

import UsageStatsCards from "../UsageStatsCards.vue";

const messages: Record<string, string> = {
  "usage.totalRequests": "总请求数",
  "usage.totalTokens": "总 Token",
  "usage.totalCost": "总消费",
  "usage.inSelectedRange": "所选范围内",
  "usage.in": "输入",
  "usage.out": "输出",
  "usage.avgDuration": "平均耗时",
  "usage.actualCost": "实际",
  "usage.standardCost": "标准",
  "usage.todayStats": "今日统计",
  "usage.todaySoFar": "从今日 00:00 到当前",
  "usage.todayRequests": "今日请求",
  "usage.todayTokens": "今日 Token",
  "usage.todayCost": "今日消费",
  "usage.todayAvgDuration": "今日平均耗时",
  "usage.cacheTokens": "缓存",
  "usage.cacheSplit": "写入 {write} / 读取 {read}",
  "usage.cacheHitRate": "命中率",
  "usage.perRequest": "每次请求",
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

const stats = {
  total_requests: 8,
  total_input_tokens: 120,
  total_output_tokens: 240,
  total_cache_creation_tokens: 12500,
  total_cache_read_tokens: 34000,
  total_cache_tokens: 46500,
  total_tokens: 396,
  cache_hit_rate: 0.896,
  total_cost: 1.2,
  total_actual_cost: 1.1,
  admin_free_requests: 0,
  admin_free_standard_cost: 0,
  average_duration_ms: 150,
  today_requests: 3,
  today_input_tokens: 30,
  today_output_tokens: 60,
  today_cache_creation_tokens: 200,
  today_cache_read_tokens: 700,
  today_cache_tokens: 900,
  today_tokens: 99,
  today_cache_hit_rate: 75,
  today_cost: 0.45,
  today_actual_cost: 0.4,
  today_average_duration_ms: 120,
};

describe("user UsageStatsCards", () => {
  it("renders cache hit rate as a standalone selected-range card", () => {
    const wrapper = mount(UsageStatsCards, {
      props: {
        stats,
      },
      global: {
        stubs: {
          Icon: true,
          PlatformIcon: true,
        },
      },
    });

    const cacheCard = wrapper.get('[data-testid="usage-cache-stats-card"]');
    expect(cacheCard.text()).toContain("命中率");
    expect(cacheCard.text()).toContain("89.6%");
    expect(cacheCard.text()).toContain("写入 12,500 / 读取 34,000");
  });

  it("keeps today usage metrics visible", () => {
    const wrapper = mount(UsageStatsCards, {
      props: {
        stats,
      },
      global: {
        stubs: {
          Icon: true,
          PlatformIcon: true,
        },
      },
    });

    const text = wrapper.text();
    expect(text).toContain("今日统计");
    expect(text).toContain("今日请求");
    expect(text).toContain("今日 Token");
    expect(text).toContain("今日消费");
    expect(text).toContain("今日平均耗时");
    expect(text).toContain("75.0%");
  });
});
