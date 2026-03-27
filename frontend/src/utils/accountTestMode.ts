export type AccountTestMode = 'real_forward' | 'health_check'

export const DEFAULT_ACCOUNT_TEST_MODE: AccountTestMode = 'real_forward'

const ACCOUNT_TEST_MODE_STORAGE_KEY = 'sub2api.admin.accounts.test_mode'

export function normalizeAccountTestMode(value: unknown): AccountTestMode {
  return value === 'health_check' ? 'health_check' : DEFAULT_ACCOUNT_TEST_MODE
}

export function loadAccountTestModePreference(): AccountTestMode {
  if (typeof window === 'undefined') {
    return DEFAULT_ACCOUNT_TEST_MODE
  }
  try {
    return normalizeAccountTestMode(window.localStorage.getItem(ACCOUNT_TEST_MODE_STORAGE_KEY))
  } catch {
    return DEFAULT_ACCOUNT_TEST_MODE
  }
}

export function saveAccountTestModePreference(mode: AccountTestMode) {
  if (typeof window === 'undefined') {
    return
  }
  try {
    window.localStorage.setItem(ACCOUNT_TEST_MODE_STORAGE_KEY, normalizeAccountTestMode(mode))
  } catch {
    // Ignore storage failures and keep the in-memory selection.
  }
}
