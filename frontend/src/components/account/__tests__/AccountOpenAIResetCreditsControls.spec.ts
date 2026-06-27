import { describe, expect, it, vi } from "vitest";
import { mount } from "@vue/test-utils";

import AccountOpenAIResetCreditsControls from "../AccountOpenAIResetCreditsControls.vue";

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  };
});

describe("AccountOpenAIResetCreditsControls", () => {
  it("keeps refresh, reset, and remaining count in one nowrap row", () => {
    const wrapper = mount(AccountOpenAIResetCreditsControls, {
      props: {
        resetStatusLabel: "3 resets left",
        resetUnsupported: false,
        resetCreditsLow: false,
        resetting: false,
        refreshing: false,
        resetDisabled: false,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    });

    expect(wrapper.classes()).toEqual(
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

    expect(refresh.element.parentElement).toBe(wrapper.element);
    expect(reset.element.parentElement).toBe(wrapper.element);
    expect(remaining.element.parentElement).toBe(wrapper.element);
    expect(refresh.classes()).toContain("shrink-0");
    expect(reset.classes()).toContain("shrink-0");
    expect(remaining.classes()).toContain("shrink-0");
  });

  it("renders refresh as an accessible icon-only button and warns at one remaining credit", () => {
    const wrapper = mount(AccountOpenAIResetCreditsControls, {
      props: {
        resetStatusLabel: "1 resets left",
        resetUnsupported: false,
        resetCreditsLow: true,
        resetting: false,
        refreshing: false,
        resetDisabled: false,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
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
});
