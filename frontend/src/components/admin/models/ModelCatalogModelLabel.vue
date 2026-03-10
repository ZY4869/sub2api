<template>
  <button
    type="button"
    class="group flex w-full items-center gap-3 rounded-xl bg-white/80 px-3 py-2 text-left transition hover:bg-primary-50 dark:bg-dark-900/60 dark:hover:bg-primary-500/10"
    @click="handleCopy"
  >
    <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gray-100 dark:bg-dark-800">
      <img v-if="iconUrl" :src="iconUrl" :alt="displayText" class="h-6 w-6 object-contain" />
      <span v-else class="text-sm font-semibold text-gray-500 dark:text-gray-400">{{ displayText.slice(0, 1) }}</span>
    </span>
    <span class="min-w-0 flex-1">
      <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">{{ displayText }}</span>
      <span class="block truncate text-xs text-gray-500 transition group-hover:text-primary-600 dark:text-gray-400 dark:group-hover:text-primary-300">
        {{ model }}
      </span>
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'
import { resolveModelCatalogDisplayName, resolveModelCatalogIcon } from '@/utils/modelCatalogPresentation'

const props = defineProps<{
  model: string
  displayName?: string
  iconKey?: string
}>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const displayText = computed(() => resolveModelCatalogDisplayName(props.model, props.displayName))
const iconUrl = computed(() => resolveModelCatalogIcon(props.iconKey))

async function handleCopy() {
  await copyToClipboard(props.model, t('admin.models.copyModelIdSuccess', { model: props.model }))
}
</script>
