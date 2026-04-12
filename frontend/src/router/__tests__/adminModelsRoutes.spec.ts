import { describe, expect, it, vi } from 'vitest'

vi.mock('@/composables/useNavigationLoading', () => ({
  useNavigationLoadingState: () => ({
    startNavigation: vi.fn(),
    endNavigation: vi.fn(),
    isLoading: { value: false }
  })
}))

vi.mock('@/composables/useRoutePrefetch', () => ({
  useRoutePrefetch: () => ({
    triggerPrefetch: vi.fn(),
    cancelPendingPrefetch: vi.fn(),
    resetPrefetchState: vi.fn()
  })
}))

import router from '@/router'

describe('admin model routes', () => {
  it('keeps the billing center as the default admin models entry', () => {
    const adminModels = router.getRoutes().find((route) => route.name === 'AdminModels')
    expect(adminModels?.redirect).toBe('/admin/models/billing')
  })

  it('redirects the legacy pricing route to the billing center', () => {
    const legacyPricing = router.getRoutes().find((route) => route.name === 'AdminModelsPricing')
    expect(legacyPricing?.redirect).toBe('/admin/models/billing')
  })
})
