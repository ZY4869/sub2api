import { describe, expect, it, vi } from 'vitest'

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

describe('api docs routes', () => {
  it('redirects the user and admin docs roots to the common protocol page', () => {
    const userRoot = router.getRoutes().find((route) => route.path === '/api-docs')
    const adminRoot = router.getRoutes().find((route) => route.path === '/admin/api-docs')

    expect(userRoot?.redirect).toBe('/api-docs/common')
    expect(adminRoot?.redirect).toBe('/admin/api-docs/common')
  })

  it('registers authenticated user and admin api docs protocol routes', () => {
    const userRoute = router.resolve('/api-docs/openai-native')
    const adminRoute = router.resolve('/admin/api-docs/openai-native')

    expect(userRoute.name).toBe('ApiDocs')
    expect(userRoute.params.pageId).toBe('openai-native')
    expect(userRoute.meta.requiresAuth).toBe(true)
    expect(userRoute.meta.requiresAdmin).toBe(false)
    expect(userRoute.meta.titleKey).toBe('ui.routeTitles.apiDocs')

    expect(adminRoute.name).toBe('AdminApiDocs')
    expect(adminRoute.params.pageId).toBe('openai-native')
    expect(adminRoute.meta.requiresAuth).toBe(true)
    expect(adminRoute.meta.requiresAdmin).toBe(true)
    expect(adminRoute.meta.titleKey).toBe('admin.apiDocs.title')
  })

  it('keeps limited and blacklist pages nested under the accounts parent layout', () => {
    const limitedRoute = router.resolve('/admin/accounts/limited')
    const blacklistRoute = router.resolve('/admin/accounts/blacklist')

    expect(limitedRoute.name).toBe('AdminAccountsLimited')
    expect(blacklistRoute.name).toBe('AdminAccountsBlacklist')

    expect(limitedRoute.matched[0]?.path).toBe('/admin/accounts')
    expect(blacklistRoute.matched[0]?.path).toBe('/admin/accounts')
    expect(limitedRoute.matched).toHaveLength(2)
    expect(blacklistRoute.matched).toHaveLength(2)
  })
})
