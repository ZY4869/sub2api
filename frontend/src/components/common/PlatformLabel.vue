<template>
  <div class="flex min-w-0 items-center gap-2 text-left" :title="title || undefined">
    <PlatformIcon v-if="platform" :platform="platform" :size="size" class="shrink-0" />
    <div class="min-w-0">
      <div class="truncate font-medium text-gray-900 dark:text-white">
        {{ label }}
      </div>
      <div
        v-if="description"
        class="truncate text-xs text-gray-500 dark:text-gray-400"
      >
        {{ description }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AccountPlatform, GroupPlatform } from '@/types'
import PlatformIcon from './PlatformIcon.vue'

const props = withDefaults(defineProps<{
  label: string
  description?: string
  platform?: AccountPlatform | GroupPlatform
  size?: 'xs' | 'sm' | 'md' | 'lg'
}>(), {
  description: '',
  size: 'sm'
})

const title = computed(() =>
  props.description ? `${props.label} ${props.description}` : props.label
)
</script>
