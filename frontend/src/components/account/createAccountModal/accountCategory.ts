export type AccountCategory = 'oauth-based' | 'apikey' | 'vertex_ai' | 'sso'

export type GeminiAccountCategory = Exclude<AccountCategory, 'sso'>
