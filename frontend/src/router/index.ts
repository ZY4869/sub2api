/**
 * Vue Router configuration for Sub2API frontend
 * Defines all application routes with lazy loading and navigation guards
 */

import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import { useNavigationLoadingState } from '@/composables/useNavigationLoading'
import { useRoutePrefetch } from '@/composables/useRoutePrefetch'
import { resolveDocumentTitle } from './title'
import i18n from '@/i18n'

/**
 * Route definitions with lazy loading
 */
const routes: RouteRecordRaw[] = [
  // ==================== Setup Routes ====================
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('@/views/setup/SetupWizardView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Setup',
      titleKey: 'setup.title'
    }
  },

  // ==================== Public Routes ====================
  {
    path: '/home',
    name: 'Home',
    component: () => import('@/views/HomeView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Home',
      titleKey: 'ui.routeTitles.home'
    }
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/auth/LoginView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Login',
      titleKey: 'common.login'
    }
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('@/views/auth/RegisterView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Register',
      titleKey: 'auth.createAccount'
    }
  },
  {
    path: '/email-verify',
    name: 'EmailVerify',
    component: () => import('@/views/auth/EmailVerifyView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Verify Email',
      titleKey: 'auth.verifyYourEmail'
    }
  },
  {
    path: '/auth/callback',
    name: 'OAuthCallback',
    component: () => import('@/views/auth/OAuthCallbackView.vue'),
    meta: {
      requiresAuth: false,
      title: 'OAuth Callback',
      titleKey: 'ui.routeTitles.oauthCallback'
    }
  },
  {
    path: '/auth/linuxdo/callback',
    name: 'LinuxDoOAuthCallback',
    component: () => import('@/views/auth/LinuxDoCallbackView.vue'),
    meta: {
      requiresAuth: false,
      title: 'LinuxDo OAuth Callback',
      titleKey: 'ui.routeTitles.linuxDoOAuthCallback'
    }
  },
  {
    path: '/forgot-password',
    name: 'ForgotPassword',
    component: () => import('@/views/auth/ForgotPasswordView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Forgot Password',
      titleKey: 'auth.forgotPasswordTitle'
    }
  },
  {
    path: '/reset-password',
    name: 'ResetPassword',
    component: () => import('@/views/auth/ResetPasswordView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Reset Password',
      titleKey: 'auth.resetPasswordTitle'
    }
  },
  {
    path: '/key-usage',
    name: 'KeyUsage',
    component: () => import('@/views/KeyUsageView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Key Usage',
      titleKey: 'keyUsage.title',
    }
  },
  {
    path: '/models',
    name: 'PublicModels',
    component: () => import('@/views/PublicModelsView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Models',
      titleKey: 'ui.routeTitles.models'
    }
  },

  // ==================== User Routes ====================
  {
    path: '/',
    redirect: '/home'
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('@/views/user/DashboardView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Dashboard',
      titleKey: 'dashboard.title',
      descriptionKey: 'dashboard.welcomeMessage'
    }
  },
  {
    path: '/keys',
    name: 'Keys',
    component: () => import('@/views/user/KeysView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'API Keys',
      titleKey: 'keys.title',
      descriptionKey: 'keys.description'
    }
  },
  {
    path: '/available-channels',
    name: 'AvailableChannels',
    component: () => import('@/views/user/AvailableChannelsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Available Channels',
      titleKey: 'availableChannels.title',
      descriptionKey: 'availableChannels.description'
    }
  },
  {
    path: '/channel-status',
    name: 'ChannelStatus',
    component: () => import('@/views/user/ChannelStatusView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Channel Status',
      titleKey: 'channelStatus.title',
      descriptionKey: 'channelStatus.description'
    }
  },
  {
    path: '/usage',
    name: 'Usage',
    component: () => import('@/views/user/UsageView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Usage Records',
      titleKey: 'usage.title',
      descriptionKey: 'usage.description'
    }
  },
  {
    path: '/redeem',
    name: 'Redeem',
    component: () => import('@/views/user/RedeemView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Redeem Code',
      titleKey: 'redeem.title',
      descriptionKey: 'redeem.description'
    }
  },
  {
    path: '/affiliate',
    name: 'Affiliate',
    component: () => import('@/views/user/AffiliateView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Affiliate',
      titleKey: 'affiliate.title',
      descriptionKey: 'affiliate.description'
    }
  },
  {
    path: '/profile',
    name: 'Profile',
    component: () => import('@/views/user/ProfileView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Profile',
      titleKey: 'profile.title',
      descriptionKey: 'profile.description'
    }
  },
  {
    path: '/api-docs',
    redirect: '/api-docs/common'
  },
  {
    path: '/api-docs/:pageId(common|openai-native|openai|anthropic|gemini|grok|deepseek|antigravity|vertex-batch|document-ai)',
    name: 'ApiDocs',
    component: () => import('@/views/user/ApiDocsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'API Docs',
      titleKey: 'ui.routeTitles.apiDocs'
    }
  },
  {
    path: '/subscriptions',
    name: 'Subscriptions',
    component: () => import('@/views/user/SubscriptionsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'My Subscriptions',
      titleKey: 'userSubscriptions.title',
      descriptionKey: 'userSubscriptions.description'
    }
  },
  {
    path: '/purchase',
    name: 'PurchaseSubscription',
    component: () => import('@/views/user/PurchaseSubscriptionView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Purchase Subscription',
      titleKey: 'purchase.title',
      descriptionKey: 'purchase.description'
    }
  },
  {
    path: '/custom/:id',
    name: 'CustomPage',
    component: () => import('@/views/user/CustomPageView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Custom Page',
      titleKey: 'customPage.title',
    }
  },

  // ==================== Admin Routes ====================
  {
    path: '/admin',
    redirect: '/admin/dashboard'
  },
  {
    path: '/admin/dashboard',
    name: 'AdminDashboard',
    component: () => import('@/views/admin/DashboardView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Admin Dashboard',
      titleKey: 'admin.dashboard.title',
      descriptionKey: 'admin.dashboard.description'
    }
  },
  {
    path: '/admin/models',
    name: 'AdminModels',
    redirect: '/admin/models/all',
    component: () => import('@/views/admin/models/ModelsLayoutView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Model Catalog',
      titleKey: 'admin.models.title',
      descriptionKey: 'admin.models.description'
    },
    children: [
      {
        path: 'billing',
        name: 'AdminModelsBilling',
        redirect: '/admin/billing/pricing',
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Billing Center',
          titleKey: 'admin.models.pages.billing.title',
          descriptionKey: 'admin.models.pages.billing.description'
        }
      },
      {
        path: 'available',
        name: 'AdminModelsAvailable',
        component: () => import('@/views/admin/models/AvailableModelsView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Available Models',
          titleKey: 'admin.models.pages.available.title',
          descriptionKey: 'admin.models.pages.available.description'
        }
      },
      {
        path: 'all',
        name: 'AdminModelsAll',
        component: () => import('@/views/admin/models/AllModelsView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'All Models',
          titleKey: 'admin.models.pages.all.title',
          descriptionKey: 'admin.models.pages.all.description'
        }
      },
      {
        path: 'pricing',
        name: 'AdminModelsPricing',
        redirect: '/admin/billing/pricing',
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Billing Center',
          titleKey: 'admin.models.pages.billing.title',
          descriptionKey: 'admin.models.pages.billing.description'
        }
      },
      {
        path: 'official',
        redirect: '/admin/billing/pricing'
      },
      {
        path: 'sale',
        redirect: '/admin/billing/pricing'
      },
      {
        path: 'relay',
        redirect: '/admin/billing/pricing'
      },
      {
        path: 'registry',
        redirect: '/admin/models/all'
      },
    ]
  },
  {
    path: '/admin/billing',
    name: 'AdminBilling',
    redirect: '/admin/billing/pricing',
    component: () => import('@/views/admin/billing/BillingLayoutView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Billing Center',
      titleKey: 'admin.billing.title',
      descriptionKey: 'admin.billing.description'
    },
    children: [
      {
        path: 'pricing',
        name: 'AdminBillingPricing',
        component: () => import('@/views/admin/billing/BillingPricingView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Billing Pricing',
          titleKey: 'admin.billing.pages.pricing.title',
          descriptionKey: 'admin.billing.pages.pricing.description'
        }
      },
      {
        path: 'public-model-catalog',
        name: 'AdminBillingPublicCatalog',
        component: () => import('@/views/admin/billing/BillingPublicCatalogView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Public Model Catalog',
          titleKey: 'admin.billing.pages.publicCatalog.title',
          descriptionKey: 'admin.billing.pages.publicCatalog.description'
        }
      },
      {
        path: 'rules',
        name: 'AdminBillingRules',
        component: () => import('@/views/admin/billing/BillingRulesView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Billing Rules',
          titleKey: 'admin.billing.pages.rules.title',
          descriptionKey: 'admin.billing.pages.rules.description'
        }
      },
    ]
  },
  {
    path: '/admin/ops',
    name: 'AdminOps',
    component: () => import('@/views/admin/ops/OpsDashboard.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Ops Monitoring',
      titleKey: 'admin.ops.title',
      descriptionKey: 'admin.ops.description'
    }
  },
  {
    path: '/admin/request-details',
    name: 'AdminRequestDetails',
    redirect: (to) => ({
      path: '/admin/usage',
      query: {
        ...to.query,
        tab: 'request_details',
      },
    }),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Request Details',
      titleKey: 'admin.requestDetails.title',
      descriptionKey: 'admin.requestDetails.description'
    }
  },
  {
    path: '/admin/users',
    name: 'AdminUsers',
    component: () => import('@/views/admin/UsersView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'User Management',
      titleKey: 'admin.users.title',
      descriptionKey: 'admin.users.description'
    }
  },
  {
    path: '/admin/affiliates',
    name: 'AdminAffiliates',
    component: () => import('@/views/admin/AffiliateUsersView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Affiliate Users',
      titleKey: 'admin.affiliates.title',
      descriptionKey: 'admin.affiliates.description'
    }
  },
  {
    path: '/admin/groups',
    name: 'AdminGroups',
    component: () => import('@/views/admin/GroupsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Group Management',
      titleKey: 'admin.groups.title',
      descriptionKey: 'admin.groups.description'
    }
  },
  {
    path: '/admin/channels',
    name: 'AdminChannels',
    component: () => import('@/views/admin/ChannelsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Channel Management',
      titleKey: 'admin.channels.title',
      descriptionKey: 'admin.channels.description'
    }
  },
  {
    path: '/admin/channel-monitors',
    name: 'AdminChannelMonitors',
    component: () => import('@/views/admin/ChannelMonitorsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Channel Monitors',
      titleKey: 'admin.channelMonitors.title',
      descriptionKey: 'admin.channelMonitors.description'
    }
  },
  {
    path: '/admin/subscriptions',
    name: 'AdminSubscriptions',
    component: () => import('@/views/admin/SubscriptionsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Subscription Management',
      titleKey: 'admin.subscriptions.title',
      descriptionKey: 'admin.subscriptions.description'
    }
  },
  {
    path: '/admin/accounts',
    component: () => import('@/views/admin/AccountsLayoutView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Account Management',
      titleKey: 'admin.accounts.title',
      descriptionKey: 'admin.accounts.description'
    },
    children: [
      {
        path: '',
        name: 'AdminAccounts',
        component: () => import('@/views/admin/AccountsView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Account Management',
          titleKey: 'admin.accounts.title',
          descriptionKey: 'admin.accounts.description'
        }
      },
      {
        path: 'limited',
        name: 'AdminAccountsLimited',
        component: () => import('@/views/admin/LimitedAccountsView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Limited Accounts',
          titleKey: 'admin.accounts.limited.title',
          descriptionKey: 'admin.accounts.limited.description'
        }
      },
      {
        path: 'blacklist',
        name: 'AdminAccountsBlacklist',
        component: () => import('@/views/admin/BlacklistedAccountsView.vue'),
        meta: {
          requiresAuth: true,
          requiresAdmin: true,
          title: 'Blacklisted Accounts',
          titleKey: 'admin.accounts.blacklist.title',
          descriptionKey: 'admin.accounts.blacklist.description'
        }
      }
    ]
  },
  {
    path: '/admin/api-docs',
    redirect: '/admin/api-docs/common'
  },
  {
    path: '/admin/api-docs/:pageId(common|openai-native|openai|anthropic|gemini|grok|deepseek|antigravity|vertex-batch|document-ai)',
    name: 'AdminApiDocs',
    component: () => import('@/views/admin/AdminApiDocsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Admin API Docs',
      titleKey: 'admin.apiDocs.title',
      descriptionKey: 'admin.apiDocs.description'
    }
  },
  {
    path: '/admin/announcements',
    name: 'AdminAnnouncements',
    component: () => import('@/views/admin/AnnouncementsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Announcements',
      titleKey: 'admin.announcements.title',
      descriptionKey: 'admin.announcements.description'
    }
  },
  {
    path: '/admin/proxies',
    name: 'AdminProxies',
    component: () => import('@/views/admin/ProxiesView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Proxy Management',
      titleKey: 'admin.proxies.title',
      descriptionKey: 'admin.proxies.description'
    }
  },
  {
    path: '/admin/redeem',
    name: 'AdminRedeem',
    component: () => import('@/views/admin/RedeemView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Redeem Code Management',
      titleKey: 'admin.redeem.title',
      descriptionKey: 'admin.redeem.description'
    }
  },
  {
    path: '/admin/promo-codes',
    name: 'AdminPromoCodes',
    component: () => import('@/views/admin/PromoCodesView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Promo Code Management',
      titleKey: 'admin.promo.title',
      descriptionKey: 'admin.promo.description'
    }
  },
  {
    path: '/admin/settings',
    name: 'AdminSettings',
    component: () => import('@/views/admin/SettingsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'System Settings',
      titleKey: 'admin.settings.title',
      descriptionKey: 'admin.settings.description'
    }
  },
  {
    path: '/admin/usage',
    name: 'AdminUsage',
    component: () => import('@/views/admin/UsageView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Usage Records',
      titleKey: 'admin.usage.title',
      descriptionKey: 'admin.usage.description'
    }
  },

  // ==================== 404 Not Found ====================
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFoundView.vue'),
    meta: {
      title: '404 Not Found',
      titleKey: 'ui.routeTitles.notFound'
    }
  }
]

