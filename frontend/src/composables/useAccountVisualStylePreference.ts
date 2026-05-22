import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { userAPI } from "@/api";
import { useAppStore } from "@/stores/app";
import { useAuthStore } from "@/stores/auth";
import type { VisualPreset, VisualPresetPreference } from "@/types";
import {
  normalizeVisualPresetPreference,
  resolveVisualPreset,
} from "@/utils/visualPreset";

const updatingState = ref(false);

export function useAccountVisualStylePreference() {
  const { t } = useI18n();
  const appStore = useAppStore();
  const authStore = useAuthStore();

  const accountVisualPresetOverride = computed<VisualPresetPreference>(() =>
    normalizeVisualPresetPreference(
      authStore.user?.account_visual_preset_override,
    ),
  );
  const resolvedAccountVisualPreset = computed<VisualPreset>(() =>
    resolveVisualPreset(
      appStore.visualPresetDefault,
      authStore.user?.visual_preset_preference,
      authStore.user?.account_visual_preset_override,
    ),
  );
  const updatingAccountVisualStyle = computed(() => updatingState.value);

  const setAccountVisualPresetOverride = async (
    preference: VisualPresetPreference,
  ) => {
    const currentUser = authStore.user;
    if (!currentUser || updatingState.value) {
      return;
    }

    const nextPreference = normalizeVisualPresetPreference(preference);
    const previousPreference = normalizeVisualPresetPreference(
      currentUser.account_visual_preset_override,
    );
    if (nextPreference === previousPreference) {
      return;
    }

    updatingState.value = true;
    authStore.setAccountVisualPresetOverride(nextPreference);

    try {
      const updatedUser = await userAPI.updateProfile({
        account_visual_preset_override: nextPreference,
      });
      authStore.setCurrentUser(updatedUser);
    } catch (error: any) {
      authStore.setAccountVisualPresetOverride(previousPreference);
      appStore.showError(
        error?.response?.data?.detail ||
          error?.message ||
          t("admin.accounts.accountVisualStyleUpdateFailed"),
      );
      throw error;
    } finally {
      updatingState.value = false;
    }
  };

  return {
    accountVisualPresetOverride,
    resolvedAccountVisualPreset,
    updatingAccountVisualStyle,
    setAccountVisualPresetOverride,
  };
}
