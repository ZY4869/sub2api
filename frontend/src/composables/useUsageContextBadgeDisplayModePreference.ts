import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { userAPI } from "@/api";
import { useAppStore } from "@/stores/app";
import { useAuthStore } from "@/stores/auth";
import type { UsageContextBadgeDisplayMode } from "@/types";
import { normalizeUsageContextBadgeDisplayMode } from "@/utils/usageModelPresentation";

const updatingState = ref(false);

export function useUsageContextBadgeDisplayModePreference() {
  const { t } = useI18n();
  const appStore = useAppStore();
  const authStore = useAuthStore();

  const usageContextBadgeDisplayMode = computed(() =>
    normalizeUsageContextBadgeDisplayMode(
      authStore.user?.usage_context_badge_display_mode,
    ),
  );
  const updatingUsageContextBadgeDisplayMode = computed(
    () => updatingState.value,
  );

  const setUsageContextBadgeDisplayMode = async (
    mode: UsageContextBadgeDisplayMode,
  ) => {
    const currentUser = authStore.user;
    if (!currentUser || updatingState.value) {
      return;
    }

    const nextMode = normalizeUsageContextBadgeDisplayMode(mode);
    const previousMode = normalizeUsageContextBadgeDisplayMode(
      currentUser.usage_context_badge_display_mode,
    );
    if (nextMode === previousMode) {
      return;
    }

    updatingState.value = true;
    authStore.setUsageContextBadgeDisplayMode(nextMode);

    try {
      const updatedUser = await userAPI.updateProfile({
        usage_context_badge_display_mode: nextMode,
      });
      authStore.setCurrentUser(updatedUser);
    } catch (error: any) {
      authStore.setUsageContextBadgeDisplayMode(previousMode);
      appStore.showError(
        error?.response?.data?.detail ||
          t("usage.contextBadgeDisplayUpdateFailed"),
      );
      throw error;
    } finally {
      updatingState.value = false;
    }
  };

  return {
    usageContextBadgeDisplayMode,
    updatingUsageContextBadgeDisplayMode,
    setUsageContextBadgeDisplayMode,
  };
}
