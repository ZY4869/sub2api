<template>
  <AppLayout v-if="isAuthenticated">
    <PublicModelCatalogContent />
  </AppLayout>

  <div
    v-else
    class="min-h-screen bg-[radial-gradient(circle_at_top_left,_rgba(14,116,144,0.12),_transparent_28%),linear-gradient(180deg,_rgba(248,250,252,1),_rgba(241,245,249,0.98))] dark:bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.12),_transparent_28%),linear-gradient(180deg,_rgba(2,6,23,1),_rgba(15,23,42,0.98))]"
  >
    <header class="border-b border-slate-200/80 bg-white/80 backdrop-blur dark:border-dark-700 dark:bg-dark-950/80">
      <div class="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
        <div class="flex items-center gap-3">
          <div class="h-10 w-10 overflow-hidden rounded-xl shadow-sm">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <div class="text-sm font-semibold text-slate-900 dark:text-white">{{ siteName }}</div>
        </div>

        <div class="flex items-center gap-3">
          <router-link
            to="/login"
            class="rounded-full border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 transition hover:border-slate-300 hover:bg-slate-50 dark:border-dark-700 dark:text-slate-200 dark:hover:bg-dark-800"
          >
            {{ t('auth.signIn') }}
          </router-link>
        </div>
      </div>
    </header>

    <main class="px-4 py-8 md:px-6 md:py-10">
      <PublicModelCatalogContent />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import PublicModelCatalogContent from '@/components/models/PublicModelCatalogContent.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const router = useRouter()
const route = useRoute()

const isAuthenticated = computed(() => authStore.isAuthenticated)
const publicModelCatalogEnabled = computed(() => appStore.publicModelCatalogEnabled)
const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.siteLogo)

watch(
  [isAuthenticated, publicModelCatalogEnabled],
  ([authenticated, enabled]) => {
    if (!authenticated && enabled === false && route.path !== '/login') {
      void router.replace({
        path: '/login',
        query: {
          redirect: route.fullPath || '/models'
        }
      })
    }
  },
  { immediate: true }
)

onMounted(() => {
  if (!appStore.publicSettingsLoaded) {
    void appStore.fetchPublicSettings()
  }
})
</script>
