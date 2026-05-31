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
  it('does not register standalone user or admin api docs routes', () => {
    const routes = router.getRoutes()
    expect(routes.find((route) => route.path === '/api-docs')).toBeUndefined()
    expect(routes.find((route) => route.path === '/admin/api-docs')).toBeUndefined()

    const userRoute = router.resolve('/api-docs/openai-native')
    const adminRoute = router.resolve('/admin/api-docs/openai-native')

    expect(userRoute.name).not.toBe('ApiDocs')
    expect(adminRoute.name).not.toBe('AdminApiDocs')
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
