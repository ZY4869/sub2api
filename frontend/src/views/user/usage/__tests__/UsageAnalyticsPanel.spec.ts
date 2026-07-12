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
        showUsageDistributionPanels: true,
        showEndpointDistributionPanel: true,
        showTokenUsageTrend: true,
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
              "collapsible",
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
              "collapsible",
              "startDate",
              "endDate",
            ],
            template: `
              <div
                data-testid="endpoint-chart"
                :data-count="endpointStats.length"
                :data-upstream-count="upstreamEndpointStats.length"
                :data-enable-breakdown="String(enableBreakdown)"
                :data-collapsible="String(collapsible)"
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
      "data-collapsible": "true",
      "data-show-source-toggle": "true",
      "data-range": "2026-03-01:2026-03-03",
    });
    expect(wrapper.get('[data-testid="token-trend"]').attributes()).toMatchObject({
      "data-count": "1",
      "data-loading": "true",
    });
  });

  it("hides model and group distribution charts when disabled", () => {
    const wrapper = mount(UsageAnalyticsPanel, {
      props: {
        trendData: [],
        modelStats: [],
        groupStats: [],
        endpointStats: [],
        upstreamEndpointStats: [],
        showUsageDistributionPanels: false,
        showEndpointDistributionPanel: true,
        showTokenUsageTrend: true,
        startDate: "2026-03-01",
        endDate: "2026-03-03",
      },
      global: {
        stubs: {
          ModelDistributionChart: {
            template: '<div data-testid="model-chart" />',
          },
          GroupDistributionChart: {
            template: '<div data-testid="group-chart" />',
          },
          EndpointDistributionChart: {
            template: '<div data-testid="endpoint-chart" />',
          },
          TokenUsageTrend: {
            template: '<div data-testid="token-trend" />',
          },
        },
      },
    });

    expect(wrapper.find('[data-testid="model-chart"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="group-chart"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="endpoint-chart"]').exists()).toBe(true);
    expect(wrapper.find('[data-testid="token-trend"]').exists()).toBe(true);
  });

  it("hides endpoint distribution while keeping token trend visible", () => {
    const wrapper = mount(UsageAnalyticsPanel, {
      props: {
        trendData: [],
        modelStats: [],
        groupStats: [],
        endpointStats: [],
        upstreamEndpointStats: [],
        showUsageDistributionPanels: false,
        showEndpointDistributionPanel: false,
        showTokenUsageTrend: true,
        startDate: "2026-03-01",
        endDate: "2026-03-03",
      },
      global: {
        stubs: {
          ModelDistributionChart: {
            template: '<div data-testid="model-chart" />',
          },
          GroupDistributionChart: {
            template: '<div data-testid="group-chart" />',
          },
          EndpointDistributionChart: {
            template: '<div data-testid="endpoint-chart" />',
          },
          TokenUsageTrend: {
            template: '<div data-testid="token-trend" />',
          },
        },
      },
    });

    expect(wrapper.find('[data-testid="endpoint-chart"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="token-trend"]').exists()).toBe(true);
  });

  it("hides the token trend without leaving an empty analytics grid", () => {
    const wrapper = mount(UsageAnalyticsPanel, {
      props: {
        trendData: [{ date: "2026-03-01", total_tokens: 120 }],
        modelStats: [],
        groupStats: [],
        endpointStats: [],
        upstreamEndpointStats: [],
        showUsageDistributionPanels: false,
        showEndpointDistributionPanel: false,
        showTokenUsageTrend: false,
        startDate: "2026-03-01",
        endDate: "2026-03-03",
      },
      global: {
        stubs: {
          TokenUsageTrend: {
            template: '<div data-testid="token-trend" />',
          },
        },
      },
    });

    expect(wrapper.find('[data-testid="token-trend"]').exists()).toBe(false);
    expect(wrapper.find('[data-testid="usage-endpoint-trend-grid"]').exists()).toBe(false);
  });
});
