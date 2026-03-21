<template>
  <div class="flex flex-wrap items-center gap-1.5">
    <span
      v-for="item in displayPlatforms"
      :key="item.key"
      class="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200"
      :title="item.key"
    >
      <ModelPlatformIcon :platform="item.key" size="sm" />
      <span class="truncate">{{ item.label }}</span>
    </span>
    <span v-if="displayPlatforms.length === 0" class="text-sm text-gray-400 dark:text-gray-500">-</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import { formatModelCatalogProvider } from '@/utils/modelCatalogPresentation'
import { normalizeLobeIconKey } from '@/utils/lobeIconResolver'

const props = withDefaults(defineProps<{
  platforms?: string[]
}>(), {
  platforms: () => []
})

const displayPlatforms = computed(() => {
  const values = (props.platforms || [])
    .map((value) => normalizeLobeIconKey(value))
    .filter((value) => value.length > 0)
  const unique = [...new Set(values)]
  const items = unique.map((key) => ({ key, label: formatModelCatalogProvider(key) }))
  return items.sort((left, right) => left.label.localeCompare(right.label))
})
</script>
