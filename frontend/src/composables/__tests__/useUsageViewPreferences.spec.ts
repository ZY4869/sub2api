import { computed, defineComponent, nextTick } from "vue";
import { mount } from "@vue/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useUsageViewPreferences } from "../useUsageViewPreferences";
import type { UsageViewPreferences } from "@/types";

const testState = vi.hoisted(() => {
  const state = {
    updateProfile: vi.fn(),
    showError: vi.fn(),
    auth: {
      user: undefined as undefined | { usage_view_preferences?: UsageViewPreferences },
      setUsageViewPreferences: vi.fn((preferences: UsageViewPreferences) => {
        if (!state.auth.user) {
          state.auth.user = {};
        }
        state.auth.user.usage_view_preferences = preferences;
      }),
      setCurrentUser: vi.fn((user: { usage_view_preferences?: UsageViewPreferences }) => {
        state.auth.user = user;
      }),
    },
    tokenDisplay: {
      mode: "m",
      setTokenDisplayMode: vi.fn((mode: "natural" | "k" | "m") => {
        state.tokenDisplay.mode = mode;
      }),
    },
  };
  return state;
});

vi.mock("vue-i18n", async () => {
  const actual = await vi.importActual<typeof import("vue-i18n")>("vue-i18n");
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  };
});

vi.mock("@/api", () => ({
  userAPI: {
    updateProfile: testState.updateProfile,
  },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: () => ({ showError: testState.showError }),
}));

vi.mock("@/stores/auth", () => ({
  useAuthStore: () => testState.auth,
}));

vi.mock("@/composables/useTokenDisplayMode", () => ({
  getPersistedTokenDisplayMode: () => testState.tokenDisplay.mode,
  useTokenDisplayMode: () => ({
    setTokenDisplayMode: testState.tokenDisplay.setTokenDisplayMode,
  }),
}));

const defaultPreferences = (): UsageViewPreferences => ({
  admin: {
    hidden_columns: ["user_agent"],
    token_display_mode: "m",
    table_density: "comfortable",
    stats_card_style: "balanced",
  },
  user: {
    hidden_columns: [],
    token_display_mode: "m",
    table_density: "comfortable",
    stats_card_style: "balanced",
  },
});

function mountHarness(page: "admin" | "user") {
  const state = {} as ReturnType<typeof useUsageViewPreferences>;
  const Harness = defineComponent({
    setup() {
      Object.assign(state, useUsageViewPreferences(page));
      return {
        pagePreferences: state.pagePreferences,
        hiddenColumns: computed(() => [...state.hiddenColumns.value].join(",")),
      };
    },
    template: `
      <div>
        <span data-test="token-mode">{{ pagePreferences.token_display_mode }}</span>
        <span data-test="density">{{ pagePreferences.table_density }}</span>
        <span data-test="style">{{ pagePreferences.stats_card_style }}</span>
        <span data-test="hidden">{{ hiddenColumns }}</span>
      </div>
    `,
  });

  const wrapper = mount(Harness);
  return { wrapper, state };
}

describe("useUsageViewPreferences", () => {
  beforeEach(() => {
    testState.auth.user = { usage_view_preferences: defaultPreferences() };
    testState.auth.setUsageViewPreferences.mockClear();
    testState.auth.setCurrentUser.mockClear();
    testState.updateProfile.mockReset();
    testState.showError.mockClear();
    testState.tokenDisplay.mode = "m";
    testState.tokenDisplay.setTokenDisplayMode.mockClear();
  });

  it("normalizes defaults and keeps admin/user column preferences separate", async () => {
    testState.auth.user = {
      usage_view_preferences: {
        admin: {
          hidden_columns: ["user_agent"],
          token_display_mode: "natural",
          table_density: "comfortable",
          stats_card_style: "balanced",
        },
        user: {
          hidden_columns: ["cache_hit"],
          token_display_mode: "k",
          table_density: "compact",
          stats_card_style: "accent",
        },
      },
    };

    const admin = mountHarness("admin");
    const user = mountHarness("user");

    expect(admin.wrapper.get('[data-test="hidden"]').text()).toBe("user_agent");
    expect(admin.wrapper.get('[data-test="token-mode"]').text()).toBe("natural");
    expect(user.wrapper.get('[data-test="hidden"]').text()).toBe("cache_hit");
    expect(user.wrapper.get('[data-test="token-mode"]').text()).toBe("k");
    expect(user.wrapper.get('[data-test="density"]').text()).toBe("compact");
    expect(user.wrapper.get('[data-test="style"]').text()).toBe("accent");
  });

  it("falls back to the local token display preference before profile preferences exist", () => {
    testState.tokenDisplay.mode = "m";
    testState.auth.user = {};

    const { wrapper } = mountHarness("user");

    expect(wrapper.get('[data-test="token-mode"]').text()).toBe("m");
    expect(wrapper.get('[data-test="hidden"]').text()).toBe("");
  });

  it("maps legacy profile token display values into the new fixed-unit modes", () => {
    testState.auth.user = {
      usage_view_preferences: {
        admin: {
          hidden_columns: [],
          token_display_mode: "full" as any,
          table_density: "comfortable",
          stats_card_style: "balanced",
        },
        user: {
          hidden_columns: [],
          token_display_mode: "compact" as any,
          table_density: "comfortable",
          stats_card_style: "balanced",
        },
      },
    };

    const admin = mountHarness("admin");
    const user = mountHarness("user");

    expect(admin.wrapper.get('[data-test="token-mode"]').text()).toBe("natural");
    expect(user.wrapper.get('[data-test="token-mode"]').text()).toBe("m");
  });

  it("optimistically saves page preferences without changing the other page", async () => {
    const saved = {
      ...defaultPreferences(),
      admin: {
        ...defaultPreferences().admin,
        hidden_columns: ["user_agent", "cache_hit"],
        token_display_mode: "k" as const,
        table_density: "compact" as const,
      },
    };
    testState.updateProfile.mockResolvedValueOnce({ usage_view_preferences: saved });
    const { state } = mountHarness("admin");

    await state.patchPagePreferences({
      hidden_columns: ["user_agent", "cache_hit"],
      token_display_mode: "k",
      table_density: "compact",
    });

    expect(testState.updateProfile).toHaveBeenCalledWith({
      usage_view_preferences: saved,
    });
    expect(testState.auth.setUsageViewPreferences).toHaveBeenCalledWith(saved);
    expect(testState.auth.setCurrentUser).toHaveBeenCalledWith({
      usage_view_preferences: saved,
    });
    expect(testState.tokenDisplay.setTokenDisplayMode).toHaveBeenLastCalledWith("k");
    expect(testState.auth.user?.usage_view_preferences?.user.hidden_columns).toEqual([]);
  });

  it("rolls back and reports an error when saving fails", async () => {
    const previous = defaultPreferences();
    testState.auth.user = { usage_view_preferences: previous };
    testState.updateProfile.mockRejectedValueOnce({
      response: { data: { detail: "save failed" } },
    });
    const { state, wrapper } = mountHarness("user");

    await expect(
      state.patchPagePreferences({
        hidden_columns: ["cache_hit"],
        token_display_mode: "natural",
      }),
    ).rejects.toBeTruthy();
    await nextTick();

    expect(testState.auth.setUsageViewPreferences).toHaveBeenLastCalledWith(previous);
    expect(testState.showError).toHaveBeenCalledWith("save failed");
    expect(wrapper.get('[data-test="hidden"]').text()).toBe("");
    expect(wrapper.get('[data-test="token-mode"]').text()).toBe("m");
  });
});
