<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <div class="text-sm text-gray-600 dark:text-gray-300">
        {{ t('common.total') }}: {{ models.length }}
        <span class="mx-2 text-gray-300 dark:text-dark-600">|</span>
        {{ t('admin.models.registry.availableStatus') }}: {{ availableCount }}
      </div>
    </div>

    <div class="divide-y divide-gray-100 dark:divide-dark-700">
      <div
        v-for="model in models"
        :key="model.id"
        class="flex flex-col gap-3 py-3 sm:flex-row sm:items-start sm:justify-between"
      >
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <p class="font-medium text-gray-900 dark:text-white">{{ model.id }}</p>
            <p v-if="model.display_name" class="text-xs text-gray-500 dark:text-gray-400">
              {{ model.display_name }}
            </p>
            <span
              class="inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium"
              :class="model.available ? availableBadgeClass : unavailableBadgeClass"
            >
              {{ model.available ? t('admin.models.registry.availableStatus') : t('admin.models.registry.unavailableStatus') }}
            </span>
          </div>
          <div class="mt-2">
            <ModelPlatformsInline :platforms="model.platforms" />
          </div>
        </div>

        <div class="flex shrink-0 items-center gap-2">
          <button
            v-if="!model.available"
            type="button"
            class="btn btn-primary btn-sm"
            :disabled="isActivating(model.id)"
            @click="emit('activate', model.id)"
          >
            {{ t('admin.models.registry.actions.activate') }}
          </button>
          <button
            v-else
            type="button"
            class="btn btn-secondary btn-sm"
            disabled
          >
            {{ t('admin.models.registry.availableStatus') }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="models.length === 0" class="py-8">
      <EmptyState :title="t('admin.models.registry.emptyTitle')" :description="t('admin.models.registry.emptyDescription')" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelRegistryDetail } from '@/api/admin/modelRegistry'
import EmptyState from '@/components/common/EmptyState.vue'
import ModelPlatformsInline from '@/components/common/ModelPlatformsInline.vue'

const props = defineProps<{
  models: ModelRegistryDetail[]
  isActivating: (modelId: string) => boolean
}>()

const emit = defineEmits<{
  (e: 'activate', modelId: string): void
}>()

const { t } = useI18n()

const availableCount = computed(() => props.models.filter((m) => m.available).length)
const availableBadgeClass = 'bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300'
const unavailableBadgeClass = 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'
</script>

