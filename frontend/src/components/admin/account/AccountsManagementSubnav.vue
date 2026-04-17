<template>
  <nav class="flex flex-wrap items-center gap-2 rounded-2xl border border-gray-200 bg-white p-2 shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <RouterLink
      v-for="item in items"
      :key="item.to"
      :to="item.to"
      class="rounded-xl px-4 py-2 text-sm font-medium transition-colors"
      :class="route.path === item.to ? 'bg-primary-600 text-white shadow-sm' : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white'"
    >
      {{ item.label }}
    </RouterLink>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const { t } = useI18n()

const items = computed(() => [
  { to: '/admin/accounts', label: t('admin.accounts.subnav.all') },
  { to: '/admin/accounts/limited', label: t('admin.accounts.subnav.limited') },
  { to: '/admin/accounts/blacklist', label: t('admin.accounts.subnav.blacklist') }
])
</script>
