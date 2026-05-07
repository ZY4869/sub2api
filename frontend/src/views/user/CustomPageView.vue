<template>
  <AppLayout>
    <div class="custom-page-layout">
      <div class="card flex-1 min-h-0 overflow-hidden">
        <div v-if="loading" class="flex h-full items-center justify-center py-12">
          <div
            class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
          ></div>
        </div>

        <div
          v-else-if="!menuItem"
          class="flex h-full items-center justify-center p-10 text-center"
        >
          <div class="max-w-md">
            <div
              class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <Icon name="link" size="lg" class="text-gray-400" />
            </div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('customPage.notFoundTitle') }}
            </h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ t('customPage.notFoundDesc') }}
            </p>
          </div>
        </div>

        <div
          v-else-if="isMarkdownMode && pageLoading"
          class="flex h-full items-center justify-center py-12"
        >
          <div
            class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
          ></div>
        </div>

        <div
          v-else-if="isMarkdownMode && pageError"
          class="flex h-full items-center justify-center p-10 text-center"
        >
          <div class="max-w-md">
            <div
              class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <Icon name="document" size="lg" class="text-gray-400" />
            </div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('customPage.loadFailedTitle') }}
            </h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ pageError || t('customPage.loadFailedDesc') }}
            </p>
          </div>
        </div>

        <div
          v-else-if="isMarkdownMode && pageContent"
          class="h-full overflow-y-auto px-4 py-5 md:px-6"
        >
          <CustomMarkdownPageContent
            :markdown="pageContent"
            :title="menuItem?.label || t('customPage.title')"
            :toc-title="t('customPage.pageToc')"
          />
        </div>

        <div
          v-else-if="!isValidUrl"
          class="flex h-full items-center justify-center p-10 text-center"
        >
          <div class="max-w-md">
            <div
              class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <Icon name="link" size="lg" class="text-gray-400" />
            </div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('customPage.notConfiguredTitle') }}
            </h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ t('customPage.notConfiguredDesc') }}
            </p>
          </div>
        </div>

        <div v-else class="custom-embed-shell">
          <a
            :href="embeddedUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="btn btn-secondary btn-sm custom-open-fab"
          >
            <Icon name="externalLink" size="sm" class="mr-1.5" :stroke-width="2" />
            {{ t('customPage.openInNewTab') }}
          </a>
          <iframe
            :src="embeddedUrl"
            class="custom-embed-frame"
            allowfullscreen
          ></iframe>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { pagesAPI } from '@/api'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import AppLayout from '@/components/layout/AppLayout.vue'
import CustomMarkdownPageContent from '@/components/custom/CustomMarkdownPageContent.vue'
import Icon from '@/components/icons/Icon.vue'
import { buildEmbeddedUrl, detectTheme } from '@/utils/embedded-url'
import { sanitizeUrl } from '@/utils/url'

const { t, locale } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const adminSettingsStore = useAdminSettingsStore()

const loading = ref(false)
const pageLoading = ref(false)
const pageError = ref('')
const pageContent = ref('')
const pageTheme = ref<'light' | 'dark'>('light')
let themeObserver: MutationObserver | null = null

const menuItemId = computed(() => String(route.params.id || '').trim())

const menuItem = computed(() => {
  const id = menuItemId.value
  if (!id) {
    return null
  }
  const publicItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
  const found = publicItems.find((item) => item.id === id) ?? null
  if (found) {
    return found
  }
  if (authStore.isAdmin) {
    return adminSettingsStore.customMenuItems.find((item) => item.id === id) ?? null
  }
  return null
})

const embeddedUrl = computed(() => {
  if (!menuItem.value) return ''
  return buildEmbeddedUrl(
    menuItem.value.url,
    authStore.user?.id,
    authStore.token,
    pageTheme.value,
    locale.value,
  )
})

const isMarkdownMode = computed(() =>
  (menuItem.value?.page_mode || 'iframe') === 'markdown' && !!menuItem.value?.page_slug,
)

const isValidUrl = computed(() => {
  const url = sanitizeUrl(embeddedUrl.value)
  return url.startsWith('http://') || url.startsWith('https://')
})

async function ensureSettingsReady() {
  const tasks: Promise<unknown>[] = []

  if (!appStore.publicSettingsLoaded) {
    tasks.push(appStore.fetchPublicSettings())
  }

  if (authStore.isAdmin && !adminSettingsStore.loaded) {
    tasks.push(adminSettingsStore.fetch())
  }

  if (tasks.length === 0) {
    return
  }

  await Promise.all(tasks)
}

async function loadMarkdownPage() {
  if (!isMarkdownMode.value || !menuItem.value?.page_slug) {
    pageContent.value = ''
    pageError.value = ''
    pageLoading.value = false
    return
  }

  if (authStore.isAdmin && menuItem.value.page_content) {
    pageContent.value = menuItem.value.page_content
    pageError.value = ''
    pageLoading.value = false
    return
  }

  pageLoading.value = true
  pageError.value = ''
  try {
    const page = await pagesAPI.getCustomPage(menuItem.value.page_slug)
    pageContent.value = page.content || ''
  } catch (error: any) {
    pageContent.value = ''
    pageError.value = error?.message || t('customPage.loadFailedDesc')
  } finally {
    pageLoading.value = false
  }
}

async function initializePage() {
  loading.value = true
  try {
    await ensureSettingsReady()
    await loadMarkdownPage()
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  pageTheme.value = detectTheme()

  if (typeof document !== 'undefined') {
    themeObserver = new MutationObserver(() => {
      pageTheme.value = detectTheme()
    })
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })
  }

  await initializePage()
})

watch(
  () => [menuItemId.value, menuItem.value?.page_slug, menuItem.value?.page_content, menuItem.value?.page_mode] as const,
  async () => {
    if (loading.value) {
      return
    }
    await loadMarkdownPage()
  },
)

watch(
  () => authStore.isAdmin,
  async (isAdmin, wasAdmin) => {
    if (isAdmin && !wasAdmin && !adminSettingsStore.loaded) {
      await adminSettingsStore.fetch()
      await loadMarkdownPage()
    }
  },
)

onUnmounted(() => {
  if (themeObserver) {
    themeObserver.disconnect()
    themeObserver = null
  }
})
</script>

<style scoped>
.custom-page-layout {
  @apply flex flex-col;
  height: calc(100vh - 64px - 4rem);
}

.custom-embed-shell {
  @apply relative;
  @apply h-full w-full overflow-hidden rounded-2xl;
  @apply bg-gradient-to-b from-gray-50 to-white dark:from-dark-900 dark:to-dark-950;
  @apply p-0;
}

.custom-open-fab {
  @apply absolute right-3 top-3 z-10;
  @apply shadow-sm backdrop-blur supports-[backdrop-filter]:bg-white/80 dark:supports-[backdrop-filter]:bg-dark-800/80;
}

.custom-embed-frame {
  display: block;
  margin: 0;
  width: 100%;
  height: 100%;
  border: 0;
  border-radius: 0;
  box-shadow: none;
  background: transparent;
}
</style>
