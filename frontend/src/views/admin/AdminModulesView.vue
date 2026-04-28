<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <header class="space-y-2">
        <p class="text-sm font-medium text-primary-600 dark:text-primary-400">
          {{ t('admin.modules.eyebrow') }}
        </p>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('admin.modules.title') }}
        </h1>
        <p class="max-w-3xl text-sm leading-6 text-gray-500 dark:text-gray-400">
          {{ t('admin.modules.description') }}
        </p>
      </header>

      <div class="space-y-8">
        <ModuleSection
          v-for="section in visibleSections"
          :key="section.id"
          :section="section"
          :collapsed="collapsedSections[section.id] === true"
          @toggle="toggleSection(section.id)"
        />
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import AppLayout from '@/components/layout/AppLayout.vue'
import ModuleSection from '@/components/admin/modules/ModuleSection.vue'
import { adminModuleSections } from '@/components/admin/modules/moduleCatalog'

const { t } = useI18n()
const authStore = useAuthStore()
const collapsedSections = reactive<Record<string, boolean>>({})

const visibleSections = computed(() =>
  adminModuleSections
    .map((section) => ({
      ...section,
      cards: section.cards.filter((card) => !authStore.isSimpleMode || !card.hideInSimpleMode)
    }))
    .filter((section) => section.cards.length > 0)
)

function toggleSection(sectionId: string) {
  collapsedSections[sectionId] = collapsedSections[sectionId] !== true
}
</script>
