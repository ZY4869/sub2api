import { mount } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";

import FailedRequestsPanel from "../FailedRequestsPanel.vue";

const messages: Record<string, string> = {
  "common.collapse": "Collapse",
  "common.expand": "Expand",
  "common.loading": "Loading...",
  "usage.failedRequests.title": "Recent Failed Requests",
  "usage.failedRequests.description": "Gateway failures for the selected filters",
  "usage.failedRequests.refresh": "Refresh failed requests",
  "usage.failedRequests.empty": "No failed requests found for the selected filters.",
  "usage.failedRequests.failedToLoad": "Failed to load failed requests. Retry or adjust the filters.",
  "usage.failedRequests.searchPlaceholder": "Search request_id / error message",
  "usage.errors.allCategories": "All categories",
  "usage.errors.allStatuses": "All status codes",
  "usage.errors.categories.auth": "Auth",
  "usage.errors.categories.rate_limit": "Rate limit",
  "usage.errors.categories.quota": "Quota",
  "usage.errors.categories.invalid_request": "Invalid request",
  "usage.errors.categories.service_unavailable": "Service unavailable",
  "usage.errors.categories.upstream": "Upstream",
  "usage.errors.categories.internal": "Internal",
  "usage.errors.categories.cyber": "Security policy",
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

const mountPanel = () =>
  mount(FailedRequestsPanel, {
    props: {
      rows: [],
      loading: false,
      error: false,
      filters: {
        q: "",
        category: "",
        statusCode: null,
      },
      hiddenColumns: new Set(["user_agent"]),
      columns: [
        { key: "created_at", label: "Time" },
        { key: "status", label: "Status" },
      ],
      alwaysVisibleColumns: ["created_at", "status"],
      sortBy: "created_at",
      sortOrder: "desc",
    },
    global: {
      stubs: {
        Icon: {
          props: ["name"],
          template: '<span :data-icon="name" />',
        },
        Select: {
          props: ["modelValue", "options"],
          template: '<select><option v-for="option in options" :key="String(option.value)">{{ option.label }}</option></select>',
        },
        UsageColumnSettingsMenu: {
          template: '<button data-testid="column-settings">Columns</button>',
        },
        FailedRequestsTable: {
          template: '<div data-testid="failed-requests-table" />',
        },
      },
    },
  });

describe("FailedRequestsPanel", () => {
  it("collapses filters and content while keeping header actions visible", async () => {
    const wrapper = mountPanel();

    expect(wrapper.text()).toContain("Recent Failed Requests");
    expect(wrapper.get("input").attributes("placeholder")).toBe("Search request_id / error message");
    expect(wrapper.text()).toContain("No failed requests found");

    const toggle = wrapper.get('[data-testid="failed-requests-collapse-toggle"]');
    expect(toggle.attributes("aria-expanded")).toBe("true");

    await toggle.trigger("click");

    expect(toggle.attributes("aria-expanded")).toBe("false");
    expect(wrapper.text()).toContain("Recent Failed Requests");
    expect(wrapper.find('[data-testid="column-settings"]').exists()).toBe(true);
    expect(wrapper.find("input").exists()).toBe(false);
    expect(wrapper.text()).not.toContain("No failed requests found");
  });
});
