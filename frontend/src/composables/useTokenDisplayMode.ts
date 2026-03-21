import { computed, ref } from "vue";
import type { TokenDisplayMode } from "@/types";
import { formatTokenCount } from "@/utils/format";

const TOKEN_DISPLAY_MODE_STORAGE_KEY = "token-display-mode";
const DEFAULT_TOKEN_DISPLAY_MODE: TokenDisplayMode = "full";

function normalizeTokenDisplayMode(
  value: string | null | undefined,
): TokenDisplayMode {
  return value === "compact" ? "compact" : DEFAULT_TOKEN_DISPLAY_MODE;
}

export function getPersistedTokenDisplayMode(): TokenDisplayMode {
  try {
    return normalizeTokenDisplayMode(
      localStorage.getItem(TOKEN_DISPLAY_MODE_STORAGE_KEY),
    );
  } catch {
    return DEFAULT_TOKEN_DISPLAY_MODE;
  }
}

export function setPersistedTokenDisplayMode(mode: TokenDisplayMode): void {
  try {
    localStorage.setItem(TOKEN_DISPLAY_MODE_STORAGE_KEY, mode);
  } catch {
    // Ignore storage failures and keep in-memory mode.
  }
}

const tokenDisplayModeState = ref<TokenDisplayMode>(
  getPersistedTokenDisplayMode(),
);

export function useTokenDisplayMode() {
  const tokenDisplayMode = computed(() => tokenDisplayModeState.value);
  const isCompactTokenDisplay = computed(
    () => tokenDisplayModeState.value === "compact",
  );

  const setTokenDisplayMode = (mode: TokenDisplayMode) => {
    tokenDisplayModeState.value = normalizeTokenDisplayMode(mode);
    setPersistedTokenDisplayMode(tokenDisplayModeState.value);
  };

  const toggleTokenDisplayMode = () => {
    setTokenDisplayMode(
      tokenDisplayModeState.value === "compact" ? "full" : "compact",
    );
  };

  const formatTokenDisplay = (
    value: number | null | undefined,
    options?: { allowBillions?: boolean },
  ) =>
    formatTokenCount(value, {
      mode: tokenDisplayModeState.value,
      allowBillions: options?.allowBillions,
    });

  return {
    tokenDisplayMode,
    isCompactTokenDisplay,
    setTokenDisplayMode,
    toggleTokenDisplayMode,
    formatTokenDisplay,
  };
}
