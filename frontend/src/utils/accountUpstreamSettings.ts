export type AccountUpstreamSettingsMode = 'create' | 'edit'

export const ACCOUNT_UPSTREAM_BASE_URL_PLACEHOLDER = 'https://cloudcode-pa.googleapis.com'
export const ACCOUNT_UPSTREAM_API_KEY_PLACEHOLDER = 'sk-...'

export function resolveAccountUpstreamApiKeyHintKey(mode: AccountUpstreamSettingsMode): string {
  return mode === 'edit'
    ? 'admin.accounts.leaveEmptyToKeep'
    : 'admin.accounts.upstream.apiKeyHint'
}
