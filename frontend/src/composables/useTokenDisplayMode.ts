import { computed, ref } from "vue";
import type { TokenDisplayMode } from "@/types";
import { formatTokenCount } from "@/utils/format";

const TOKEN_DISPLAY_MODE_STORAGE_KEY = "token-display-mode";
const DEFAULT_TOKEN_DISPLAY_MODE: TokenDisplayMode = "m";

function normalizeTokenDisplayMode(
  value: string | null | undefined,
): TokenDisplayMode {
  if (value === "natural" || value === "k" || value === "m") {
    return value;
  }
  if (value === "full") return "natural";
  if (value === "compact") return "m";
  return DEFAULT_TOKEN_DISPLAY_MODE;
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
    () => tokenDisplayModeState.value !== "natural",
  );

  const setTokenDisplayMode = (mode: TokenDisplayMode) => {
    tokenDisplayModeState.value = normalizeTokenDisplayMode(mode);
    setPersistedTokenDisplayMode(tokenDisplayModeState.value);
  };

  const toggleTokenDisplayMode = () => {
    setTokenDisplayMode(tokenDisplayModeState.value === "natural" ? "m" : "natural");
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
