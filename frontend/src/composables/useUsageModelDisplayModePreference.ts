import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { userAPI } from "@/api";
import { useAppStore } from "@/stores/app";
import { useAuthStore } from "@/stores/auth";
import type { UsageModelDisplayMode } from "@/types";
import { normalizeUsageModelDisplayMode } from "@/utils/usageModelPresentation";

const updatingState = ref(false);

export function useUsageModelDisplayModePreference() {
  const { t } = useI18n();
  const appStore = useAppStore();
  const authStore = useAuthStore();

  const usageModelDisplayMode = computed(() =>
    normalizeUsageModelDisplayMode(authStore.user?.usage_model_display_mode),
  );
  const updatingUsageModelDisplayMode = computed(() => updatingState.value);

  const setUsageModelDisplayMode = async (mode: UsageModelDisplayMode) => {
    const currentUser = authStore.user;
    if (!currentUser || updatingState.value) {
      return;
    }

    const nextMode = normalizeUsageModelDisplayMode(mode);
    const previousMode = normalizeUsageModelDisplayMode(
      currentUser.usage_model_display_mode,
    );
    if (nextMode === previousMode) {
      return;
    }

    updatingState.value = true;
    authStore.setUsageModelDisplayMode(nextMode);

    try {
      const updatedUser = await userAPI.updateProfile({
        usage_model_display_mode: nextMode,
      });
      authStore.setCurrentUser(updatedUser);
    } catch (error: any) {
      authStore.setUsageModelDisplayMode(previousMode);
      appStore.showError(
        error?.response?.data?.detail || t("usage.modelDisplayUpdateFailed"),
      );
      throw error;
    } finally {
      updatingState.value = false;
    }
  };

  return {
    usageModelDisplayMode,
    updatingUsageModelDisplayMode,
    setUsageModelDisplayMode,
  };
}
