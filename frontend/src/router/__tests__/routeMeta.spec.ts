import { describe, expect, it } from 'vitest'
import { vi } from 'vitest'

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false },
  }),
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn(),
  }),
}))

import router from '@/router'

describe('router route meta titles', () => {
  it('provides title keys for public utility routes that used to fall back to english', () => {
    const titleKeys = new Map(
      router.getRoutes().map((route) => [String(route.name || ''), String(route.meta.titleKey || '')])
    )

    expect(titleKeys.get('Setup')).toBe('setup.title')
    expect(titleKeys.get('Home')).toBe('ui.routeTitles.home')
    expect(titleKeys.get('PublicModels')).toBe('ui.routeTitles.models')
    expect(titleKeys.get('EmailVerify')).toBe('auth.verifyYourEmail')
    expect(titleKeys.get('OAuthCallback')).toBe('ui.routeTitles.oauthCallback')
    expect(titleKeys.get('LinuxDoOAuthCallback')).toBe('ui.routeTitles.linuxDoOAuthCallback')
    expect(titleKeys.get('ResetPassword')).toBe('auth.resetPasswordTitle')
    expect(titleKeys.get('KeyUsage')).toBe('keyUsage.title')
    expect(titleKeys.get('NotFound')).toBe('ui.routeTitles.notFound')
  })
})
