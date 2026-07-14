import { describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";

import AccountOpenAIResetCreditsControls from "../AccountOpenAIResetCreditsControls.vue";

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === "admin.accounts.usageWindow.resetCreditExpiresAt") {
          return `Expires ${params?.time}`;
        }
        return key;
      },
    }),
  };
});

function mountControls(overrides = {}) {
  return mount(AccountOpenAIResetCreditsControls, {
    props: {
      resetStatusLabel: "3 resets left",
      resetUnsupported: false,
      resetCreditsLow: false,
      resetting: false,
      refreshing: false,
      resetDisabled: false,
      ...overrides,
    },
    global: {
      stubs: {
        Icon: true,
      },
    },
  });
}

describe("AccountOpenAIResetCreditsControls", () => {
  it("keeps refresh, reset, and remaining count in one nowrap row", () => {
    const wrapper = mountControls();
    const row = wrapper.get('[data-testid="account-usage-reset-credits-row"]');

    expect(row.classes()).toEqual(
      expect.arrayContaining([
        "inline-flex",
        "flex-nowrap",
        "whitespace-nowrap",
        "w-max",
        "max-w-full",
      ]),
    );

    const refresh = wrapper.get('[data-testid="account-usage-reset-credits-refresh"]');
    const reset = wrapper.get('[data-testid="account-usage-reset-quota-button"]');
    const remaining = wrapper.get('[data-testid="account-usage-reset-quota-remaining"]');

    expect(refresh.element.parentElement).toBe(row.element);
    expect(reset.element.parentElement).toBe(row.element);
    expect(remaining.element.parentElement).toBe(row.element);
    expect(refresh.classes()).toContain("shrink-0");
    expect(reset.classes()).toContain("shrink-0");
    expect(remaining.classes()).toContain("shrink-0");
  });

  it("renders refresh as an accessible icon-only button and warns at one remaining credit", () => {
    const wrapper = mountControls({
      resetStatusLabel: "1 resets left",
      resetCreditsLow: true,
    });

    const refresh = wrapper.get('[data-testid="account-usage-reset-credits-refresh"]');
    const remaining = wrapper.get('[data-testid="account-usage-reset-quota-remaining"]');

    expect(refresh.text()).toBe("");
    expect(refresh.attributes("aria-label")).toBe(
      "admin.accounts.usageWindow.refreshResetCreditsTitle",
    );
    expect(remaining.classes()).toContain("bg-amber-50");
    expect(remaining.classes()).toContain("text-amber-700");
  });

  it("shows expiry +N and emits toggle details", async () => {
    const wrapper = mountControls({
      earliestExpiryText: "2026-08-01 00:00",
      expiryExtraCount: 2,
      detailRows: [
        {
          id: "credit-1",
          status: "granted",
          expiresAt: "2026-08-01T00:00:00Z",
          expiresAtText: "2026-08-01 00:00",
        },
        {
          id: "credit-2",
          status: "granted",
          expiresAt: "2026-08-02T00:00:00Z",
          expiresAtText: "2026-08-02 00:00",
        },
      ],
      detailsExpanded: true,
    });

    const expiry = wrapper.get('[data-testid="account-usage-reset-quota-expiry"]');
    expect(expiry.text()).toContain("Expires 2026-08-01 00:00");
    expect(wrapper.get('[data-testid="account-usage-reset-quota-expiry-extra"]').text()).toBe("+2");
    expect(wrapper.get('[data-testid="account-usage-reset-quota-expiry-details"]').text()).toContain("2026-08-02 00:00");

    await expiry.trigger("click");
    expect(wrapper.emitted("toggle-details")).toHaveLength(1);
  });

  it("keeps unsupported, unknown, zero, and loading states disabled or styled", () => {
    const unsupported = mountControls({
      resetUnsupported: true,
      resetStatusLabel: "unsupported",
      resetDisabled: true,
    });
    expect(unsupported.get('[data-testid="account-usage-reset-quota-remaining"]').classes()).toContain("bg-gray-50");
    expect(unsupported.get('[data-testid="account-usage-reset-quota-button"]').attributes("disabled")).toBeDefined();

    const unknown = mountControls({
      resetUnknown: true,
      resetStatusLabel: "-- resets left",
    });
    expect(unknown.get('[data-testid="account-usage-reset-quota-remaining"]').classes()).toContain("bg-gray-50");

    const zero = mountControls({
      resetZero: true,
      resetStatusLabel: "0 resets left",
    });
    expect(zero.get('[data-testid="account-usage-reset-quota-remaining"]').classes()).toContain("bg-rose-50");

    const loading = mountControls({
      refreshing: true,
    });
    expect(loading.get('[data-testid="account-usage-reset-credits-refresh"]').attributes("disabled")).toBeDefined();
  });
});
