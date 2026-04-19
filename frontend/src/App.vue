<script setup lang="ts">
import { RouterView, useRouter, useRoute } from 'vue-router'
import { onMounted, onBeforeUnmount, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toast from '@/components/common/Toast.vue'
import NavigationProgress from '@/components/common/NavigationProgress.vue'
import { resolveDocumentTitle } from '@/router/title'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore, useAuthStore, useSubscriptionStore, useAnnouncementStore } from '@/stores'
import { getSetupStatus } from '@/api/setup'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const subscriptionStore = useSubscriptionStore()
const announcementStore = useAnnouncementStore()

/**
 * Update favicon dynamically
 * @param logoUrl - URL of the logo to use as favicon
 */
function updateFavicon(logoUrl: string) {
  // Find existing favicon link or create new one
  let link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.type = logoUrl.endsWith('.svg') ? 'image/svg+xml' : 'image/x-icon'
  link.href = logoUrl
}

// Watch for site settings changes and update favicon/title
watch(
  () => appStore.siteLogo,
  (newLogo) => {
    if (newLogo) {
      updateFavicon(newLogo)
    }
  },
  { immediate: true }
)

// Watch for authentication state and manage subscription data + announcements
function onVisibilityChange() {
  if (document.visibilityState === 'visible' && authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
  }
}

watch(
  () => authStore.isAuthenticated,
  (isAuthenticated, oldValue) => {
    if (isAuthenticated) {
      // User logged in: preload subscriptions and start polling
      subscriptionStore.fetchActiveSubscriptions().catch((error) => {
        console.error('Failed to preload subscriptions:', error)
      })
      subscriptionStore.startPolling()

      // Announcements: new login vs page refresh restore
      if (oldValue === false) {
        // New login: delay 3s then force fetch
        setTimeout(() => announcementStore.fetchAnnouncements(true), 3000)
      } else {
        // Page refresh restore (oldValue was undefined)
        announcementStore.fetchAnnouncements()
      }

      // Register visibility change listener
      document.addEventListener('visibilitychange', onVisibilityChange)
    } else {
      // User logged out: clear data and stop polling
      subscriptionStore.clear()
      announcementStore.reset()
      document.removeEventListener('visibilitychange', onVisibilityChange)
    }
  },
  { immediate: true }
)

// Route change trigger (throttled by store)
router.afterEach(() => {
  if (authStore.isAuthenticated) {
    announcementStore.fetchAnnouncements()
  }
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', onVisibilityChange)
})

onMounted(async () => {
  // Check if setup is needed
  try {
    const status = await getSetupStatus()
    if (status.needs_setup && route.path !== '/setup') {
      router.replace('/setup')
      return
    }
  } catch {
    // If setup endpoint fails, assume normal mode and continue
  }

  // Load public settings into appStore (will be cached for other components)
  await appStore.fetchPublicSettings()

  // Re-resolve document title now that siteName is available
  document.title = resolveDocumentTitle(route.meta.title, appStore.siteName, route.meta.titleKey as string)
})
</script>

<template>
  <NavigationProgress />
  <div
    v-if="appStore.maintenanceModeEnabled"
    class="border-b border-amber-200 bg-amber-50/95 backdrop-blur dark:border-amber-700/40 dark:bg-amber-950/80"
  >
    <div class="mx-auto flex max-w-7xl items-center gap-2 px-4 py-2 text-sm text-amber-900 dark:text-amber-100">
      <Icon
        name="exclamationTriangle"
        size="sm"
        class="shrink-0 text-amber-600 dark:text-amber-300"
      />
      <span class="font-medium">{{ t('auth.maintenanceModeTitle') }}</span>
      <span class="hidden sm:inline">·</span>
      <span class="text-xs sm:text-sm">{{ t('auth.maintenanceModeMessage') }}</span>
    </div>
  </div>
  <RouterView />
  <Toast />
  <AnnouncementPopup />
</template>
