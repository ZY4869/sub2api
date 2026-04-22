<template>
  <div class="group relative inline-flex">
    <span
      :class="[
        'inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-xs font-medium',
        toneClass,
      ]"
    >
      <ModelIcon
        v-if="model"
        :model="model"
        :display-name="modelDisplayName || label"
        size="12px"
      />
      <Icon v-else name="exclamationTriangle" size="xs" :stroke-width="2" />
      <span>{{ label }}</span>
      <span v-if="countdown" class="text-[10px] opacity-70">{{ countdown }}</span>
    </span>
    <div
      v-if="tooltip"
      class="pointer-events-none absolute bottom-full left-1/2 z-50 mb-2 w-56 -translate-x-1/2 whitespace-normal rounded bg-gray-900 px-3 py-2 text-center text-xs leading-relaxed text-white opacity-0 transition-opacity group-hover:opacity-100 dark:bg-gray-700"
    >
      {{ tooltip }}
      <div
        class="absolute left-1/2 top-full -translate-x-1/2 border-4 border-transparent border-t-gray-900 dark:border-t-gray-700"
      ></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(
  defineProps<{
    tone: 'purple' | 'amber' | 'red'
    label: string
    countdown?: string | null
    tooltip?: string | null
    model?: string | null
    modelDisplayName?: string | null
  }>(),
  {
    countdown: null,
    tooltip: null,
    model: null,
    modelDisplayName: null,
  },
)

const toneClass = computed(() => {
  switch (props.tone) {
    case 'amber':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
    case 'red':
      return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
    default:
      return 'bg-purple-100 text-purple-700 dark:bg-purple-900/30 dark:text-purple-400'
  }
})
</script>
