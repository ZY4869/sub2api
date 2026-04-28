<template>
  <section class="space-y-4" data-testid="module-section">
    <button
      type="button"
      class="flex w-full items-center justify-between gap-4 rounded-lg border border-gray-200 bg-white px-4 py-3 text-left transition hover:border-primary-200 hover:bg-primary-50/40 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 dark:border-dark-700 dark:bg-dark-800 dark:hover:border-primary-700 dark:hover:bg-primary-900/10 dark:focus:ring-offset-dark-950"
      :aria-expanded="!collapsed"
      :aria-controls="`module-section-${section.id}`"
      :data-testid="`module-section-toggle-${section.id}`"
      @click="$emit('toggle')"
    >
      <span class="min-w-0">
        <span class="block text-lg font-semibold text-gray-900 dark:text-white">
          {{ t(section.titleKey) }}
        </span>
        <span class="mt-1 block text-sm text-gray-500 dark:text-gray-400">
          {{ t(section.descriptionKey) }}
        </span>
      </span>
      <Icon
        name="chevronDown"
        size="md"
        :class="[
          'flex-shrink-0 text-gray-400 transition-transform dark:text-gray-500',
          collapsed && '-rotate-90'
        ]"
      />
    </button>

    <div
      v-show="!collapsed"
      :id="`module-section-${section.id}`"
      class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3"
      :data-testid="`module-section-content-${section.id}`"
    >
      <ModuleCard v-for="card in section.cards" :key="card.id" :card="card" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import ModuleCard from './ModuleCard.vue'
import type { AdminModuleSection } from './moduleCatalog'

defineProps<{
  section: AdminModuleSection
  collapsed: boolean
}>()

defineEmits<{
  toggle: []
}>()

const { t } = useI18n()
</script>
