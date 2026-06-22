import { reactive, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import { userAPI } from '@/api'
import type { AccountDaily5HTriggerSettings, AccountDaily5HTriggerSettingsView } from '@/types'

export function useAccountsDaily5HTrigger(ctx: any) {
  const {
    appStore,
    authStore,
    daily5HTriggerSettingsLoading,
    daily5HTriggerSettingsSaving,
    showDaily5HTriggerSettings,
    t
  } = ctx

const createDefaultDaily5HTriggerSettings =
  (): AccountDaily5HTriggerSettings => ({
    enabled: false,
    selected_account_types: ["chatgpt_oauth"],
    include_paused_accounts: false,
    ignore_free_accounts: false,
    skip_cn_holidays_and_weekends: false,
    openai_model_mode: { mode: "auto", fixed_model_id: "" },
    anthropic_model_mode: { mode: "auto", fixed_model_id: "" },
    gemini_model_mode: { mode: "auto", fixed_model_id: "" },
  });
const daily5HTriggerSettingsView = reactive<AccountDaily5HTriggerSettingsView>({
  settings: createDefaultDaily5HTriggerSettings(),
  candidates: [],
});

const applyDaily5HTriggerSettingsView = (
  value?: AccountDaily5HTriggerSettingsView | null,
) => {
  daily5HTriggerSettingsView.settings =
    value?.settings || createDefaultDaily5HTriggerSettings();
  daily5HTriggerSettingsView.candidates = Array.isArray(value?.candidates)
    ? value.candidates
    : [];
};

const loadDaily5HTriggerSettings = async () => {
  daily5HTriggerSettingsLoading.value = true;
  try {
    const view = await adminAPI.accounts.getDaily5HTriggerSettings();
    applyDaily5HTriggerSettingsView(view);
  } catch (error: any) {
    console.error("Failed to load daily 5H trigger settings:", error);
    appStore.showError(
      error?.message || t("admin.accounts.daily5h.loadFailed"),
    );
  } finally {
    daily5HTriggerSettingsLoading.value = false;
  }
};

const updateDaily5HTriggerSettings = async (
  settings: AccountDaily5HTriggerSettings,
) => {
  const view = await adminAPI.accounts.updateDaily5HTriggerSettings(settings);
  applyDaily5HTriggerSettingsView(view);
};

const handleToggleDaily5HTrigger = async () => {
  if (daily5HTriggerSettingsLoading.value || daily5HTriggerSettingsSaving.value) {
    return;
  }
  daily5HTriggerSettingsSaving.value = true;
  try {
    await updateDaily5HTriggerSettings({
      ...daily5HTriggerSettingsView.settings,
      enabled: !daily5HTriggerSettingsView.settings.enabled,
    });
    appStore.showSuccess(t("admin.accounts.daily5h.updateSuccess"));
  } catch (error: any) {
    console.error("Failed to toggle daily 5H trigger settings:", error);
    appStore.showError(
      error?.message || t("admin.accounts.daily5h.updateFailed"),
    );
  } finally {
    daily5HTriggerSettingsSaving.value = false;
  }
};

const handleSaveDaily5HTriggerSettings = async (
  settings: AccountDaily5HTriggerSettings,
) => {
  daily5HTriggerSettingsSaving.value = true;
  try {
    await updateDaily5HTriggerSettings(settings);
    showDaily5HTriggerSettings.value = false;
    appStore.showSuccess(t("admin.accounts.daily5h.updateSuccess"));
  } catch (error: any) {
    console.error("Failed to save daily 5H trigger settings:", error);
    appStore.showError(
      error?.message || t("admin.accounts.daily5h.updateFailed"),
    );
  } finally {
    daily5HTriggerSettingsSaving.value = false;
  }
};

const handleOpenDaily5HTriggerSettings = async () => {
  showDaily5HTriggerSettings.value = true;
  if (!daily5HTriggerSettingsLoading.value) {
    await loadDaily5HTriggerSettings();
  }
};

const accountRealtimeCountdownUpdating = ref(false);

const handleToggleAccountRealtimeCountdown = async () => {
  const currentUser = authStore.user;
  if (!currentUser || accountRealtimeCountdownUpdating.value) {
    return;
  }

  const previousValue = currentUser.account_realtime_countdown_enabled !== false;
  const nextValue = !previousValue;
  accountRealtimeCountdownUpdating.value = true;
  authStore.setAccountRealtimeCountdownEnabled(nextValue);

  try {
    const updatedUser = await userAPI.updateProfile({
      account_realtime_countdown_enabled: nextValue,
    });
    authStore.setCurrentUser(updatedUser);
  } catch (error: any) {
    authStore.setAccountRealtimeCountdownEnabled(previousValue);
    appStore.showError(
      error?.message || t("admin.accounts.accountRealtimeCountdownUpdateFailed"),
    );
  } finally {
    accountRealtimeCountdownUpdating.value = false;
  }
};


  return {
    createDefaultDaily5HTriggerSettings,
    daily5HTriggerSettingsView,
    applyDaily5HTriggerSettingsView,
    loadDaily5HTriggerSettings,
    updateDaily5HTriggerSettings,
    handleToggleDaily5HTrigger,
    handleSaveDaily5HTriggerSettings,
    handleOpenDaily5HTriggerSettings,
    accountRealtimeCountdownUpdating,
    handleToggleAccountRealtimeCountdown
  }
}
