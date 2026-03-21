<template>
  <button
    type="button"
    class="group flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left transition hover:bg-primary-50 dark:hover:bg-primary-500/10"
    @click="handleCopy"
  >
    <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gray-100 dark:bg-dark-800">
      <ModelIcon
        :model="model"
        :provider="provider"
        :icon-key="iconKey"
        :display-name="displayText"
        size="20px"
      />
    </span>
    <span class="min-w-0 flex-1">
      <span class="block truncate text-sm font-semibold text-gray-900 dark:text-white">{{ displayText }}</span>
      <span class="flex items-center gap-2 text-xs text-gray-500 transition group-hover:text-primary-600 dark:text-gray-400 dark:group-hover:text-primary-300">
        <span class="truncate">{{ model }}</span>
        <span
          v-if="showTierBadge"
          class="inline-flex shrink-0 rounded-full bg-violet-100 px-1.5 py-0.5 text-[10px] font-semibold text-violet-700 dark:bg-violet-500/15 dark:text-violet-300"
        >
          {{ tierBadgeLabel }}
        </span>
      </span>
    </span>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useClipboard } from '@/composables/useClipboard'
import ModelIcon from '@/components/common/ModelIcon.vue'
import { resolveModelCatalogDisplayName } from '@/utils/modelCatalogPresentation'

const props = withDefaults(defineProps<{
  model: string
  displayName?: string
  iconKey?: string
  provider?: string
  platforms?: string[]
  showTierBadge?: boolean
  tierBadgeLabel?: string
}>(), {
  showTierBadge: false,
  tierBadgeLabel: ''
})

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const displayText = computed(() => resolveModelCatalogDisplayName(props.model, props.displayName))

async function handleCopy() {
  await copyToClipboard(props.model, t('admin.models.copyModelIdSuccess', { model: props.model }))
}
</script>
