import type { AuthInputMethod } from '@/composables/useAccountOAuth'

export interface OAuthFlowExposed {
  authCode: string
  oauthState: string
  projectId: string
  sessionKey: string
  refreshToken: string
  inputMethod: AuthInputMethod
  reset: () => void
}
