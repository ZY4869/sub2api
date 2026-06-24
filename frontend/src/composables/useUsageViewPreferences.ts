import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { userAPI } from "@/api";
import { useAppStore } from "@/stores/app";
import { useAuthStore } from "@/stores/auth";
import {
  getPersistedTokenDisplayMode,
  useTokenDisplayMode,
} from "@/composables/useTokenDisplayMode";
import type {
  UsageViewPage,
  UsageViewPagePreferences,
  UsageViewPreferences,
} from "@/types";

const updatingState = ref(false);

const DEFAULT_ADMIN_HIDDEN_COLUMNS = ["user_agent"];

const defaultPagePreferences = (page: UsageViewPage): UsageViewPagePreferences => ({
  hidden_columns: page === "admin" ? [...DEFAULT_ADMIN_HIDDEN_COLUMNS] : [],
  token_display_mode: "full",
  table_density: "comfortable",
  stats_card_style: "balanced",
});

function normalizePagePreferences(
  page: UsageViewPage,
  input?: Partial<UsageViewPagePreferences>,
): UsageViewPagePreferences {
  const defaults = defaultPagePreferences(page);
  return {
    hidden_columns: Array.isArray(input?.hidden_columns)
      ? [...new Set(input.hidden_columns.filter((key) => typeof key === "string" && key.trim()).map((key) => key.trim()))]
      : defaults.hidden_columns,
    token_display_mode: input?.token_display_mode === "compact" || input?.token_display_mode === "full"
      ? input.token_display_mode
      : defaults.token_display_mode,
    table_density: input?.table_density === "compact" || input?.table_density === "comfortable"
      ? input.table_density
      : defaults.table_density,
    stats_card_style: input?.stats_card_style === "accent" || input?.stats_card_style === "balanced"
      ? input.stats_card_style
      : defaults.stats_card_style,
  };
}

function normalizePreferences(input?: Partial<UsageViewPreferences>): UsageViewPreferences {
  return {
    admin: normalizePagePreferences("admin", input?.admin),
    user: normalizePagePreferences("user", input?.user),
  };
}

export function useUsageViewPreferences(page: UsageViewPage) {
  const { t } = useI18n();
  const appStore = useAppStore();
  const authStore = useAuthStore();
  const { setTokenDisplayMode } = useTokenDisplayMode();

  const preferences = computed(() => {
    const normalized = normalizePreferences(authStore.user?.usage_view_preferences);
    if (!authStore.user?.usage_view_preferences) {
      normalized[page].token_display_mode = getPersistedTokenDisplayMode();
    }
    return normalized;
  });

  const pagePreferences = computed(() => preferences.value[page]);
  const hiddenColumns = computed(() => new Set(pagePreferences.value.hidden_columns));
  const updatingUsageViewPreferences = computed(() => updatingState.value);

  watch(
    () => pagePreferences.value.token_display_mode,
    (mode) => {
      setTokenDisplayMode(mode);
    },
    { immediate: true },
  );

  const setPagePreferences = async (nextPagePreferences: UsageViewPagePreferences) => {
    const currentUser = authStore.user;
    if (!currentUser || updatingState.value) {
      return;
    }

    const previous = normalizePreferences(currentUser.usage_view_preferences);
    const next = normalizePreferences({
      ...previous,
      [page]: normalizePagePreferences(page, nextPagePreferences),
    });

    updatingState.value = true;
    authStore.setUsageViewPreferences(next);
    setTokenDisplayMode(next[page].token_display_mode);

    try {
      const updatedUser = await userAPI.updateProfile({
        usage_view_preferences: next,
      });
      authStore.setCurrentUser(updatedUser);
      setTokenDisplayMode(next[page].token_display_mode);
    } catch (error: any) {
      authStore.setUsageViewPreferences(previous);
      setTokenDisplayMode(previous[page].token_display_mode);
      appStore.showError(
        error?.response?.data?.detail || t("usage.displaySettingsUpdateFailed"),
      );
      throw error;
    } finally {
      updatingState.value = false;
    }
  };

  const patchPagePreferences = async (patch: Partial<UsageViewPagePreferences>) => {
    await setPagePreferences({
      ...pagePreferences.value,
      ...patch,
    });
  };

  const setHiddenColumns = async (columns: string[]) => {
    await patchPagePreferences({ hidden_columns: columns });
  };

  const toggleColumn = async (key: string) => {
    const next = new Set(pagePreferences.value.hidden_columns);
    if (next.has(key)) {
      next.delete(key);
    } else {
      next.add(key);
    }
    await setHiddenColumns([...next]);
  };

  return {
    preferences,
    pagePreferences,
    hiddenColumns,
    updatingUsageViewPreferences,
    setPagePreferences,
    patchPagePreferences,
    setHiddenColumns,
    toggleColumn,
  };
}
