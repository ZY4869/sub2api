import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { userAPI } from "@/api";
import { useAppStore } from "@/stores/app";
import { useAuthStore } from "@/stores/auth";
import type {
  AccountGroupDisplayMode,
  AccountTodayStatsWindow,
} from "@/types";
import {
  normalizeAccountGroupDisplayMode,
  normalizeAccountTodayStatsWindows,
} from "@/utils/accountDisplayPreferences";

const updatingState = ref(false);

export function useAccountDisplayPreferences() {
  const { t } = useI18n();
  const appStore = useAppStore();
  const authStore = useAuthStore();

  const accountTodayStatsWindows = computed(() =>
    normalizeAccountTodayStatsWindows(
      authStore.user?.account_today_stats_windows,
    ),
  );
  const accountGroupDisplayMode = computed(() =>
    normalizeAccountGroupDisplayMode(
      authStore.user?.account_group_display_mode,
    ),
  );
  const updatingAccountDisplayPreferences = computed(
    () => updatingState.value,
  );

  const setAccountDisplayPreferences = async (next: {
    todayStatsWindows: AccountTodayStatsWindow[];
    groupDisplayMode: AccountGroupDisplayMode;
  }) => {
    const currentUser = authStore.user;
    if (!currentUser || updatingState.value) {
      return;
    }

    const nextWindows = normalizeAccountTodayStatsWindows(
      next.todayStatsWindows,
    );
    const nextMode = normalizeAccountGroupDisplayMode(next.groupDisplayMode);
    const previousWindows = normalizeAccountTodayStatsWindows(
      currentUser.account_today_stats_windows,
    );
    const previousMode = normalizeAccountGroupDisplayMode(
      currentUser.account_group_display_mode,
    );

    if (
      nextMode === previousMode &&
      nextWindows.join(",") === previousWindows.join(",")
    ) {
      return;
    }

    updatingState.value = true;
    authStore.setAccountTodayStatsWindows(nextWindows);
    authStore.setAccountGroupDisplayMode(nextMode);

    try {
      const updatedUser = await userAPI.updateProfile({
        account_today_stats_windows: nextWindows,
        account_group_display_mode: nextMode,
      });
      authStore.setCurrentUser({
        ...updatedUser,
        account_today_stats_windows: nextWindows,
        account_group_display_mode: nextMode,
      });
    } catch (error: any) {
      authStore.setAccountTodayStatsWindows(previousWindows);
      authStore.setAccountGroupDisplayMode(previousMode);
      appStore.showError(
        error?.response?.data?.detail ||
          error?.message ||
          t("admin.accounts.displayOptimization.saveFailed"),
      );
      throw error;
    } finally {
      updatingState.value = false;
    }
  };

  return {
    accountTodayStatsWindows,
    accountGroupDisplayMode,
    updatingAccountDisplayPreferences,
    setAccountDisplayPreferences,
  };
}

