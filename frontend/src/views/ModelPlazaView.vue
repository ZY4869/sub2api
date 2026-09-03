<template>
  <AppLayout v-if="isAuthenticated">
    <ModelPlazaContent />
  </AppLayout>

  <div v-else class="min-h-screen overflow-x-hidden bg-gray-50 dark:bg-dark-950">
    <header class="border-b border-slate-200/80 bg-white/80 backdrop-blur dark:border-dark-700 dark:bg-dark-950/80">
      <div class="mx-auto flex max-w-7xl items-center justify-between gap-3 px-3 py-3 sm:px-6 sm:py-4">
        <router-link to="/home" class="flex min-w-0 flex-1 items-center gap-3">
          <div class="h-10 w-10 shrink-0 overflow-hidden rounded-xl shadow-sm">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <div class="truncate text-sm font-semibold text-slate-900 dark:text-white">{{ siteName }}</div>
        </router-link>

        <div class="flex shrink-0 flex-wrap items-center justify-end gap-2 sm:gap-3">
          <router-link
            to="/models"
            class="rounded-lg border border-slate-200 px-3 py-2 text-sm font-medium text-slate-700 transition hover:border-slate-300 hover:bg-slate-50 dark:border-dark-700 dark:text-slate-200 dark:hover:bg-dark-800 sm:px-4"
          >
            {{ t('nav.modelsCatalog') }}
          </router-link>
          <router-link
            to="/login"
            class="rounded-lg border border-primary-200 bg-primary-50 px-3 py-2 text-sm font-medium text-primary-700 transition hover:border-primary-300 dark:border-primary-500/30 dark:bg-primary-500/10 dark:text-primary-200 sm:px-4"
          >
            {{ t('auth.signIn') }}
          </router-link>
        </div>
      </div>
    </header>

    <main class="px-4 py-8 md:px-6 md:py-10">
      <ModelPlazaContent />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ModelPlazaContent from '@/components/models/ModelPlazaContent.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()

const isAuthenticated = computed(() => authStore.isAuthenticated)
const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => appStore.siteLogo)

onMounted(() => {
  if (!appStore.publicSettingsLoaded) {
    void appStore.fetchPublicSettings()
  }
})
</script>
