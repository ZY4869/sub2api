<template>
  <RouterLink
    :to="card.to"
    class="group flex h-full min-h-36 flex-col rounded-lg border border-gray-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-primary-200 hover:shadow-md focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 dark:border-dark-700 dark:bg-dark-800 dark:hover:border-primary-700 dark:focus:ring-offset-dark-950"
    :data-testid="`module-card-${card.id}`"
  >
    <div class="flex items-start justify-between gap-3">
      <span :class="['rounded-lg p-2.5', accentClass.icon]">
        <Icon :name="card.icon" size="lg" />
      </span>
      <Icon
        name="arrowRight"
        size="sm"
        class="mt-2 text-gray-400 transition group-hover:translate-x-0.5 group-hover:text-primary-600 dark:text-gray-500 dark:group-hover:text-primary-400"
      />
    </div>
    <div class="mt-4 flex-1">
      <h3 class="text-base font-semibold text-gray-900 dark:text-white">
        {{ t(card.titleKey) }}
      </h3>
      <p class="mt-2 text-sm leading-6 text-gray-500 dark:text-gray-400">
        {{ t(card.descriptionKey) }}
      </p>
    </div>
    <span :class="['mt-4 h-1 w-12 rounded-full transition group-hover:w-16', accentClass.line]"></span>
  </RouterLink>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { AdminModuleAccent, AdminModuleCard } from './moduleCatalog'

const props = defineProps<{
  card: AdminModuleCard
}>()

const { t } = useI18n()

const accentClasses: Record<AdminModuleAccent, { icon: string; line: string }> = {
  emerald: {
    icon: 'bg-emerald-50 text-emerald-600 dark:bg-emerald-900/30 dark:text-emerald-300',
    line: 'bg-emerald-500'
  },
  sky: {
    icon: 'bg-sky-50 text-sky-600 dark:bg-sky-900/30 dark:text-sky-300',
    line: 'bg-sky-500'
  },
  violet: {
    icon: 'bg-violet-50 text-violet-600 dark:bg-violet-900/30 dark:text-violet-300',
    line: 'bg-violet-500'
  },
  amber: {
    icon: 'bg-amber-50 text-amber-600 dark:bg-amber-900/30 dark:text-amber-300',
    line: 'bg-amber-500'
  },
  rose: {
    icon: 'bg-rose-50 text-rose-600 dark:bg-rose-900/30 dark:text-rose-300',
    line: 'bg-rose-500'
  },
  cyan: {
    icon: 'bg-cyan-50 text-cyan-600 dark:bg-cyan-900/30 dark:text-cyan-300',
    line: 'bg-cyan-500'
  }
}

const accentClass = computed(() => accentClasses[props.card.accent])
</script>
