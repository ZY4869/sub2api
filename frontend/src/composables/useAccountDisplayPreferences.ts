import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { userAPI } from "@/api";
import { useAppStore } from "@/stores/app";
import { useAuthStore } from "@/stores/auth";
import type {
  AccountGroupDisplayMode,
  AccountStatusDisplayMode,
  AccountTodayStatsCycleMode,
  AccountTodayStatsWindow,
} from "@/types";
import {
  normalizeAccountGroupDisplayMode,
  normalizeAccountStatusDisplayMode,
  normalizeAccountTodayStatsCycleMode,
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
  const accountTodayStatsCycleMode = computed(() =>
    normalizeAccountTodayStatsCycleMode(
      authStore.user?.account_today_stats_cycle_mode,
    ),
  );
  const accountGroupDisplayMode = computed(() =>
    normalizeAccountGroupDisplayMode(
      authStore.user?.account_group_display_mode,
    ),
  );
  const accountStatusDisplayMode = computed(() =>
    normalizeAccountStatusDisplayMode(
      authStore.user?.account_status_display_mode,
    ),
  );
  const updatingAccountDisplayPreferences = computed(
    () => updatingState.value,
  );

  const setAccountDisplayPreferences = async (next: {
    todayStatsWindows: AccountTodayStatsWindow[];
    todayStatsCycleMode: AccountTodayStatsCycleMode;
    groupDisplayMode: AccountGroupDisplayMode;
    statusDisplayMode: AccountStatusDisplayMode;
  }) => {
    const currentUser = authStore.user;
    if (!currentUser || updatingState.value) {
      return;
    }

    const nextWindows = normalizeAccountTodayStatsWindows(
      next.todayStatsWindows,
    );
    const nextCycleMode = normalizeAccountTodayStatsCycleMode(
      next.todayStatsCycleMode,
    );
    const nextMode = normalizeAccountGroupDisplayMode(next.groupDisplayMode);
    const nextStatusMode = normalizeAccountStatusDisplayMode(
      next.statusDisplayMode,
    );
    const previousWindows = normalizeAccountTodayStatsWindows(
      currentUser.account_today_stats_windows,
    );
    const previousCycleMode = normalizeAccountTodayStatsCycleMode(
      currentUser.account_today_stats_cycle_mode,
    );
    const previousMode = normalizeAccountGroupDisplayMode(
      currentUser.account_group_display_mode,
    );
    const previousStatusMode = normalizeAccountStatusDisplayMode(
      currentUser.account_status_display_mode,
    );

    if (
      nextMode === previousMode &&
      nextStatusMode === previousStatusMode &&
      nextCycleMode === previousCycleMode &&
      nextWindows.join(",") === previousWindows.join(",")
    ) {
      return;
    }

    updatingState.value = true;
    authStore.setAccountTodayStatsWindows(nextWindows);
    authStore.setAccountTodayStatsCycleMode(nextCycleMode);
    authStore.setAccountGroupDisplayMode(nextMode);
    authStore.setAccountStatusDisplayMode(nextStatusMode);

    try {
      const updatedUser = await userAPI.updateProfile({
        account_today_stats_windows: nextWindows,
        account_today_stats_cycle_mode: nextCycleMode,
        account_group_display_mode: nextMode,
        account_status_display_mode: nextStatusMode,
      });
      authStore.setCurrentUser({
        ...updatedUser,
        account_today_stats_windows: nextWindows,
        account_today_stats_cycle_mode: nextCycleMode,
        account_group_display_mode: nextMode,
        account_status_display_mode: nextStatusMode,
      });
    } catch (error: any) {
      authStore.setAccountTodayStatsWindows(previousWindows);
      authStore.setAccountTodayStatsCycleMode(previousCycleMode);
      authStore.setAccountGroupDisplayMode(previousMode);
      authStore.setAccountStatusDisplayMode(previousStatusMode);
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
    accountTodayStatsCycleMode,
    accountGroupDisplayMode,
    accountStatusDisplayMode,
    updatingAccountDisplayPreferences,
    setAccountDisplayPreferences,
  };
}
