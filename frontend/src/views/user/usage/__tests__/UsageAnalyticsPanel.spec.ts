import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";

import UsageAnalyticsPanel from "../UsageAnalyticsPanel.vue";

describe("UsageAnalyticsPanel", () => {
  it("passes user analytics data to charts without enabling admin breakdowns", () => {
    const wrapper = mount(UsageAnalyticsPanel, {
      props: {
        trendData: [{ date: "2026-03-01", total_tokens: 120 }],
        modelStats: [{ model: "gpt-5.4", total_tokens: 120 }],
        groupStats: [{ group: "default", total_tokens: 120 }],
        endpointStats: [
          { endpoint: "/v1/messages", requests: 2, total_tokens: 120 },
        ],
        upstreamEndpointStats: [
          { endpoint: "/v1/chat/completions", requests: 2, total_tokens: 120 },
        ],
        trendLoading: true,
        modelLoading: false,
        groupLoading: true,
        endpointLoading: false,
        startDate: "2026-03-01",
        endDate: "2026-03-03",
      },
      global: {
        stubs: {
          ModelDistributionChart: {
            props: [
              "metric",
              "modelStats",
              "loading",
              "showMetricToggle",
              "enableBreakdown",
            ],
            template: `
              <div
                data-testid="model-chart"
                :data-count="modelStats.length"
                :data-enable-breakdown="String(enableBreakdown)"
                :data-loading="String(loading)"
                :data-show-toggle="String(showMetricToggle)"
              />
            `,
          },
          GroupDistributionChart: {
            props: [
              "metric",
              "groupStats",
              "loading",
              "showMetricToggle",
              "enableBreakdown",
              "startDate",
              "endDate",
            ],
            template: `
              <div
                data-testid="group-chart"
                :data-count="groupStats.length"
                :data-enable-breakdown="String(enableBreakdown)"
                :data-loading="String(loading)"
                :data-range="startDate + ':' + endDate"
              />
            `,
          },
          EndpointDistributionChart: {
            props: [
              "metric",
              "source",
              "endpointStats",
              "upstreamEndpointStats",
              "loading",
              "showMetricToggle",
              "showSourceToggle",
              "enableBreakdown",
              "startDate",
              "endDate",
            ],
            template: `
              <div
                data-testid="endpoint-chart"
                :data-count="endpointStats.length"
                :data-upstream-count="upstreamEndpointStats.length"
                :data-enable-breakdown="String(enableBreakdown)"
                :data-show-source-toggle="String(showSourceToggle)"
                :data-range="startDate + ':' + endDate"
              />
            `,
          },
          TokenUsageTrend: {
            props: ["trendData", "loading"],
            template: `
              <div
                data-testid="token-trend"
                :data-count="trendData.length"
                :data-loading="String(loading)"
              />
            `,
          },
        },
      },
    });

    expect(wrapper.get('[data-testid="model-chart"]').attributes()).toMatchObject({
      "data-count": "1",
      "data-enable-breakdown": "false",
      "data-loading": "false",
      "data-show-toggle": "true",
    });
    expect(wrapper.get('[data-testid="group-chart"]').attributes()).toMatchObject({
      "data-count": "1",
      "data-enable-breakdown": "false",
      "data-loading": "true",
      "data-range": "2026-03-01:2026-03-03",
    });
    expect(wrapper.get('[data-testid="endpoint-chart"]').attributes()).toMatchObject({
      "data-count": "1",
      "data-upstream-count": "1",
      "data-enable-breakdown": "false",
      "data-show-source-toggle": "true",
      "data-range": "2026-03-01:2026-03-03",
    });
    expect(wrapper.get('[data-testid="token-trend"]').attributes()).toMatchObject({
      "data-count": "1",
      "data-loading": "true",
    });
  });
});
