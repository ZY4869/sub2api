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
  it('keeps all models as the default admin models entry', () => {
    const adminModels = router.getRoutes().find((route) => route.name === 'AdminModels')
    expect(adminModels?.redirect).toBe('/admin/models/all')
  })

  it('redirects the legacy pricing route to the billing center', () => {
    const legacyPricing = router.getRoutes().find((route) => route.name === 'AdminModelsPricing')
    expect(legacyPricing?.redirect).toBe('/admin/billing/pricing')
  })

  it('redirects the other legacy billing-only routes to the billing center', () => {
    const legacyPaths = ['/admin/models/billing', '/admin/models/official', '/admin/models/sale', '/admin/models/relay']
    for (const legacyPath of legacyPaths) {
      const route = router.getRoutes().find((item) => item.path === legacyPath)
      expect(route?.redirect).toBe('/admin/billing/pricing')
    }
  })

  it('registers the standalone admin billing route group', () => {
    const adminBilling = router.getRoutes().find((route) => route.name === 'AdminBilling')
    expect(adminBilling?.redirect).toBe('/admin/billing/pricing')
  })

  it('registers billing pricing and billing rules child routes', () => {
    const billingPricing = router.getRoutes().find((route) => route.name === 'AdminBillingPricing')
    const billingRules = router.getRoutes().find((route) => route.name === 'AdminBillingRules')

    expect(billingPricing?.path).toBe('/admin/billing/pricing')
    expect(billingRules?.path).toBe('/admin/billing/rules')
  })
})