/**
 * Create router instance
 */
const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior(_to, _from, savedPosition) {
    // Scroll to saved position when using browser back/forward
    if (savedPosition) {
      return savedPosition
    }
    // Scroll to top for new routes
    return { top: 0 }
  }
})

/**
 * Navigation guard: Authentication check
 */
let authInitialized = false

// 初始化导航加载状态和预加载
const navigationLoading = useNavigationLoadingState()
// 延迟初始化预加载，传入 router 实例
let routePrefetch: ReturnType<typeof useRoutePrefetch> | null = null
const BACKEND_MODE_ALLOWED_PATHS = ['/login', '/key-usage', '/setup', '/models']
const LOGIN_ENTRY_PATHS = ['/login', '/register']

function isAllowedPath(path: string, allowedPaths: string[]) {
  return allowedPaths.some((allowedPath) => path === allowedPath || path.startsWith(`${allowedPath}/`))
}

function showMaintenanceRedirectNotice(appStore: ReturnType<typeof useAppStore>) {
  appStore.showWarning(i18n.global.t('auth.maintenanceModeMessage'))
}

router.beforeEach((to, _from, next) => {
  // 开始导航加载状态
  navigationLoading.startNavigation()

  const authStore = useAuthStore()

  // Restore auth state from localStorage on first navigation (page refresh)
  if (!authInitialized) {
    authStore.checkAuth()
    authInitialized = true
  }

  // Set page title
  const appStore = useAppStore()
  // For custom pages, use menu item label as document title
  if (to.name === 'CustomPage') {
    const id = to.params.id as string
    const publicItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
    const adminSettingsStore = useAdminSettingsStore()
    const menuItem = publicItems.find((item) => item.id === id)
      ?? (authStore.isAdmin ? adminSettingsStore.customMenuItems.find((item) => item.id === id) : undefined)
    if (menuItem?.label) {
      const siteName = appStore.siteName || 'Sub2API'
      document.title = `${menuItem.label} - ${siteName}`
    } else {
      document.title = resolveDocumentTitle(to.meta.title, appStore.siteName, to.meta.titleKey as string)
    }
  } else {
    document.title = resolveDocumentTitle(to.meta.title, appStore.siteName, to.meta.titleKey as string)
  }

  // Check if route requires authentication
  const requiresAuth = to.meta.requiresAuth !== false // Default to true
  const requiresAdmin = to.meta.requiresAdmin === true

  // If route doesn't require auth, allow access
  if (!requiresAuth) {
    // If already authenticated and trying to access login/register, redirect to appropriate dashboard
    if (authStore.isAuthenticated && LOGIN_ENTRY_PATHS.includes(to.path)) {
      if (appStore.maintenanceModeEnabled && !authStore.isAdmin) {
        next()
        return
      }
      if (appStore.backendModeEnabled) {
        if (authStore.isAdmin) {
          next('/admin/dashboard')
          return
        }
        next()
        return
      }
      // Admin users go to admin dashboard, regular users go to user dashboard
      next(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
      return
    }
    // Backend mode: block public pages for unauthenticated users (except login, key-usage, setup)
    if (!appStore.maintenanceModeEnabled && appStore.backendModeEnabled && !authStore.isAuthenticated) {
      const isAllowed = isAllowedPath(to.path, BACKEND_MODE_ALLOWED_PATHS)
      if (!isAllowed) {
        next('/login')
        return
      }
    }
    next()
    return
  }

  // Route requires authentication
  if (!authStore.isAuthenticated) {
    if (appStore.maintenanceModeEnabled) {
      showMaintenanceRedirectNotice(appStore)
    }
    // Not authenticated, redirect to login
    next({
      path: '/login',
      query: { redirect: to.fullPath } // Save intended destination
    })
    return
  }

  if (appStore.maintenanceModeEnabled && !authStore.isAdmin) {
    showMaintenanceRedirectNotice(appStore)
    next('/login')
    return
  }

  // Check admin requirement
  if (requiresAdmin && !authStore.isAdmin) {
    // User is authenticated but not admin, redirect to user dashboard
    next('/dashboard')
    return
  }

  // 简易模式下限制访问某些页面
  if (authStore.isSimpleMode) {
    const restrictedPaths = [
      '/admin/groups',
      '/admin/channels',
      '/admin/subscriptions',
      '/admin/redeem',
      '/subscriptions',
      '/redeem'
    ]

    if (restrictedPaths.some((path) => to.path.startsWith(path))) {
      // 简易模式下访问受限页面,重定向到仪表板
      next(authStore.isAdmin ? '/admin/dashboard' : '/dashboard')
      return
    }
  }

  // Backend mode: admin gets full access, non-admin blocked
  if (appStore.backendModeEnabled) {
    if (authStore.isAuthenticated && authStore.isAdmin) {
      next()
      return
    }
    const isAllowed = isAllowedPath(to.path, BACKEND_MODE_ALLOWED_PATHS)
    if (!isAllowed) {
      next('/login')
      return
    }
  }

  // All checks passed, allow navigation
  next()
})

/**
 * Navigation guard: End loading and trigger prefetch
 */
router.afterEach((to) => {
  // 结束导航加载状态
  navigationLoading.endNavigation()

  // 懒初始化预加载（首次导航时创建，传入 router 实例）
  if (!routePrefetch) {
    routePrefetch = useRoutePrefetch(router)
  }
  // 触发路由预加载（在浏览器空闲时执行）
  routePrefetch.triggerPrefetch(to)
})

/**
 * Navigation guard: Error handling
 * Handles dynamic import failures caused by deployment updates
 */
router.onError((error) => {
  console.error('Router error:', error)

  // Check if this is a dynamic import failure (chunk loading error)
  const isChunkLoadError =
    error.message?.includes('Failed to fetch dynamically imported module') ||
    error.message?.includes('Loading chunk') ||
    error.message?.includes('Loading CSS chunk') ||
    error.name === 'ChunkLoadError'

  if (isChunkLoadError) {
    // Avoid infinite reload loop by checking sessionStorage
    const reloadKey = 'chunk_reload_attempted'
    const lastReload = sessionStorage.getItem(reloadKey)
    const now = Date.now()

    // Allow reload if never attempted or more than 10 seconds ago
    if (!lastReload || now - parseInt(lastReload) > 10000) {
      sessionStorage.setItem(reloadKey, now.toString())
      console.warn('Chunk load error detected, reloading page to fetch latest version...')
      window.location.reload()
    } else {
      console.error('Chunk load error persists after reload. Please clear browser cache.')
    }
  }
})

export default router
